package workflows

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Activities implement the actual business logic for audit workflows

type AuditActivities struct {
	db                 *sqlx.DB
	dataQualityService *services.DataQualityService
}

// NewAuditActivities creates audit activity implementations
func NewAuditActivities(db *sqlx.DB) *AuditActivities {
	return &AuditActivities{
		db:                 db,
		dataQualityService: services.NewDataQualityService(db),
	}
}

// ComputeDataQualityActivity computes data quality metrics for sources
func (a *AuditActivities) ComputeDataQualityActivity(ctx context.Context, sources []string, tenantID string) (*services.DataQuality, error) {
	return a.dataQualityService.ComputeQuality(ctx, sources, tenantID)
}

// FetchLastHashActivity retrieves the last hash for a tenant
// TODO: Replace SQL with Hasura GraphQL query:
//
//	query GetLastHash($tenantId: uuid!) {
//	  audit_log(where: {tenant_id: {_eq: $tenantId}}, order_by: {timestamp: desc}, limit: 1) {
//	    hash
//	  }
//	}
//
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (a *AuditActivities) FetchLastHashActivity(ctx context.Context, tenantID string) (string, error) {
	var lastHash string
	query := `
		SELECT hash FROM audit_log 
		WHERE tenant_id = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`
	err := a.db.GetContext(ctx, &lastHash, query, tenantID)
	if err == sql.ErrNoRows {
		return "", nil // No previous hash
	}
	return lastHash, err
}

