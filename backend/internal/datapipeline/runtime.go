package datapipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Source produces batches. emit returning an error stops the stream.
type Source interface {
	Stream(ctx context.Context, rc *RunContext, batchSize int, emit func([]Row) error) error
}

// RunContext is what every node may know about the current run.
type RunContext struct {
	RunID    string
	TenantID string
	Spec     *Spec
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
	Nodes      []NodeStats
	RecordsIn  int64 // rows read from sources
	RecordsOut int64 // rows accepted by sinks
	Errors     int64
}

const defaultBatchSize = 2000

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
	children := map[string][]string{}
	for _, e := range spec.Edges {
		children[e.From] = append(children[e.From], e.To)
	}

	stats := map[string]*NodeStats{}
	procs := map[string]Processor{}
	for i, id := range order {
		n := nodes[id]
		stats[id] = &NodeStats{NodeID: id, Label: n.Label, Type: n.Type, OrderIndex: i, Status: "COMPLETED"}
		switch n.Type {
		case NodeFileSource, NodeBOSource:
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
		if n.Type != NodeFileSource && n.Type != NodeBOSource {
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
		err = src.Stream(ctx, rc, batchSize, func(rows []Row) error {
			st.In += int64(len(rows))
			st.Out += int64(len(rows))
			sum.RecordsIn += int64(len(rows))
			for _, k := range children[id] {
				if err := push(k, rows2copy(rows, len(children[id]) > 1)); err != nil {
					return err
				}
			}
			return ctx.Err()
		})
		st.Duration += time.Since(start)
		if err != nil {
			st.Status, st.Err = "FAILED", err.Error()
			runErr = fmt.Errorf("node %q: %w", id, err)
			break
		}
	}

	if cerr := closeAll(runErr); cerr != nil && runErr == nil {
		runErr = cerr
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
