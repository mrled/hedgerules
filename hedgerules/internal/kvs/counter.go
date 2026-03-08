package kvs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
)

// CountingKVSClient wraps a KVSClient and counts API calls made.
type CountingKVSClient struct {
	Client KVSClient
	Calls  int
}

func (c *CountingKVSClient) DescribeKeyValueStore(ctx context.Context, params *cloudfrontkeyvaluestore.DescribeKeyValueStoreInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.DescribeKeyValueStoreOutput, error) {
	c.Calls++
	return c.Client.DescribeKeyValueStore(ctx, params, optFns...)
}

func (c *CountingKVSClient) ListKeys(ctx context.Context, params *cloudfrontkeyvaluestore.ListKeysInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.ListKeysOutput, error) {
	c.Calls++
	return c.Client.ListKeys(ctx, params, optFns...)
}

func (c *CountingKVSClient) UpdateKeys(ctx context.Context, params *cloudfrontkeyvaluestore.UpdateKeysInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.UpdateKeysOutput, error) {
	c.Calls++
	return c.Client.UpdateKeys(ctx, params, optFns...)
}
