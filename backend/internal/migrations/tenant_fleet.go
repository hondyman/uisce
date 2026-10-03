package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// Connector opens the database for a target and returns a function that releases it. It is
// injected so the orchestrator never decides how a tenant database is reached or authorized.
type Connector func(ctx context.Context, t Target) (db *sql.DB, release func(), err error)

// Applier applies one target. *TenantRunner is the production implementation.
type Applier interface {
	Apply(ctx context.Context, db *sql.DB, t Target) (Report, error)
}

// Fleet rolls migrations out across many tenant databases in waves.
//
// A wave is a group of targets run together (up to Concurrency at a time). After each wave, if
// more than MaxWaveFailures targets failed, the rollout STOPS and every remaining target is
// reported skipped: a bad migration reaches one wave of tenants, not all of them. State lives in
// each database's migration log, so a rerun, after a failure or a killed orchestrator, resumes
// and never re-executes an applied file.
type Fleet struct {
	Runner          Applier
	Connect         Connector
	WaveSize        int // targets per wave; required
	Concurrency     int // parallel targets within a wave; 0 means 1
	MaxWaveFailures int // failed targets tolerated per wave before stopping; 0 means none
}

// TargetResult is one target's outcome.
type TargetResult struct {
	Report  Report `json:"report"`
	Skipped bool   `json:"skipped,omitempty"`
}

// FleetReport is the outcome of a rollout. Done is true only when every target is Done.
type FleetReport struct {
	Waves   [][]TargetResult `json:"waves"`
	Failed  int              `json:"failed"`
	Skipped int              `json:"skipped"`
	Done    bool             `json:"done"`
}

// Apply runs the targets, in order, wave by wave.
func (f *Fleet) Apply(ctx context.Context, targets []Target) (FleetReport, error) {
	if f.Runner == nil || f.Connect == nil || f.WaveSize <= 0 {
		return FleetReport{}, errors.New("migrations: Fleet needs a Runner, a Connect and a positive WaveSize")
	}
	seen := map[string]bool{}
	for _, t := range targets {
		if err := t.Validate(); err != nil {
			return FleetReport{}, err
		}
		if seen[t.String()] {
			return FleetReport{}, fmt.Errorf("migrations: target %s listed twice", t)
		}
		seen[t.String()] = true
	}
	conc := f.Concurrency
	if conc <= 0 {
		conc = 1
	}

	var rep FleetReport
	stopped := false
	for start := 0; start < len(targets); start += f.WaveSize {
		end := start + f.WaveSize
		if end > len(targets) {
			end = len(targets)
		}
		wave := targets[start:end]
		results := make([]TargetResult, len(wave))

		if stopped || ctx.Err() != nil {
			for i, t := range wave {
				results[i] = skipped(t)
				rep.Skipped++
			}
			rep.Waves = append(rep.Waves, results)
			continue
		}

		sem := make(chan struct{}, conc)
		var wg sync.WaitGroup
		for i, t := range wave {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, t Target) {
				defer wg.Done()
				defer func() { <-sem }()
				results[i] = f.one(ctx, t)
			}(i, t)
		}
		wg.Wait()

		failed := 0
		for _, r := range results {
			if !r.Report.Done {
				failed++
			}
		}
		rep.Failed += failed
		if failed > f.MaxWaveFailures {
			stopped = true
		}
		rep.Waves = append(rep.Waves, results)
	}

	rep.Done = rep.Failed == 0 && rep.Skipped == 0
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	return rep, nil
}

func skipped(t Target) TargetResult {
	return TargetResult{Skipped: true, Report: Report{Target: t.String(), Error: "skipped: an earlier wave failed or the rollout was cancelled"}}
}

func (f *Fleet) one(ctx context.Context, t Target) TargetResult {
	db, release, err := f.Connect(ctx, t)
	if err != nil {
		return TargetResult{Report: Report{Target: t.String(), Error: "connect: " + err.Error()}}
	}
	if release != nil {
		defer release()
	}
	rep, _ := f.Runner.Apply(ctx, db, t) // the Report carries the error
	return TargetResult{Report: rep}
}
