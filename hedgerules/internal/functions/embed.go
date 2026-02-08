package functions

import (
	_ "embed"
	"fmt"
)

//go:embed viewer-request.js
var ViewerRequestJS []byte

//go:embed viewer-response.js
var ViewerResponseJS []byte

// BuildFunctionCode prepends injected variables to the JS source.
// It injects the KVS ID and the debug headers toggle.
func BuildFunctionCode(jsSource []byte, kvsID string, debugHeaders bool) []byte {
	header := fmt.Sprintf("var kvsId = '%s';\nvar debugHeaders = %v;\n", kvsID, debugHeaders)
	return append([]byte(header), jsSource...)
}
