package main

import (
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "dispatch":
			RunDispatch()
		case "websocket":
			RunWebSocketTest()
		default:
			RunDispatch()
		}
	} else {
		RunDispatch()
	}
}
