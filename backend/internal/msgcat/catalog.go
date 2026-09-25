package msgcat

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
)

type key struct{ set, nbr int }

// byLang is one message's text per language.
type byLang map[string]Entry

type snapshot struct {
	entries map[key]byLang
	loaded  time.Time
}

// Catalog renders catalog messages. Core text and each tenant's overrides
// are cached for TTL (edits made on this instance invalidate at once; other
// instances pick them up within TTL).
type Catalog struct {
	store   *Store
	ttl     time.Duration
	mu      sync.Mutex
	core    *snapshot
	tenants map[string]*snapshot
}

func NewCatalog(store *Store) *Catalog {
	return &Catalog{store: store, ttl: 30 * time.Second, tenants: map[string]*snapshot{}}
}

func index(entries []Entry) map[key]byLang {
	m := map[key]byLang{}
	for _, e := range entries {
		k := key{e.SetNbr, e.MessageNbr}
		if m[k] == nil {
			m[k] = byLang{}
		}
		m[k][e.Language] = e
	}
	return m
}

func (c *Catalog) coreSnapshot(ctx context.Context) (map[key]byLang, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.core != nil && time.Since(c.core.loaded) < c.ttl {
		return c.core.entries, nil
	}
	entries, err := c.store.CoreEntries(ctx)
	if err != nil {
		if c.core != nil {
			return c.core.entries, nil // serve stale over failing
		}
		return nil, err
	}
	c.core = &snapshot{index(entries), time.Now()}
	return c.core.entries, nil
}

func (c *Catalog) tenantSnapshot(ctx context.Context, tenantID string) (map[key]byLang, error) {
	if tenantID == "" {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := c.tenants[tenantID]; s != nil && time.Since(s.loaded) < c.ttl {
		return s.entries, nil
	}
	entries, err := c.store.TenantEntries(ctx, tenantID)
	if err != nil {
		if s := c.tenants[tenantID]; s != nil {
			return s.entries, nil
		}
		return nil, err
	}
	c.tenants[tenantID] = &snapshot{index(entries), time.Now()}
	return c.tenants[tenantID].entries, nil
}

// Invalidate drops cached text after an edit (tenantID "" = core).
func (c *Catalog) Invalidate(tenantID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if tenantID == "" {
		c.core = nil
		return
	}
	delete(c.tenants, tenantID)
}

// Rendered is a message ready to show a user.
type Rendered struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	UserAction string `json:"user_action,omitempty"`
	Language   string `json:"language"`
	Status     int    `json:"-"`
}

// Last resort when the catalog cannot be read at all: the envelope still
// never carries an internal cause.
const unavailableText = "An internal error occurred (ref: %1). Please contact support."

// Lookup finds the text for a message: for each preferred language (which
// always ends with English), the tenant's own text first, then the core
// catalog's. It reports false when the message is in neither.
func (c *Catalog) Lookup(ctx context.Context, tenantID string, langs []string, set, nbr int) (Entry, bool) {
	core, err := c.coreSnapshot(ctx)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("msgcat: loading core catalog: %v", err)
	}
	tenant, err := c.tenantSnapshot(ctx, tenantID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("msgcat: loading catalog overrides for tenant %s: %v", tenantID, err)
	}
	k := key{set, nbr}
	for _, l := range langs {
		if e, ok := tenant[k][l]; ok {
			return e, true
		}
		if e, ok := core[k][l]; ok {
			return e, true
		}
	}
	return Entry{}, false
}

// Render renders e for a tenant in the caller's preferred languages. A code
// missing from the catalog renders as the internal-error message with the
// given reference, so a gap in the catalog never leaks anything either.
func (c *Catalog) Render(ctx context.Context, tenantID string, langs []string, e *Error, ref string) Rendered {
	entry, ok := c.Lookup(ctx, tenantID, langs, e.Set, e.Nbr)
	if !ok {
		logging.GetLogger().Sugar().Errorf("msgcat: message %s is not in the catalog (ref %s)", e.Code(), ref)
		e = Internal(ref)
		if entry, ok = c.Lookup(ctx, tenantID, langs, e.Set, e.Nbr); !ok {
			entry = Entry{Language: BaseLanguage, Severity: string(SeverityFatal), Text: unavailableText}
		}
	}
	status := e.Status
	if status == 0 {
		status = http.StatusBadRequest
		if entry.Severity == string(SeverityFatal) {
			status = http.StatusInternalServerError
		}
	}
	return Rendered{
		Code:       e.Code(),
		Severity:   entry.Severity,
		Message:    Format(entry.Text, e.Params),
		UserAction: Format(entry.UserAction, e.Params),
		Language:   entry.Language,
		Status:     status,
	}
}
