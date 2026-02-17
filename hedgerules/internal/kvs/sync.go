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

// maxKeysPerBatch is the AWS CloudFront KVS limit for UpdateKeys API.
// See: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cloudfront-limits.html
const maxKeysPerBatch = 50

// Sync applies a SyncPlan to a CloudFront KVS using the batch UpdateKeys API.
// Large operations are automatically split into multiple batches to respect AWS limits.
func Sync(ctx context.Context, client KVSClient, kvsARN string, etag string, plan *SyncPlan) error {
	if len(plan.Puts) == 0 && len(plan.Deletes) == 0 {
		return nil
	}

	// Convert entries to API types
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

	// Process operations in batches
	currentETag := etag
	putsIdx := 0
	deletesIdx := 0

	for putsIdx < len(puts) || deletesIdx < len(deletes) {
		// Determine batch size for this iteration
		remainingPuts := len(puts) - putsIdx
		remainingDeletes := len(deletes) - deletesIdx
		remaining := remainingPuts + remainingDeletes

		batchSize := min(maxKeysPerBatch, remaining)

		// Allocate batch operations
		batchPuts := make([]cfkvstypes.PutKeyRequestListItem, 0, batchSize)
		batchDeletes := make([]cfkvstypes.DeleteKeyRequestListItem, 0, batchSize)

		// Fill batch with puts first, then deletes
		for len(batchPuts)+len(batchDeletes) < batchSize && putsIdx < len(puts) {
			batchPuts = append(batchPuts, puts[putsIdx])
			putsIdx++
		}
		for len(batchPuts)+len(batchDeletes) < batchSize && deletesIdx < len(deletes) {
			batchDeletes = append(batchDeletes, deletes[deletesIdx])
			deletesIdx++
		}

		// Execute batch
		resp, err := client.UpdateKeys(ctx, &cloudfrontkeyvaluestore.UpdateKeysInput{
			KvsARN:  &kvsARN,
			IfMatch: &currentETag,
			Puts:    batchPuts,
			Deletes: batchDeletes,
		})
		if err != nil {
			return fmt.Errorf("updating KVS keys (batch %d/%d puts, %d deletes): %w",
				putsIdx, len(puts), deletesIdx, err)
		}

		// Update ETag for next batch
		if resp.ETag != nil {
			currentETag = *resp.ETag
		}
	}

	return nil
}
