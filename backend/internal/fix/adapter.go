package fix

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
)

// ComplianceEvaluator is the legacy in-process compliance interface.
// Preserved for backward compatibility with existing tests; the FIX-over-
// pipeline path routes compliance through the data-pipeline's `fix_compliance`
// tile (HANDOFF_FIX_OVER_PIPELINE.md §9). New code should not implement or
// depend on this — see the doc §3 "What to delete" for the cleanup path.
type ComplianceEvaluator interface {
	EvaluateTrade(trade ExternalTradeItem) EvaluateResult
}

type EvaluateResult struct {
	Approved        bool
	CanOverride     bool
	HighestSeverity string
	Violations      []Violation
}

type Violation struct {
	RuleID    string
	FieldPath string
	Message   string
}

type ExternalTradeItem struct {
	ExternalOrderID string
	PortfolioID     string
	ISIN            string
	Symbol          string
	Quantity        float64
	Price           float64
	Side            string
	Account         string
}

// TenantResolver maps a quickfix SessionID to its owning tenant and broker.
// Implementation is the Layer-1 admin API's session registry, loaded from
// fix_tenant_config. Per HANDOFF_FIX_OVER_PIPELINE.md §4, this is the GSIFI
// boundary — every message must be tenant-tagged at adapter dispatch, never
// inferred from message content.
type TenantResolver interface {
	Resolve(sessionID quickfix.SessionID) (tenantID uuid.UUID, brokerID uuid.UUID, ok bool)
}

// InboundSink receives tenant-tagged, per-message PipelineRecord-shaped
// values for downstream consumption by the data-pipeline's fix_listener tile.
// The implementation is in-process (a buffered channel fan-out) — Layer 2
// (the data-pipeline) connects to it via the fix_listener source driver.
type InboundSink interface {
	Emit(ctx context.Context, rec InboundRecord) error
}

// InboundSinkFunc adapts a function to InboundSink.
type InboundSinkFunc func(ctx context.Context, rec InboundRecord) error

func (f InboundSinkFunc) Emit(ctx context.Context, rec InboundRecord) error {
	if f == nil {
		return nil
	}
	return f(ctx, rec)
}

// InboundRecord is the shape handed off from adapter → fix_listener.
// Mirrors the JSON shape documented in HANDOFF_FIX_OVER_PIPELINE.md §9.
type InboundRecord struct {
	RawBytes   []byte    `json:"raw_bytes"`
	TenantID   uuid.UUID `json:"tenant_id"`
	BrokerID   uuid.UUID `json:"broker_id"`
	SessionID  string    `json:"session_id"`
	MsgType    string    `json:"msg_type"`
	ReceivedAt time.Time `json:"received_at"`
	MsgSeqNum  int       `json:"msg_seq_num"`
	ClOrdID    string    `json:"cl_ord_id,omitempty"` // when available from message header
	ExecType   string    `json:"exec_type,omitempty"`
	OrdStatus  string    `json:"ord_status,omitempty"`
	LastQty    string    `json:"last_qty,omitempty"`
	LastPx     string    `json:"last_px,omitempty"`
	ExecID     string    `json:"exec_id,omitempty"`
}

// StaticTenantResolver maps every session to a fixed tenant/broker.
// Used by the demo/sample FIX agent so inbound ExecutionReports are
// tagged without a full fix_tenant_config lookup.
type StaticTenantResolver struct {
	TenantID uuid.UUID
	BrokerID uuid.UUID
}

func (s StaticTenantResolver) Resolve(sessionID quickfix.SessionID) (uuid.UUID, uuid.UUID, bool) {
	if s.TenantID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false
	}
	return s.TenantID, s.BrokerID, true
}

