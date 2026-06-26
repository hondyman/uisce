package main

import (
	"fmt"
	"os"
	"path/filepath"

	"cube-gonja/config"
	"cube-gonja/internal/render"
)

func main() {
	cfg := config.FromEnv()
	dir := cfg.OutputDir
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if _, err := os.Stat(dir); err != nil {
		fmt.Println("no model dir:", dir)
		os.Exit(1)
	}

	var failed bool
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(path) != ".yml" {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			failed = true
			fmt.Println("-", path, "read error:", rerr)
			return nil
		}
		if verr := render.ValidateYAMLHardBinding(path, b, cfg.AllowedDataSource); verr != nil {
			failed = true
			fmt.Println("-", verr)
		}
		return nil
	})
	if failed {
		fmt.Println("❌ data_source validation failed")
		os.Exit(2)
	}
	fmt.Println("✅ data_source validation passed")
}
