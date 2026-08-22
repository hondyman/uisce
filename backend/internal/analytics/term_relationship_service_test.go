package analytics

import (
	"context"
	"testing"
)

func TestTermRelationshipService_FallbackDisambiguation_Account(t *testing.T) {
	svc := &TermRelationshipService{}

	disambig := svc.buildFallbackDisambiguation("Account Code")
	if disambig == nil {
		t.Fatalf("expected disambiguation result for Account Code, got nil")
	}

	if disambig.PrimaryTerm.TermName != "Account Code" {
		t.Errorf("expected primary term 'Account Code', got '%s'", disambig.PrimaryTerm.TermName)
	}

	if len(disambig.RelatedTerms) == 0 {
		t.Fatalf("expected related terms for Account Code, got 0")
	}

	foundAlloc := false
	foundCust := false
	for _, term := range disambig.RelatedTerms {
		if term.TermName == "Allocation Account Code" {
			foundAlloc = true
		}
		if term.TermName == "Custodial Account Code" {
			foundCust = true
		}
	}

	if !foundAlloc {
		t.Errorf("expected to find 'Allocation Account Code' in related terms")
	}
	if !foundCust {
		t.Errorf("expected to find 'Custodial Account Code' in related terms")
	}
}

func TestTermRelationshipService_FallbackDisambiguation_Symbology(t *testing.T) {
	svc := &TermRelationshipService{}

	disambig := svc.buildFallbackDisambiguation("ISIN")
	if disambig == nil {
		t.Fatalf("expected disambiguation result for ISIN, got nil")
	}

	if len(disambig.RelatedTerms) == 0 {
		t.Fatalf("expected related terms for ISIN, got 0")
	}

	foundCUSIP := false
	foundSEDOL := false
	foundFIGI := false
	for _, term := range disambig.RelatedTerms {
		if term.TermName == "CUSIP" {
			foundCUSIP = true
		}
		if term.TermName == "SEDOL" {
			foundSEDOL = true
		}
		if term.TermName == "FIGI" {
			foundFIGI = true
		}
	}

	if !foundCUSIP || !foundSEDOL || !foundFIGI {
		t.Errorf("expected to find CUSIP, SEDOL, and FIGI in symbology family")
	}
}

func TestTermRelationshipService_SuggestRelatedTermsForColumn(t *testing.T) {
	svc := &TermRelationshipService{}

	// Test account column suggestion
	res, err := svc.SuggestRelatedTermsForColumn(context.Background(), "t1", "ds1", "alloc_acct_no", "trades")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) == 0 {
		t.Fatalf("expected suggestions for alloc_acct_no, got none")
	}

	// Test cusip column suggestion
	resCusip, err := svc.SuggestRelatedTermsForColumn(context.Background(), "t1", "ds1", "cusip_id", "securities")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resCusip) == 0 {
		t.Fatalf("expected suggestions for cusip_id, got none")
	}
}

