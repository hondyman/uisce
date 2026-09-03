package semantic_bridge

import (
	"testing"

	"github.com/google/uuid"
)

func buildChain(t *testing.T, key []byte, payloadHashes []string) []ledgerRow {
	t.Helper()
	rows := make([]ledgerRow, len(payloadHashes))
	prev := ""
	for i, ph := range payloadHashes {
		sig := signChainEntry(key, prev, ph)
		rows[i] = ledgerRow{ID: uuid.New(), PayloadHash: ph, PrevHMAC: prev, HMACSig: sig}
		prev = sig
	}
	return rows
}

func TestVerifyChain_IntactChainPasses(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	rows := buildChain(t, key, []string{"hashA", "hashB", "hashC"})

	if broken := verifyChain(key, rows); broken != uuid.Nil {
		t.Fatalf("expected an intact chain to verify clean, but row %s failed", broken)
	}
}

func TestVerifyChain_EditedPayloadHashIsDetected(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	rows := buildChain(t, key, []string{"hashA", "hashB", "hashC"})

	tamperedID := rows[1].ID
	rows[1].PayloadHash = "hashB-edited-by-someone-with-db-access"

	broken := verifyChain(key, rows)
	if broken != tamperedID {
		t.Fatalf("expected tamper detection to flag row %s, got %s", tamperedID, broken)
	}
}

func TestVerifyChain_DeletedRowBreaksTheChainAfterIt(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	rows := buildChain(t, key, []string{"hashA", "hashB", "hashC"})

	// Simulate deleting row 1 (hashB) — row 2 (hashC) now links to row 0's
	// signature instead of the deleted row's, which won't match.
	withGap := []ledgerRow{rows[0], rows[2]}

	broken := verifyChain(key, withGap)
	if broken != rows[2].ID {
		t.Fatalf("expected the row after the deletion (%s) to fail verification, got %s", rows[2].ID, broken)
	}
}

func TestVerifyChain_WrongKeyFailsEverything(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	wrongKey := []byte("fedcba9876543210fedcba9876543210")
	rows := buildChain(t, key, []string{"hashA", "hashB"})

	broken := verifyChain(wrongKey, rows)
	if broken != rows[0].ID {
		t.Fatalf("expected verification with the wrong key to fail at the first row (%s), got %s", rows[0].ID, broken)
	}
}

func TestVerifyChain_EmptyChainIsTriviallyIntact(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	if broken := verifyChain(key, nil); broken != uuid.Nil {
		t.Fatalf("expected an empty chain to verify clean, got %s", broken)
	}
}
