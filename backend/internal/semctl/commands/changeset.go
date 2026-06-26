package commands

import (
	"flag"
	"fmt"
	"os"
)

func RunChangeSet(args []string) {
	csCmd := flag.NewFlagSet("changeset", flag.ExitOnError)
	// action := csCmd.String("action", "create", "Action: create, list")
	csCmd.Parse(args)

	if len(args) < 1 {
		fmt.Println("Usage: semctl changeset <action> [args]")
		return
	}

	action := args[0]
	switch action {
	case "create":
		// TODO: Implement create logic (bundling diffs)
		fmt.Println("ChangeSet created: CS-MOCK-1234 (Mock)")
	default:
		fmt.Printf("Unknown changeset action: %s\n", action)
		os.Exit(1)
	}
}
