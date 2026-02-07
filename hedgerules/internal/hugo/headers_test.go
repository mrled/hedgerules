package hugo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHeaders(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "/": {
    "X-Frame-Options": "DENY",
    "X-Content-Type-Options": "nosniff"
  },
  "/blog/my-post/": {
    "Cache-Control": "max-age=3600"
  }
}`
	os.WriteFile(filepath.Join(dir, "_hedge_headers.json"), []byte(content), 0644)

	entries, err := ParseHeaders(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	result := make(map[string]string)
	for _, e := range entries {
		result[e.Key] = e.Value
	}

	// Check root headers
	root, ok := result["/"]
	if !ok {
		t.Fatal("missing / entry")
	}
	if !strings.Contains(root, "X-Frame-Options: DENY") {
		t.Errorf("root missing X-Frame-Options header, got: %s", root)
	}
	if !strings.Contains(root, "X-Content-Type-Options: nosniff") {
		t.Errorf("root missing X-Content-Type-Options header, got: %s", root)
	}

	// Check blog post headers
	blog, ok := result["/blog/my-post/"]
	if !ok {
		t.Fatal("missing /blog/my-post/ entry")
	}
	if blog != "Cache-Control: max-age=3600" {
		t.Errorf("blog entry: expected 'Cache-Control: max-age=3600', got '%s'", blog)
	}
}

func TestParseHeaders_NoFile(t *testing.T) {
	dir := t.TempDir()
	entries, err := ParseHeaders(dir)
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil for missing file, got %v", entries)
	}
}

func TestParseHeaders_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_hedge_headers.json"), []byte("{invalid"), 0644)

	_, err := ParseHeaders(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseHeaders_EmptyJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_hedge_headers.json"), []byte("{}"), 0644)

	entries, err := ParseHeaders(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}
