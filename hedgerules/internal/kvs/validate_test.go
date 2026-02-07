package kvs

import (
	"strings"
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	d := &Data{
		Entries: []Entry{
			{Key: "/blog", Value: "/blog/"},
			{Key: "/about", Value: "/about/"},
		},
	}
	errs := d.Validate()
	if len(errs) > 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_EmptyData(t *testing.T) {
	d := &Data{}
	errs := d.Validate()
	if len(errs) > 0 {
		t.Errorf("expected no errors for empty data, got %v", errs)
	}
}

func TestValidate_KeyTooLong(t *testing.T) {
	d := &Data{
		Entries: []Entry{
			{Key: "/" + strings.Repeat("a", 512), Value: "/dest"},
		},
	}
	errs := d.Validate()
	if len(errs) == 0 {
		t.Fatal("expected validation error for key > 512 bytes")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "key exceeds") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected key size error, got %v", errs)
	}
}

func TestValidate_EntryTooLong(t *testing.T) {
	d := &Data{
		Entries: []Entry{
			{Key: "/ok", Value: strings.Repeat("x", 1022)},
		},
	}
	errs := d.Validate()
	if len(errs) == 0 {
		t.Fatal("expected validation error for entry > 1024 bytes")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "key+value exceeds") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected entry size error, got %v", errs)
	}
}

func TestValidate_TotalTooLarge(t *testing.T) {
	var entries []Entry
	// Each entry is ~1000 bytes, need >5242 to exceed 5MB
	val := strings.Repeat("x", 990)
	for i := 0; i < 5300; i++ {
		entries = append(entries, Entry{Key: strings.Repeat("a", 10), Value: val})
	}
	d := &Data{Entries: entries}
	errs := d.Validate()
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "total data exceeds") {
			found = true
		}
	}
	if !found {
		t.Error("expected total size error")
	}
}

func TestValidate_ExactBoundaries(t *testing.T) {
	// Key exactly 512 bytes should pass
	d := &Data{
		Entries: []Entry{
			{Key: strings.Repeat("a", 512), Value: strings.Repeat("b", 512)},
		},
	}
	errs := d.Validate()
	if len(errs) > 0 {
		t.Errorf("expected key of exactly 512 bytes to pass, got %v", errs)
	}

	// Entry exactly 1024 bytes should pass
	d = &Data{
		Entries: []Entry{
			{Key: strings.Repeat("a", 100), Value: strings.Repeat("b", 924)},
		},
	}
	errs = d.Validate()
	if len(errs) > 0 {
		t.Errorf("expected entry of exactly 1024 bytes to pass, got %v", errs)
	}
}