// Adapter is the FIX acceptor's application callbacks layer.
// Routes inbound messages by (tenant_id, broker_id) into the data-pipeline
// per HANDOFF_FIX_OVER_PIPELINE.md §3 ("Rewrite adapter.go"). Replaces the
// old hardcoded MsgType=D switch with a generic dispatch.
type Adapter struct {
	app        *fixApplication
	sessionMap map[quickfix.SessionID]struct{}
	sessionMu  sync.RWMutex

	// Legacy compliance evaluator. Only invoked when no TenantResolver
	// is configured (dev/smoke mode). Production deploys must configure
	// a TenantResolver + InboundSink so messages flow through the
	// data-pipeline.
	evaluator ComplianceEvaluator

	// New FIX-over-pipeline dependencies. nil-safe — when nil, the
	// adapter falls back to the legacy evaluator path. See §3 of the
	// handoff for the migration sequence.
	resolver TenantResolver
	sink     InboundSink
	db       *sql.DB // for fix_session_log INSERTs; nil-safe

	// Latency-budget tracking for Amendment 2 fallback (HANDOFF §7).
	// For each inbound NewOrderSingle we send an immediate PendingNew,
	// then start a timer keyed by ClOrdID. If the timer fires before the
	// business-level ExecutionReport (MsgType=8) arrives, we send a
	// ExecType=8 Rejected. Cancelled when an ExecutionReport for the
	// same ClOrdID arrives.
	latencyBudget time.Duration
	pendingMu     sync.Mutex
	pending       map[string]*pendingReject // key: "<sessionID>|<clOrdID>"

	// sender is the FIX-protocol-level sender. Defaults to a wrapper
	// around quickfix.SendToTarget; tests inject a fake to capture
	// outbound messages for the latency-budget race tests.
	sender Sender
}

// pendingReject tracks a scheduled latency-budget rejection. The timer
// field is the real *time.Timer in production; the fake in tests.
type pendingReject struct {
	clOrdID   string
	sessionID quickfix.SessionID
	timer     LatencyTimer
	mu        sync.Mutex
	cancelled bool
}

// Sender is the FIX-protocol send boundary. Indirected so the latency-
// budget test can capture what the adapter would have sent without
// requiring a real quickfix session.
type Sender interface {
	Send(msg *quickfix.Message, sessionID quickfix.SessionID) error
}

// quickfixSender is the production Sender — wraps quickfix.SendToTarget.
type quickfixSender struct{}

func (quickfixSender) Send(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return quickfix.SendToTarget(msg, sessionID)
}

// LatencyTimer is the timer abstraction the latency-budget path uses.
// Production wraps *time.Timer; tests inject a controllable fake.
type LatencyTimer interface {
	Stop() bool
}

type fixApplication struct {
	onLogon   func(sessionID quickfix.SessionID)
	onLogout  func(sessionID quickfix.SessionID)
	onMsg     func(sessionID quickfix.SessionID, msg *quickfix.Message) error
	sessionMu sync.RWMutex
}

func (a *fixApplication) OnCreate(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Session created: %s", sessionID)
}

func (a *fixApplication) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logon: %s", sessionID)
}

func (a *fixApplication) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logout: %s", sessionID)
}

func (a *fixApplication) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {}

func (a *fixApplication) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

func (a *fixApplication) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

func (a *fixApplication) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	a.sessionMu.RLock()
	handler := a.onMsg
	a.sessionMu.RUnlock()
	if handler != nil {
		if err := handler(sessionID, msg); err != nil {
			return quickfix.NewMessageRejectError(err.Error(), 0, nil)
		}
	}
	return nil
}

// NewAdapter creates the legacy in-process adapter (ComplianceEvaluator path).
// Kept for existing server_test.go and any smoke deployments; the production
// path is NewPipelineAdapter below.
func NewAdapter(evaluator ComplianceEvaluator) *Adapter {
	app := &fixApplication{}
	adapter := &Adapter{
		app:        app,
		sessionMap: make(map[quickfix.SessionID]struct{}),
		evaluator:  evaluator,
	}

	app.onLogon = func(sessionID quickfix.SessionID) {
		adapter.sessionMu.Lock()
		adapter.sessionMap[sessionID] = struct{}{}
		adapter.sessionMu.Unlock()
	}

	app.onLogout = func(sessionID quickfix.SessionID) {
		adapter.sessionMu.Lock()
		delete(adapter.sessionMap, sessionID)
		adapter.sessionMu.Unlock()
	}

	app.onMsg = func(sessionID quickfix.SessionID, msg *quickfix.Message) error {
		msgType, _ := msg.Header.GetString(tagMsgType)
		if msgType == "D" {
			trade := TranslateNewOrderSingle(msg)
			result := adapter.evaluator.EvaluateTrade(trade)
			reply := BuildExecutionReport(trade, result)
			if err := quickfix.SendToTarget(reply, sessionID); err != nil {
				log.Printf("[FIX] Failed to send ExecutionReport: %v", err)
			}
		}
		return nil
	}

	return adapter
}

