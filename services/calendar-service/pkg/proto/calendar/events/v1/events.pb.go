// Package eventsv1 is a minimal stub of the calendar-service proto types
// that the upstream `internal/publisher` (Redpanda) and
// `services/trading-consumer` services depend on.
//
// The original `.proto` definitions and generated code were lost when
// `services/calendar-service/` was deleted (only README.md remains).
// Rather than break both consumers, this stub keeps the type names and
// the proto.Message contract, so existing call sites that already pass
// values around still compile.
//
// TODO(calendar-service-revival): Regenerate from the original .proto
// once it's reinstated. The wire format here is intentionally trivial
// (no business fields are persisted to disk in this stub).
package eventsv1

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CalendarEvent is the canonical event message produced by the publisher
// and consumed by trading-consumer. Stub structure — see TODO above.
//
// Field-naming follows the patterns the two existing call sites use
// (Region, Exchange, CalendarDate, IsBusinessDay, HolidayName, etc.) so
// the publisher's struct literals continue to satisfy the type checker.
type CalendarEvent struct {
	EventId         string                 `protobuf:"bytes,1,opt,name=event_id,proto3"`
	EventType       string                 `protobuf:"bytes,2,opt,name=event_type,proto3"`
	TenantId        string                 `protobuf:"bytes,3,opt,name=tenant_id,proto3"`
	Date            string                 `protobuf:"bytes,4,opt,name=date,proto3"` // ISO YYYY-MM-DD
	Holiday         string                 `protobuf:"bytes,5,opt,name=holiday,proto3"`
	IsBusiness      bool                   `protobuf:"varint,6,opt,name=is_business,proto3"`
	OccurredAt      *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=occurred_at,proto3"`
	Source          string                 `protobuf:"bytes,8,opt,name=source,proto3"`
	Region          string                 `protobuf:"bytes,9,opt,name=region,proto3"`
	Exchange        string                 `protobuf:"bytes,10,opt,name=exchange,proto3"`
	CalendarDate    string                 `protobuf:"bytes,11,opt,name=calendar_date,proto3"`
	IsBusinessDay   bool                   `protobuf:"varint,12,opt,name=is_business_day,proto3"`
	HolidayName     string                 `protobuf:"bytes,13,opt,name=holiday_name,proto3"`
	SourceSystem    string                 `protobuf:"bytes,14,opt,name=source_system,proto3"`
	ConfidenceScore int32                  `protobuf:"varint,15,opt,name=confidence_score,proto3"`

	RuleApplied           string `protobuf:"bytes,16,opt,name=rule_applied,proto3"`
	SemanticTermVersion   string `protobuf:"bytes,17,opt,name=semantic_term_version,proto3"`
	BusinessObjectVersion string `protobuf:"bytes,18,opt,name=business_object_version,proto3"`
}

func (m *CalendarEvent) Reset() { *m = CalendarEvent{} }
func (m *CalendarEvent) String() string {
	return fmt.Sprintf("CalendarEvent{event_id=%s event_type=%s tenant_id=%s date=%s}", m.EventId, m.EventType, m.TenantId, m.Date)
}
func (*CalendarEvent) ProtoMessage() {}

// CalendarConflict signals a conflict between two ingestion records.
type CalendarConflict struct {
	EventId    string                 `protobuf:"bytes,1,opt,name=event_id,proto3"`
	TenantId   string                 `protobuf:"bytes,2,opt,name=tenant_id,proto3"`
	Date       string                 `protobuf:"bytes,3,opt,name=date,proto3"`
	Reason     string                 `protobuf:"bytes,4,opt,name=reason,proto3"`
	SourceA    string                 `protobuf:"bytes,5,opt,name=source_a,proto3"`
	SourceB    string                 `protobuf:"bytes,6,opt,name=source_b,proto3"`
	Severity   string                 `protobuf:"bytes,7,opt,name=severity,proto3"`
	OccurredAt *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=occurred_at,proto3"`
	Region     string                 `protobuf:"bytes,9,opt,name=region,proto3"`
}

func (m *CalendarConflict) Reset() { *m = CalendarConflict{} }
func (m *CalendarConflict) String() string {
	return fmt.Sprintf("CalendarConflict{event_id=%s tenant_id=%s date=%s severity=%s}", m.EventId, m.TenantId, m.Date, m.Severity)
}
func (*CalendarConflict) ProtoMessage() {}

// IngestionEvent reports the start/complete lifecycle of an ingestion job.
type IngestionEvent struct {
	EventId            string                 `protobuf:"bytes,1,opt,name=event_id,proto3"`
	Phase              string                 `protobuf:"bytes,2,opt,name=phase,proto3"`
	TenantId           string                 `protobuf:"bytes,3,opt,name=tenant_id,proto3"`
	IngestionId        string                 `protobuf:"bytes,4,opt,name=ingestion_id,proto3"`
	RecordsCreated     int32                  `protobuf:"varint,5,opt,name=records_created,proto3"`
	RecordsUpdated     int32                  `protobuf:"varint,6,opt,name=records_updated,proto3"`
	RecordsDeleted     int32                  `protobuf:"varint,7,opt,name=records_deleted,proto3"`
	ConflictsDetected  int32                  `protobuf:"varint,8,opt,name=conflicts_detected,proto3"`
	Duration           int64                  `protobuf:"varint,9,opt,name=duration_ms,proto3"`
	TriggeredBy        string                 `protobuf:"bytes,10,opt,name=triggered_by,proto3"`
	OccurredAt         *timestamppb.Timestamp `protobuf:"bytes,11,opt,name=occurred_at,proto3"`
	Success            bool                   `protobuf:"varint,12,opt,name=success,proto3"`
	Region             string                 `protobuf:"bytes,13,opt,name=region,proto3"`
	Year               int32                  `protobuf:"varint,14,opt,name=year,proto3"`
	ForceRefresh       bool                   `protobuf:"varint,15,opt,name=force_refresh,proto3"`
	RecordsIngested    int32                  `protobuf:"varint,16,opt,name=records_ingested,proto3"`
	ConflictsResolved  int32                  `protobuf:"varint,17,opt,name=conflicts_resolved,proto3"`
	ConflictsEscalated int32                  `protobuf:"varint,18,opt,name=conflicts_escalated,proto3"`
	SourcesQueried     int32                  `protobuf:"varint,19,opt,name=sources_queried,proto3"`
	SourcesSucceeded   int32                  `protobuf:"varint,20,opt,name=sources_succeeded,proto3"`
	SourcesFailed      int32                  `protobuf:"varint,21,opt,name=sources_failed,proto3"`
	ErrorMessages      []string               `protobuf:"bytes,22,rep,name=error_messages,proto3"`
	DurationMs         int64                  `protobuf:"varint,23,opt,name=duration_ms,proto3"`
}

func (m *IngestionEvent) Reset() { *m = IngestionEvent{} }
func (m *IngestionEvent) String() string {
	return fmt.Sprintf("IngestionEvent{event_id=%s phase=%s tenant_id=%s}", m.EventId, m.Phase, m.TenantId)
}
func (*IngestionEvent) ProtoMessage() {}

// _ keeps the proto import live even if a future refactor drops the only
// direct use.
var _ = proto.Marshal
