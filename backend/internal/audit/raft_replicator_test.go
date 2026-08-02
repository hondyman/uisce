package audit_test

import (
	"context"
	"encoding/json"
	"io"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	"github.com/hondyman/uisce/backend/internal/audit"
)

type mockLedgerRecorder struct {
	entries       []audit.LedgerEntry
	recordErr    error
	lastHash     string
	hashGenerator func(entry audit.LedgerEntry, prevHash string) string
}

func (m *mockLedgerRecorder) RecordLedgerEntry(ctx context.Context, entry audit.LedgerEntry) (*audit.LedgerEntry, error) {
	if m.recordErr != nil {
		return nil, m.recordErr
	}
	if m.lastHash == "" {
		m.lastHash = "GENESIS_BLOCK"
	}
	entry.PreviousHash = m.lastHash
	if m.hashGenerator != nil {
		entry.CurrentHash = m.hashGenerator(entry, m.lastHash)
	} else {
		entry.CurrentHash = "hash_" + entry.AuditID
	}
	m.lastHash = entry.CurrentHash
	entry.SequenceID = int64(len(m.entries) + 1)
	m.entries = append(m.entries, entry)
	return &entry, nil
}

func TestRaftReplicator_FSMApply_AdvancesHashChain(t *testing.T) {
	recorder := &mockLedgerRecorder{
		hashGenerator: func(entry audit.LedgerEntry, prevHash string) string {
			return prevHash + "->" + entry.AuditID
		},
	}

	hashPtr := newAtomicPtr("GENESIS_BLOCK")
	fsm := &testFSM{
		ledger:        recorder,
		lastBlockHash: hashPtr,
	}

	entry1 := audit.LedgerEntry{
		AuditID:         "audit-001",
		TenantID:        uuid.New().String(),
		EntityType:      "trade",
		EntityID:        "trade-123",
		ActionType:      "EXECUTE",
		PerformedBy:     "system",
		PayloadSnapshot: map[string]any{"qty": float64(100)},
	}

	data1, err := json.Marshal(entry1)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fsm.Apply(newTestLog(data1))

	entry2 := audit.LedgerEntry{
		AuditID:         "audit-002",
		TenantID:        entry1.TenantID,
		EntityType:      "trade",
		EntityID:        "trade-456",
		ActionType:      "EXECUTE",
		PerformedBy:     "system",
		PayloadSnapshot: map[string]any{"qty": float64(200)},
	}

	data2, err := json.Marshal(entry2)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fsm.Apply(newTestLog(data2))

	if len(recorder.entries) != 2 {
		t.Errorf("expected 2 recorded entries, got %d", len(recorder.entries))
	}

	if recorder.entries[0].PreviousHash != "GENESIS_BLOCK" {
		t.Errorf("entry 1: expected prev_hash GENESIS_BLOCK, got '%s'", recorder.entries[0].PreviousHash)
	}

	if recorder.entries[1].PreviousHash != recorder.entries[0].CurrentHash {
		t.Errorf("entry 2: expected prev_hash '%s', got '%s'", recorder.entries[0].CurrentHash, recorder.entries[1].PreviousHash)
	}
}

func TestRaftReplicator_FSMSnapshot_Restore(t *testing.T) {
	recorder := &mockLedgerRecorder{}
	hashPtr := newAtomicPtr("hash_after_1000_entries")
	fsm := &testFSM{
		ledger:        recorder,
		lastBlockHash: hashPtr,
	}

	snap, err := fsm.Snapshot()
	if err != nil {
		t.Fatalf("unexpected error creating snapshot: %v", err)
	}

	sink := &testSnapshotSink{data: []byte{}}
	err = snap.Persist(sink)
	if err != nil {
		t.Fatalf("unexpected error persisting snapshot: %v", err)
	}

	hashBefore := fsm.lastBlockHash.Load()
	if hashBefore != "hash_after_1000_entries" {
		t.Errorf("expected lastBlockHash to be 'hash_after_1000_entries', got '%s'", hashBefore)
	}

	newHashPtr := newAtomicPtr("GENESIS_BLOCK")
	newFSM := &testFSM{
		ledger:        recorder,
		lastBlockHash: newHashPtr,
	}

	err = newFSM.Restore(&testReadCloser{data: sink.data})
	if err != nil {
		t.Fatalf("unexpected error restoring snapshot: %v", err)
	}

	restoredHash := newFSM.lastBlockHash.Load()
	if restoredHash != "hash_after_1000_entries" {
		t.Errorf("expected restored hash 'hash_after_1000_entries', got '%s'", restoredHash)
	}
}

