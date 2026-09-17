package mcp

import "log"

// RegisterHook is invoked from TraceRegister so tests can trace wiring
// call sites → mux entry. Production leaves it nil; stderr still gets the log.
var RegisterHook func(source string)

// TraceRegister records that an MCP mux registration is about to run.
// Call this at the api.go wiring site, not only inside RegisterRoutes, so
// a Walk dump can be paired with the source line that produced the entry.
func TraceRegister(source string) {
	log.Printf("[MCP-REGISTER] %s", source)
	if RegisterHook != nil {
		RegisterHook(source)
	}
}
