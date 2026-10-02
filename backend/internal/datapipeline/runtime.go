package datapipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Row is one record flowing through the pipeline. Num is the 1-based row
// number in the original source and is preserved through transforms so sinks
// can write staging._source_row_num and errors can point at the file row.
type Row struct {
	Num  int
	Data map[string]any
}

// Reject is a row a node refused, with the reason.
type Reject struct {
	Row    Row
	NodeID string
	Field  string
	Reason string
}

// Result of processing one batch.
type Result struct {
	Out      []Row
	Rejected []Reject
	// Warnings are rows that passed but tripped a non-blocking rule.
	Warnings []Reject
}

// Processor is a non-source node. Open/Close bracket the run so sinks can
// create and finalize bookkeeping (e.g. staging._load_run); Close receives the
// run error (nil on success) so a sink can mark the run FAILED.
type Processor interface {
	Open(ctx context.Context, rc *RunContext) error
	Process(ctx context.Context, rows []Row) (Result, error)
	Close(ctx context.Context, runErr error) error
}

// Source produces batches. rejected are rows the source itself refused (e.g.
// a value that does not fit the file contract). emit returning an error
// stops the stream.
type Source interface {
	Stream(ctx context.Context, rc *RunContext, batchSize int, emit func(rows []Row, rejected []Reject) error) error
}

// RunContext is what every node may know about the current run.
type RunContext struct {
	RunID    string
	TenantID string
	Spec     *Spec
	// Preview: MaxRows stops each source after that many rows, SampleRows
	// keeps the first rows leaving every node, and DryRun makes sinks write
	// nothing (a BO sink still has every row judged by the rule engine).
	MaxRows    int
	SampleRows int
	DryRun     bool
	// Loads: the load run each staging sink claimed (node id -> load run
	// id), for the master steps that follow them.
	Loads map[string]string
}

// MasterRequest asks for a committed staging load to be mastered.
type MasterRequest struct {
	TenantID      string
	Entity        string
	StagingTable  string
	LoadRunID     string
	PipelineRunID string
}

// MasterResult is the mastering run a master step started.
type MasterResult struct {
	NodeID        string `json:"node_id"`
	Entity        string `json:"entity"`
	LoadRunID     string `json:"load_run_id"`
	RunID         string `json:"run_id"`
	Status        string `json:"status"`
	Records       int    `json:"records"`
	Valid         int    `json:"valid"`
	Invalid       int    `json:"invalid"`
	Published     int    `json:"published"`
	HeldForReview int    `json:"held_for_review"`
	Exceptions    int    `json:"exceptions"`
	Replayed      bool   `json:"replayed,omitempty"`
}

// Masterer masters committed staging loads (the mastering engine, wired by
// the API server; nil: master steps can't run).
type Masterer interface {
	MasterLoad(ctx context.Context, r MasterRequest) (*MasterResult, error)
}

// Recorder receives per-run observability. Implementations persist to
// data_pipeline_runs / data_pipeline_step_telemetry / staging._mapping_error.
type Recorder interface {
	NodeDone(ctx context.Context, s NodeStats)
	Rejected(ctx context.Context, r Reject)
	Warned(ctx context.Context, r Reject)
}

// NodeStats mirrors data_pipeline_step_telemetry.
type NodeStats struct {
	NodeID     string
	Label      string
	Type       string
	In         int64
	Out        int64
	Errors     int64
	Warnings   int64
	Duration   time.Duration
	Status     string // COMPLETED | FAILED
	Err        string
	OrderIndex int
}

// Factory builds the runtime for a node. Deps (engine, BO client, DB) are
// closed over by the factory so the runner stays pure and testable.
type Factory interface {
	Source(n Node) (Source, error)
	Processor(n Node) (Processor, error)
}

// Summary is the run outcome.
type Summary struct {
	Nodes []NodeStats
	// Mastering: the mastering runs the pipeline's master steps started.
	Mastering  []MasterResult   `json:"mastering,omitempty"`
	Samples    map[string][]Row `json:"samples,omitempty"` // preview only
	RecordsIn  int64            // rows read from sources
	RecordsOut int64            // rows accepted by sinks
	Errors     int64
}

const defaultBatchSize = 2000

// errPreviewDone stops a source once a preview has enough rows.
var errPreviewDone = errors.New("preview row limit reached")