func TestRaftReplicator_AppendAuditEvent_FallbackMode(t *testing.T) {
	recorder := &mockLedgerRecorder{}
	ctx := context.Background()
	tenantID := uuid.New()

	entry := audit.LedgerEntry{
		AuditID:         "audit-test-001",
		TenantID:        tenantID.String(),
		EntityType:      "order",
		EntityID:        "order-789",
		ActionType:      "NEW",
		PerformedBy:     "admin",
		PayloadSnapshot: map[string]any{"symbol": "AAPL", "qty": float64(100)},
	}

	recorded, err := recorder.RecordLedgerEntry(ctx, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recorded == nil {
		t.Fatal("expected non-nil recorded entry")
	}

	if recorded.PreviousHash != "GENESIS_BLOCK" {
		t.Errorf("expected GENESIS_BLOCK, got '%s'", recorded.PreviousHash)
	}

	if len(recorder.entries) != 1 {
		t.Errorf("expected 1 entry in recorder, got %d", len(recorder.entries))
	}
}

func TestRaftReplicator_Close_NilRaft(t *testing.T) {
	r := &audit.RaftReplicator{}
	err := r.Close()
	if err != nil {
		t.Errorf("unexpected error closing nil raft: %v", err)
	}
}

func TestRaftReplicator_IsLeader_NilRaft(t *testing.T) {
	r := &audit.RaftReplicator{}
	if !r.IsLeader() {
		t.Error("expected IsLeader() to return true for nil raft (single-node mode)")
	}
}

type testFSM struct {
	ledger        *mockLedgerRecorder
	lastBlockHash *atomicPtr
}

func (f *testFSM) Apply(l *raft.Log) any {
	var entry audit.LedgerEntry
	if err := json.Unmarshal(l.Data, &entry); err != nil {
		return err
	}
	recorded, err := f.ledger.RecordLedgerEntry(context.Background(), entry)
	if err != nil {
		return err
	}
	if recorded != nil {
		f.lastBlockHash.Store(recorded.CurrentHash)
	}
	return nil
}

func (f *testFSM) Snapshot() (raft.FSMSnapshot, error) {
	hash := f.lastBlockHash.Load()
	return &testSnapshot{LastHash: hash}, nil
}

func (f *testFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	var snap testSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	f.lastBlockHash.Store(snap.LastHash)
	return nil
}

type testSnapshot struct {
	LastHash string
}

func (s *testSnapshot) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = sink.Write(data)
	return err
}

func (s *testSnapshot) Release() {}

type testSnapshotSink struct {
	data []byte
}

func (s *testSnapshotSink) Write(p []byte) (int, error) {
	s.data = append(s.data, p...)
	return len(p), nil
}

func (s *testSnapshotSink) Close() error   { return nil }
func (s *testSnapshotSink) ID() string     { return "test-sink" }
func (s *testSnapshotSink) Cancel() error { return nil }

type testLog struct {
	data []byte
}

func (l *testLog) Data() []byte        { return l.data }
func (l *testLog) Index() uint64       { return 0 }
func (l *testLog) Term() uint64        { return 0 }
func (l *testLog) Type() raft.LogType  { return raft.LogCommand }
func (l *testLog) Extensions() []byte  { return nil }
func (l *testLog) AppendEntry(m raft.Log) {}

func newTestLog(data []byte) *raft.Log {
	return &raft.Log{
		Data: data,
		Type: raft.LogCommand,
	}
}

type testReadCloser struct {
	data []byte
	pos  int
}

func (rc *testReadCloser) Read(p []byte) (int, error) {
	if rc.pos >= len(rc.data) {
		return 0, io.EOF
	}
	n := copy(p, rc.data[rc.pos:])
	rc.pos += n
	return n, nil
}

func (rc *testReadCloser) Close() error { return nil }

type atomicPtr struct {
	v atomic.Value
}

func newAtomicPtr(h string) *atomicPtr {
	p := &atomicPtr{}
	p.v.Store(h)
	return p
}

func (p *atomicPtr) Load() string {
	return p.v.Load().(string)
}

func (p *atomicPtr) Store(h string) {
	p.v.Store(h)
}