// NewPipelineAdapter creates the FIX-over-pipeline adapter. Messages are
// dispatched into the data-pipeline via sink; tenant and broker are
// resolved from the session registry via resolver. db is used for the
// append-only fix_session_log audit (nil-safe — skips logging when nil).
// latencyBudget is the per-tenant pipeline latency budget (typically
// loaded from fix_tenant_config.pipeline_latency_budget_ms); the
// Amendment 2 fallback ExecType=8 Reject fires after this delay if no
// business-level ExecutionReport has arrived.
//
// Per HANDOFF_FIX_OVER_PIPELINE.md §7, the adapter sends an immediate
// PendingNew ExecutionReport for inbound NewOrderSingle so the broker's
// ack deadline is met; the business-level ExecutionReport flows back
// asynchronously through the data-pipeline (fix_execution_writer →
// fix_sender).
func NewPipelineAdapter(resolver TenantResolver, sink InboundSink, db *sql.DB, latencyBudget time.Duration) *Adapter {
	return newAdapter(resolver, sink, db, latencyBudget, quickfixSender{})
}

// NewPipelineAdapterForTest is the test-only constructor that accepts
// an injectable Sender. Production code must use NewPipelineAdapter;
// this variant exists for the latency-budget race tests.
func NewPipelineAdapterForTest(resolver TenantResolver, sink InboundSink, db *sql.DB, latencyBudget time.Duration, sender Sender) *Adapter {
	return newAdapter(resolver, sink, db, latencyBudget, sender)
}

func newAdapter(resolver TenantResolver, sink InboundSink, db *sql.DB, latencyBudget time.Duration, sender Sender) *Adapter {
	app := &fixApplication{}
	adapter := &Adapter{
		app:           app,
		sessionMap:    make(map[quickfix.SessionID]struct{}),
		resolver:      resolver,
		sink:          sink,
		db:            db,
		latencyBudget: latencyBudget,
		pending:       make(map[string]*pendingReject),
		sender:        sender,
	}

	app.onLogon = func(sessionID quickfix.SessionID) {
		adapter.sessionMu.Lock()
		adapter.sessionMap[sessionID] = struct{}{}
		adapter.sessionMu.Unlock()
		adapter.logSessionEvent(sessionID, "logon", 0, "", "", nil)
	}

	app.onLogout = func(sessionID quickfix.SessionID) {
		adapter.sessionMu.Lock()
		delete(adapter.sessionMap, sessionID)
		adapter.sessionMu.Unlock()
		// Cancel any pending rejection timers for this session; their
		// orders won't be filled.
		adapter.cancelPendingForSession(sessionID.String())
		adapter.logSessionEvent(sessionID, "logout", 0, "", "", nil)
	}

	app.onMsg = func(sessionID quickfix.SessionID, msg *quickfix.Message) error {
		return adapter.dispatchInbound(sessionID, msg)
	}

	return adapter
}

