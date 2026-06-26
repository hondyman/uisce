package bundles

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// In-memory sample data for PoC
var sampleEnts = map[string]Entitlement{
	"e1": {ID: "e1", Name: "campaign_view", Resource: "campaigns", Action: "read"},
	"e2": {ID: "e2", Name: "customer_segment", Resource: "customers", Action: "read"},
	"e3": {ID: "e3", Name: "conversion_rate", Resource: "analytics", Action: "read"},
	"e4": {ID: "e4", Name: "orders_write", Resource: "orders", Action: "write"},
	"e5": {ID: "e5", Name: "billing_view", Resource: "billing", Action: "read"},
}

type proposalRow struct {
	ID         string          `db:"id"`
	BundleID   sql.NullString  `db:"bundle_id"`
	ChangeType sql.NullString  `db:"change_type"`
	Details    json.RawMessage `db:"details"`
	Fitness    sql.NullFloat64 `db:"fitness_score"`
	Risk       sql.NullFloat64 `db:"risk_score"`
	Status     sql.NullString  `db:"status"`
}

func getProposal(db *sqlx.DB, id string) (*proposalRow, error) {
	if db == nil {
		return nil, fmt.Errorf("no db")
	}
	var p proposalRow
	err := db.Get(&p, `SELECT id, bundle_id, change_type, details, fitness_score, risk_score, status FROM bundle_change_proposal WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GuardrailConfig holds runtime-configured guardrail rules.
type GuardrailConfig struct {
	SoDPairs  [][2]string `json:"sod_pairs" yaml:"sod_pairs"`
	Certified []string    `json:"certified" yaml:"certified"`
}

// GuardrailCache holds the cached config plus metadata.
type GuardrailCache struct {
	Config     *GuardrailConfig `json:"config"`
	LastLoaded time.Time        `json:"last_loaded"`
	Source     string           `json:"source"`
}

// In-memory cache for guardrails (optional). Reload with ReloadGuardrails.
var guardrailsCache *GuardrailCache
var guardrailsMutex sync.RWMutex
var initDBFunc = InitDBFromConfig

// ReloadGuardrails loads guardrails from the DB (or YAML fallback) into the in-memory cache.
func ReloadGuardrails(db *sqlx.DB) error {
	cfg, src, err := loadGuardrails(db)
	if err != nil {
		return err
	}
	guardrailsMutex.Lock()
	guardrailsCache = &GuardrailCache{Config: cfg, LastLoaded: time.Now(), Source: src}
	guardrailsMutex.Unlock()
	logging.GetLogger().Info("guardrails_reload", zap.Int("sod_pairs", len(cfg.SoDPairs)), zap.Int("certified", len(cfg.Certified)), zap.String("source", src), zap.Time("last_loaded", guardrailsCache.LastLoaded))
	return nil
}

// getGuardrails returns cached guardrails if present, otherwise loads and caches them.
func getGuardrails(db *sqlx.DB) (*GuardrailConfig, error) {
	// If no DB provided, always attempt to (re)load from YAML paths to ensure
	// tests and local workflows that write guardrails.yaml are honored.
	if db == nil {
		cfg, src, err := loadGuardrails(nil)
		if err != nil {
			return nil, err
		}
		guardrailsMutex.Lock()
		guardrailsCache = &GuardrailCache{Config: cfg, LastLoaded: time.Now(), Source: src}
		guardrailsMutex.Unlock()
		logging.GetLogger().Info("guardrails_load", zap.Int("sod_pairs", len(cfg.SoDPairs)), zap.Int("certified", len(cfg.Certified)), zap.String("source", src), zap.Time("last_loaded", guardrailsCache.LastLoaded))
		return cfg, nil
	}

	guardrailsMutex.RLock()
	if guardrailsCache != nil {
		cfg := guardrailsCache.Config
		guardrailsMutex.RUnlock()
		return cfg, nil
	}
	guardrailsMutex.RUnlock()
	cfg, src, err := loadGuardrails(db)
	if err != nil {
		return nil, err
	}
	guardrailsMutex.Lock()
	guardrailsCache = &GuardrailCache{Config: cfg, LastLoaded: time.Now(), Source: src}
	guardrailsMutex.Unlock()
	logging.GetLogger().Info("guardrails_load", zap.Int("sod_pairs", len(cfg.SoDPairs)), zap.Int("certified", len(cfg.Certified)), zap.String("source", src), zap.Time("last_loaded", guardrailsCache.LastLoaded))
	return cfg, nil
}

// loadGuardrails tries DB first; if db==nil or no rules, falls back to YAML (env GUARDRAILS_PATH or common files)
func loadGuardrails(db *sqlx.DB) (*GuardrailConfig, string, error) {
	cfg := &GuardrailConfig{}
	sourced := "none"
	if db != nil {
		// table schema: guardrail_rules(type TEXT, data JSONB)
		rows, err := db.Queryx(`SELECT type, data FROM guardrail_rules`)
		if err == nil {
			defer rows.Close()
			found := false
			for rows.Next() {
				var typ string
				var data []byte
				if err := rows.Scan(&typ, &data); err != nil {
					continue
				}
				switch typ {
				case "sod":
					var payload struct {
						Pairs [][2]string `json:"pairs"`
					}
					_ = json.Unmarshal(data, &payload)
					if len(payload.Pairs) > 0 {
						cfg.SoDPairs = append(cfg.SoDPairs, payload.Pairs...)
						found = true
					}
				case "certified":
					var payload struct {
						Claims []string `json:"claims"`
					}
					_ = json.Unmarshal(data, &payload)
					if len(payload.Claims) > 0 {
						cfg.Certified = append(cfg.Certified, payload.Claims...)
						found = true
					}
				}
			}
			if found {
				sourced = "db"
				return cfg, sourced, nil
			}
		}
	}

	// If none found in DB, load from YAML config paths
	tryPaths := []string{}
	if p := os.Getenv("GUARDRAILS_PATH"); p != "" {
		tryPaths = append(tryPaths, p)
	}
	tryPaths = append(tryPaths, "guardrails.yaml", "../guardrails.yaml", "../../guardrails.yaml")
	for _, p := range tryPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var y GuardrailConfig
		if err := yaml.Unmarshal(b, &y); err != nil {
			continue
		}
		if len(y.SoDPairs) > 0 {
			cfg.SoDPairs = y.SoDPairs
		}
		if len(y.Certified) > 0 {
			cfg.Certified = y.Certified
		}
		if len(cfg.SoDPairs) > 0 || len(cfg.Certified) > 0 {
			sourced = "yaml"
		}
		break
	}
	return cfg, sourced, nil
}

// loadGuardrailsFromSource attempts to load guardrails from a specific source.
// source can be "db" or "yaml". If source is empty, falls back to default behavior.
func loadGuardrailsFromSource(db *sqlx.DB, source string) (*GuardrailConfig, string, error) {
	switch strings.ToLower(source) {
	case "db":
		if db == nil {
			return &GuardrailConfig{}, "db", fmt.Errorf("no db configured")
		}
		// attempt DB only
		cfg := &GuardrailConfig{}
		rows, err := db.Queryx(`SELECT type, data FROM guardrail_rules`)
		if err != nil {
			return cfg, "db", err
		}
		defer rows.Close()
		found := false
		for rows.Next() {
			var typ string
			var data []byte
			if err := rows.Scan(&typ, &data); err != nil {
				continue
			}
			switch typ {
			case "sod":
				var payload struct {
					Pairs [][2]string `json:"pairs"`
				}
				_ = json.Unmarshal(data, &payload)
				if len(payload.Pairs) > 0 {
					cfg.SoDPairs = append(cfg.SoDPairs, payload.Pairs...)
					found = true
				}
			case "certified":
				var payload struct {
					Claims []string `json:"claims"`
				}
				_ = json.Unmarshal(data, &payload)
				if len(payload.Claims) > 0 {
					cfg.Certified = append(cfg.Certified, payload.Claims...)
					found = true
				}
			}
		}
		if !found {
			return cfg, "db", fmt.Errorf("no guardrails found in db")
		}
		return cfg, "db", nil
	case "yaml":
		// load from YAML only
		cfg := &GuardrailConfig{}
		tryPaths := []string{}
		if p := os.Getenv("GUARDRAILS_PATH"); p != "" {
			tryPaths = append(tryPaths, p)
		}
		tryPaths = append(tryPaths, "guardrails.yaml", "../guardrails.yaml", "../../guardrails.yaml")
		for _, p := range tryPaths {
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var y GuardrailConfig
			if err := yaml.Unmarshal(b, &y); err != nil {
				continue
			}
			if len(y.SoDPairs) > 0 {
				cfg.SoDPairs = y.SoDPairs
			}
			if len(y.Certified) > 0 {
				cfg.Certified = y.Certified
			}
			return cfg, "yaml", nil
		}
		return cfg, "yaml", fmt.Errorf("no yaml guardrails found")
	default:
		return loadGuardrails(db)
	}
}

func evaluateGuardrails(db *sqlx.DB, details json.RawMessage) (bool, []string, error) {
	var d struct {
		Claims      []string `json:"claims"`
		Description string   `json:"description"`
	}
	if err := json.Unmarshal(details, &d); err != nil {
		return false, nil, err
	}

	cfg, err := getGuardrails(db)
	if err != nil {
		return false, nil, err
	}

	// debug: log loaded config and incoming claims
	logging.GetLogger().Debug("evaluateGuardrails: cfg", zap.Any("sod_pairs", cfg.SoDPairs), zap.Any("certified", cfg.Certified), zap.Any("claims_in", d.Claims))

	reasons := []string{}
	claimSet := map[string]bool{}
	for _, c := range d.Claims {
		claimSet[c] = true
	}

	for _, pair := range cfg.SoDPairs {
		if len(pair) >= 2 {
			if claimSet[pair[0]] && claimSet[pair[1]] {
				reasons = append(reasons, fmt.Sprintf("SoD conflict between %s and %s", pair[0], pair[1]))
			}
		}
	}

	certMap := map[string]bool{}
	for _, c := range cfg.Certified {
		certMap[c] = true
	}
	for _, c := range d.Claims {
		if certMap[c] {
			reasons = append(reasons, fmt.Sprintf("Certified asset present: %s requires explicit review", c))
		}
	}

	ok := len(reasons) == 0
	return ok, reasons, nil
}

// applyProposal writes a new claim_bundle and claim_bundle_item rows for change_type='add'
func applyProposal(db *sqlx.DB, proposalID, actor string) error {
	p, err := getProposal(db, proposalID)
	if err != nil {
		return err
	}
	// parse details
	var d struct {
		Claims      []string `json:"claims"`
		Description string   `json:"description"`
	}
	if err := json.Unmarshal(p.Details, &d); err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// create bundle
	bundleID := uuidNew()
	name := d.Description
	if name == "" {
		name = "Auto bundle " + bundleID[:8]
	}
	now := time.Now()
	_, err = tx.Exec(`INSERT INTO claim_bundle (id, name, version, domain, description, created_by, created_at, status, risk_level) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		bundleID, name, 1, "poctenant", d.Description, actor, now, "active", "low")
	if err != nil {
		return err
	}
	// insert items
	for _, claim := range d.Claims {
		itemID := uuidNew()
		// store claim id in model_id for PoC and permission 'read'
		_, err = tx.Exec(`INSERT INTO claim_bundle_item (id, bundle_id, model_id, permission, scope) VALUES ($1,$2,$3,$4,$5)`, itemID, bundleID, claim, "read", json.RawMessage(`[]`))
		if err != nil {
			return err
		}
	}
	// mark proposal decided/applied
	_, err = tx.Exec(`UPDATE bundle_change_proposal SET status='auto_applied', decided_at=$1, decided_by=$2 WHERE id=$3`, now, actor, proposalID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RegisterRoutes pins all bundle handlers onto the provided chi router
func RegisterRoutes(r chi.Router) {
	r.Route("/bundles", func(r chi.Router) {
		r.Use(AdminAuthMiddleware())
		r.Post("/analyze", TriggerAnalyzeHandler)
		r.Get("/proposals", ListProposalsHandler)
		r.Post("/proposals/{id}/approve", ApproveProposalHandler)
		r.Post("/proposals/{id}/apply", ApplyProposalHandler)
		r.Post("/proposals/{id}/reject", RejectProposalHandler)
		r.Post("/candidates/generate", GenerateBundlesHandler)

		r.Route("/guardrails", func(r chi.Router) {
			r.Get("/", ListGuardrailsHandler)
			r.Post("/", CreateGuardrailHandler)
			r.Get("/cache", GetGuardrailsCacheHandler)
			r.Post("/reload", ReloadGuardrailsHandler)
			r.Post("/force-load", ForceLoadGuardrailsHandler)
			r.Put("/{id}", UpdateGuardrailHandler)
			r.Delete("/{id}", DeleteGuardrailHandler)
		})
	})
}

// uuidNew returns a UUID string (simple wrapper)
func uuidNew() string { return fmt.Sprintf("id-%d", time.Now().UnixNano()) }

// Approve/Apply/Reject HTTP handlers
func ApproveProposalHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Approver string `json:"approver"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p, err := getProposal(db, id)
	if err != nil {
		http.Error(w, "proposal not found", http.StatusNotFound)
		return
	}
	// evaluate guardrails
	ok, reasons, err := evaluateGuardrails(db, p.Details)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		// require steward review: set status=approved but not applied
		_, _ = db.Exec(`UPDATE bundle_change_proposal SET status='approved', decided_at=$1, decided_by=$2 WHERE id=$3`, now, body.Approver, id)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "approved_pending_review", "reasons": reasons})
		return
	}
	// auto-apply low-risk
	if p.Risk.Valid && p.Risk.Float64 < 30 {
		if err := applyProposal(db, id, body.Approver); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "auto_applied"})
		return
	}
	// mark approved (manual apply later)
	_, _ = db.Exec(`UPDATE bundle_change_proposal SET status='approved', decided_at=$1, decided_by=$2 WHERE id=$3`, now, body.Approver, id)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "approved"})
}

func ApplyProposalHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Force bool   `json:"force"`
		Actor string `json:"actor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db, err := InitDBFromConfig("../config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p, err := getProposal(db, id)
	if err != nil {
		http.Error(w, "proposal not found", http.StatusNotFound)
		return
	}
	ok, reasons, err := evaluateGuardrails(db, p.Details)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok && !body.Force {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "guardrail failure", "reasons": reasons})
		return
	}
	actor := body.Actor
	if actor == "" {
		actor = "system"
	}
	if err := applyProposal(db, id, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "applied"})
}

func RejectProposalHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Approver string `json:"approver"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	_, err = db.Exec(`UPDATE bundle_change_proposal SET status='rejected', decided_at=$1, decided_by=$2 WHERE id=$3`, now, body.Approver, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "rejected"})
}

var sampleEvents = []UsageEvent{
	{Timestamp: time.Now().Add(-24 * time.Hour), UserID: "u1", TenantID: "t1", EntitlementID: "e1", Count: 10},
	{Timestamp: time.Now().Add(-48 * time.Hour), UserID: "u1", TenantID: "t1", EntitlementID: "e2", Count: 5},
	{Timestamp: time.Now().Add(-2 * 24 * time.Hour), UserID: "u2", TenantID: "t1", EntitlementID: "e1", Count: 3},
	{Timestamp: time.Now().Add(-3 * 24 * time.Hour), UserID: "u2", TenantID: "t1", EntitlementID: "e3", Count: 7},
	{Timestamp: time.Now().Add(-10 * 24 * time.Hour), UserID: "u3", TenantID: "t1", EntitlementID: "e5", Count: 20},
	{Timestamp: time.Now().Add(-5 * 24 * time.Hour), UserID: "u4", TenantID: "t1", EntitlementID: "e4", Count: 1},
}

// GenerateBundlesHandler runs the miner and returns candidates (JSON)
func GenerateBundlesHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
		MinUsage int    `json:"min_usage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Accept empty body -> defaults
		req.TenantID = "t1"
		req.MinUsage = 1
	}
	if req.TenantID == "" {
		req.TenantID = "t1"
	}
	if req.MinUsage <= 0 {
		req.MinUsage = 1
	}

	candidates := MineCandidates(sampleEnts, sampleEvents, req.TenantID, req.MinUsage)
	// attempt to persist candidates if DB is configured
	if db != nil {
		if err := PersistCandidates(db, candidates); err != nil {
			http.Error(w, "failed to persist candidates: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"candidates": candidates})
}

// Exported helpers for PoC main to avoid cycle
func SampleEnts() map[string]Entitlement { return sampleEnts }
func SampleEvents() []UsageEvent         { return sampleEvents }

// --- Persistence helpers (optional, use Postgres if available) ---

var db *sqlx.DB
var dbOnce sync.Once

// InitDB initializes package DB using env POLICIES_DB or config.yaml dsn
func InitDBFromConfig(configPath string) (*sqlx.DB, error) {
	dbOnce.Do(func() {})
	if db != nil {
		return db, nil
	}

	candidates := resolvePolicyDBDSNs(configPath)
	if len(candidates) == 0 {
		return nil, nil
	}

	var lastErr error
	for _, candidate := range candidates {
		conn, err := sqlx.Connect("postgres", candidate)
		if err != nil {
			lastErr = err
			logging.GetLogger().Sugar().Warnw("policy_db_connect_failed", "dsn", sanitizeDSN(candidate), "error", err.Error())
			continue
		}

		db = conn
		_, _ = db.Exec(`
			CREATE TABLE IF NOT EXISTS candidate_bundles (
				id TEXT PRIMARY KEY,
				tenant_id TEXT,
				name TEXT,
				description TEXT,
				claims JSONB,
				scope TEXT,
				score DOUBLE PRECISION,
				risk DOUBLE PRECISION,
				explanations JSONB,
				status TEXT,
				created_at TIMESTAMP WITH TIME ZONE
			)
		`)
		return db, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to connect to policy database: %w", lastErr)
	}
	return nil, nil
}

func resolvePolicyDBDSNs(configPath string) []string {
	seen := map[string]struct{}{}
	dsns := []string{}

	add := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		dsns = append(dsns, trimmed)
		seen[trimmed] = struct{}{}
	}

	add(os.Getenv("POLICY_DB_URL"))

	tryPaths := []string{}
	if configPath != "" {
		tryPaths = append(tryPaths, configPath)
	}
	tryPaths = append(tryPaths, "config.yaml", "../config.yaml", "../../config.yaml")
	for _, p := range tryPaths {
		f, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var conf struct {
			Dsn string `yaml:"dsn"`
		}
		if err := yaml.Unmarshal(f, &conf); err != nil {
			continue
		}
		add(conf.Dsn)
	}

	add(os.Getenv("ALPHA_DATABASE_URL"))
	add(os.Getenv("ALPHA_PG_DSN"))
	add(os.Getenv("DATABASE_URL"))
	add(os.Getenv("ROLE_DATABASE_URL"))
	add("postgres://postgres@localhost:5432/alpha?sslmode=disable")

	// 100.84.126.19 fallback -> localhost
	existing := append([]string(nil), dsns...)
	for _, candidate := range existing {
		if fallback := dockerHostFallback(candidate); fallback != "" {
			add(fallback)
		}
	}

	return dsns
}

func dockerHostFallback(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if !strings.EqualFold(host, "100.84.126.19") {
		return ""
	}
	port := parsed.Port()
	if port == "" {
		port = "5432"
	}
	parsed.Host = net.JoinHostPort("localhost", port)
	return parsed.String()
}

func sanitizeDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "unknown"
	}
	parsed.User = nil
	if parsed.RawQuery != "" {
		parsed.RawQuery = ""
	}
	if parsed.Fragment != "" {
		parsed.Fragment = ""
	}
	return parsed.String()
}

