package hugo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrled/hedgerules/hedgerules/internal/kvs"
)

// ParseHeaders reads _hedge_headers.json and returns header entries.
// The JSON format is: { "/path": { "Header-Name": "value", ... }, ... }
// Each entry's value is newline-delimited "Header-Name: value" strings.
func ParseHeaders(outputDir string) ([]kvs.Entry, error) {
	path := filepath.Join(outputDir, "_hedge_headers.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil // No headers file is fine
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var raw map[string]map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var entries []kvs.Entry
	for urlPath, headerMap := range raw {
		var lines []string
		for name, value := range headerMap {
			lines = append(lines, fmt.Sprintf("%s: %s", name, value))
		}
		entries = append(entries, kvs.Entry{
			Key:   urlPath,
			Value: strings.Join(lines, "\n"),
		})
	}

	return entries, nil
}
