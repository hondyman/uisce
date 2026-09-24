package swift

import (
	"testing"
	"time"
)

// Sample ISO 20022 pacs.008 (FI-to-FI Customer Credit Transfer).
// Uses the urn:iso:std:iso:20022:tech:xsd:pacs.008.001.09 namespace.
const samplePacs008 = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pacs.008.001.09">
  <FIToFICstmrCdtTrf>
    <GrpHdr>
      <MsgId>PACS008-TEST-001</MsgId>
      <CreDtTm>2026-10-03T14:30:00Z</CreDtTm>
      <NbOfTxs>1</NbOfTxs>
      <TtlIntrBkSttlmAmt Ccy="USD">1000000.00</TtlIntrBkSttlmAmt>
      <SttlmInf>
        <SttlmMtd>INDA</SttlmMtd>
      </SttlmInf>
    </GrpHdr>
    <CdtTrfTxInf>
      <PmtId>
        <EndToEndId>E2E-REF-TEST-001</EndToEndId>
        <UETR>550e8400-e29b-41d4-a716-446655440000</UETR>
      </PmtId>
      <IntrBkSttlmAmt Ccy="USD">1000000.00</IntrBkSttlmAmt>
      <IntrBkSttlmDt>2026-10-03</IntrBkSttlmDt>
      <Dbtr>
        <FinInstnId>
          <BICFI>TESTUS33XXX</BICFI>
        </FinInstnId>
      </Dbtr>
      <Cdtr>
        <FinInstnId>
          <BICFI>CUSTGB2LXXX</BICFI>
        </FinInstnId>
      </Cdtr>
    </CdtTrfTxInf>
  </FIToFICstmrCdtTrf>
</Document>`

// Sample pacs.009 (Financial Institution Credit Transfer — bank-to-bank).
const samplePacs009 = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pacs.009.001.08">
  <FinInstnCdtTrf>
    <GrpHdr>
      <MsgId>PACS009-TEST-001</MsgId>
      <CreDtTm>2026-10-03T14:30:00Z</CreDtTm>
      <NbOfTxs>1</NbOfTxs>
    </GrpHdr>
    <CdtTrfTxInf>
      <PmtId>
        <EndToEndId>FI-E2E-001</EndToEndId>
        <UETR>660e8400-e29b-41d4-a716-446655440001</UETR>
      </PmtId>
      <IntrBkSttlmAmt Ccy="EUR">500000.00</IntrBkSttlmAmt>
    </CdtTrfTxInf>
  </FinInstnCdtTrf>
</Document>`

// Sample camt.056 (FI-to-FI Payment Cancellation Request — the MX recall message).
const sampleCamt056 = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.056.001.08">
  <FIToFIPmtCxlReq>
    <GrpHdr>
      <MsgId>CAMT056-TEST-001</MsgId>
      <CreDtTm>2026-10-03T15:00:00Z</CreDtTm>
      <NbOfTxs>1</NbOfTxs>
    </GrpHdr>
    <TxInf>
      <CxlId>CXL-001</CxlId>
      <OrgnlGrpInf>
        <OrgnlMsgId>PACS008-TEST-001</OrgnlMsgId>
      </OrgnlGrpInf>
    </TxInf>
  </FIToFIPmtCxlReq>
