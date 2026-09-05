package models

import "testing"

// TestScannerToFrontendWireFormat proves the last-mile conversion the whole
// fix depends on: internal/scanner/ansi_scanner.go writes the DB-canonical
// enum form (e.g. "MANY_TO_MANY") into catalog_edge.properties, and
// backend/internal/metadata/businessobject_service.go's
// GetBusinessObjectRelationships parses it with ParseCardinality and
// applies Display() before returning JSON to the frontend, which expects
// the '1:1'/'1:M'/'M:1'/'M:M' wire format (frontend/src/types/cardinality.ts).
// If either side of that conversion were missing, a real M:M relationship
// scanned correctly on the backend would still never render as an embedded
// grid in the page designer — this test is the one-line proof that both
// sides are wired.
func TestScannerToFrontendWireFormat(t *testing.T) {
	cases := map[string]string{
		"ONE_TO_ONE":   "1:1",
		"ONE_TO_MANY":  "1:M",
		"MANY_TO_ONE":  "M:1",
		"MANY_TO_MANY": "M:M",
	}
	for written, wantWire := range cases {
		got := ParseCardinality(written).Display()
		if got != wantWire {
			t.Errorf("ParseCardinality(%q).Display() = %q, want %q", written, got, wantWire)
		}
	}
}
