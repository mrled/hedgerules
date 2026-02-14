package hugo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/micahrl/hedgerules/hedgerules/internal/kvs"
)

// ScanDirectories walks outputDir and returns index redirect entries.
// For each directory, it creates a redirect from /dir to /dir/.
func ScanDirectories(outputDir string) ([]kvs.Entry, error) {
	var entries []kvs.Entry

	info, err := os.Stat(outputDir)
	if err != nil {
		return nil, fmt.Errorf("stat output directory %s: %w", outputDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("output directory is not a directory: %s", outputDir)
	}

	err = filepath.WalkDir(outputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// Skip the root output directory itself
		if path == outputDir {
			return nil
		}

		rel, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}
		// Convert to URL path with forward slashes
		urlPath := "/" + filepath.ToSlash(rel)
		entries = append(entries, kvs.Entry{
			Key:   urlPath,
			Value: urlPath + "/",
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking output directory: %w", err)
	}

	return entries, nil
}
