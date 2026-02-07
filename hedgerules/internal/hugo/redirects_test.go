package hugo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/micahrl/hedgerules/internal/kvs"
)

func TestParseRedirects(t *testing.T) {
	dir := t.TempDir()
	content := `/old-page /new-page
/blog/old /blog/new 301
# This is a comment

/with-spaces   /destination
`
	os.WriteFile(filepath.Join(dir, "_redirects"), []byte(content), 0644)

	entries, err := ParseRedirects(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"/old-page":    "/new-page",
		"/blog/old":    "/blog/new",
		"/with-spaces": "/destination",
	}

	if len(entries) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(entries))
	}

	for _, e := range entries {
		want, ok := expected[e.Key]
		if !ok {
			t.Errorf("unexpected entry: %s", e.Key)
			continue
		}
		if e.Value != want {
			t.Errorf("entry %s: expected %s, got %s", e.Key, want, e.Value)
		}
	}
}

func TestParseRedirects_NoFile(t *testing.T) {
	dir := t.TempDir()
	entries, err := ParseRedirects(dir)
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil for missing file, got %v", entries)
	}
}

func TestParseRedirects_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_redirects"), []byte(""), 0644)

	entries, err := ParseRedirects(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestMergeRedirects(t *testing.T) {
	dirEntries := []kvs.Entry{
		{Key: "/blog", Value: "/blog/"},
		{Key: "/about", Value: "/about/"},
	}
	fileEntries := []kvs.Entry{
		{Key: "/blog", Value: "/new-blog/"},   // Override
		{Key: "/custom", Value: "/redirect/"}, // New
	}

	merged := MergeRedirects(dirEntries, fileEntries)

	result := make(map[string]string)
	for _, e := range merged {
		result[e.Key] = e.Value
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// /blog should be overridden by file
	if result["/blog"] != "/new-blog/" {
		t.Errorf("/blog: expected /new-blog/, got %s", result["/blog"])
	}
	// /about should remain from dirs
	if result["/about"] != "/about/" {
		t.Errorf("/about: expected /about/, got %s", result["/about"])
	}
	// /custom should be from file
	if result["/custom"] != "/redirect/" {
		t.Errorf("/custom: expected /redirect/, got %s", result["/custom"])
	}
}

func TestMergeRedirects_Empty(t *testing.T) {
	merged := MergeRedirects(nil, nil)
	if len(merged) != 0 {
		t.Errorf("expected 0 entries, got %d", len(merged))
	}
}
