package kvs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	cfkvstypes "github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
)

// KVSClient abstracts the CloudFront KeyValueStore API.
type KVSClient interface {
	DescribeKeyValueStore(ctx context.Context, params *cloudfrontkeyvaluestore.DescribeKeyValueStoreInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.DescribeKeyValueStoreOutput, error)
	ListKeys(ctx context.Context, params *cloudfrontkeyvaluestore.ListKeysInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.ListKeysOutput, error)
	UpdateKeys(ctx context.Context, params *cloudfrontkeyvaluestore.UpdateKeysInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.UpdateKeysOutput, error)
}

// ComputeSyncPlan compares desired state against existing KVS state.
// existingKeys maps key -> value for all current KVS entries.
func ComputeSyncPlan(desired *Data, existingKeys map[string]string) *SyncPlan {
	plan := &SyncPlan{}

	desiredMap := make(map[string]string, len(desired.Entries))
	for _, e := range desired.Entries {
		desiredMap[e.Key] = e.Value
	}

	// Find puts: keys that are new or have changed values
	for _, e := range desired.Entries {
		existing, ok := existingKeys[e.Key]
		if !ok || existing != e.Value {
			plan.Puts = append(plan.Puts, e)
		}
	}

	// Find deletes: keys in existing that aren't in desired
	for key := range existingKeys {
		if _, ok := desiredMap[key]; !ok {
			plan.Deletes = append(plan.Deletes, key)
		}
	}

	return plan
}

// FetchExistingKeys retrieves all current keys and values from a KVS.
func FetchExistingKeys(ctx context.Context, client KVSClient, kvsARN string) (map[string]string, string, error) {
	// Get ETag
	desc, err := client.DescribeKeyValueStore(ctx, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: &kvsARN,
	})
	if err != nil {
		return nil, "", fmt.Errorf("describing KVS: %w", err)
	}
	etag := *desc.ETag

	existing := make(map[string]string)
	var nextToken *string
	for {
		resp, err := client.ListKeys(ctx, &cloudfrontkeyvaluestore.ListKeysInput{
			KvsARN:    &kvsARN,
			NextToken: nextToken,
		})
		if err != nil {
			return nil, "", fmt.Errorf("listing KVS keys: %w", err)
		}
		for _, item := range resp.Items {
			existing[*item.Key] = *item.Value
		}
		nextToken = resp.NextToken
		if nextToken == nil {
			break
		}
	}

	return existing, etag, nil
}

// Sync applies a SyncPlan to a CloudFront KVS using the batch UpdateKeys API.
func Sync(ctx context.Context, client KVSClient, kvsARN string, etag string, plan *SyncPlan) error {
	if len(plan.Puts) == 0 && len(plan.Deletes) == 0 {
		return nil
	}

	var puts []cfkvstypes.PutKeyRequestListItem
	for _, e := range plan.Puts {
		key := e.Key
		value := e.Value
		puts = append(puts, cfkvstypes.PutKeyRequestListItem{
			Key:   &key,
			Value: &value,
		})
	}

	var deletes []cfkvstypes.DeleteKeyRequestListItem
	for _, key := range plan.Deletes {
		k := key
		deletes = append(deletes, cfkvstypes.DeleteKeyRequestListItem{
			Key: &k,
		})
	}

	_, err := client.UpdateKeys(ctx, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  &kvsARN,
		IfMatch: &etag,
		Puts:    puts,
		Deletes: deletes,
	})
	if err != nil {
		return fmt.Errorf("updating KVS keys: %w", err)
	}

	return nil
}
