package fix

import (
	"database/sql"
	"os"
	"testing"

	"github.com/quickfixgo/quickfix"
	_ "github.com/lib/pq"
)

// TestPostgresMessageStoreFactory_NextAndIncr exercises the basic
// sequence-number persistence path against a real (or test) Postgres.
// Skipped if FIX_TEST_DATABASE_URL is unset so unit-only CI doesn't fail.
func TestPostgresMessageStoreFactory_NextAndIncr(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	factory := NewPostgresMessageStoreFactory(db)
	sid := quickfix.SessionID{BeginString: "FIX.4.4", SenderCompID: "TEST", TargetCompID: "BROKER"}
	store, err := factory.Create(sid)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer store.Close()
	defer store.Reset()

	if got := store.NextSenderMsgSeqNum(); got != 1 {
		t.Fatalf("NextSenderMsgSeqNum after create: want 1, got %d", got)
	}

	if err := store.IncrNextSenderMsgSeqNum(); err != nil {
		t.Fatalf("IncrNextSenderMsgSeqNum: %v", err)
	}
	if got := store.NextSenderMsgSeqNum(); got != 2 {
		t.Fatalf("NextSenderMsgSeqNum after incr: want 2, got %d", got)
	}

	// Persistence: a fresh store over the same session_id must see the
	// incremented value, not 1.
	store2, err := factory.Create(sid)
	if err != nil {
		t.Fatalf("Create (fresh): %v", err)
	}
	defer store2.Close()

	if got := store2.NextSenderMsgSeqNum(); got != 2 {
		t.Fatalf("NextSenderMsgSeqNum after restart: want 2, got %d", got)
	}
}

// TestPostgresMessageStoreFactory_SaveAndGetMessages exercises the
// message-log half of the contract.
func TestPostgresMessageStoreFactory_SaveAndGetMessages(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	factory := NewPostgresMessageStoreFactory(db)
	sid := quickfix.SessionID{BeginString: "FIX.4.4", SenderCompID: "TEST2", TargetCompID: "BROKER"}
	store, err := factory.Create(sid)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer store.Close()
	defer store.Reset()

	msg1 := []byte("8=FIX.4.4\x019=100\x0135=D\x01...")
	msg2 := []byte("8=FIX.4.4\x019=120\x0135=D\x01...")

	if err := store.SaveMessageAndIncrNextSenderMsgSeqNum(1, msg1); err != nil {
		t.Fatalf("SaveMessageAndIncrNextSenderMsgSeqNum(1): %v", err)
	}
	if err := store.SaveMessageAndIncrNextSenderMsgSeqNum(2, msg2); err != nil {
		t.Fatalf("SaveMessageAndIncrNextSenderMsgSeqNum(2): %v", err)
	}

	got, err := store.GetMessages(1, 2)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetMessages returned %d messages, want 2", len(got))
	}
	if string(got[0]) != string(msg1) || string(got[1]) != string(msg2) {
		t.Fatalf("GetMessages returned wrong bytes")
	}
}

// TestPostgresMessageStoreFactory_Reset exercises the admin-reset path.
func TestPostgresMessageStoreFactory_Reset(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	factory := NewPostgresMessageStoreFactory(db)
	sid := quickfix.SessionID{BeginString: "FIX.4.4", SenderCompID: "RESET", TargetCompID: "BROKER"}
	store, err := factory.Create(sid)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer store.Close()
	defer store.Reset()

	if err := store.IncrNextSenderMsgSeqNum(); err != nil {
		t.Fatalf("IncrNextSenderMsgSeqNum: %v", err)
	}
	if err := store.SaveMessage(1, []byte("test")); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	if err := store.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if got := store.NextSenderMsgSeqNum(); got != 1 {
		t.Fatalf("NextSenderMsgSeqNum after Reset: want 1, got %d", got)
	}

	got, err := store.GetMessages(1, 1)
	if err != nil {
		t.Fatalf("GetMessages after Reset: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetMessages after Reset: want 0, got %d", len(got))
	}
}

