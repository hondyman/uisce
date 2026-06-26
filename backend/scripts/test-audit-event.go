package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

func main() {
	log.Println("Testing Audit Event Publishing...")

	// Initialize publisher
	publisher, err := audit.InitializeAuditPublisher("localhost:19092")
	if err != nil {
		log.Fatalf("Failed to initialize publisher: %v", err)
	}
	defer publisher.Close()

	ctx := context.Background()

	// Test 1: Job Run Event
	log.Println("\n📋 Publishing test job run event...")
	jobRunEvent := audit.JobRunCompletedEvent{
		RunID:        uuid.New().String(),
		JobID:        "test-positions-preagg",
		TenantID:     "tenant-test-001",
		StartTS:      time.Now().Add(-5 * time.Minute),
		EndTS:        time.Now(),
		Status:       audit.JobStatusFailed,
		ErrorMessage: "Test error: null pointer in activity",
		SemanticContext: json.RawMessage(`{
			"semantic_terms": ["st-positions", "st-client-address"],
			"business_objects": ["Positions", "Client"]
		}`),
		ComplianceContext: json.RawMessage(`{
			"pii_fields": ["client_address"],
			"sensitivity": "HIGH",
			"residency": "EU"
		}`),
		SLOContext: json.RawMessage(`{
			"target_duration_seconds": 300,
			"actual_duration_seconds": 310,
			"breach": true
		}`),
		Metadata: json.RawMessage(`{
			"retry_count": 2,
			"worker_queue": "default"
		}`),
	}

	if err := publisher.PublishJobRun(ctx, jobRunEvent); err != nil {
		log.Printf("❌ Failed to publish job run: %v", err)
	} else {
		log.Println("✅ Job run event published successfully")
	}

	// Test 2: Compliance Violation Event
	log.Println("\n🚨 Publishing test compliance violation...")
	violationEvent := audit.ComplianceViolationEvent{
		ViolationID:     uuid.New().String(),
		TenantID:        "tenant-test-001",
		JobRunID:        jobRunEvent.RunID,
		ViolationType:   "PII_EXPOSURE",
		ViolatedAt:      time.Now(),
		Severity:        audit.ViolationSeverityHigh,
		PIIExposed:      false,
		AffectedRecords: 0,
		ComplianceRefs:  []string{"GDPR", "CCPA"},
		Narrative:       "Job attempted to process EU data without proper residency check",
		Metadata: json.RawMessage(`{
			"detected_by": "compliance_engine",
			"auto_blocked": true
		}`),
	}

	if err := publisher.PublishComplianceViolation(ctx, violationEvent); err != nil {
		log.Printf("❌ Failed to publish violation: %v", err)
	} else {
		log.Println("✅ Compliance violation published successfully")
	}

	// Test 3: ChangeSet Event
	log.Println("\n📝 Publishing test changeset...")
	changeSetEvent := audit.ChangeSetCreatedEvent{
		ChangesetID: uuid.New().String(),
		Type:        "BUSINESS_TERM_UPDATE",
		Actor:       "test.user@example.com",
		TenantID:    "tenant-test-001",
		CreatedAt:   time.Now(),
		SemanticImpact: json.RawMessage(`{
			"affected_terms": ["st-client-address"],
			"affected_jobs": ["test-positions-preagg"],
			"affected_dags": ["daily-positions"]
		}`),
		ComplianceImpact: json.RawMessage(`{
			"new_restrictions": ["EU_RESIDENCY_REQUIRED"],
			"affected_regulations": ["GDPR"]
		}`),
		TenantImpact: json.RawMessage(`{
			"type": "SINGLE",
			"tenants": ["tenant-test-001"]
		}`),
		AISummary: json.RawMessage(`{
			"title": "Client Address sensitivity elevated to HIGH",
			"description": "This change requires all jobs processing client addresses to enforce EU residency checks"
		}`),
		AIRisk: json.RawMessage(`{
			"riskLevel": "MEDIUM",
			"riskScore": 0.6,
			"rationale": "Changes compliance posture for 14 jobs"
		}`),
		PayloadOld: json.RawMessage(`{
			"sensitivity": "MEDIUM"
		}`),
		PayloadNew: json.RawMessage(`{
			"sensitivity": "HIGH"
		}`),
		Approvers: []string{"compliance@example.com", "data.steward@example.com"},
		Status:    audit.ChangeSetStatusPending,
	}

	if err := publisher.PublishChangeSet(ctx, changeSetEvent); err != nil {
		log.Printf("❌ Failed to publish changeset: %v", err)
	} else {
		log.Println("✅ ChangeSet published successfully")
	}

	// Test 4: Semantic Snapshot Event
	log.Println("\n🔍 Publishing test semantic snapshot...")
	snapshotEvent := audit.SemanticSnapshotEvent{
		SnapshotID:     uuid.New().String(),
		SemanticTermID: "st-client-address",
		Version:        43,
		Timestamp:      time.Now(),
		Definition:     "Physical address of the client",
		BusinessTermID: "bt-client-address",
		TenantID:       "tenant-test-001",
		Compliance: json.RawMessage(`{
			"sensitivity": "HIGH",
			"pii": true,
			"regulations": ["GDPR", "CCPA"]
		}`),
		Lineage: json.RawMessage(`{
			"upstream": ["raw.clients.address"],
			"downstream": ["positions_view", "client_report"]
		}`),
		Metadata: json.RawMessage(`{
			"changed_by": "test.user@example.com",
			"change_reason": "Compliance review"
		}`),
	}

	if err := publisher.PublishSemanticSnapshot(ctx, snapshotEvent); err != nil {
		log.Printf("❌ Failed to publish snapshot: %v", err)
	} else {
		log.Println("✅ Semantic snapshot published successfully")
	}

	log.Println("\n✅ All test events published successfully!")

	log.Println("\n📊 Next steps:")
	log.Println("1. Check Redpanda Console: http://localhost:8080")
	log.Println("2. Query in Trino:")
	log.Println("   docker exec -it audit-trino trino")
	log.Println("   USE iceberg.audit;")
	log.Println("   SELECT * FROM scheduler_job_runs;")
	log.Println("3. Test API:")
	log.Println("   curl -H 'X-Tenant-ID: tenant-test-001' http://localhost:8080/api/audit/job-runs")
}
