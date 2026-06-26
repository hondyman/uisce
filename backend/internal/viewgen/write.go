package viewgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteViews writes the generated views into a directory as individual JSON files.
func WriteViews(dir string, res Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	for _, v := range res.Views {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal view %s: %w", v.Name, err)
		}
		fp := filepath.Join(dir, v.Name+".json")
		if err := os.WriteFile(fp, b, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", fp, err)
		}
	}
	return nil
}
