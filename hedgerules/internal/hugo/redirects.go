package hugo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/micahrl/hedgerules/hedgerules/internal/kvs"
)

// ParseRedirects reads the _hedge_redirects.txt file and returns redirect entries.
// Lines are whitespace-separated: source destination [status].
// Empty lines and lines starting with # are ignored.
func ParseRedirects(outputDir string) ([]kvs.Entry, error) {
	path := filepath.Join(outputDir, "_hedge_redirects.txt")

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil // No _hedge_redirects.txt file is fine
	}
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var entries []kvs.Entry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Fprintf(os.Stderr, "warning: invalid redirect on line %d: %s\n", lineNum, line)
			continue
		}

		entries = append(entries, kvs.Entry{
			Key:   parts[0],
			Value: parts[1],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return entries, nil
}

// MergeRedirects merges directory redirects with file redirects.
// File redirects take precedence over directory redirects.
func MergeRedirects(dirEntries, fileEntries []kvs.Entry) []kvs.Entry {
	merged := make(map[string]string)

	// Directory entries first (lower priority)
	for _, e := range dirEntries {
		merged[e.Key] = e.Value
	}

	// File entries override
	for _, e := range fileEntries {
		merged[e.Key] = e.Value
	}

	entries := make([]kvs.Entry, 0, len(merged))
	for k, v := range merged {
		entries = append(entries, kvs.Entry{Key: k, Value: v})
	}
	return entries
}
