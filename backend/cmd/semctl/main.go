package main

import (
	"fmt"
	"os"

	"github.com/hondyman/semlayer/backend/internal/semctl/commands"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	// Parse flags after subcommand
	// But we need to let subcommand handle its own flags

	switch subcommand {
	case "init":
		commands.RunInit(os.Args[2:])
	case "pull":
		commands.RunPull(os.Args[2:])
	case "diff":
		commands.RunDiff(os.Args[2:])
	case "validate":
		commands.RunValidate(os.Args[2:])
	case "changeset":
		commands.RunChangeSet(os.Args[2:])
	case "sdk":
		commands.RunSDK(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: semctl <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  init       Initialize semantic configuration")
	fmt.Println("  pull       Pull semantic objects from remote")
	fmt.Println("  diff       Diff local objects against remote")
	fmt.Println("  validate   Validate local objects")
	fmt.Println("  changeset  Manage ChangeSets")
	fmt.Println("  sdk        Generate SDKs")
}
