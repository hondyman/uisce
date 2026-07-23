package api

import (
	"encoding/json"
	"testing"
)

type progressPayload struct {
	Type    string      `json:"type"`
	Current int         `json:"current"`
	Total   int         `json:"total"`
	Message string      `json:"message"`
	Results interface{} `json:"results"`
}

type completedPayload struct {
	Type    string        `json:"type"`
	Current int           `json:"current"`
	Total   int           `json:"total"`
	Message string        `json:"message"`
	Results []interface{} `json:"results"`
}

func TestWSPayloadShapes(t *testing.T) {
	// progress - use typed struct so json.Marshal order is deterministic
	prog := progressPayload{Type: "progress", Current: 1, Total: 10, Message: "working", Results: nil}
	b, err := json.Marshal(prog)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	// exact JSON ordering expected from struct field order
	expectedProg := `{"type":"progress","current":1,"total":10,"message":"working","results":null}`
	if string(b) != expectedProg {
		t.Fatalf("progress payload mismatch:\n got: %s\nwant: %s", string(b), expectedProg)
	}

	// failed - reuse progress struct with different values
	fail := progressPayload{Type: "failed", Current: 0, Total: 0, Message: "error", Results: nil}
	b2, err := json.Marshal(fail)
	if err != nil {
		t.Fatalf("failed to marshal failed payload: %v", err)
	}
	if string(b2) != `{"type":"failed","current":0,"total":0,"message":"error","results":null}` {
		t.Fatalf("failed payload mismatch: %s", string(b2))
	}

	// completed - ensure results ordering via typed struct
	completed := completedPayload{Type: "completed", Current: 100, Total: 100, Message: "done", Results: []interface{}{map[string]interface{}{"a": 1}}}
	b3, err := json.Marshal(completed)
	if err != nil {
		t.Fatalf("failed to marshal completed: %v", err)
	}
	expectedCompletedPrefix := `{"type":"completed","current":100,"total":100,"message":"done","results":` // results may contain object ordering
	if len(b3) < len(expectedCompletedPrefix) || string(b3[:len(expectedCompletedPrefix)]) != expectedCompletedPrefix {
		t.Fatalf("completed payload prefix mismatch: %s", string(b3))
	}
}
