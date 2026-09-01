package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestStatementFreezerService_SealVerifyAndTamperDetection(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	service := reporting.NewStatementFreezerService(nil, "test-secret-key-123")

	effectiveDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	rawArrowBatch := []byte("mock-arrow-columnar-record-batch-bytes")
	astPlanJSON := []byte(`{"calculation":"HurdleWaterfall","gamma":0.20}`)
	pdfBinary := []byte("%PDF-1.4 mock pdf content for institutional statement")

	// 1. Seal Statement & Generate Passport
	passport, err := service.FreezeAndSealStatement(
		ctx,
		tenantID,
		"STMT-2026-Q1-001",
		effectiveDate,
		"Chief Compliance Officer (CCO)",
		rawArrowBatch,
		astPlanJSON,
		pdfBinary,
		"s3://vault/statements/2026-Q1-001.pdf",
	)
	if err != nil {
		t.Fatalf("sealing failed: %v", err)
	}

	if passport.Status != reporting.SnapshotStatusSealed {
		t.Errorf("expected status %s, got %s", reporting.SnapshotStatusSealed, passport.Status)
	}

	// 2. Verify Valid Unaltered Artifact
	isValid, reason := service.VerifyStatementIntegrity(*passport, pdfBinary)
	if !isValid {
		t.Fatalf("expected valid verification, failed with: %s", reason)
	}

	// 3. Verify Tamper Detection on Modified Binary
	tamperedPDF := []byte("%PDF-1.4 TAMPERED content by bad actor")
	isTamperedValid, tamperReason := service.VerifyStatementIntegrity(*passport, tamperedPDF)
	if isTamperedValid {
		t.Fatalf("security violation: tampered binary passed verification")
	}

	if tamperReason == "" {
		t.Errorf("expected tamper reason explanation")
	}
}
