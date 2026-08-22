package analytics

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestEvolutionDaemon_DetectSymbologyFamily(t *testing.T) {
	daemon := NewMetadataEvolutionDaemon(nil, NewRuleBasedTermDisambiguator())

	// Test ISIN Name
	edgeType, family := daemon.DetectSymbologyFamily("isin_code", nil)
	if edgeType != "IS_PEER_IDENTIFIER_OF" || family != "ISO_6166_SECURITY_IDENTIFIERS" {
		t.Errorf("expected ISIN family, got %s, %s", edgeType, family)
	}

	// Test CUSIP Name
	edgeType, family = daemon.DetectSymbologyFamily("primary_cusip", nil)
	if edgeType != "IS_PEER_IDENTIFIER_OF" || family != "NORTH_AMERICA_SECURITY_IDENTIFIERS" {
		t.Errorf("expected CUSIP family, got %s, %s", edgeType, family)
	}

	// Test Sample Values Regex Check (US0378331005 for Apple ISIN)
	props := map[string]interface{}{
		"sample_values": []string{"US0378331005", "GB0002634946"},
	}
	propsBytes, _ := json.Marshal(props)

	edgeType, family = daemon.DetectSymbologyFamily("security_identifier", propsBytes)
	if edgeType != "IS_PEER_IDENTIFIER_OF" || family != "ISO_6166_SECURITY_IDENTIFIERS" {
		t.Errorf("expected ISIN family from sample values, got %s, %s", edgeType, family)
	}

	// Test CUSIP 9-char Sample Value
	cusipProps := map[string]interface{}{
		"sample_values": []string{"037833100"},
	}
	cusipPropsBytes, _ := json.Marshal(cusipProps)
	edgeType, family = daemon.DetectSymbologyFamily("id_val", cusipPropsBytes)
	if edgeType != "IS_PEER_IDENTIFIER_OF" || family != "NORTH_AMERICA_SECURITY_IDENTIFIERS" {
		t.Errorf("expected CUSIP family from sample values, got %s, %s", edgeType, family)
	}
}

func TestEvolutionDaemon_ClassifierRules(t *testing.T) {
	classifier := NewRuleBasedTermDisambiguator()

	// 1. Test Exact Synonyms
	rationale, isSynonym := classifier.CompareNodes("cust_id", "customer_identifier")
	if !isSynonym {
		t.Errorf("expected cust_id and customer_identifier to be recognized as synonyms")
	}

	// 2. Test Account Differentiations
	rationale, isSynonym = classifier.CompareNodes("Allocation Account Code", "Custodial Account Code")
	if isSynonym {
		t.Errorf("expected Allocation Account Code and Custodial Account Code to be differentiated, not synonymous")
	}
	if rationale == "" {
		t.Errorf("expected non-empty differentiation rationale")
	}

	// 3. Test Symbology Differentiations
	rationale, isSynonym = classifier.CompareNodes("ISIN", "CUSIP")
	if isSynonym {
		t.Errorf("expected ISIN and CUSIP to be differentiated peers, not exact synonyms")
	}
	if rationale == "" {
		t.Errorf("expected non-empty symbology differentiation rationale")
	}
}

func TestEvolutionDaemon_ProcessCatalogEvent_NilSafety(t *testing.T) {
	daemon := NewMetadataEvolutionDaemon(nil, nil)

	evt := CatalogMutationEvent{
		TenantID:     uuid.New(),
		DatasourceID: uuid.New(),
		NodeID:       uuid.New(),
		Action:       "INSERT",
	}

	err := daemon.ProcessCatalogEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("expected no error when processing event with nil db, got: %v", err)
	}
}
