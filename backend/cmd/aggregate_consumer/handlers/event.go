package handlers

// Event is the canonical Debezium change-event payload delivered to handlers.
// Constructed by main.go from the raw JSON envelope; handlers never see the
// raw bytes except through the DLQ path.
type Event struct {
	Op             string         `json:"op"`              // c | u | d | r (snapshot read)
	TS             int64          `json:"ts_ms"`            // ts_ms from Debezium
	Table          string         `json:"table"`            // source.schema (or source.table for cross-DB events)
	LSN            string         `json:"lsn"`              // source.lsn, used for dedupe keying
	Before         map[string]any `json:"before,omitempty"` // full previous row (REPLICA IDENTITY FULL)
	After          map[string]any `json:"after,omitempty"`  // full current row
	ChangedColumns []string       `json:"-"`                // derived: columns whose value differs between Before and After; empty for create (all columns are "new")
}