</Document>`

// TestDetectMXMsgType_Pacs008 — namespace detection for pacs.008.
func TestDetectMXMsgType_Pacs008(t *testing.T) {
	got := DetectMXMsgType([]byte(samplePacs008))
	if got != "pacs.008" {
		t.Errorf("DetectMXMsgType pacs.008: got %q, want %q", got, "pacs.008")
	}
}

// TestDetectMXMsgType_Pacs009 — namespace detection for pacs.009.
func TestDetectMXMsgType_Pacs009(t *testing.T) {
	got := DetectMXMsgType([]byte(samplePacs009))
	if got != "pacs.009" {
		t.Errorf("DetectMXMsgType pacs.009: got %q, want %q", got, "pacs.009")
	}
}

// TestDetectMXMsgType_Camt056 — namespace detection for camt.056 (recall).
func TestDetectMXMsgType_Camt056(t *testing.T) {
	got := DetectMXMsgType([]byte(sampleCamt056))
	if got != "camt.056" {
		t.Errorf("DetectMXMsgType camt.056: got %q, want %q", got, "camt.056")
	}
}

// TestDetectMXMsgType_Unknown — unknown/garbage XML must not panic.
func TestDetectMXMsgType_Unknown(t *testing.T) {
	got := DetectMXMsgType([]byte(`<Document xmlns="urn:some:other:schema"><Foo/></Document>`))
	if got == "pacs.008" || got == "pacs.009" || got == "camt.056" {
		t.Errorf("DetectMXMsgType unknown: should not match known type, got %q", got)
	}
}

// TestParseMX_Pacs008_MsgID — ParseMX extracts MsgId from GrpHdr.
func TestParseMX_Pacs008_MsgID(t *testing.T) {
	msg, err := ParseMX([]byte(samplePacs008))
	if err != nil {
		t.Fatalf("ParseMX pacs.008: unexpected error: %v", err)
	}
	if msg.MsgType != "pacs.008" {
		t.Errorf("MsgType: got %q, want %q", msg.MsgType, "pacs.008")
	}
	if msg.MsgID != "PACS008-TEST-001" {
		t.Errorf("MsgID: got %q, want %q", msg.MsgID, "PACS008-TEST-001")
	}
}

// TestParseMX_Pacs008_CreDtTm — ParseMX parses creation datetime.
func TestParseMX_Pacs008_CreDtTm(t *testing.T) {
	msg, err := ParseMX([]byte(samplePacs008))
	if err != nil {
		t.Fatalf("ParseMX pacs.008: %v", err)
	}
	want := time.Date(2026, 10, 3, 14, 30, 0, 0, time.UTC)
	if !msg.CreDtTm.Equal(want) {
		t.Errorf("CreDtTm: got %v, want %v", msg.CreDtTm, want)
	}
}

// TestParseMX_Pacs008_FieldExtraction — ParseMX extracts key XPath fields
// that the seed maps onto semantic terms (EndToEndId, UETR, settlement amount).
func TestParseMX_Pacs008_FieldExtraction(t *testing.T) {
	msg, err := ParseMX([]byte(samplePacs008))
	if err != nil {
		t.Fatalf("ParseMX pacs.008: %v", err)
	}

	// These XPath keys must match what the seed file registers in swift_field_map.
	// If they drift, the field_map tile silently misses them — this test catches that.
	checks := map[string]string{
		"Document/FIToFICstmrCdtTrf/GrpHdr/MsgId":                         "PACS008-TEST-001",
		"Document/FIToFICstmrCdtTrf/CdtTrfTxInf/PmtId/EndToEndId":         "E2E-REF-TEST-001",
		"Document/FIToFICstmrCdtTrf/CdtTrfTxInf/PmtId/UETR":               "550e8400-e29b-41d4-a716-446655440000",
		"Document/FIToFICstmrCdtTrf/CdtTrfTxInf/IntrBkSttlmAmt":           "1000000.00",
		"Document/FIToFICstmrCdtTrf/CdtTrfTxInf/Dbtr/FinInstnId/BICFI":    "TESTUS33XXX",
		"Document/FIToFICstmrCdtTrf/CdtTrfTxInf/Cdtr/FinInstnId/BICFI":    "CUSTGB2LXXX",
		"Document/FIToFICstmrCdtTrf/CdtTrfTxInf/IntrBkSttlmDt":            "2026-10-03",
	}
	for xpath, want := range checks {
		got, ok := msg.Fields[xpath]
		if !ok {
			t.Errorf("field %q: not found in parsed output (field_map tile will miss it)", xpath)
			continue
		}
		if got != want {
			t.Errorf("field %q: got %q, want %q", xpath, got, want)
		}
	}
}

// TestParseMX_Camt056_MsgID — ParseMX handles camt.056 recall message.
func TestParseMX_Camt056_MsgID(t *testing.T) {
	msg, err := ParseMX([]byte(sampleCamt056))
	if err != nil {
		t.Fatalf("ParseMX camt.056: %v", err)
	}
	if msg.MsgType != "camt.056" {
		t.Errorf("MsgType: got %q, want %q", msg.MsgType, "camt.056")
	}
	if msg.MsgID != "CAMT056-TEST-001" {
		t.Errorf("MsgID: got %q, want %q", msg.MsgID, "CAMT056-TEST-001")
	}
	// OriginalMsgId must be extractable — SWIFTRecallActivity needs it
	key := "Document/FIToFIPmtCxlReq/TxInf/OrgnlGrpInf/OrgnlMsgId"
	if got := msg.Fields[key]; got != "PACS008-TEST-001" {
		t.Errorf("OrgnlMsgId: got %q, want %q", got, "PACS008-TEST-001")
	}
}

// TestParseMX_EmptyDoc — empty input must not panic, returns error or empty message.
func TestParseMX_EmptyDoc(t *testing.T) {
	msg, err := ParseMX([]byte{})
	// Either nil msg+error or empty msg+no-error is acceptable; no panic is the contract.
	if err == nil && msg == nil {
		t.Error("ParseMX empty: both msg and err are nil — expected at least one non-nil")
	}
}
