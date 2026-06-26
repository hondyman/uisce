package main

import (
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "client":
			RunWebSocketClientMain()
		case "server":
			RunWebSocketServerMain()
		default:
			RunWebSocketServerMain()
		}
	} else {
		RunWebSocketServerMain()
	}
}
