package kvs

import (
	"sort"
	"testing"
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
			{Key: "/keep", Value: "/keep/"},      // unchanged
			{Key: "/update", Value: "/new-dest/"}, // value changed
			{Key: "/new", Value: "/new/"},          // new key
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