// dispatchInbound handles a single inbound application message. Tags it
// with (tenant_id, broker_id) via the resolver, emits an InboundRecord
// for the data-pipeline's fix_listener, and (for NewOrderSingle) sends
// the immediate PendingNew reply per §7 of the handoff.
func (a *Adapter) dispatchInbound(sessionID quickfix.SessionID, msg *quickfix.Message) error {
	if a.resolver == nil || a.sink == nil {
		// Adapter is in legacy mode (NewAdapter path) or partially
		// configured. Fall back: the onMsg handler installed by
		// NewAdapter will not be invoked here — the two paths are
		// mutually exclusive. Returning nil without dispatch is
		// safer than misrouting.
		return nil
	}

	tenantID, brokerID, ok := a.resolver.Resolve(sessionID)
	if !ok {
		log.Printf("[FIX] dispatchInbound: no tenant resolution for session %s; dropping", sessionID)
		// Drop the message rather than guess at the tenant. GSIFI
		// compliance: never infer tenant from message content.
		return nil
	}

	// Serialize the message bytes for the data-pipeline consumer.
	rawBytes, err := serializeFIXMessage(msg)
	if err != nil {
		log.Printf("[FIX] serialize message: %v", err)
		return nil
	}

	msgType, _ := msg.Header.GetString(tagMsgType)
	seqNum := 0
	if n, err := msg.Header.GetInt(tagMsgSeqNum); err == nil {
		seqNum = n
	}

	clOrdID, _ := msg.Body.GetString(tagClOrdID)
	execType, _ := msg.Body.GetString(tagExecType)
	ordStatus, _ := msg.Body.GetString(tagOrdStatus)
	lastQty, _ := msg.Body.GetString(tag.LastQty)
	lastPx, _ := msg.Body.GetString(tag.LastPx)
	execID, _ := msg.Body.GetString(tag.ExecID)

	rec := InboundRecord{
		RawBytes:   rawBytes,
		TenantID:   tenantID,
		BrokerID:   brokerID,
		SessionID:  sessionID.String(),
		MsgType:    msgType,
		ReceivedAt: time.Now().UTC(),
		MsgSeqNum:  seqNum,
		ClOrdID:    clOrdID,
		ExecType:   execType,
		OrdStatus:  ordStatus,
		LastQty:    lastQty,
		LastPx:     lastPx,
		ExecID:     execID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.sink.Emit(ctx, rec); err != nil {
		log.Printf("[FIX] sink.Emit: %v", err)
		// Don't return an error here — that would tell quickfix to
		// reject the message at the FIX-protocol level, when the
		// broker has already received it. The data-pipeline's error
		// policy (fail_fast / skip_and_log / dead_letter) handles
		// downstream consequences; the broker will eventually
		// disconnect on latency budget breach and the timeout path
		// (§7) will fire.
	}

	// Audit log (nil-safe — skipped when db is unset).
	a.logSessionEvent(sessionID, "msg_in", seqNum, msgType, clOrdID, rawBytes)

	// Amendment 2: send immediate PendingNew for inbound NewOrderSingle
	// so the broker's ack deadline is met. The business-level
	// ExecutionReport flows back asynchronously via fix_sender.
	//
	// If the business-level ExecutionReport (MsgType=8) doesn't arrive
	// within latencyBudget, the timer we schedule below fires and sends
	// ExecType=8 Rejected. This is the half of Amendment 2 that prevents
	// the broker from disconnecting on silence — see HANDOFF_FIX_OVER_PIPELINE.md
	// §7 latency budget and the §14 "Outbound/ExecType=8 Rejected on
	// timeout" gotcha.
	if msgType == "D" {
		pending := buildPendingNew(clOrdID)
		if err := quickfix.SendToTarget(pending, sessionID); err != nil {
			log.Printf("[FIX] SendToTarget(PendingNew) failed: %v", err)
		}
		a.logSessionEvent(sessionID, "msg_out", seqNum, "8", clOrdID, nil)
		a.scheduleLatencyReject(sessionID, clOrdID)
	}

	// If the inbound message IS a business-level ExecutionReport
	// (MsgType=8), cancel any pending rejection timer for the same
	// ClOrdID — the order is being acknowledged.
	if msgType == "8" && clOrdID != "" {
		a.cancelPendingReject(sessionID.String(), clOrdID)
	}

	return nil
}

// scheduleLatencyReject arms a timer that, if it fires before the
// matching business-level ExecutionReport arrives, sends ExecType=8
// (Rejected) back to the broker. See HANDOFF_FIX_OVER_PIPELINE.md §7.
//
// The timer implementation is abstracted via LatencyTimer so tests
// can use a controllable fake clock instead of real time.
func (a *Adapter) scheduleLatencyReject(sessionID quickfix.SessionID, clOrdID string) {
	if a.latencyBudget <= 0 {
		return
	}
	if clOrdID == "" {
		return
	}
	key := sessionID.String() + "|" + clOrdID
	sid := sessionID

	a.pendingMu.Lock()
	defer a.pendingMu.Unlock()

	// Cancel any existing timer for the same key (re-arming).
	if existing, ok := a.pending[key]; ok {
		existing.mu.Lock()
		existing.cancelled = true
		existing.mu.Unlock()
		existing.timer.Stop()
	}

	pr := &pendingReject{
		clOrdID:   clOrdID,
		sessionID: sid,
	}

	// Capture pr in the closure so the timer fire function knows
	// which pendingReject to update.
	pr.timer = newLatencyTimer(a.latencyBudget, func() {
		// Check cancellation under the pr mutex so a cancel-and-fire
		// race resolves deterministically: if cancelled, drop without
		// sending. If not cancelled, send Reject.
		pr.mu.Lock()
		cancelled := pr.cancelled
		pr.mu.Unlock()

		a.pendingMu.Lock()
		delete(a.pending, key)
		a.pendingMu.Unlock()

		if cancelled {
			return
		}

		reject := buildRejected(sid, clOrdID, "pipeline timeout")
		if err := a.sender.Send(reject, sid); err != nil {
			log.Printf("[FIX] SendToTarget(Rejected) failed: %v", err)
		}
		a.logSessionEvent(sid, "msg_out", 0, "8", clOrdID, nil)
		a.logSessionEvent(sid, "reject", 0, "8", clOrdID, nil)
	})

	a.pending[key] = pr
}

// cancelPendingReject marks the timer for (sessionID, clOrdID) as
// cancelled and stops it. Called when the business-level ExecutionReport
// arrives in time. The "mark + stop" pair is needed because Stop()
// alone doesn't synchronize with the fire goroutine — a fire that's
// already started must see cancelled=true to drop without sending.
func (a *Adapter) cancelPendingReject(sessionID, clOrdID string) {
	key := sessionID + "|" + clOrdID
	a.pendingMu.Lock()
	pr, ok := a.pending[key]
	a.pendingMu.Unlock()
	if !ok {
		return
	}

	pr.mu.Lock()
	pr.cancelled = true
	pr.mu.Unlock()
	pr.timer.Stop()

	a.pendingMu.Lock()
	delete(a.pending, key)
	a.pendingMu.Unlock()
}

// cancelPendingForSession cancels all pending rejection timers for a
// session — called on logout so timers don't fire on a dead session.
func (a *Adapter) cancelPendingForSession(sessionID string) {
	prefix := sessionID + "|"
	a.pendingMu.Lock()
	var toCancel []*pendingReject
	var keys []string
	for k, pr := range a.pending {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			toCancel = append(toCancel, pr)
			keys = append(keys, k)
		}
	}
	a.pendingMu.Unlock()

	for _, pr := range toCancel {
		pr.mu.Lock()
		pr.cancelled = true
		pr.mu.Unlock()
		pr.timer.Stop()
	}

	a.pendingMu.Lock()
	for _, k := range keys {
		delete(a.pending, k)
	}
	a.pendingMu.Unlock()
}

