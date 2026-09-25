package schedule

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Runner runs one kind of target (report, saved query, data pipeline, ...).
// Runners live next to the thing they run and register with the scheduler,
// so the scheduler itself knows nothing about reports or pipelines.
type Runner interface {
	Kind() string
	// Label is the target kind in plain words ("Report").
	Label() string
	// Check confirms the target exists and userID may run it in the tenant,
	// and the params are acceptable, when a schedule is created or changed.
	Check(ctx context.Context, tenantID, userID, ref string, params map[string]any) error
	// Targets lists what userID can schedule in the tenant, for pickers.
	Targets(ctx context.Context, tenantID, userID string) ([]TargetInfo, error)
	// Run runs the target once as the schedule's owner. It must be
	// idempotent enough that a firing is never run twice (the engine does
	// not retry a failed run). Errors should be msgcat errors; anything else
	// is recorded as an internal error.
	Run(ctx context.Context, rc RunContext) (*Outcome, error)
}

// TargetInfo is one schedulable thing.
type TargetInfo struct {
	Ref         string `json:"ref"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RunContext is everything a runner gets for one run.
type RunContext struct {
	RunID        string
	ScheduleID   string
	TenantID     string
	OwnerID      string
	DatasourceID string
	Region       string
	Ref          string
	Params       map[string]any
	ScheduledFor time.Time
	// Heartbeat reports progress so long runs are not considered stuck.
	Heartbeat func(details ...any)
}

// Outcome is what a run produced.
type Outcome struct {
	Rows    int            `json:"rows,omitempty"`
	Summary string         `json:"summary,omitempty"`
	Refs    map[string]any `json:"refs,omitempty"` // e.g. report execution id
	// Output is an optional file the run produced, kept with the run.
	Output *Output `json:"-"`
}

// Output is a file produced by a run.
type Output struct {
	FileName    string
	ContentType string
	Content     []byte
	Rows        int
}

// Registry holds the runners by kind.
type Registry struct {
	mu      sync.RWMutex
	runners map[string]Runner
}

func NewRegistry(rs ...Runner) *Registry {
	r := &Registry{runners: map[string]Runner{}}
	for _, x := range rs {
		r.Register(x)
	}
	return r
}

func (r *Registry) Register(x Runner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runners[x.Kind()] = x
}

func (r *Registry) Get(kind string) (Runner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.runners[kind]
	if !ok {
		return nil, msgUnknownKind(kind)
	}
	return x, nil
}

// KindInfo is a schedulable kind.
type KindInfo struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

func (r *Registry) Kinds() []KindInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]KindInfo, 0, len(r.runners))
	for k, x := range r.runners {
		out = append(out, KindInfo{Kind: k, Label: x.Label()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}
