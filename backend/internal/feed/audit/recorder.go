package audit

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/canonical"
)

// Service implements the AuditRecorder interface with hash chaining
type Service struct {
	mu      sync.RWMutex
	records map[string][]AuditRecord // traceID -> ordered records
}

func NewService() *Service {
	return &Service{
		records: make(map[string][]AuditRecord),
	}
}

func (s *Service) Record(traceID, eventType, actor, action, target string, logicSnapshot map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.records[traceID]
	sequenceNumber := len(records) + 1

	var previousHash string
	if len(records) > 0 {
		previousHash = records[len(records)-1].CurrentHash
	} else {
		previousHash = "genesis"
	}

	record := AuditRecord{
		TraceID:        traceID,
		Timestamp:      time.Now(),
		EventType:      eventType,
		Actor:          actor,
		Action:         action,
		Target:         target,
		LogicSnapshot:  logicSnapshot,
		PreviousHash:   previousHash,
		SequenceNumber: sequenceNumber,
	}

	// Calculate current hash using deterministic canonicalization
	record.CurrentHash = s.calculateHash(record)

	s.records[traceID] = append(records, record)
	return nil
}

func (s *Service) GetTrail(traceID string) ([]AuditRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, exists := s.records[traceID]
	if !exists {
		return []AuditRecord{}, nil
	}
	return records, nil
}

func (s *Service) GetEvidenceBundle(actionID string) (*EvidenceBundle, error) {
	// In this simplified version, actionID == traceID
	records, err := s.GetTrail(actionID)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no audit trail found for action: %s", actionID)
	}

	bundle := &EvidenceBundle{
		ActionID:       actionID,
		TraceID:        actionID,
		AuditRecords:   records,
		StartTime:      records[0].Timestamp,
		EndTime:        records[len(records)-1].Timestamp,
		HashChainValid: s.VerifyHashChain(records),
	}

	// Extract metadata from first and last records
	if firstSnapshot := records[0].LogicSnapshot; firstSnapshot != nil {
		if clientID, ok := firstSnapshot["client_id"].(string); ok {
			bundle.ClientID = clientID
		}
		if actionType, ok := firstSnapshot["action_type"].(string); ok {
			bundle.ActionType = actionType
		}
	}

	// Determine status from last record
	lastRecord := records[len(records)-1]
	switch lastRecord.EventType {
	case "trade_executed":
		bundle.Status = "executed"
	case "approval_created":
		bundle.Status = "pending"
	case "approval_decided":
		if decision, ok := lastRecord.LogicSnapshot["approved"].(bool); ok && decision {
			bundle.Status = "approved"
		} else {
			bundle.Status = "rejected"
		}
	}

	return bundle, nil
}

func (s *Service) VerifyHashChain(records []AuditRecord) bool {
	if len(records) == 0 {
		return true
	}

	for i, record := range records {
		expectedPreviousHash := "genesis"
		if i > 0 {
			expectedPreviousHash = records[i-1].CurrentHash
		}

		if record.PreviousHash != expectedPreviousHash {
			return false
		}

		// Recalculate hash and verify
		calculatedHash := s.calculateHash(record)
		if calculatedHash != record.CurrentHash {
			return false
		}
	}

	return true
}

func (s *Service) calculateHash(record AuditRecord) string {
	// Create a copy without current hash for hashing
	hashInput := struct {
		TraceID        string
		Timestamp      time.Time
		EventType      string
		Actor          string
		Action         string
		Target         string
		LogicSnapshot  map[string]interface{}
		PreviousHash   string
		SequenceNumber int
	}{
		TraceID:        record.TraceID,
		Timestamp:      record.Timestamp,
		EventType:      record.EventType,
		Actor:          record.Actor,
		Action:         record.Action,
		Target:         record.Target,
		LogicSnapshot:  record.LogicSnapshot,
		PreviousHash:   record.PreviousHash,
		SequenceNumber: record.SequenceNumber,
	}

	// Use deterministic canonical JSON for consistent hashing
	data, _ := canonical.MarshalDeterministic(hashInput)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

