package db

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
)

// TestWithGoldCopySync_NoConnectionLeak proves the safety property this
// helper exists for: a pooled connection that ran an elevated
// uisce_gold_copy_sync transaction must come back to its plain connecting
// role for the very next, unrelated query. If this ever fails, the failure
// mode is a real cross-tenant data exposure, not a test artifact.
func TestWithGoldCopySync_NoConnectionLeak(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("UISCE_TEST_DB_DSN not set, skipping gold-copy-sync role isolation test")
	}

	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	var haveRole bool
	if err := dbConn.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync')").Scan(&haveRole); err != nil {
		t.Fatalf("checking for uisce_gold_copy_sync role: %v", err)
	}
	if !haveRole {
		t.Skip("uisce_gold_copy_sync role does not exist on test database; skipping")
	}

	dbConn.SetMaxOpenConns(1) // force physical connection reuse across sub-tests
	ctx := context.Background()

	t.Run("role_reverts_after_commit", func(t *testing.T) {
		var insideRole string
		err := WithGoldCopySync(ctx, dbConn, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, "SELECT current_user").Scan(&insideRole)
		})
		if err != nil {
			t.Fatalf("WithGoldCopySync: %v", err)
		}
		if insideRole != "uisce_gold_copy_sync" {
			t.Fatalf("expected current_user=uisce_gold_copy_sync inside the callback, got %q", insideRole)
		}

		var afterRole string
		if err := dbConn.QueryRowContext(ctx, "SELECT current_user").Scan(&afterRole); err != nil {
			t.Fatalf("query after commit: %v", err)
		}
		if afterRole == "uisce_gold_copy_sync" {
			t.Fatalf("LEAK: reused connection still shows uisce_gold_copy_sync after WithGoldCopySync committed")
		}
	})

	t.Run("role_reverts_after_fn_error", func(t *testing.T) {
		sentinelErr := WithGoldCopySync(ctx, dbConn, func(tx *sql.Tx) error {
			return sql.ErrNoRows // any error — forces the rollback path
		})
		if sentinelErr == nil {
			t.Fatal("expected an error from WithGoldCopySync when fn returns one")
		}

		var afterRole string
		if err := dbConn.QueryRowContext(ctx, "SELECT current_user").Scan(&afterRole); err != nil {
			t.Fatalf("query after rollback: %v", err)
		}
		if afterRole == "uisce_gold_copy_sync" {
			t.Fatalf("LEAK: reused connection still shows uisce_gold_copy_sync after WithGoldCopySync rolled back")
		}
	})

	t.Run("role_reverts_after_panic", func(t *testing.T) {
		func() {
			defer func() {
				_ = recover() // WithGoldCopySync re-panics after rolling back; that's expected here
			}()
			_ = WithGoldCopySync(ctx, dbConn, func(tx *sql.Tx) error {
				panic("simulated failure mid cross-tenant operation")
			})
		}()

		var afterRole string
		if err := dbConn.QueryRowContext(ctx, "SELECT current_user").Scan(&afterRole); err != nil {
			t.Fatalf("query after panic recovery: %v", err)
		}
		if afterRole == "uisce_gold_copy_sync" {
			t.Fatalf("LEAK: reused connection still shows uisce_gold_copy_sync after a panic mid-transaction")
		}
	})

	t.Run("no_leak_under_concurrency", func(t *testing.T) {
		dbConn.SetMaxOpenConns(5)
		defer dbConn.SetMaxOpenConns(1)

		var wg sync.WaitGroup
		leaks := make(chan string, 100)
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				if n%3 == 0 {
					if err := WithGoldCopySync(ctx, dbConn, func(tx *sql.Tx) error {
						return nil
					}); err != nil {
						leaks <- err.Error()
					}
					return
				}
				var role string
				if err := dbConn.QueryRowContext(ctx, "SELECT current_user").Scan(&role); err != nil {
					leaks <- err.Error()
					return
				}
				if role == "uisce_gold_copy_sync" {
					leaks <- "LEAK: ordinary query observed uisce_gold_copy_sync role"
				}
			}(i)
		}
		wg.Wait()
		close(leaks)
		for l := range leaks {
			t.Error(l)
		}
	})

	t.Run("scoped_to_five_tables_only", func(t *testing.T) {
		err := WithGoldCopySync(ctx, dbConn, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "SELECT 1 FROM public.report_templates LIMIT 1")
			return err
		})
		if err == nil {
			t.Fatal("expected permission denied reading report_templates under uisce_gold_copy_sync; got nil error")
		}
	})
}