// TestPostgresMessageStoreFactory_FiftyMessages_ResumeAfterRestart
// exercises the doc's §8 (Amendment 3) acceptance test:
//   1. Logon, send 50 messages (MsgSeqNum 1..50).
//   2. Restart the acceptor process (drop the existing store, create a
//      fresh one over the same session_id).
//   3. Logon again.
//   4. Assert MsgSeqNum resumes at 51 — not 1. The broker would
//      force-logout if quickfix tried to send MsgSeqNum=1 after seeing
//      MsgSeqNum=50.
//
// A test that only round-trips a single message proves persistence
// but not the broker-resume scenario. This is the test that proves the
// Postgres message store is fit for purpose.
func TestPostgresMessageStoreFactory_FiftyMessages_ResumeAfterRestart(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	factory := NewPostgresMessageStoreFactory(db)
	sid := quickfix.SessionID{BeginString: "FIX.4.4", SenderCompID: "FIFTY", TargetCompID: "BROKER"}

	// Pre-cleanup so the test is hermetic.
	defer func() {
		store, err := factory.Create(sid)
		if err != nil {
			return
		}
		_ = store.Reset()
		_ = store.Close()
	}()

	// Phase 1: send 50 messages.
	store1, err := factory.Create(sid)
	if err != nil {
		t.Fatalf("Create phase 1: %v", err)
	}
	for i := 1; i <= 50; i++ {
		msg := []byte{0x01, 0x02, byte(i)}
		if err := store1.SaveMessageAndIncrNextSenderMsgSeqNum(i, msg); err != nil {
			t.Fatalf("phase 1: SaveMessageAndIncrNextSenderMsgSeqNum(%d): %v", i, err)
		}
	}
	if got := store1.NextSenderMsgSeqNum(); got != 51 {
		t.Fatalf("phase 1: NextSenderMsgSeqNum after 50 msgs: want 51, got %d", got)
	}
	// Simulate process restart: drop the in-process store without
	// touching Postgres.
	_ = store1.Close()

	// Phase 2: fresh store over the same session_id, post-restart.
	store2, err := factory.Create(sid)
	if err != nil {
		t.Fatalf("Create phase 2: %v", err)
	}
	defer store2.Close()

	if got := store2.NextSenderMsgSeqNum(); got != 51 {
		t.Fatalf("phase 2 (post-restart): NextSenderMsgSeqNum: want 51 (broker expects this), got %d — broker would force-logout", got)
	}
	if got := store2.NextTargetMsgSeqNum(); got != 1 {
		t.Fatalf("phase 2 (post-restart): NextTargetMsgSeqNum: want 1, got %d", got)
	}

	// GetMessages(1, 50) must return all 50 stored messages so
	// ResendRequest can replay them on demand.
	msgs, err := store2.GetMessages(1, 50)
	if err != nil {
		t.Fatalf("phase 2: GetMessages: %v", err)
	}
	if len(msgs) != 50 {
		t.Fatalf("phase 2: GetMessages returned %d, want 50", len(msgs))
	}

	// Phase 3: send one more message, must use MsgSeqNum=51 (not 1).
	if err := store2.SaveMessageAndIncrNextSenderMsgSeqNum(51, []byte{0x01, 0x02, 51}); err != nil {
		t.Fatalf("phase 3: SaveMessageAndIncrNextSenderMsgSeqNum(51): %v", err)
	}
	if got := store2.NextSenderMsgSeqNum(); got != 52 {
		t.Fatalf("phase 3: NextSenderMsgSeqNum: want 52, got %d", got)
	}
}

// openTestDB opens a Postgres connection from FIX_TEST_DATABASE_URL,
// skipping the test if it's not set (so unit-only CI works).
// Schema is the responsibility of the test runner: apply migrations
// 003–008 against the test DB before invoking these tests.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("FIX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("FIX_TEST_DATABASE_URL not set; skipping Postgres store test")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test DB: %v", err)
	}

	// Best-effort: clear any rows from a previous failed run so the
	// assertions don't collide. Safe to fail silently; assertions will
	// then fail loudly.
	_, _ = db.Exec(`DELETE FROM fix_message_store WHERE session_id LIKE 'FIX.4.4:%'`)
	_, _ = db.Exec(`DELETE FROM fix_session_state WHERE session_id LIKE 'FIX.4.4:%'`)

	return db
}
