package functions

import (
	"strings"
	"testing"
)

func TestViewerRequestJSEmbedded(t *testing.T) {
	if len(ViewerRequestJS) == 0 {
		t.Fatal("ViewerRequestJS is empty")
	}
	content := string(ViewerRequestJS)
	if !strings.Contains(content, "async function handler") {
		t.Error("ViewerRequestJS missing handler function")
	}
	if !strings.Contains(content, "kvs.get") {
		t.Error("ViewerRequestJS missing KVS lookup")
	}
}

func TestViewerResponseJSEmbedded(t *testing.T) {
	if len(ViewerResponseJS) == 0 {
		t.Fatal("ViewerResponseJS is empty")
	}
	content := string(ViewerResponseJS)
	if !strings.Contains(content, "async function handler") {
		t.Error("ViewerResponseJS missing handler function")
	}
	if !strings.Contains(content, "kvs.get") {
		t.Error("ViewerResponseJS missing KVS lookup")
	}
}

func TestBuildFunctionCode(t *testing.T) {
	js := []byte("function handler() {}")
	kvsID := "arn:aws:cloudfront::123:key-value-store/abc"

	result := BuildFunctionCode(js, kvsID)
	code := string(result)

	if !strings.HasPrefix(code, "var kvsId = '"+kvsID+"';") {
		t.Errorf("expected kvsId prefix, got: %s", code[:80])
	}
	if !strings.Contains(code, "function handler() {}") {
		t.Error("original JS code missing from result")
	}
}