// PersistCandidates inserts candidate bundles into table candidate_bundles (id PK)
func PersistCandidates(db *sqlx.DB, candidates []CandidateBundle) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, c := range candidates {
		claimsJSON, _ := json.Marshal(c.Claims)
		explanationsJSON, _ := json.Marshal(c.Explanations)
		_, err := tx.Exec(`INSERT INTO candidate_bundles (id, tenant_id, name, description, claims, scope, score, risk, explanations, status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, description=EXCLUDED.description, claims=EXCLUDED.claims, score=EXCLUDED.score, risk=EXCLUDED.risk, explanations=EXCLUDED.explanations, status=EXCLUDED.status, created_at=EXCLUDED.created_at`,
			c.ID, c.TenantID, c.Name, c.Description, claimsJSON, c.Scope, c.Score, c.Risk, explanationsJSON, c.Status, c.CreatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ApproveBundle sets status=approved
func ApproveBundle(db *sqlx.DB, id string) error {
	_, err := db.Exec(`UPDATE candidate_bundles SET status='approved' WHERE id=$1`, id)
	return err
}

// RejectBundle sets status=rejected
func RejectBundle(db *sqlx.DB, id string) error {
	_, err := db.Exec(`UPDATE candidate_bundles SET status='rejected' WHERE id=$1`, id)
	return err
}

// ListPersistedCandidates returns persisted candidate bundles for a tenant (limit>0)
func ListPersistedCandidates(db *sqlx.DB, tenantID string, limit int) ([]CandidateBundle, error) {
	if db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	var rows *sqlx.Rows
	var err error
	if tenantID != "" {
		rows, err = db.Queryx(`SELECT id, tenant_id, name, description, claims, scope, score, risk, explanations, status, created_at FROM candidate_bundles WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	} else {
		rows, err = db.Queryx(`SELECT id, tenant_id, name, description, claims, scope, score, risk, explanations, status, created_at FROM candidate_bundles ORDER BY created_at DESC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CandidateBundle{}
	for rows.Next() {
		var cb CandidateBundle
		var claimsB []byte
		var explB []byte
		var desc sql.NullString
		var scope sql.NullString
		var status sql.NullString
		if err := rows.Scan(&cb.ID, &cb.TenantID, &cb.Name, &desc, &claimsB, &scope, &cb.Score, &cb.Risk, &explB, &status, &cb.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			cb.Description = desc.String
		} else {
			cb.Description = ""
		}
		if scope.Valid {
			cb.Scope = scope.String
		} else {
			cb.Scope = ""
		}
		if status.Valid {
			cb.Status = status.String
		} else {
			cb.Status = ""
		}
		if len(claimsB) > 0 {
			_ = json.Unmarshal(claimsB, &cb.Claims)
		}
		if len(explB) > 0 {
			_ = json.Unmarshal(explB, &cb.Explanations)
		}
		out = append(out, cb)
	}
	return out, nil
}

// HTTP handler: trigger analysis and proposal creation
func TriggerAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	created, err := AnalyzeAndPropose(db, "t1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"created": created})
}

// HTTP handler: list proposals
func ListProposalsHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	list, err := ListProposals(db, status, 200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"proposals": list})
}

// --- Admin handlers for guardrail rules (CRUD) ---
// GuardrailRule represents a DB row
type GuardrailRule struct {
	ID   string          `db:"id" json:"id"`
	Type string          `db:"type" json:"type"`
	Data json.RawMessage `db:"data" json:"data"`
}

func CreateGuardrailHandler(w http.ResponseWriter, r *http.Request) {
	var in GuardrailRule
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// validate payload per rule type
	if err := validateGuardrailData(in.Type, in.Data); err != nil {
		http.Error(w, fmt.Sprintf("invalid data for type %s: %v", in.Type, err), http.StatusBadRequest)
		return
	}
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil || db == nil {
		http.Error(w, "no db configured", http.StatusBadRequest)
		return
	}
	id := uuidNew()
	_, err = db.Exec(`INSERT INTO guardrail_rules (id, type, data) VALUES ($1,$2,$3)`, id, in.Type, in.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	in.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(in)
}

func ListGuardrailsHandler(w http.ResponseWriter, r *http.Request) {
	db, _ := InitDBFromConfig("../config.yaml")
	out := []GuardrailRule{}
	if db != nil {
		rows, err := db.Queryx(`SELECT id, type, data FROM guardrail_rules`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r GuardrailRule
				var data []byte
				if err := rows.Scan(&r.ID, &r.Type, &data); err != nil {
					continue
				}
				r.Data = json.RawMessage(data)
				out = append(out, r)
			}
		}
	}
	// also include cache metadata if present
	guardrailsMutex.RLock()
	cached := guardrailsCache
	guardrailsMutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if cached != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"rules": out, "cache": cached})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"rules": out})
}

