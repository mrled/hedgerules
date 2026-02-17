package kvs

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
)

func TestComputeSyncPlan_NewKeys(t *testing.T) {
	desired := &Data{
		Entries: []Entry{
			{Key: "/blog", Value: "/blog/"},
			{Key: "/about", Value: "/about/"},
		},
	}
	existing := map[string]string{}

	plan := ComputeSyncPlan(desired, existing)
	if len(plan.Puts) != 2 {
		t.Errorf("expected 2 puts, got %d", len(plan.Puts))
	}
	if len(plan.Deletes) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(plan.Deletes))
	}
}

func TestComputeSyncPlan_DeleteOldKeys(t *testing.T) {
	desired := &Data{
		Entries: []Entry{
			{Key: "/blog", Value: "/blog/"},
		},
	}
	existing := map[string]string{
		"/blog":  "/blog/",
		"/old":   "/old/",
		"/stale": "/stale/",
	}

	plan := ComputeSyncPlan(desired, existing)
	if len(plan.Puts) != 0 {
		t.Errorf("expected 0 puts (unchanged), got %d", len(plan.Puts))
	}
	sort.Strings(plan.Deletes)
	if len(plan.Deletes) != 2 {
		t.Fatalf("expected 2 deletes, got %d", len(plan.Deletes))
	}
	if plan.Deletes[0] != "/old" || plan.Deletes[1] != "/stale" {
		t.Errorf("expected /old and /stale deletes, got %v", plan.Deletes)
	}
}

func TestComputeSyncPlan_UpdateChangedValues(t *testing.T) {
	desired := &Data{
		Entries: []Entry{
			{Key: "/blog", Value: "/new-blog/"},
		},
	}
	existing := map[string]string{
		"/blog": "/blog/",
	}

	plan := ComputeSyncPlan(desired, existing)
	if len(plan.Puts) != 1 {
		t.Errorf("expected 1 put for changed value, got %d", len(plan.Puts))
	}
	if len(plan.Deletes) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(plan.Deletes))
	}
}

func TestComputeSyncPlan_NoChanges(t *testing.T) {
	desired := &Data{
		Entries: []Entry{
			{Key: "/blog", Value: "/blog/"},
		},
	}
	existing := map[string]string{
		"/blog": "/blog/",
	}

	plan := ComputeSyncPlan(desired, existing)
	if len(plan.Puts) != 0 {
		t.Errorf("expected 0 puts, got %d", len(plan.Puts))
	}
	if len(plan.Deletes) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(plan.Deletes))
	}
}

func TestComputeSyncPlan_Mixed(t *testing.T) {
	desired := &Data{
		Entries: []Entry{
			{Key: "/keep", Value: "/keep/"},       // unchanged
			{Key: "/update", Value: "/new-dest/"}, // value changed
			{Key: "/new", Value: "/new/"},         // new key
		},
	}
	existing := map[string]string{
		"/keep":   "/keep/",
		"/update": "/old-dest/",
		"/remove": "/remove/",
	}

	plan := ComputeSyncPlan(desired, existing)

	// Should put /update (changed) and /new (new)
	putKeys := make(map[string]bool)
	for _, p := range plan.Puts {
		putKeys[p.Key] = true
	}
	if len(plan.Puts) != 2 {
		t.Errorf("expected 2 puts, got %d: %v", len(plan.Puts), plan.Puts)
	}
	if !putKeys["/update"] || !putKeys["/new"] {
		t.Errorf("expected puts for /update and /new, got %v", plan.Puts)
	}

	// Should delete /remove
	if len(plan.Deletes) != 1 || plan.Deletes[0] != "/remove" {
		t.Errorf("expected delete for /remove, got %v", plan.Deletes)
	}
}

// mockKVSClient for testing batched Sync operations.
type mockKVSClient struct {
	updateKeysCalls []mockUpdateKeysCall
	nextETag        int
}

type mockUpdateKeysCall struct {
	putCount    int
	deleteCount int
	etag        string
}

func (m *mockKVSClient) DescribeKeyValueStore(ctx context.Context, params *cloudfrontkeyvaluestore.DescribeKeyValueStoreInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.DescribeKeyValueStoreOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockKVSClient) ListKeys(ctx context.Context, params *cloudfrontkeyvaluestore.ListKeysInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.ListKeysOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockKVSClient) UpdateKeys(ctx context.Context, params *cloudfrontkeyvaluestore.UpdateKeysInput, optFns ...func(*cloudfrontkeyvaluestore.Options)) (*cloudfrontkeyvaluestore.UpdateKeysOutput, error) {
	call := mockUpdateKeysCall{
		putCount:    len(params.Puts),
		deleteCount: len(params.Deletes),
		etag:        *params.IfMatch,
	}
	m.updateKeysCalls = append(m.updateKeysCalls, call)

	// Return new ETag for next batch
	m.nextETag++
	etag := fmt.Sprintf("etag-%d", m.nextETag)
	return &cloudfrontkeyvaluestore.UpdateKeysOutput{
		ETag: &etag,
	}, nil
}

