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
	if !strings.Contains(content, "{/path}") {
		t.Error("ViewerResponseJS missing {/path} token substitution")
	}
	if !strings.Contains(content, "x-hedgerules-patterns") {
		t.Error("ViewerResponseJS missing debug header x-hedgerules-patterns")
	}
	if !strings.Contains(content, "x-hedgerules-matched") {
		t.Error("ViewerResponseJS missing debug header x-hedgerules-matched")
	}
	if !strings.Contains(content, "x-hedgerules-error") {
		t.Error("ViewerResponseJS missing error debug header x-hedgerules-error")
	}
	if !strings.Contains(content, "debugHeaders") {
		t.Error("ViewerResponseJS missing debugHeaders conditional")
	}
}

func TestBuildFunctionCode(t *testing.T) {
	js := []byte("function handler() {}")
	kvsID := "arn:aws:cloudfront::123:key-value-store/abc"

	result := BuildFunctionCode(js, kvsID, false)
	code := string(result)

	if !strings.HasPrefix(code, "var kvsId = '"+kvsID+"';") {
		t.Errorf("expected kvsId prefix, got: %s", code[:80])
	}
	if !strings.Contains(code, "var debugHeaders = false;") {
		t.Error("expected debugHeaders = false")
	}
	if !strings.Contains(code, "function handler() {}") {
		t.Error("original JS code missing from result")
	}
}

func TestBuildFunctionCode_DebugEnabled(t *testing.T) {
	js := []byte("function handler() {}")
	kvsID := "arn:aws:cloudfront::123:key-value-store/abc"

	result := BuildFunctionCode(js, kvsID, true)
	code := string(result)

	if !strings.Contains(code, "var debugHeaders = true;") {
		t.Error("expected debugHeaders = true")
	}
	if !strings.Contains(code, "var kvsId = '"+kvsID+"';") {
		t.Error("expected kvsId variable")
	}
}