// ComputeHashActivity computes hash for audit entry including data quality
func (a *AuditActivities) ComputeHashActivity(ctx context.Context, event AuditEvent, prevHash string, dq *services.DataQuality) (string, error) {
	// Create hash input with all components
	hashInput := struct {
		Event       AuditEvent
		PrevHash    string
		DataQuality *services.DataQuality
	}{
		Event:       event,
		PrevHash:    prevHash,
		DataQuality: dq,
	}

	data, err := json.Marshal(hashInput)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// PersistAuditLogActivity saves audit entry with data quality to database
// TODO: Replace SQL with Hasura GraphQL mutation:
//
//	mutation InsertAuditLog($object: audit_log_insert_input!) {
//	  insert_audit_log_one(object: $object) {
//	    id
//	    hash
//	  }
//	}
//
// Variables: {"object": {"id": "...", "timestamp": "...", "tenant_id": "...", "user_id": "...",
//
//	"question": "...", "answer": "...", "provider": "...", "confidence": 0.95,
//	"sources": [...], "caveats": [...], "hash": "...", "prev_hash": "...",
//	"data_quality": {...}, "version": 1}}
//
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (a *AuditActivities) PersistAuditLogActivity(ctx context.Context, event AuditEvent, hash string, prevHash string, dq *services.DataQuality) error {
	dqJSON, err := json.Marshal(dq)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO audit_log (
			id, timestamp, tenant_id, user_id, question, answer,
			provider, confidence, sources, caveats,
			hash, prev_hash, data_quality, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	_, err = a.db.ExecContext(ctx, query,
		event.ID, event.Timestamp, event.TenantID, event.UserID,
		event.Question, event.Answer, event.Provider, event.Confidence,
		pq.Array(event.Sources), pq.Array(event.Caveats),
		hash, prevHash, dqJSON, event.Version,
	)

	return err
}

// UpdateLastHashActivity updates the last hash pointer for tenant
// TODO: Replace SQL with Hasura GraphQL mutation (upsert):
//
//	mutation UpsertTenantHash($tenantId: uuid!, $hash: String!) {
//	  insert_tenant_last_hash_one(
//	    object: {tenant_id: $tenantId, last_hash: $hash, updated_at: "now()"},
//	    on_conflict: {constraint: tenant_last_hash_pkey, update_columns: [last_hash, updated_at]}
//	  ) {
//	    tenant_id
//	    last_hash
//	  }
//	}
//
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (a *AuditActivities) UpdateLastHashActivity(ctx context.Context, tenantID string, newHash string) error {
	query := `
		INSERT INTO tenant_last_hash (tenant_id, last_hash, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (tenant_id) 
		DO UPDATE SET last_hash = $2, updated_at = NOW()
	`
	_, err := a.db.ExecContext(ctx, query, tenantID, newHash)
	return err
}

// FetchAuditLogsActivity retrieves all audit logs for a tenant
// TODO: Replace SQL with Hasura GraphQL query:
//
//	query GetAuditLogs($tenantId: uuid!) {
//	  audit_log(where: {tenant_id: {_eq: $tenantId}}, order_by: {timestamp: asc}) {
//	    id
//	    tenant_id
//	    question
//	    answer
//	    sources
//	    caveats
//	    hash
//	    prev_hash
//	    data_quality
//	    timestamp
//	  }
//	}
//
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (a *AuditActivities) FetchAuditLogsActivity(ctx context.Context, tenantID string) ([]AuditLogEntry, error) {
	query := `
		SELECT 
			id, tenant_id, question, answer, sources, caveats,
			hash, prev_hash, data_quality, timestamp
		FROM audit_log
		WHERE tenant_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := a.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLogEntry
	for rows.Next() {
		var log AuditLogEntry
		var dqJSON []byte

		err := rows.Scan(
			&log.ID, &log.TenantID, &log.Question, &log.Answer,
			pq.Array(&log.Sources), pq.Array(&log.Caveats),
			&log.Hash, &log.PrevHash, &dqJSON, &log.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if len(dqJSON) > 0 {
			var dq services.DataQuality
			if err := json.Unmarshal(dqJSON, &dq); err == nil {
				log.DataQuality = &dq
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// ValidateChainActivity validates hash chain integrity
func (a *AuditActivities) ValidateChainActivity(ctx context.Context, logs []AuditLogEntry) (bool, error) {
	for i := 1; i < len(logs); i++ {
		expectedPrev := logs[i-1].Hash
		actualPrev := logs[i].PrevHash

		if expectedPrev != actualPrev {
			return true, nil // Chain broken
		}
	}
	return false, nil // Chain intact
}

// ValidateSLAComplianceActivity checks data quality SLA violations
func (a *AuditActivities) ValidateSLAComplianceActivity(ctx context.Context, logs []AuditLogEntry) (int, error) {
	violations := 0

	for _, log := range logs {
		if log.DataQuality != nil {
			// Count RED freshness as SLA violation
			if log.DataQuality.FreshnessStatus == "RED" {
				violations++
			}

			// Count high null rates as violation (>10%)
			if log.DataQuality.NullRate > 0.10 {
				violations++
			}
		}
	}

	return violations, nil
}

// EmitAlertActivity sends alerts for chain or SLA violations
func (a *AuditActivities) EmitAlertActivity(ctx context.Context, tenantID string, chainBroken bool, slaViolations int) error {
	alert := fmt.Sprintf(
		"AUDIT ALERT for tenant %s: chain_broken=%v, sla_violations=%d",
		tenantID, chainBroken, slaViolations,
	)

	// In production, send to Slack, PagerDuty, email, etc.
	fmt.Println(alert)

	// Store alert in database
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation InsertAuditAlert($object: audit_alerts_insert_input!) {
	//   insert_audit_alerts_one(object: $object) {
	//     id
	//     alert_type
	//   }
	// }
	// Variables: {"object": {"tenant_id": "...", "alert_type": "critical|warning|info",
	//   "message": "...", "created_at": "now()"}}
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		INSERT INTO audit_alerts (tenant_id, alert_type, message, created_at)
		VALUES ($1, $2, $3, NOW())
	`

	alertType := "info"
	if chainBroken {
		alertType = "critical"
	} else if slaViolations > 10 {
		alertType = "warning"
	}

	_, err := a.db.ExecContext(ctx, query, tenantID, alertType, alert)
	return err
}
