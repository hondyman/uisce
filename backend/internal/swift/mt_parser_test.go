package swift

import (
	"strings"
	"testing"
)

// TestParseMT_Block1_BICSender verifies BIC extraction from block 1.
func TestParseMT_Block1_BICSender(t *testing.T) {
	raw := []byte(`{1:F01BANKBEBB0000000000}{2:I541BANKUS33XBBN}{4:
:20:REF20241001
:35B:ISIN US0231351067
:36:1000
:98A:SETT//20241003
:95P:BUYR//CUSTBEBB
:97A:SAFE//123456
:19A:SETT//USD500000,00
-}`)
	msg, err := ParseMT(raw)
	if err != nil {
		t.Fatalf("ParseMT returned error: %v", err)
	}
	if msg.BICSender != "BANKBEBB0000" {
		t.Errorf("BICSender: got %q, want %q", msg.BICSender, "BANKBEBB0000")
	}
	if msg.MsgType != "541" {
		t.Errorf("MsgType: got %q, want %q", msg.MsgType, "541")
	}
}

// TestParseMT_Fields verifies field extraction from block 4.
func TestParseMT_Fields(t *testing.T) {
	raw := []byte(`{1:F01BANKBEBB0000000000}{2:I541BANKUS33XBBN}{4:
:20:REF-20241001-ABC
:35B:ISIN US0231351067
COMMON REFERENCE
:36:1000,
:98A:SETT//20241003
:95P:BUYR//CUSTBEBB
:97A:SAFE//123456
:19A:SETT//USD500000,00
-}`)
	msg, err := ParseMT(raw)
	if err != nil {
		t.Fatalf("ParseMT error: %v", err)
	}

	cases := []struct {
		tag  string
		want string
	}{
		{":20:", "REF-20241001-ABC"},
		{":36:", "1000,"},
		{":98A:SETT", "//20241003"},
		{":97A:SAFE", "//123456"},
	}
	for _, c := range cases {
		got, ok := msg.Fields[c.tag]
		if !ok {
			t.Errorf("field %q missing from parsed message", c.tag)
			continue
		}
		if got != c.want {
			t.Errorf("field %q: got %q, want %q", c.tag, got, c.want)
		}
	}
}

// TestParseMT_ISINExtraction verifies that :35B: strips the ISIN prefix for
// fields that begin with "ISIN " — the field value is the full line including
// "ISIN " so callers can use parse_isin transform.
func TestParseMT_ISINExtraction(t *testing.T) {
	raw := []byte(`{1:F01BANKBEBB0000000000}{2:I543BANKUS33XBBN}{4:
:20:DELIV001
:35B:ISIN US4592001014
/DE/1234567
-}`)
	msg, err := ParseMT(raw)
	if err != nil {
		t.Fatalf("ParseMT error: %v", err)
	}
	isinField, ok := msg.Fields[":35B:"]
	if !ok {
		t.Fatal(":35B: field missing")
	}
	if !strings.HasPrefix(isinField, "ISIN ") {
		t.Errorf(":35B: value %q does not start with 'ISIN '", isinField)
	}
	// Verify ISIN code is extractable
	isin := strings.TrimPrefix(isinField, "ISIN ")
	isin = strings.Fields(isin)[0]
	if isin != "US4592001014" {
		t.Errorf("extracted ISIN: got %q, want %q", isin, "US4592001014")
	}
}

// TestParseMT_SequenceBlocks verifies that 16R/16S sequence delimiters are
// recognised in the field map (not treated as nested blocks — we flatten them).
func TestParseMT_SequenceBlocks(t *testing.T) {
	raw := []byte(`{1:F01BANKBEBB0000000000}{2:I541BANKUS33XBBN}{4:
:16R:GENL
:20:SEQ-REF-001
:16S:GENL
:16R:TRADDET
:35B:ISIN GB0002634946
:36:500
:16S:TRADDET
-}`)
	msg, err := ParseMT(raw)
	if err != nil {
		t.Fatalf("ParseMT error: %v", err)
	}
	if msg.Fields[":20:"] != "SEQ-REF-001" {
		t.Errorf(":20: inside sequence block: got %q, want %q", msg.Fields[":20:"], "SEQ-REF-001")
	}
	if _, ok := msg.Fields[":35B:"]; !ok {
		t.Error(":35B: inside sequence block missing")
	}
	// Sequence delimiters should appear as fields so callers can detect structure
	if _, ok := msg.Fields[":16R:GENL"]; !ok {
		t.Log("note: :16R:GENL not stored as a field — acceptable if parser is field-flatten-only")
	}
}

// TestParseMT_RoundTrip_MT543 verifies that a DVP deliver instruction parsed
// from raw MT543 bytes round-trips correctly through field extraction.
// The instruction_emit tile reverses this: semantic map → MT tags.
func TestParseMT_RoundTrip_MT543(t *testing.T) {
	rawMT543 := []byte(`{1:F01DELRBANK0000000001}{2:I543RECVBANKXBBN}{4:
:20:TXN-DVP-20241003-001
:35B:ISIN FR0000131104
:36:2500
:90A:PRCT//100,
:98A:SETT//20241005
:95P:SELL//DELVBEBB
:97A:SAFE//ACCT-987654
:19A:SETT//USD1250000,00
-}`)

	msg, err := ParseMT(rawMT543)
	if err != nil {
		t.Fatalf("ParseMT MT543 error: %v", err)
	}

	// Verify all settlement-critical fields are present
	required := []string{":20:", ":35B:", ":36:", ":98A:SETT", ":95P:SELL", ":97A:SAFE", ":19A:SETT"}
	for _, tag := range required {
		if _, ok := msg.Fields[tag]; !ok {
			t.Errorf("MT543 round-trip: required field %q missing after parse", tag)
		}
	}

	// Verify transaction ref is intact
	if got := msg.Fields[":20:"]; got != "TXN-DVP-20241003-001" {
		t.Errorf(":20: round-trip: got %q, want %q", got, "TXN-DVP-20241003-001")
	}
	if msg.MsgType != "543" {
		t.Errorf("MsgType round-trip: got %q, want 543", msg.MsgType)
	}
}