func TestTermRelationshipService_TaxonomyL3(t *testing.T) {
	svc := &TermRelationshipService{}

	// 1. Test ListL3Classifications
	classifications, err := svc.ListL3Classifications(context.Background(), "t1")
	if err != nil {
		t.Fatalf("unexpected error listing L3 classifications: %v", err)
	}
	if len(classifications) != 18 {
		t.Errorf("expected 18 L3 classifications, got %d", len(classifications))
	}

	// 2. Test SuggestL3Classification for Trade Allocation
	allocMatch := svc.SuggestL3Classification("Allocation Account Code", "alloc_code")
	if allocMatch == nil || allocMatch.Name != "Trade Allocation" {
		t.Errorf("expected suggestion 'Trade Allocation', got %+v", allocMatch)
	}
	if allocMatch.DomainName != "Trading & Execution (OMS/EMS)" {
		t.Errorf("expected Domain 'Trading & Execution (OMS/EMS)', got '%s'", allocMatch.DomainName)
	}

	// 3. Test SuggestL3Classification for Custodial Safekeeping
	custMatch := svc.SuggestL3Classification("Custodian Identifier", "custodian_id")
	if custMatch == nil || custMatch.Name != "Custodial Safekeeping" {
		t.Errorf("expected suggestion 'Custodial Safekeeping', got %+v", custMatch)
	}

	// 4. Test SuggestL3Classification for Symbology
	symbMatch := svc.SuggestL3Classification("ISIN", "security_isin")
	if symbMatch == nil || symbMatch.Name != "Instrument Symbology" {
		t.Errorf("expected suggestion 'Instrument Symbology', got %+v", symbMatch)
	}

	// 5. Test SuggestL3Classification for Monetary Amounts
	amtMatch := svc.SuggestL3Classification("Currency Code", "asset_crrncy_code")
	if amtMatch == nil || amtMatch.Name != "Monetary Amounts" {
		t.Errorf("expected suggestion 'Monetary Amounts', got %+v", amtMatch)
	}
}

func TestTermRelationshipService_VisualizeLens(t *testing.T) {
	svc := &TermRelationshipService{}

	// Test 1: VisualizeLens for ISIN in SUBTYPE_AND_PEERS
	res, err := svc.VisualizeLens(context.Background(), "test-tenant", "ISIN", VisualizeLensRequest{
		LensType: LensSubtypeAndPeers,
	})
	if err != nil {
		t.Fatalf("unexpected error visualizing ISIN: %v", err)
	}
	if res.FocalNodeID != "ISIN" {
		t.Errorf("expected FocalNodeID 'ISIN', got '%s'", res.FocalNodeID)
	}
	if len(res.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes for ISIN subtype lens, got %d", len(res.Nodes))
	}
	foundISIN := false
	for _, n := range res.Nodes {
		if n.NodeName == "ISIN" && n.IsFocal {
			foundISIN = true
		}
	}
	if !foundISIN {
		t.Errorf("expected focal node to be ISIN")
	}

	// Test 2: VisualizeLens for Arbitrary Term in TAXONOMY_HIERARCHY
	resTax, err := svc.VisualizeLens(context.Background(), "test-tenant", "Trade Allocation", VisualizeLensRequest{
		LensType: LensTaxonomyHierarchy,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resTax.Nodes) != 4 {
		t.Errorf("expected 4 nodes (L1, L2, L3, Focal) in taxonomy hierarchy, got %d", len(resTax.Nodes))
	}
	if len(resTax.Edges) != 3 {
		t.Errorf("expected 3 edges in taxonomy hierarchy, got %d", len(resTax.Edges))
	}

	// Test 3: VisualizeLens for Metric in SEMANTIC_CALCULATION_MESH
	resCalc, err := svc.VisualizeLens(context.Background(), "test-tenant", "Total Realized PnL", VisualizeLensRequest{
		LensType: LensSemanticCalculationMesh,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resCalc.Nodes) == 0 || len(resCalc.Edges) == 0 {
		t.Errorf("expected calculation mesh nodes and edges for Total Realized PnL")
	}

	// Test 4: VisualizeLens for Term in PHYSICAL_ERD
	resERD, err := svc.VisualizeLens(context.Background(), "test-tenant", "Order Direction", VisualizeLensRequest{
		LensType: LensPhysicalERD,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resERD.Nodes) < 2 {
		t.Errorf("expected at least 2 tables in ERD, got %d", len(resERD.Nodes))
	}

	// Test 5: VisualizeLens for Term in PIPELINE_IMPACT
	resPipe, err := svc.VisualizeLens(context.Background(), "test-tenant", "Execution Price", VisualizeLensRequest{
		LensType: LensPipelineImpact,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resPipe.Nodes) != 5 {
		t.Errorf("expected 5 pipeline tiers, got %d", len(resPipe.Nodes))
	}
}
