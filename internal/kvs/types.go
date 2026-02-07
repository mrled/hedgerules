package kvs

// Entry is a single key-value pair destined for CloudFront KVS.
type Entry struct {
	Key   string
	Value string
}

// Data holds all entries for a single KVS.
type Data struct {
	Entries []Entry
}

// SyncPlan describes what operations are needed to bring KVS to desired state.
type SyncPlan struct {
	Puts    []Entry  // Keys to add or update
	Deletes []string // Keys to remove
}