// Run executes the spec. The graph is a forest (each non-source node has one
// parent, validated by Spec.Validate), so batches flow depth-first from each
// source through its children with no buffering between nodes.
func Run(ctx context.Context, spec *Spec, rc *RunContext, f Factory, rec Recorder) (*Summary, error) {
	if errs := spec.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("invalid pipeline: %w", errors.Join(errs...))
	}
	order, err := spec.TopoOrder()
	if err != nil {
		return nil, err
	}
	rc.Spec = spec
	batchSize := spec.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	failFast := spec.ErrorPolicy == ErrorPolicyFailFast

	nodes := map[string]Node{}
	for _, n := range spec.Nodes {
		nodes[n.ID] = n
	}
	// Master steps run after the stream, once every sink has committed; they
	// are not in the stream (a staging sink with one is still a sink).
	children := map[string][]string{}
	masterOf := map[string]string{} // master node -> its staging sink
	for _, e := range spec.Edges {
		if nodes[e.To].Type == NodeMaster {
			masterOf[e.To] = e.From
			continue
		}
		children[e.From] = append(children[e.From], e.To)
	}
	if rc.Loads == nil {
		rc.Loads = map[string]string{}
	}

	stats := map[string]*NodeStats{}
	procs := map[string]Processor{}
	for i, id := range order {
		n := nodes[id]
		stats[id] = &NodeStats{NodeID: id, Label: n.Label, Type: n.Type, OrderIndex: i, Status: "COMPLETED"}
		switch n.Type {
		case NodeFileSource, NodeBOSource, NodeQueueSource, NodeMaster:
		default:
			p, err := f.Processor(n)
			if err != nil {
				return nil, fmt.Errorf("node %q: %w", id, err)
			}
			procs[id] = p
		}
	}

	// Open sinks/processors in order; on failure close what opened.
	var opened []string
	closeAll := func(runErr error) error {
		var errs []error
		for i := len(opened) - 1; i >= 0; i-- {
			if err := procs[opened[i]].Close(ctx, runErr); err != nil {
				errs = append(errs, fmt.Errorf("close %q: %w", opened[i], err))
			}
		}
		return errors.Join(errs...)
	}
	for _, id := range order {
		p, ok := procs[id]
		if !ok {
			continue
		}
		if err := p.Open(ctx, rc); err != nil {
			stats[id].Status, stats[id].Err = "FAILED", err.Error()
			_ = closeAll(err)
			return summarize(order, stats, nodes), fmt.Errorf("open %q: %w", id, err)
		}
		opened = append(opened, id)
	}

	sum := &Summary{}
	sample := func(id string, rows []Row) {
		if rc.SampleRows <= 0 {
			return
		}
		if sum.Samples == nil {
			sum.Samples = map[string][]Row{}
		}
		for _, r := range rows {
			if len(sum.Samples[id]) >= rc.SampleRows {
				return
			}
			sum.Samples[id] = append(sum.Samples[id], r)
		}
	}
	var push func(id string, rows []Row) error
	push = func(id string, rows []Row) error {
		if len(rows) == 0 {
			return nil
		}
		st := stats[id]
		start := time.Now()
		st.In += int64(len(rows))
		res, err := procs[id].Process(ctx, rows)
		st.Duration += time.Since(start)
		if err != nil {
			st.Status, st.Err = "FAILED", err.Error()
			return fmt.Errorf("node %q: %w", id, err)
		}
		st.Out += int64(len(res.Out))
		sample(id, res.Out)
		st.Errors += int64(len(res.Rejected))
		sum.Errors += int64(len(res.Rejected))
		for _, w := range res.Warnings {
			w.NodeID = id
			st.Warnings++
			if rec != nil {
				rec.Warned(ctx, w)
			}
		}
		for _, rj := range res.Rejected {
			rj.NodeID = id
			if rec != nil {
				rec.Rejected(ctx, rj)
			}
			if failFast {
				st.Status, st.Err = "FAILED", rj.Reason
				return fmt.Errorf("node %q row %d: %s", id, rj.Row.Num, rj.Reason)
			}
		}
		kids := children[id]
		if len(kids) == 0 { // sink
			sum.RecordsOut += int64(len(res.Out))
			return nil
		}
		for _, k := range kids {
			// Copy so a child that mutates rows cannot affect a sibling.
			cp := rows2copy(res.Out, len(kids) > 1)
			if err := push(k, cp); err != nil {
				return err
			}
		}
		return nil
	}

	var runErr error
	for _, id := range order {
		n := nodes[id]
		if n.Type != NodeFileSource && n.Type != NodeBOSource && n.Type != NodeQueueSource {
			continue
		}
		src, err := f.Source(n)
		if err != nil {
			runErr = fmt.Errorf("node %q: %w", id, err)
			stats[id].Status, stats[id].Err = "FAILED", err.Error()
			break
		}
		st := stats[id]
		start := time.Now()
		seen := 0
		err = src.Stream(ctx, rc, batchSize, func(rows []Row, rejected []Reject) error {
			if rc.MaxRows > 0 {
				if seen >= rc.MaxRows {
					return errPreviewDone
				}
				if left := rc.MaxRows - seen; len(rows) > left {
					rows = rows[:left]
				}
				seen += len(rows)
			}
			sample(id, rows)
			st.In += int64(len(rows) + len(rejected))
			st.Out += int64(len(rows))
			st.Errors += int64(len(rejected))
			sum.RecordsIn += int64(len(rows) + len(rejected))
			sum.Errors += int64(len(rejected))
			for _, rj := range rejected {
				rj.NodeID = id
				if rec != nil {
					rec.Rejected(ctx, rj)
				}
				if failFast {
					return fmt.Errorf("row %d: %s", rj.Row.Num, rj.Reason)
				}
			}
			for _, k := range children[id] {
				if err := push(k, rows2copy(rows, len(children[id]) > 1)); err != nil {
					return err
				}
			}
			return ctx.Err()
		})
		st.Duration += time.Since(start)
		if errors.Is(err, errPreviewDone) {
			err = nil
		}
		if err != nil {
			st.Status, st.Err = "FAILED", err.Error()
			runErr = fmt.Errorf("node %q: %w", id, err)
			break
		}
	}

	if cerr := closeAll(runErr); cerr != nil && runErr == nil {
		runErr = cerr
	}
	// Master steps: every sink has committed; master each staging load.
	for _, id := range order {
		parent, ok := masterOf[id]
		if !ok {
			continue
		}
		st := stats[id]
		if runErr != nil || rc.DryRun {
			st.Status = "SKIPPED"
			continue
		}
		res, err := masterStep(ctx, f, rc, nodes[id], nodes[parent])
		if res != nil {
			sum.Mastering = append(sum.Mastering, *res)
			st.In, st.Out, st.Errors = int64(res.Records), int64(res.Published), int64(res.Exceptions)
		}
		if err != nil {
			st.Status, st.Err = "FAILED", err.Error()
			runErr = fmt.Errorf("node %q: the load is committed but mastering it failed (master it from the mastering console): %w", id, err)
			continue
		}
		// A run that rejected every record produced nothing, but the engine
		// reports it as PARTIAL, not an error — so it used to finish as
		// COMPLETED with zero published. Rejecting all of them is always a
		// configuration fault, never a data-quality verdict.
		if res != nil && res.Records > 0 && res.Invalid == res.Records {
			st.Status = "FAILED"
			st.Err = fmt.Sprintf("mastering rejected all %d records (%d valid, %d published)", res.Records, res.Valid, res.Published)
			runErr = fmt.Errorf("node %q: the load is committed but mastering it failed (master it from the mastering console): %w",
				id, errors.New(st.Err))
		}
	}
	out := summarize(order, stats, nodes)
	sum.Nodes = out.Nodes
	if rec != nil {
		for _, s := range sum.Nodes {
			rec.NodeDone(ctx, s)
		}
	}
	return sum, runErr
}