// logSessionEvent writes one row to fix_session_log if db is configured.
// Best-effort: errors are logged but never returned. Per §3 of the
// handoff, this is observability, not a workflow source of truth.
func (a *Adapter) logSessionEvent(sessionID quickfix.SessionID, eventType string, seqNum int, msgType, clOrdID string, rawExcerpt []byte) {
	if a.db == nil {
		return
	}

	tenantID, brokerID, ok := a.resolveForLog(sessionID)
	if !ok {
		return
	}

	var rawStr sql.NullString
	if len(rawExcerpt) > 0 {
		// Cap at 512 bytes (matches the schema). Base64-encode so
		// binary-safe SOH-separated FIX messages round-trip.
		if len(rawExcerpt) > 512 {
			rawExcerpt = rawExcerpt[:512]
		}
		rawStr = sql.NullString{String: base64.StdEncoding.EncodeToString(rawExcerpt), Valid: true}
	}

	_, err := a.db.ExecContext(context.Background(), `
		INSERT INTO fix_session_log (
			tenant_id, broker_id, session_id, event_type, msg_seq_num,
			msg_type, cl_ord_id, raw_excerpt, occurred_at
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, NOW())
	`,
		tenantID, brokerID, sessionID.String(), eventType, seqNum,
		msgType, clOrdID, rawStr,
	)
	if err != nil {
		log.Printf("[FIX] fix_session_log insert failed: %v", err)
	}
}

// resolveForLog re-uses the resolver for the audit-log write. Splitting
// from dispatchInbound keeps the dispatch path single-responsibility.
func (a *Adapter) resolveForLog(sessionID quickfix.SessionID) (uuid.UUID, uuid.UUID, bool) {
	if a.resolver == nil {
		return uuid.Nil, uuid.Nil, false
	}
	return a.resolver.Resolve(sessionID)
}

// CreateAcceptor builds the quickfix.Acceptor with the Postgres-backed
// message store (Amendment 3). db is the same *sql.DB used elsewhere in
// the acceptor process. Passing nil for db falls back to the in-memory
// store — only acceptable for tests / dev with `allow_seq_reset=true`
// per HANDOFF_FIX_OVER_PIPELINE.md §8.
// SendMessage dispatches an application message on a session the
// acceptor already owns. Temporal activities must call this via the
// admin API — never quickfix.SendToTarget from a workflow/activity
// that does not hold the socket.
func (a *Adapter) SendMessage(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	if a == nil {
		return fmt.Errorf("adapter is nil")
	}
	if a.sender != nil {
		return a.sender.Send(msg, sessionID)
	}
	return quickfix.SendToTarget(msg, sessionID)
}

