package functions

import (
	_ "embed"
	"fmt"
)

//go:embed viewer-request.js
var ViewerRequestJS []byte

//go:embed viewer-response.js
var ViewerResponseJS []byte

// BuildFunctionCode prepends the KVS ID constant to the JS source.
func BuildFunctionCode(jsSource []byte, kvsID string) []byte {
	header := fmt.Sprintf("var kvsId = '%s';\n", kvsID)
	return append([]byte(header), jsSource...)
}
