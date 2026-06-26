package recon

import (
	"testing"
	"time"
)

func TestReconMatcher(t *testing.T) {
	matcher := NewReconMatcher()

	now := time.Now()

	candidates := []InternalLedgerEntry{
		{ID: "INT-1", Amount: 100.0, Date: now, Description: "AMAZON INC"},
		{ID: "INT-2", Amount: 500.0, Date: now.AddDate(0, 0, -5), Description: "GOOGLE"},
		{ID: "INT-3", Amount: 100.0, Date: now.AddDate(0, 0, 1), Description: "AMZN MKTPLACE"},
	}

	// Case 1: Perfect Match
	ext1 := ExternalTransaction{
		ID:          "EXT-1",
		Amount:      100.0,
		Date:        now,
		Description: "AMAZON INC",
	}
	result1 := matcher.FindBestMatch(ext1, candidates)
	if result1.InternalID != "INT-1" {
		t.Errorf("Expected INT-1, got %s", result1.InternalID)
	}

	// Case 2: Fuzzy Description Match ("AMZN MKTPLACE" vs "AMAZON INC" is tricky, but "AMZN MKTPLACE" vs "AMZN MKTPLACE" is exact)
	// Let's test the fuzzy logic specifically.
	// "AMZN MKTPLACE" (INT-3) vs "AMAZON MARKETPLACE" (EXT-2)
	ext2 := ExternalTransaction{
		ID:          "EXT-2",
		Amount:      100.0,
		Date:        now.AddDate(0, 0, 1),
		Description: "AMAZON MARKETPLACE",
	}
	result2 := matcher.FindBestMatch(ext2, candidates)
	
	// INT-3 has exact amount (50pts), close date (10pts), and fuzzy match.
	// INT-1 has exact amount (50pts), close date (8pts), and fuzzy match.
	// INT-3 should win due to date proximity and description similarity to "AMAZON MARKETPLACE"
	if result2.InternalID != "INT-3" {
		t.Errorf("Expected INT-3, got %s (Score: %d)", result2.InternalID, result2.Score)
	}

	// Case 3: No Match (Amount Mismatch)
	ext3 := ExternalTransaction{
		ID:          "EXT-3",
		Amount:      999.0,
		Date:        now,
		Description: "UNKNOWN",
	}
	result3 := matcher.FindBestMatch(ext3, candidates)
	if result3.IsMatch {
		t.Errorf("Expected No Match, got %s", result3.InternalID)
	}
}