// ForceLoadGuardrailsHandler forces loading from a specific source (db|yaml) and updates the cache.
func ForceLoadGuardrailsHandler(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	db, _ := initDBFunc("../config.yaml")
	cfg, src, err := loadGuardrailsFromSource(db, source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	guardrailsMutex.Lock()
	guardrailsCache = &GuardrailCache{Config: cfg, LastLoaded: time.Now(), Source: src}
	guardrailsMutex.Unlock()
	// structured log
	logging.GetLogger().Info("guardrails_forced_load", zap.Int("sod_pairs", len(cfg.SoDPairs)), zap.Int("certified", len(cfg.Certified)), zap.String("source", src), zap.Time("last_loaded", guardrailsCache.LastLoaded))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"loaded": true, "cache": guardrailsCache})
}

func DeleteGuardrailHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil || db == nil {
		http.Error(w, "no db configured", http.StatusBadRequest)
		return
	}
	_, err = db.Exec(`DELETE FROM guardrail_rules WHERE id=$1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "deleted": true})
}

func UpdateGuardrailHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in GuardrailRule
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// validate payload per rule type
	if err := validateGuardrailData(in.Type, in.Data); err != nil {
		http.Error(w, fmt.Sprintf("invalid data for type %s: %v", in.Type, err), http.StatusBadRequest)
		return
	}
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil || db == nil {
		http.Error(w, "no db configured", http.StatusBadRequest)
		return
	}
	_, err = db.Exec(`UPDATE guardrail_rules SET type=$1, data=$2 WHERE id=$3`, in.Type, in.Data, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	in.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(in)
}

// ReloadGuardrailsHandler is an admin endpoint to reload the in-memory guardrail cache.
func ReloadGuardrailsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := InitDBFromConfig("../config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := ReloadGuardrails(db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	guardrailsMutex.RLock()
	cached := guardrailsCache
	guardrailsMutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if cached == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"reloaded": true, "cache": nil})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"reloaded": true, "cache": cached})
}

// GetGuardrailsCacheHandler returns the cached guardrail configuration (admin-only).
func GetGuardrailsCacheHandler(w http.ResponseWriter, r *http.Request) {
	guardrailsMutex.RLock()
	cached := guardrailsCache
	guardrailsMutex.RUnlock()
	if cached == nil {
		http.Error(w, "no cache loaded", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"cache": cached})
}

// AdminAuthMiddleware enforces simple RBAC for admin endpoints.
// Allows GET without auth, but requires either header X-User-Role: admin
// or Authorization: Bearer <ADMIN_API_KEY> for modifying requests (POST/PUT/DELETE).
func AdminAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// allow health/read operations
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			// allow if header role is admin
			if strings.ToLower(r.Header.Get("X-User-Role")) == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			// allow if Authorization Bearer token matches env ADMIN_API_KEY
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
				if token != "" && os.Getenv("ADMIN_API_KEY") != "" && token == os.Getenv("ADMIN_API_KEY") {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "admin required"})
		})
	}
}

// validateGuardrailData verifies the JSON shape for known types.
func validateGuardrailData(typ string, data json.RawMessage) error {
	// Strict JSON Schema validation per type using gojsonschema
	var schemaStr string
	switch typ {
	case "sod":
		// pairs: array of arrays with exactly 2 non-empty strings
		schemaStr = `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"pairs": {
					"type": "array",
					"items": {
						"type": "array",
						"minItems": 2,
						"maxItems": 2,
						"items": { "type": "string", "minLength": 1 }
					},
					"minItems": 1
				}
			},
			"required": ["pairs"],
			"additionalProperties": false
		}`
	case "certified":
		// claims: non-empty array of non-empty strings
		schemaStr = `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"claims": { "type": "array", "minItems": 1, "items": { "type": "string", "minLength": 1 } }
			},
			"required": ["claims"],
			"additionalProperties": false
		}`
	default:
		return fmt.Errorf("unknown guardrail type: %s", typ)
	}

	schemaLoader := gojsonschema.NewStringLoader(schemaStr)
	documentLoader := gojsonschema.NewBytesLoader(data)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		parts := []string{}
		for _, e := range result.Errors() {
			parts = append(parts, e.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(parts, "; "))
	}
	return nil
}
