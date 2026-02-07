package kvs

import "fmt"

const (
	MaxKeyBytes   = 512
	MaxEntryBytes = 1024    // key + value
	MaxTotalBytes = 5242880 // 5 MB
)

// ValidationError describes a single constraint violation.
type ValidationError struct {
	Key     string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Key, e.Message)
}

// DataStats holds summary size information for a Data set.
type DataStats struct {
	NumKeys    int
	TotalBytes int
}

// Stats returns the number of keys and total byte size of the data.
func (d *Data) Stats() DataStats {
	total := 0
	for _, e := range d.Entries {
		total += len([]byte(e.Key)) + len([]byte(e.Value))
	}
	return DataStats{NumKeys: len(d.Entries), TotalBytes: total}
}

// Validate checks all KVS constraints. Returns nil if valid.
func (d *Data) Validate() []ValidationError {
	var errs []ValidationError
	totalSize := 0

	for _, e := range d.Entries {
		keySize := len([]byte(e.Key))
		entrySize := keySize + len([]byte(e.Value))

		if keySize > MaxKeyBytes {
			errs = append(errs, ValidationError{
				Key:     e.Key,
				Message: fmt.Sprintf("key exceeds %d bytes (%d bytes)", MaxKeyBytes, keySize),
			})
		}

		if entrySize > MaxEntryBytes {
			errs = append(errs, ValidationError{
				Key:     e.Key,
				Message: fmt.Sprintf("key+value exceeds %d bytes (%d bytes)", MaxEntryBytes, entrySize),
			})
		}

		totalSize += entrySize
	}

	if totalSize > MaxTotalBytes {
		errs = append(errs, ValidationError{
			Key:     "(total)",
			Message: fmt.Sprintf("total data exceeds %d bytes (%d bytes)", MaxTotalBytes, totalSize),
		})
	}

	return errs
}