func summarize(order []string, stats map[string]*NodeStats, _ map[string]Node) *Summary {
	s := &Summary{}
	for _, id := range order {
		s.Nodes = append(s.Nodes, *stats[id])
	}
	return s
}

func rows2copy(rows []Row, need bool) []Row {
	if !need {
		return rows
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		d := make(map[string]any, len(r.Data))
		for k, v := range r.Data {
			d[k] = v
		}
		out[i] = Row{Num: r.Num, Data: d}
	}
	return out
}

// decodeConfig unmarshals a node config into v.
func decodeConfig(n Node, v any) error {
	if err := json.Unmarshal(n.Config, v); err != nil {
		return fmt.Errorf("node %q config: %w", n.ID, err)
	}
	return nil
}

// masterStep masters the load a staging sink committed.
func masterStep(ctx context.Context, f Factory, rc *RunContext, n, staging Node) (*MasterResult, error) {
	var c MasterConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	var sc StagingSinkConfig
	if err := decodeConfig(staging, &sc); err != nil {
		return nil, err
	}
	m, ok := f.(interface{ Masterer() Masterer })
	if !ok || m.Masterer() == nil {
		return nil, fmt.Errorf("mastering is not configured for this environment")
	}
	load := rc.Loads[staging.ID]
	if load == "" {
		return nil, fmt.Errorf("staging load %q claimed no load run", staging.ID)
	}
	res, err := m.Masterer().MasterLoad(context.WithoutCancel(ctx), MasterRequest{TenantID: rc.TenantID,
		Entity: strings.ToLower(strings.TrimSpace(c.Entity)), StagingTable: sc.Table, LoadRunID: load, PipelineRunID: rc.RunID})
	if res != nil {
		res.NodeID, res.LoadRunID = n.ID, load
	}
	return res, err
}
