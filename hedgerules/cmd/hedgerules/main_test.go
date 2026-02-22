package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAtFile_PlainValue(t *testing.T) {
	result, err := resolveAtFile("my-kvs-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-kvs-name" {
		t.Errorf("expected %q, got %q", "my-kvs-name", result)
	}
}

func TestResolveAtFile_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "kvs.txt")
	if err := os.WriteFile(f, []byte("  my-kvs-name\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveAtFile("@" + f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-kvs-name" {
		t.Errorf("expected %q, got %q", "my-kvs-name", result)
	}
}

func TestResolveAtFile_MissingFile(t *testing.T) {
	_, err := resolveAtFile("@/nonexistent/path/kvs.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
