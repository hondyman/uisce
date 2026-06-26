package commands

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hondyman/semlayer/backend/internal/semctl/config"
	"github.com/nsf/jsondiff"
)

func RunDiff(args []string) {
	diffCmd := flag.NewFlagSet("diff", flag.ExitOnError)
	envFlag := diffCmd.String("env", "dev", "Environment to diff against")
	dirFlag := diffCmd.String("dir", "./semantic", "Local directory to diff")
	diffCmd.Parse(args)

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	envCfg, ok := cfg.Envs[*envFlag]
	if !ok {
		fmt.Printf("Environment '%s' not found\n", *envFlag)
		os.Exit(1)
	}

	// Diff APIs
	fmt.Println("Diffing APIs...")
	localAPIPath := filepath.Join(*dirFlag, "apis", "endpoints.json")
	diffFile(localAPIPath, envCfg.URL+"/api/api-studio/endpoints?env="+*envFlag)

	// Diff Pages
	fmt.Println("Diffing Pages...")
	localPagePath := filepath.Join(*dirFlag, "pages", "pages.json")
	diffFile(localPagePath, envCfg.URL+"/api/page-studio/pages?env="+*envFlag)
}

func diffFile(localPath, remoteURL string) {
	localData, err := os.ReadFile(localPath)
	if err != nil {
		fmt.Printf("  [!] Could not read local file %s: %v\n", localPath, err)
		return
	}

	resp, err := http.Get(remoteURL)
	if err != nil {
		fmt.Printf("  [!] Could not fetch remote %s: %v\n", remoteURL, err)
		return
	}
	defer resp.Body.Close()
	remoteData, _ := io.ReadAll(resp.Body)

	opts := jsondiff.DefaultConsoleOptions()
	diffEnum, diffStr := jsondiff.Compare(localData, remoteData, &opts)

	if diffEnum == jsondiff.FullMatch {
		fmt.Printf("  %s: No changes\n", filepath.Base(localPath))
	} else {
		fmt.Printf("  %s: Changes found\n%s\n", filepath.Base(localPath), diffStr)
	}
}
