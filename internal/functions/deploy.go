package functions

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

// CFClient abstracts the CloudFront Functions API.
type CFClient interface {
	DescribeFunction(ctx context.Context, params *cloudfront.DescribeFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.DescribeFunctionOutput, error)
	CreateFunction(ctx context.Context, params *cloudfront.CreateFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.CreateFunctionOutput, error)
	UpdateFunction(ctx context.Context, params *cloudfront.UpdateFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateFunctionOutput, error)
	PublishFunction(ctx context.Context, params *cloudfront.PublishFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.PublishFunctionOutput, error)
}

// KVSARNResolver abstracts CloudFront KVS ARN resolution.
type KVSARNResolver interface {
	ListKeyValueStores(ctx context.Context, params *cloudfront.ListKeyValueStoresInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListKeyValueStoresOutput, error)
}

// ResolveKVSARN resolves a KVS name to its ARN by listing all KVS and matching by name.
func ResolveKVSARN(ctx context.Context, client KVSARNResolver, kvsName string) (string, error) {
	var marker *string
	for {
		resp, err := client.ListKeyValueStores(ctx, &cloudfront.ListKeyValueStoresInput{
			Marker: marker,
		})
		if err != nil {
			return "", fmt.Errorf("listing key value stores: %w", err)
		}
		if resp.KeyValueStoreList != nil {
			for _, item := range resp.KeyValueStoreList.Items {
				if item.Name != nil && *item.Name == kvsName {
					if item.ARN != nil {
						return *item.ARN, nil
					}
				}
			}
			marker = resp.KeyValueStoreList.NextMarker
		}
		if marker == nil {
			break
		}
	}
	return "", fmt.Errorf("key value store not found: %s", kvsName)
}

// DeployFunction creates or updates a CloudFront Function with the given
// JS source code and KVS association. It publishes the function to LIVE stage.
func DeployFunction(ctx context.Context, client CFClient, name string, code []byte, kvsARN string) error {
	runtime := cftypes.FunctionRuntimeCloudfrontJs20
	kvAssoc := &cftypes.KeyValueStoreAssociations{
		Quantity: int32Ptr(1),
		Items: []cftypes.KeyValueStoreAssociation{
			{KeyValueStoreARN: &kvsARN},
		},
	}

	// Check if function already exists
	var etag string
	descResp, err := client.DescribeFunction(ctx, &cloudfront.DescribeFunctionInput{
		Name:  &name,
		Stage: cftypes.FunctionStageDevelopment,
	})

	var notFound *cftypes.NoSuchFunctionExists
	if err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("describing function %s: %w", name, err)
	}

	if descResp != nil && descResp.ETag != nil {
		// Function exists, update it
		etag = *descResp.ETag
		updateResp, err := client.UpdateFunction(ctx, &cloudfront.UpdateFunctionInput{
			Name:         &name,
			IfMatch:      &etag,
			FunctionCode: code,
			FunctionConfig: &cftypes.FunctionConfig{
				Comment:                  strPtr(fmt.Sprintf("Managed by hedgerules: %s", name)),
				Runtime:                  runtime,
				KeyValueStoreAssociations: kvAssoc,
			},
		})
		if err != nil {
			return fmt.Errorf("updating function %s: %w", name, err)
		}
		etag = *updateResp.ETag
	} else {
		// Function doesn't exist, create it
		createResp, err := client.CreateFunction(ctx, &cloudfront.CreateFunctionInput{
			Name:         &name,
			FunctionCode: code,
			FunctionConfig: &cftypes.FunctionConfig{
				Comment:                  strPtr(fmt.Sprintf("Managed by hedgerules: %s", name)),
				Runtime:                  runtime,
				KeyValueStoreAssociations: kvAssoc,
			},
		})
		if err != nil {
			return fmt.Errorf("creating function %s: %w", name, err)
		}
		etag = *createResp.ETag
	}

	// Publish to LIVE stage
	_, err = client.PublishFunction(ctx, &cloudfront.PublishFunctionInput{
		Name:    &name,
		IfMatch: &etag,
	})
	if err != nil {
		return fmt.Errorf("publishing function %s: %w", name, err)
	}

	return nil
}

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
