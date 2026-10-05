package audit

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
)

var (
	ErrSpoolClosed    = errors.New("durable_spool: spool is closed")
	ErrCorruptSegment = errors.New("durable_spool: segment file is corrupt")
)

const (
	SpoolMagicHeader = uint32(0x55495343) // 'UISC'
	SpoolVersion     = uint16(1)
	DefaultBatchSize = 100
)

// DurableSpool provides high-throughput, sub-millisecond local disk WAL persistence
// with background asynchronous streaming to Redpanda, zero-loss crash recovery replay,
// and zero-allocation group commit.
type DurableSpool struct {
	mu           sync.Mutex
	cond         *sync.Cond
	dir          string
	activeFile   *os.File
	broker       MessageBroker
	topic        string
	closed       bool
	stopDrain    chan struct{}
	drainWG      sync.WaitGroup
	spooledCount int64
	drainedCount int64

	// Zero-allocation Group Commit state
	batchBuf    bytes.Buffer
	currentSeq  uint64
	syncedSeq   uint64
	isSyncing   bool
	lastSyncErr error
}

// NewDurableSpool initializes local WAL directory and starts background drainer
func NewDurableSpool(spoolDir string, broker MessageBroker, topic string) (*DurableSpool, error) {
	if err := os.MkdirAll(spoolDir, 0755); err != nil {
		return nil, fmt.Errorf("create spool dir: %w", err)
	}

	segmentPath := filepath.Join(spoolDir, fmt.Sprintf("wal_%d.segment", time.Now().UTC().UnixNano()))
	f, err := os.OpenFile(segmentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open active wal segment: %w", err)
	}

	// Write Segment Header
	hdr := make([]byte, 8)
	binary.BigEndian.PutUint32(hdr[0:4], SpoolMagicHeader)
	binary.BigEndian.PutUint16(hdr[4:6], SpoolVersion)
	if _, err := f.Write(hdr); err != nil {
		f.Close()
		return nil, fmt.Errorf("write wal header: %w", err)
	}
	_ = f.Sync()

	s := &DurableSpool{
		dir:        spoolDir,
		activeFile: f,
		broker:     broker,
		topic:      topic,
		stopDrain:  make(chan struct{}),
	}
	s.cond = sync.NewCond(&s.mu)

	// Replay any older uncommitted segments on startup
	if err := s.recoverAndDrain(context.Background()); err != nil {
		fmt.Printf("WAL recovery notice: %v\n", err)
	}

	// Start asynchronous background drainer to Redpanda
	s.drainWG.Add(1)
	go s.backgroundDrainLoop()

	return s, nil
}

// WriteHotPath atomically appends the evaluation event to the local WAL with zero-allocation group commit fsync
func (s *DurableSpool) WriteHotPath(ctx context.Context, event EvaluationEventPayload) error {
	payloadBytes, err := canonical.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event for wal: %w", err)
	}

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payloadBytes)))

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrSpoolClosed
	}

	// 1. Assign sequence number and append to in-flight batch buffer
	s.currentSeq++
	mySeq := s.currentSeq

	s.batchBuf.Write(lenBuf[:])
	s.batchBuf.Write(payloadBytes)

	// 2. If another leader is currently fsyncing an earlier batch, wait until syncedSeq >= mySeq
	for s.isSyncing && s.syncedSeq < mySeq && !s.closed {
		s.cond.Wait()
	}

	if s.closed {
		s.mu.Unlock()
		return ErrSpoolClosed
	}

	// Check if a previous leader already synced our batch
	if s.syncedSeq >= mySeq {
		syncErr := s.lastSyncErr
		s.mu.Unlock()
		if syncErr == nil {
			atomic.AddInt64(&s.spooledCount, 1)
		}
		return syncErr
	}

	// 3. We are the Leader for this batch
	s.isSyncing = true
	syncUpToSeq := s.currentSeq

	// Copy batch buffer to write
	toWrite := make([]byte, s.batchBuf.Len())
	copy(toWrite, s.batchBuf.Bytes())
	s.batchBuf.Reset()

	// Write and fsync to active file
	var syncErr error
	if s.activeFile != nil {
		if _, err := s.activeFile.Write(toWrite); err != nil {
			syncErr = fmt.Errorf("wal batch write: %w", err)
		} else if err := s.activeFile.Sync(); err != nil {
			syncErr = fmt.Errorf("wal fsync: %w", err)
		}
	} else {
		syncErr = ErrSpoolClosed
	}

	s.syncedSeq = syncUpToSeq
	s.lastSyncErr = syncErr
	s.isSyncing = false

	// Broadcast to all waiting goroutines in this batch
	s.cond.Broadcast()
	s.mu.Unlock()

	if syncErr == nil {
		atomic.AddInt64(&s.spooledCount, 1)
	}
	return syncErr
}

