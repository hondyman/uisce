package ormmove

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/hondyman/uisce/backend/internal/migrations"
)

// Connector opens a tenant's target database and returns a function that releases
// it. It is the same seam migrations.Fleet uses, and it is injected for the same
// reason: the orchestrator must never decide how a tenant database is reached or
// authorized. That decision belongs to the provisioning saga and tenantdb.
type Connector func(ctx context.Context, t migrations.Target) (db *sql.DB, release func(), err error)

// Fleet runs the move across many tenants in waves, with the same stop rule
// migrations.Fleet has for migrations: after a wave, if more than MaxWaveFailures
// targets failed, the rollout STOPS and every remaining tenant is reported
// skipped. A tenant that will not move — a bad row, a count that will not
// reconcile, an unreachable database — must not take the fleet down with it, and
// must not silently be counted as done either.
type Fleet struct {
	Mover   *Mover
	Connect Connector

	WaveSize        int // tenants per wave; required
	Concurrency     int // tenants in parallel within a wave; 0 means 1
	MaxWaveFailures int // failed tenants tolerated per wave before stopping; 0 means none
}

// TargetResult is one tenant's outcome.
type TargetResult struct {
	Target  string `json:"target"`
	Report  Report `json:"report"`
	Skipped bool   `json:"skipped,omitempty"`
}

// FleetReport is the outcome of a rollout. Done is true only when every target is.
type FleetReport struct {
	Waves   [][]TargetResult `json:"waves"`
	Moved   int              `json:"moved"`
	Failed  int              `json:"failed"`
	Skipped int              `json:"skipped"`
	Done    bool             `json:"done"`
}

// Apply moves every target, in order, wave by wave. The returned error is only
// for a rollout that could not start or was cancelled; per-tenant failures live in
// the report, because the whole point is that one bad tenant is reported rather
// than aborting the rest.
func (f *Fleet) Apply(ctx context.Context, targets []migrations.Target) (FleetReport, error) {
	if f.Mover == nil || f.Connect == nil || f.WaveSize <= 0 {
		return FleetReport{}, errors.New("ormmove: Fleet needs a Mover, a Connect and a positive WaveSize")
	}
	seen := map[string]bool{}
	for _, t := range targets {
		if err := t.Validate(); err != nil {
			return FleetReport{}, err
		}
		if seen[t.String()] {
			return FleetReport{}, fmt.Errorf("ormmove: target %s listed twice", t)
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
				results[i] = TargetResult{Target: t.String(), Skipped: true,
					Report: Report{TenantID: t.TenantID, Error: "skipped: an earlier wave failed or the rollout was cancelled"}}
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
			go func(i int, t migrations.Target) {
				defer wg.Done()
				defer func() { <-sem }()
				results[i] = f.one(ctx, t)
			}(i, t)
		}
		wg.Wait()

		failed := 0
		for _, r := range results {
			switch {
			case r.Skipped:
				rep.Skipped++
			case r.Report.Done:
				rep.Moved++
			default:
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

func (f *Fleet) one(ctx context.Context, t migrations.Target) TargetResult {
	out := TargetResult{Target: t.String()}
	db, release, err := f.Connect(ctx, t)
	if err != nil {
		out.Report = Report{TenantID: t.TenantID, Error: "connect: " + err.Error()}
		return out
	}
	if release != nil {
		defer release()
	}
	rep, _ := f.Mover.Move(ctx, db, t.TenantID) // the Report carries the error
	out.Report = rep
	return out
}
