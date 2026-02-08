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
	os.WriteFile(filepath.Join(dir, "_hedge_redirects.txt"), []byte(content), 0644)

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
	os.WriteFile(filepath.Join(dir, "_hedge_redirects.txt"), []byte(""), 0644)

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

func TestResolveChains(t *testing.T) {
	entries := []kvs.Entry{
		{Key: "/a", Value: "/b"},
		{Key: "/b", Value: "/c"},
	}
	resolved, err := ResolveChains(entries)
	if err != nil {
		t.Fatal(err)
	}

	result := make(map[string]string)
	for _, e := range resolved {
		result[e.Key] = e.Value
	}

	if result["/a"] != "/c" {
		t.Errorf("/a: expected /c, got %s", result["/a"])
	}
	if result["/b"] != "/c" {
		t.Errorf("/b: expected /c, got %s", result["/b"])
	}
}

func TestResolveChains_MultiHop(t *testing.T) {
	entries := []kvs.Entry{
		{Key: "/a", Value: "/b"},
		{Key: "/b", Value: "/c"},
		{Key: "/c", Value: "/d"},
		{Key: "/d", Value: "/e"},
	}
	resolved, err := ResolveChains(entries)
	if err != nil {
		t.Fatal(err)
	}

	result := make(map[string]string)
	for _, e := range resolved {
		result[e.Key] = e.Value
	}

	for _, key := range []string{"/a", "/b", "/c", "/d"} {
		if result[key] != "/e" {
			t.Errorf("%s: expected /e, got %s", key, result[key])
		}
	}
}

func TestResolveChains_Cycle(t *testing.T) {
	entries := []kvs.Entry{
		{Key: "/a", Value: "/b"},
		{Key: "/b", Value: "/c"},
		{Key: "/c", Value: "/a"},
	}
	_, err := ResolveChains(entries)
	if err == nil {
		t.Fatal("expected error for redirect cycle, got nil")
	}
}

func TestResolveChains_NoChains(t *testing.T) {
	entries := []kvs.Entry{
		{Key: "/old", Value: "/new"},
		{Key: "/blog", Value: "/blog/"},
		{Key: "/about", Value: "/about/"},
	}
	resolved, err := ResolveChains(entries)
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(resolved))
	}

	result := make(map[string]string)
	for _, e := range resolved {
		result[e.Key] = e.Value
	}

	if result["/old"] != "/new" {
		t.Errorf("/old: expected /new, got %s", result["/old"])
	}
	if result["/blog"] != "/blog/" {
		t.Errorf("/blog: expected /blog/, got %s", result["/blog"])
	}
	if result["/about"] != "/about/" {
		t.Errorf("/about: expected /about/, got %s", result["/about"])
	}
}
