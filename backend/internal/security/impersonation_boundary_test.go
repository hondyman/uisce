package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// Test helpers
// ============================================================================

// boundaryDB creates an isolated in-memory SQLite database and the minimal
// schema required for the boundary tests.
func boundaryDB(t *testing.T) *sql.DB {
	t.Helper()
	// Use a shared-cache in-memory database and force a single connection so
	// that every statement in the test hits the same SQLite connection. Without
	// this, connection pooling can create separate :memory: databases and the
	// table created at setup may not exist for later queries.
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory test db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema := `
		CREATE TABLE staff_tenant_assignments (
			assignment_id TEXT PRIMARY KEY,
			operator_user_id TEXT NOT NULL,
			target_tenant_id TEXT NOT NULL,
			granted_by TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);

		CREATE INDEX idx_staff_tenant_assignments_active
			ON staff_tenant_assignments (operator_user_id, target_tenant_id, expires_at);

		CREATE TABLE bo_mutations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bo_key TEXT NOT NULL,
			transition TEXT NOT NULL,
			payload TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create boundary test schema: %v", err)
	}
	return db
}

// boundaryService constructs a ContextExchangeService wired to the test DB.
func boundaryService(t *testing.T, db *sql.DB) *ContextExchangeService {
	t.Helper()
	return NewContextExchangeService(
		NewPlatformAdminAuditLogger(db),
		ImpersonationPolicy{},
	)
}

// seedAssignment inserts a staff_tenant_assignments row. SQLite's
// CURRENT_TIMESTAMP is returned as "YYYY-MM-DD HH:MM:SS" in UTC, so we store
// the lease in the same textual format to make <=/> comparisons deterministic.
func seedAssignment(t *testing.T, db *sql.DB, operatorID string, tenantID uuid.UUID, expiresAt time.Time) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO staff_tenant_assignments (assignment_id, operator_user_id, target_tenant_id, granted_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		uuid.New().String(), operatorID, tenantID.String(), "boundary-test", expiresAt.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("failed to seed assignment: %v", err)
	}
}

// impersonationContext returns a context carrying an active impersonation
// security context. This mirrors what AuthContextMiddleware attaches after it
// validates an impersonation context token.
func impersonationContext(t *testing.T, userID, tenantID, role, mode string) context.Context {
	t.Helper()
	secCtx := &Context{
		UserID:                 userID,
		TenantID:               tenantID,
		IsGlobalAdmin:          role == RoleGlobalAdmin || role == RoleGlobalOps,
		ImpersonationActive:    true,
		RealAdminUserID:        userID,
		ImpersonationSessionID: uuid.New().String(),
		ImpersonationMode:      mode,
		ImpersonationAdminRole: role,
	}
	return WithContext(context.Background(), secCtx)
}

// transitionBO is a test-only stand-in for engine.TransitionBO. It mimics the
// transaction-boundary checks that real BO state-machine handlers must perform
// before committing any state change.
func transitionBO(ctx context.Context, db *sql.DB, svc *ContextExchangeService, boKey, transition string, payload []byte) error {
	secCtx, ok := FromContext(ctx)
	if !ok || secCtx == nil {
		return errors.New("no security context in request")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Gate 1: impersonation write authorization. Read-only sessions (helpdesk
	// always, or any role explicitly in read_only mode) are forbidden from state
	// mutations. Break-glass sessions are additionally restricted by role/BO.
	if secCtx.ImpersonationActive {
		if secCtx.ImpersonationMode != string(ModeBreakGlass) {
			return fmt.Errorf("403 Forbidden: %w", ErrImpersonationWriteForbidden)
		}
		if !svc.policy.CanBreakGlassForBO(secCtx.ImpersonationAdminRole, boKey) {
			return fmt.Errorf("403 Forbidden: role %s cannot perform break_glass on %s", secCtx.ImpersonationAdminRole, boKey)
		}
	}

	// Gate 2: active lease check for helpdesk/professional_services. Global
	// admins/ops bypass this gate because their authority comes from the role
	// itself, not a time-bounded tenant assignment.
	if secCtx.ImpersonationActive {
		tenantID, err := uuid.Parse(secCtx.TenantID)
		if err != nil {
			return fmt.Errorf("invalid tenant_id in security context: %w", err)
		}
		if err := svc.VerifyImpersonationAuthority(ctx, tx, SubjectAttributes{
			UserID:       secCtx.RealAdminUserID,
			OperatorRole: secCtx.ImpersonationAdminRole,
		}, tenantID); err != nil {
			return fmt.Errorf("403 Forbidden: %w", err)
		}
	}

	// Mutation would happen here in the real engine.
	payloadJSON, _ := json.Marshal(payload)
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO bo_mutations (bo_key, transition, payload) VALUES ($1, $2, $3)",
		boKey, transition, string(payloadJSON),
	); err != nil {
		return fmt.Errorf("mutation insert failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// mutationCount returns the number of rows written to the dummy mutation table.
func mutationCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM bo_mutations").Scan(&count); err != nil {
		t.Fatalf("failed to count mutations: %v", err)
	}
	return count
}

// assignmentPruner mimics the background AssignmentPruner: it periodically
// deletes staff_tenant_assignments rows whose lease has expired. The returned
// stop function must be called to end the goroutine.
func assignmentPruner(t *testing.T, db *sql.DB, interval time.Duration) func() {
	t.Helper()
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_, _ = db.ExecContext(context.Background(),
					"DELETE FROM staff_tenant_assignments WHERE expires_at <= CURRENT_TIMESTAMP")
			}
		}
	}()
	return func() { close(stop) }
}

// ============================================================================
// 1. Helpdesk Mutation Block Test
// ============================================================================

func TestHelpdeskReadOnlyABACEnforcement(t *testing.T) {
	db := boundaryDB(t)
	svc := boundaryService(t, db)
	tenantID := uuid.New()
	ctx := impersonationContext(t, "usr_jim_helpdesk", tenantID.String(), RoleHelpdesk, string(ModeReadOnly))

	t.Run("Reject State Mutation Attempt", func(t *testing.T) {
		err := transitionBO(ctx, db, svc, "support_ticket", "resolve", []byte(`{"status":"resolved"}`))
		if err == nil {
			t.Fatal("expected helpdesk state mutation to be rejected, got nil")
		}
		if !errors.Is(err, ErrImpersonationWriteForbidden) {
			t.Fatalf("expected ErrImpersonationWriteForbidden, got: %v", err)
		}
		if mutationCount(t, db) != 0 {
			t.Fatalf("expected zero database writes, found %d mutation rows", mutationCount(t, db))
		}
	})

	t.Run("Allow Read Path Without Mutation", func(t *testing.T) {
		// A read-only operation would not invoke transitionBO in the real engine,
		// so the absence of a mutation row after the rejected write is the
		// invariant we actually care about. This subtest documents that contract.
		if mutationCount(t, db) != 0 {
			t.Fatalf("read-only path leaked %d mutation rows", mutationCount(t, db))
		}
	})
}

// ============================================================================
// 2. Professional Services Boundary Leak Test
// ============================================================================

func TestProfessionalServicesLeaseBoundaries(t *testing.T) {
	db := boundaryDB(t)
	svc := boundaryService(t, db)

	sarahID := "f6be74ab-29c7-45e7-af85-dd3f63649d71"
	tenantA := uuid.New()
	tenantB := uuid.New()

	seedAssignment(t, db, sarahID, tenantA, time.Now().UTC().Add(1*time.Hour))

	t.Run("Valid Lease Context Selection - Tenant A", func(t *testing.T) {
		err := svc.VerifyImpersonationAuthority(context.Background(), nil, SubjectAttributes{
			UserID:       sarahID,
			OperatorRole: RoleProfessionalServices,
		}, tenantA)
		if err != nil {
			t.Errorf("expected access granted for valid consulting lease window, got: %v", err)
		}
	})

	t.Run("Cross-Tenant Context Attempt - Target Tenant B Interception", func(t *testing.T) {
		err := svc.VerifyImpersonationAuthority(context.Background(), nil, SubjectAttributes{
			UserID:       sarahID,
			OperatorRole: RoleProfessionalServices,
		}, tenantB)
		if err == nil {
			t.Fatal("CRITICAL FLAW: verification engine allowed a consultant to cross context bounds into an unassigned tenant fence")
		}
		if !errors.Is(err, ErrImpersonationLeaseViolation) {
			t.Fatalf("expected ErrImpersonationLeaseViolation, got: %v", err)
		}
	})

	t.Run("Break-Glass Mutation Within Lease Succeeds", func(t *testing.T) {
		ctx := impersonationContext(t, sarahID, tenantA.String(), RoleProfessionalServices, string(ModeBreakGlass))
		if err := transitionBO(ctx, db, svc, "tenant_config", "apply", []byte(`{"theme":"dark"}`)); err != nil {
			t.Fatalf("expected legitimate break-glass mutation within lease to succeed, got: %v", err)
		}
		if got := mutationCount(t, db); got != 1 {
			t.Fatalf("expected 1 committed mutation, got %d", got)
		}
	})

	t.Run("Break-Glass Mutation Outside Lease Is Blocked", func(t *testing.T) {
		ctx := impersonationContext(t, sarahID, tenantB.String(), RoleProfessionalServices, string(ModeBreakGlass))
		err := transitionBO(ctx, db, svc, "tenant_config", "apply", []byte(`{"theme":"light"}`))
		if err == nil {
			t.Fatal("expected break-glass mutation outside lease to be rejected")
		}
		if !errors.Is(err, ErrImpersonationLeaseViolation) {
			t.Fatalf("expected ErrImpersonationLeaseViolation at transaction boundary, got: %v", err)
		}
		if got := mutationCount(t, db); got != 1 {
			t.Fatalf("cross-tenant attempt leaked mutations: expected 1, got %d", got)
		}
	})
}

// ============================================================================
// 3. Temporal Real-Time Expiry Race Test
// ============================================================================

func TestTemporalRealTimeExpiryRace(t *testing.T) {
	db := boundaryDB(t)
	svc := boundaryService(t, db)

	consultantID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	tenantID := uuid.New()

	// Lease expires 1s from now. The pruner will reap it shortly after.
	seedAssignment(t, db, consultantID, tenantID, time.Now().UTC().Add(1*time.Second))

	stopPruner := assignmentPruner(t, db, 100*time.Millisecond)
	defer stopPruner()

	// Wait until the lease has expired and the pruner has had at least one
	// chance to delete the row.
	time.Sleep(1500 * time.Millisecond)

	t.Run("Expired Lease Rejected By Authority Check", func(t *testing.T) {
		err := svc.VerifyImpersonationAuthority(context.Background(), nil, SubjectAttributes{
			UserID:       consultantID,
			OperatorRole: RoleProfessionalServices,
		}, tenantID)
		if err == nil {
			t.Fatal("expected expired lease to be rejected, got nil")
		}
		if !errors.Is(err, ErrImpersonationLeaseViolation) {
			t.Fatalf("expected ErrImpersonationLeaseViolation after expiry, got: %v", err)
		}
	})

	t.Run("Mid-Flight State Transition Rolls Back When Lease Expires", func(t *testing.T) {
		ctx := impersonationContext(t, consultantID, tenantID.String(), RoleProfessionalServices, string(ModeBreakGlass))
		err := transitionBO(ctx, db, svc, "tenant_config", "apply", []byte(`{"flag":"on"}`))
		if err == nil {
			t.Fatal("expected mid-flight transition to roll back after lease expiry")
		}
		if !errors.Is(err, ErrImpersonationLeaseViolation) {
			t.Fatalf("expected ErrImpersonationLeaseViolation at transaction boundary, got: %v", err)
		}
		if got := mutationCount(t, db); got != 0 {
			t.Fatalf("expired lease leaked %d mutation rows", got)
		}
	})
}