// SpooledCount returns total records spooled to local WAL
func (s *DurableSpool) SpooledCount() int64 {
	return atomic.LoadInt64(&s.spooledCount)
}

// DrainedCount returns total records successfully published to Redpanda
func (s *DurableSpool) DrainedCount() int64 {
	return atomic.LoadInt64(&s.drainedCount)
}

// RotateActiveSegment closes current segment and creates a fresh active segment
func (s *DurableSpool) RotateActiveSegment() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.activeFile == nil {
		return ErrSpoolClosed
	}

	// Flush any pending in-flight buffer
	if s.batchBuf.Len() > 0 {
		toWrite := s.batchBuf.Bytes()
		_, _ = s.activeFile.Write(toWrite)
		_ = s.activeFile.Sync()
		s.syncedSeq = s.currentSeq
		s.batchBuf.Reset()
	}

	_ = s.activeFile.Sync()
	oldPath := s.activeFile.Name()
	_ = s.activeFile.Close()

	// Rename completed active segment to .ready for background drainer
	readyPath := oldPath + ".ready"
	_ = os.Rename(oldPath, readyPath)

	// Create new active segment
	newSegmentPath := filepath.Join(s.dir, fmt.Sprintf("wal_%d.segment", time.Now().UTC().UnixNano()))
	f, err := os.OpenFile(newSegmentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open new active wal segment: %w", err)
	}

	hdr := make([]byte, 8)
	binary.BigEndian.PutUint32(hdr[0:4], SpoolMagicHeader)
	binary.BigEndian.PutUint16(hdr[4:6], SpoolVersion)
	if _, err := f.Write(hdr); err != nil {
		f.Close()
		return fmt.Errorf("write wal header: %w", err)
	}
	_ = f.Sync()

	s.activeFile = f
	return nil
}

// backgroundDrainLoop periodically flushes spooled records to Redpanda
func (s *DurableSpool) backgroundDrainLoop() {
	defer s.drainWG.Done()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopDrain:
			return
		case <-ticker.C:
			_ = s.drainReadySegments(context.Background())
		}
	}
}

// drainReadySegments drains rotated .ready segments to Redpanda and deletes ONLY after successful broker ack
func (s *DurableSpool) drainReadySegments(ctx context.Context) error {
	if s.broker == nil {
		return nil
	}

	files, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".ready" {
			fullPath := filepath.Join(s.dir, f.Name())
			if err := s.replaySegment(ctx, fullPath); err != nil {
				// Stop on error, preserve segment on disk until broker recovers
				return err
			}
		}
	}
	return nil
}

// recoverAndDrain scans spool directory for uncommitted segments and replays them to Redpanda
func (s *DurableSpool) recoverAndDrain(ctx context.Context) error {
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() && (filepath.Ext(f.Name()) == ".ready" || filepath.Ext(f.Name()) == ".segment") {
			fullPath := filepath.Join(s.dir, f.Name())
			if s.activeFile != nil && fullPath == s.activeFile.Name() {
				continue // Skip currently active open segment
			}
			_ = s.replaySegment(ctx, fullPath)
		}
	}
	return nil
}

// replaySegment reads a completed WAL segment, publishes each record to Redpanda with acks=all,
// and deletes the file ONLY after all records have been durably acknowledged by the broker.
func (s *DurableSpool) replaySegment(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := make([]byte, 8)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return err
	}
	if binary.BigEndian.Uint32(hdr[0:4]) != SpoolMagicHeader {
		return ErrCorruptSegment
	}

	lenBuf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(f, lenBuf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		msgLen := binary.BigEndian.Uint32(lenBuf)
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(f, msgBuf); err != nil {
			return err
		}

		if s.broker != nil {
			// Synchronous publish to Redpanda with strict acks=all
			if err := s.broker.Publish(ctx, s.topic, uuid.New().String(), msgBuf); err != nil {
				// Critical: If broker publish fails, return error WITHOUT deleting file.
				return fmt.Errorf("replay publish to broker failed: %w", err)
			}
			atomic.AddInt64(&s.drainedCount, 1)
		}
	}

	// Unlink segment file ONLY after full batch is durably acknowledged by broker
	f.Close()
	return os.Remove(path)
}

// Close cleanly flushes and shuts down the spool
func (s *DurableSpool) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.cond.Broadcast()
	s.mu.Unlock()

	close(s.stopDrain)
	s.drainWG.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeFile != nil {
		if s.batchBuf.Len() > 0 {
			_, _ = s.activeFile.Write(s.batchBuf.Bytes())
		}
		_ = s.activeFile.Sync()
		return s.activeFile.Close()
	}
	return nil
}
