package hugo

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestScanDirectories(t *testing.T) {
	// Create temp directory structure
	dir := t.TempDir()
	// public/
	//   blog/
	//     2024/
	//   about/
	os.MkdirAll(filepath.Join(dir, "blog", "2024"), 0755)
	os.MkdirAll(filepath.Join(dir, "about"), 0755)
	// Create some files to make sure they're not included
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>"), 0644)
	os.WriteFile(filepath.Join(dir, "blog", "index.html"), []byte("<html>"), 0644)

	entries, err := ScanDirectories(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Sort for deterministic comparison
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	expected := map[string]string{
		"/about":     "/about/",
		"/blog":      "/blog/",
		"/blog/2024": "/blog/2024/",
	}

	if len(entries) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(entries), entries)
	}

	for _, e := range entries {
		want, ok := expected[e.Key]
		if !ok {
			t.Errorf("unexpected entry: %s -> %s", e.Key, e.Value)
			continue
		}
		if e.Value != want {
			t.Errorf("entry %s: expected value %s, got %s", e.Key, want, e.Value)
		}
	}
}

func TestScanDirectories_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ScanDirectories(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dir, got %d", len(entries))
	}
}

func TestScanDirectories_NotExists(t *testing.T) {
	_, err := ScanDirectories("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}
