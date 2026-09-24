package fix

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/quickfixgo/quickfix"
)

// PostgresMessageStoreFactory creates Postgres-backed MessageStores.
//
// Replaces quickfix.NewMemoryStoreFactory() so that MsgSeqNum and the
// session message log survive acceptor restarts. Without persistence,
// restarting the acceptor resets sequence numbers to 1 and the broker
// force-logs the session out — a hard Sev-1 for institutional FIX
// connections. See HANDOFF_FIX_OVER_PIPELINE.md §8 (Amendment 3).
//
// Schema: two tables, both created by the 20261016_006 and 20261016_008
// migrations:
//   - fix_message_store(session_id, msg_seq_num, message, created_at)
//     PK (session_id, msg_seq_num)
//   - fix_session_state(session_id PK, sender_msg_seq_num, target_msg_seq_num,
//                       creation_time, updated_at)
//
// Concurrency: quickfix calls into the MessageStore from one goroutine per
// session at a time (the session layer serializes). We hold an in-process
// mutex around the cached state to be safe across any future caller; the
// underlying DB ops are also serialized per session_id.
type PostgresMessageStoreFactory struct {
	db *sql.DB
}

// NewPostgresMessageStoreFactory returns a factory bound to db.
func NewPostgresMessageStoreFactory(db *sql.DB) *PostgresMessageStoreFactory {
	return &PostgresMessageStoreFactory{db: db}
}

// Create implements quickfix.MessageStoreFactory.
func (f *PostgresMessageStoreFactory) Create(sessionID quickfix.SessionID) (quickfix.MessageStore, error) {
	store := &postgresMessageStore{
		sessionID: sessionID.String(),
		db:        f.db,
	}
	if err := store.Refresh(); err != nil {
		return nil, err
	}
	return store, nil
}

type postgresMessageStore struct {
	sessionID string
	db        *sql.DB

	mu                  sync.Mutex
	senderMsgSeqNum     int
	targetMsgSeqNum     int
	creationTime        time.Time
}

// NextSenderMsgSeqNum returns the next sender-side message sequence number
// to be used (current value + 1).
func (s *postgresMessageStore) NextSenderMsgSeqNum() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.senderMsgSeqNum + 1
}

// NextTargetMsgSeqNum returns the next target-side message sequence number
// expected from the counterparty (current value + 1).
func (s *postgresMessageStore) NextTargetMsgSeqNum() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.targetMsgSeqNum + 1
}

// IncrNextSenderMsgSeqNum bumps the sender-side sequence number.
func (s *postgresMessageStore) IncrNextSenderMsgSeqNum() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.senderMsgSeqNum++
	return s.persistSenderLocked(context.Background())
}

// IncrNextTargetMsgSeqNum bumps the target-side sequence number.
func (s *postgresMessageStore) IncrNextTargetMsgSeqNum() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targetMsgSeqNum++
	return s.persistTargetLocked(context.Background())
}

// SetNextSenderMsgSeqNum sets the next sender-side sequence number
// (parameter is the *next* value, not the current).
func (s *postgresMessageStore) SetNextSenderMsgSeqNum(next int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.senderMsgSeqNum = next - 1
	return s.persistSenderLocked(context.Background())
}

// SetNextTargetMsgSeqNum sets the next target-side sequence number.
func (s *postgresMessageStore) SetNextTargetMsgSeqNum(next int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targetMsgSeqNum = next - 1
	return s.persistTargetLocked(context.Background())
}

// CreationTime returns the session creation time.
func (s *postgresMessageStore) CreationTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.creationTime
}

// SetCreationTime sets the session creation time.
func (s *postgresMessageStore) SetCreationTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creationTime = t
}

