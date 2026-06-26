package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hondyman/semlayer/backend/internal/semctl/codegen"
)

func RunSDK(args []string) {
	sdkCmd := flag.NewFlagSet("sdk", flag.ExitOnError)
	langFlag := sdkCmd.String("lang", "", "Language to generate (ts)")
	outFlag := sdkCmd.String("out", "", "Output directory")
	dirFlag := sdkCmd.String("dir", "./semantic", "Input semantic directory")

	// Handle 'generate' subcommand if passed as first arg
	if len(args) > 0 && args[0] == "generate" {
		sdkCmd.Parse(args[1:])
	} else {
		sdkCmd.Parse(args)
	}

	if *langFlag == "" || *outFlag == "" {
		fmt.Println("Usage: semctl sdk generate --lang <lang> --out <dir>")
		os.Exit(1)
	}

	// Load endpoints
	endpointsPath := filepath.Join(*dirFlag, "apis", "endpoints.json")
	if _, err := os.Stat(endpointsPath); os.IsNotExist(err) {
		fmt.Printf("Endpoints file not found at %s. Run 'semctl pull' first.\n", endpointsPath)
		os.Exit(1)
	}

	if *langFlag == "ts" {
		if err := codegen.GenerateTypeScript(endpointsPath, *outFlag); err != nil {
			fmt.Printf("Error generating TS SDK: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("TS SDK generated in %s\n", *outFlag)
	} else {
		fmt.Printf("Language '%s' not supported yet\n", *langFlag)
		os.Exit(1)
	}
}