func TestSync_SmallPlan(t *testing.T) {
	mock := &mockKVSClient{nextETag: 0}
	plan := &SyncPlan{
		Puts: []Entry{
			{Key: "/blog", Value: "/blog/"},
			{Key: "/about", Value: "/about/"},
		},
		Deletes: []string{"/old"},
	}

	err := Sync(context.Background(), mock, "arn:test", "etag-0", plan)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(mock.updateKeysCalls) != 1 {
		t.Fatalf("expected 1 UpdateKeys call, got %d", len(mock.updateKeysCalls))
	}

	call := mock.updateKeysCalls[0]
	if call.putCount != 2 {
		t.Errorf("expected 2 puts, got %d", call.putCount)
	}
	if call.deleteCount != 1 {
		t.Errorf("expected 1 delete, got %d", call.deleteCount)
	}
	if call.etag != "etag-0" {
		t.Errorf("expected etag-0, got %s", call.etag)
	}
}

func TestSync_ExactlyMaxBatch(t *testing.T) {
	mock := &mockKVSClient{nextETag: 0}

	// Create exactly 50 puts
	puts := make([]Entry, maxKeysPerBatch)
	for i := 0; i < maxKeysPerBatch; i++ {
		puts[i] = Entry{Key: fmt.Sprintf("/page%d", i), Value: fmt.Sprintf("/dest%d/", i)}
	}

	plan := &SyncPlan{Puts: puts}

	err := Sync(context.Background(), mock, "arn:test", "etag-0", plan)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(mock.updateKeysCalls) != 1 {
		t.Fatalf("expected 1 UpdateKeys call for exactly 50 keys, got %d", len(mock.updateKeysCalls))
	}

	call := mock.updateKeysCalls[0]
	if call.putCount != maxKeysPerBatch {
		t.Errorf("expected %d puts, got %d", maxKeysPerBatch, call.putCount)
	}
}

func TestSync_MultipleBatches(t *testing.T) {
	mock := &mockKVSClient{nextETag: 0}

	// Create 125 puts (should result in 3 batches: 50, 50, 25)
	puts := make([]Entry, 125)
	for i := 0; i < 125; i++ {
		puts[i] = Entry{Key: fmt.Sprintf("/page%d", i), Value: fmt.Sprintf("/dest%d/", i)}
	}

	plan := &SyncPlan{Puts: puts}

	err := Sync(context.Background(), mock, "arn:test", "etag-0", plan)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(mock.updateKeysCalls) != 3 {
		t.Fatalf("expected 3 UpdateKeys calls, got %d", len(mock.updateKeysCalls))
	}

	// Verify batch 1: 50 puts
	if mock.updateKeysCalls[0].putCount != 50 {
		t.Errorf("batch 1: expected 50 puts, got %d", mock.updateKeysCalls[0].putCount)
	}
	if mock.updateKeysCalls[0].etag != "etag-0" {
		t.Errorf("batch 1: expected etag-0, got %s", mock.updateKeysCalls[0].etag)
	}

	// Verify batch 2: 50 puts, uses updated ETag
	if mock.updateKeysCalls[1].putCount != 50 {
		t.Errorf("batch 2: expected 50 puts, got %d", mock.updateKeysCalls[1].putCount)
	}
	if mock.updateKeysCalls[1].etag != "etag-1" {
		t.Errorf("batch 2: expected etag-1, got %s", mock.updateKeysCalls[1].etag)
	}

	// Verify batch 3: 25 puts, uses updated ETag
	if mock.updateKeysCalls[2].putCount != 25 {
		t.Errorf("batch 3: expected 25 puts, got %d", mock.updateKeysCalls[2].putCount)
	}
	if mock.updateKeysCalls[2].etag != "etag-2" {
		t.Errorf("batch 3: expected etag-2, got %s", mock.updateKeysCalls[2].etag)
	}
}

func TestSync_MixedPutsAndDeletes(t *testing.T) {
	mock := &mockKVSClient{nextETag: 0}

	// Create 30 puts and 30 deletes (total 60, should result in 2 batches)
	puts := make([]Entry, 30)
	for i := 0; i < 30; i++ {
		puts[i] = Entry{Key: fmt.Sprintf("/new%d", i), Value: fmt.Sprintf("/dest%d/", i)}
	}

	deletes := make([]string, 30)
	for i := 0; i < 30; i++ {
		deletes[i] = fmt.Sprintf("/old%d", i)
	}

	plan := &SyncPlan{
		Puts:    puts,
		Deletes: deletes,
	}

	err := Sync(context.Background(), mock, "arn:test", "etag-0", plan)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(mock.updateKeysCalls) != 2 {
		t.Fatalf("expected 2 UpdateKeys calls, got %d", len(mock.updateKeysCalls))
	}

	// Batch 1: should have 30 puts + 20 deletes (fills to 50)
	call1 := mock.updateKeysCalls[0]
	if call1.putCount != 30 {
		t.Errorf("batch 1: expected 30 puts, got %d", call1.putCount)
	}
	if call1.deleteCount != 20 {
		t.Errorf("batch 1: expected 20 deletes, got %d", call1.deleteCount)
	}

	// Batch 2: should have 0 puts + 10 deletes (remaining)
	call2 := mock.updateKeysCalls[1]
	if call2.putCount != 0 {
		t.Errorf("batch 2: expected 0 puts, got %d", call2.putCount)
	}
	if call2.deleteCount != 10 {
		t.Errorf("batch 2: expected 10 deletes, got %d", call2.deleteCount)
	}
}

func TestSync_EmptyPlan(t *testing.T) {
	mock := &mockKVSClient{nextETag: 0}
	plan := &SyncPlan{}

	err := Sync(context.Background(), mock, "arn:test", "etag-0", plan)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(mock.updateKeysCalls) != 0 {
		t.Errorf("expected 0 UpdateKeys calls for empty plan, got %d", len(mock.updateKeysCalls))
	}
}