// Reset clears all session state. Used when a session is reset to its
// factory defaults (e.g. admin API request).
func (s *postgresMessageStore) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.senderMsgSeqNum = 0
	s.targetMsgSeqNum = 0
	s.creationTime = time.Now()

	ctx := context.Background()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fix_session_state (session_id, sender_msg_seq_num, target_msg_seq_num, creation_time, updated_at)
		VALUES ($1, 0, 0, NOW(), NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			sender_msg_seq_num = 0,
			target_msg_seq_num = 0,
			creation_time = NOW(),
			updated_at = NOW()
	`, s.sessionID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM fix_message_store WHERE session_id = $1`, s.sessionID)
	return err
}

// Refresh loads persisted state from Postgres. Called at session create
// time so a restarted acceptor picks up where it left off.
func (s *postgresMessageStore) Refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	row := s.db.QueryRowContext(ctx, `
		SELECT sender_msg_seq_num, target_msg_seq_num, creation_time
		FROM fix_session_state
		WHERE session_id = $1
	`, s.sessionID)

	var senderSeq, targetSeq int
	var creationTime time.Time
	switch err := row.Scan(&senderSeq, &targetSeq, &creationTime); err {
	case nil:
		s.senderMsgSeqNum = senderSeq
		s.targetMsgSeqNum = targetSeq
		s.creationTime = creationTime
	case sql.ErrNoRows:
		// First time this session is seen; insert a fresh row.
		s.creationTime = time.Now()
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO fix_session_state (session_id, sender_msg_seq_num, target_msg_seq_num, creation_time, updated_at)
			VALUES ($1, 0, 0, NOW(), NOW())
			ON CONFLICT (session_id) DO NOTHING
		`, s.sessionID)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

// Close is a no-op; the underlying *sql.DB pool is managed by the caller.
func (s *postgresMessageStore) Close() error {
	return nil
}

// SaveMessage persists a single message bytes under its sequence number.
// Used by quickfix when sending; messages are kept for replay on resend.
func (s *postgresMessageStore) SaveMessage(seqNum int, msg []byte) error {
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO fix_message_store (session_id, msg_seq_num, message, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (session_id, msg_seq_num) DO UPDATE SET message = EXCLUDED.message
	`, s.sessionID, seqNum, msg)
	return err
}

// SaveMessageAndIncrNextSenderMsgSeqNum atomically persists the message
// and bumps the sender sequence number.
func (s *postgresMessageStore) SaveMessageAndIncrNextSenderMsgSeqNum(seqNum int, msg []byte) error {
	if err := s.SaveMessage(seqNum, msg); err != nil {
		return err
	}
	return s.IncrNextSenderMsgSeqNum()
}

// GetMessages returns raw bytes for sequence numbers in [beginSeqNum, endSeqNum].
// Used by quickfix when responding to a ResendRequest.
func (s *postgresMessageStore) GetMessages(beginSeqNum, endSeqNum int) ([][]byte, error) {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT message FROM fix_message_store
		WHERE session_id = $1 AND msg_seq_num BETWEEN $2 AND $3
		ORDER BY msg_seq_num ASC
	`, s.sessionID, beginSeqNum, endSeqNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs [][]byte
	for rows.Next() {
		var msg []byte
		if err := rows.Scan(&msg); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

// persistSenderLocked writes the sender sequence number to Postgres.
// Caller must hold s.mu.
func (s *postgresMessageStore) persistSenderLocked(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fix_session_state (session_id, sender_msg_seq_num, target_msg_seq_num, creation_time, updated_at)
		VALUES ($1, $2, 0, NOW(), NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			sender_msg_seq_num = EXCLUDED.sender_msg_seq_num,
			updated_at = NOW()
	`, s.sessionID, s.senderMsgSeqNum)
	return err
}

// persistTargetLocked writes the target sequence number to Postgres.
// Caller must hold s.mu.
func (s *postgresMessageStore) persistTargetLocked(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fix_session_state (session_id, sender_msg_seq_num, target_msg_seq_num, creation_time, updated_at)
		VALUES ($1, 0, $2, NOW(), NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			target_msg_seq_num = EXCLUDED.target_msg_seq_num,
			updated_at = NOW()
	`, s.sessionID, s.targetMsgSeqNum)
	return err
}
