package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

type RaftReplicator struct {
	raft           *raft.Raft
	ledger         LedgerRecorder
	lastBlockHash  atomic.Pointer[string]
	snapshotHash   atomic.Pointer[string]
	bootstrapPeers []raft.Server
}

type LedgerRecorder interface {
	RecordLedgerEntry(ctx context.Context, entry LedgerEntry) (*LedgerEntry, error)
}

type ledgerServiceAdapter struct {
	inner *LedgerService
}

func (a *ledgerServiceAdapter) RecordLedgerEntry(ctx context.Context, entry LedgerEntry) (*LedgerEntry, error) {
	return a.inner.RecordLedgerEntry(ctx, entry)
}

type replicatorFSM struct {
	ledger       LedgerRecorder
	lastBlockHash *atomic.Pointer[string]
}

func (f *replicatorFSM) Apply(l *raft.Log) any {
	var entry LedgerEntry
	if err := json.Unmarshal(l.Data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal ledger entry: %w", err)
	}
	recorded, err := f.ledger.RecordLedgerEntry(context.Background(), entry)
	if err != nil {
		return err
	}
	if recorded != nil {
		h := recorded.CurrentHash
		f.lastBlockHash.Store(&h)
	}
	return nil
}

func (f *replicatorFSM) Snapshot() (raft.FSMSnapshot, error) {
	hash := f.lastBlockHash.Load()
	if hash == nil {
		h := "GENESIS_BLOCK"
		hash = &h
	}
	return &replicatorSnapshot{LastHash: *hash}, nil
}

func (f *replicatorFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("failed to read snapshot data: %w", err)
	}
	var snap replicatorSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}
	h := snap.LastHash
	f.lastBlockHash.Store(&h)
	return nil
}

type replicatorSnapshot struct {
	LastHash string
}

func (s *replicatorSnapshot) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}
	_, err = sink.Write(data)
	return err
}

func (s *replicatorSnapshot) Release() {}

type RaftConfig struct {
	NodeID       string
	RaftAddr     string
	DataDir      string
	BindAddr     string
	BootstrapPeers []string
	JoinAddrs    []string
}

func NewRaftReplicator(ledger *LedgerService, cfg *RaftConfig) (*RaftReplicator, error) {
	if cfg.NodeID == "" {
		return nil, fmt.Errorf("RaftConfig.NodeID is required")
	}
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("RaftConfig.DataDir is required")
	}

	adapter := &ledgerServiceAdapter{inner: ledger}

	lastHash := "GENESIS_BLOCK"
	r := &RaftReplicator{
		ledger:        adapter,
		bootstrapPeers: make([]raft.Server, 0),
	}
	r.lastBlockHash.Store(&lastHash)

	if cfg.RaftAddr == "" {
		return r, nil
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(cfg.NodeID)

	bindHost := "127.0.0.1"
	bindPort := "0"
	if cfg.BindAddr != "" {
		bindHost, bindPort, _ = splitHostPort(cfg.BindAddr)
	} else if cfg.RaftAddr != "" {
		bindHost, bindPort, _ = splitHostPort(cfg.RaftAddr)
	}
	raftAddrHost, _, _ := splitHostPort(cfg.RaftAddr)
	if raftAddrHost != "" && bindHost == "127.0.0.1" {
		bindHost = raftAddrHost
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", bindHost, bindPort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve raft tcp addr: %w", err)
	}

	transport, err := raft.NewTCPTransport(cfg.RaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft tcp transport: %w", err)
	}

	snapshotDir := filepath.Join(cfg.DataDir, "raft_snapshots")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create raft snapshot dir: %w", err)
	}

	logDir := filepath.Join(cfg.DataDir, "raft_log.db")
	logStore, err := raftboltdb.NewBoltStore(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize raft bolt store: %w", err)
	}

	snapshots, err := raft.NewFileSnapshotStore(snapshotDir, 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize raft snapshots: %w", err)
	}

	fsm := &replicatorFSM{
		ledger:        adapter,
		lastBlockHash: &r.lastBlockHash,
	}

	initialCfg := raft.Configuration{
		Servers: []raft.Server{{
			ID:      raft.ServerID(cfg.NodeID),
			Address: raft.ServerAddress(cfg.RaftAddr),
		}},
	}
	raftInstance, err := raft.NewRaft(config, fsm, logStore, logStore, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to start raft instance: %w", err)
	}

	r.raft = raftInstance

	if len(cfg.JoinAddrs) == 0 {
		bootstrapFuture := r.raft.BootstrapCluster(initialCfg)
		if err := bootstrapFuture.Error(); err != nil {
			return nil, fmt.Errorf("failed to bootstrap raft cluster: %w", err)
		}
	} else {
		for _, joinAddr := range cfg.JoinAddrs {
			addPeerFuture := r.raft.AddPeer(raft.ServerAddress(joinAddr))
			if err := addPeerFuture.Error(); err != nil {
				log.Printf("[Raft] failed to add peer %s: %v (will retry on next entry)", joinAddr, err)
			}
		}
	}

	return r, nil
}

func splitHostPort(addr string) (host, port string, err error) {
	_ = addr
	if idx := lastIndexByte(addr, ':'); idx >= 0 {
		host = addr[:idx]
		port = addr[idx+1:]
	} else {
		host = addr
		port = "0"
	}
	return host, port, nil
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func (r *RaftReplicator) AppendAuditEvent(
	ctx context.Context,
	tenantID uuid.UUID,
	entityType string,
	entityID string,
	actionType string,
	performedBy string,
	payload map[string]any,
) (*LedgerEntry, error) {
	entry := LedgerEntry{
		AuditID:         uuid.New().String(),
		TenantID:        tenantID.String(),
		EntityType:      entityType,
		EntityID:        entityID,
		ActionType:      actionType,
		PayloadSnapshot: payload,
		PerformedBy:     performedBy,
	}

	if r.raft == nil {
		return r.ledger.RecordLedgerEntry(ctx, entry)
	}

	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry for raft: %w", err)
	}

	future := r.raft.Apply(entryBytes, 5*time.Second)
	if err := future.Error(); err != nil {
		return nil, fmt.Errorf("raft consensus replication failed: %w", err)
	}

	result := future.Response()
	if err, ok := result.(error); ok && err != nil {
		return nil, err
	}

	return r.ledger.RecordLedgerEntry(ctx, entry)
}

func (r *RaftReplicator) IsLeader() bool {
	if r.raft == nil {
		return true
	}
	return r.raft.State() == raft.Leader
}

func (r *RaftReplicator) Close() error {
	if r.raft != nil {
		f := r.raft.Shutdown()
		return f.Error()
	}
	return nil
}