func (a *Adapter) CreateAcceptor(settings *quickfix.Settings, db *sql.DB) (*quickfix.Acceptor, error) {
	var store quickfix.MessageStoreFactory
	if db != nil {
		store = NewPostgresMessageStoreFactory(db)
	} else {
		log.Printf("[FIX] WARNING: db is nil; falling back to in-memory message store. " +
			"This will reset MsgSeqNum on acceptor restart. See HANDOFF_FIX_OVER_PIPELINE.md §8.")
		store = quickfix.NewMemoryStoreFactory()
	}

	logFactory := quickfix.NewScreenLogFactory()
	acceptor, err := quickfix.NewAcceptor(a.app, store, settings, logFactory)
	if err != nil {
		return nil, fmt.Errorf("create acceptor: %w", err)
	}
	return acceptor, nil
}

// serializeFIXMessage converts a *quickfix.Message into raw bytes that
// can be reconstructed downstream by the fix_decode tile. Uses quickfix's
// own String() round-trip for now; production may switch to a more
// efficient format (e.g. direct field-map serialization) once
// performance is measured.
func serializeFIXMessage(msg *quickfix.Message) ([]byte, error) {
	return []byte(msg.String()), nil
}

// newLatencyTimer is the LatencyTimer factory. Production wraps
// time.AfterFunc; tests override this variable.
var newLatencyTimer = func(d time.Duration, f func()) LatencyTimer {
	t := time.AfterFunc(d, f)
	return &realLatencyTimer{t: t}
}

// SetLatencyTimerFactory replaces the LatencyTimer factory and returns
// the previous one. Tests use this to inject a controllable fake clock.
// The returned previous factory must be restored (typically via
// t.Cleanup) so production behavior is not affected across tests.
func SetLatencyTimerFactory(f func(d time.Duration, fire func()) LatencyTimer) func(d time.Duration, fire func()) LatencyTimer {
	prev := newLatencyTimer
	newLatencyTimer = f
	return prev
}

// realLatencyTimer is the production LatencyTimer implementation.
type realLatencyTimer struct {
	t *time.Timer
}

// Stop implements LatencyTimer.
func (r *realLatencyTimer) Stop() bool {
	return r.t.Stop()
}

// buildPendingNew constructs an ExecutionReport with ExecType=A (PendingNew)
// for the Amendment 2 immediate-reply path. Returns nil if clOrdID is
// blank (the broker can't correlate an empty ClOrdID).
func buildPendingNew(clOrdID string) *quickfix.Message {
	if clOrdID == "" {
		return nil
	}
	msg := quickfix.NewMessage()
	msg.Header.SetString(tagBeginString, "FIX.4.4")
	msg.Header.SetInt(tagBodyLength, 0)
	msg.Header.SetString(tagMsgType, "8") // ExecutionReport

	msg.Body.SetString(tagClOrdID, clOrdID)
	msg.Body.SetString(tagOrderID, clOrdID)
	msg.Body.SetString(tagExecType, "A")  // PendingNew
	msg.Body.SetString(tagOrdStatus, "A") // PendingNew
	return msg
}

// buildRejected constructs an ExecutionReport with ExecType=8 (Rejected)
// for the latency-budget fallback (HANDOFF_FIX_OVER_PIPELINE.md §7).
// Used when the data-pipeline doesn't emit a terminal ExecutionReport
// within pipeline_latency_budget_ms — better to reject explicitly than
// let the broker disconnect on silence.
func buildRejected(sessionID quickfix.SessionID, clOrdID, reason string) *quickfix.Message {
	if clOrdID == "" {
		return nil
	}
	msg := quickfix.NewMessage()
	msg.Header.SetString(tagBeginString, "FIX.4.4")
	msg.Header.SetInt(tagBodyLength, 0)
	msg.Header.SetString(tagMsgType, "8") // ExecutionReport

	msg.Body.SetString(tagClOrdID, clOrdID)
	msg.Body.SetString(tagOrderID, clOrdID)
	msg.Body.SetString(tagExecType, "8")  // Rejected
	msg.Body.SetString(tagOrdStatus, "8") // Rejected
	msg.Body.SetString(tag.Text, reason)
	return msg
}

// EncodeInboundRecordJSON is a small helper for the data-pipeline consumer
// when it needs to serialize an InboundRecord (e.g. for cross-process
// transport). Kept here to keep all inbound-record code in one place.
func EncodeInboundRecordJSON(rec InboundRecord) ([]byte, error) {
	return json.Marshal(rec)
}
