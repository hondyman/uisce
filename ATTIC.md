# attic/rebalancing-worker

This branch is the final state of the `rebalancing/worker` Go module, parked intact.
**Nothing here is broken by the parking — the module was already broken when it was parked.**
This file exists so a cold reader can answer "why was this parked?" without archaeology.

Removed from `main` on 2026-10-04. Do not resurrect it by copying files back one at a time; see
*Resurrection* below.

## What it was

A Temporal-based portfolio rebalancing worker: 17 files, **3,099 lines**, its own Go module, its own
`Dockerfile`. Portfolio rebalancing with per-trade constraints, ESG preference and risk-appetite
personalisation, tax-lot analysis, drift calculation, a Finnhub price client, an XAI rebalance
client, a Postgres repository layer, and six workflow/activity files.

That is investment-domain thinking, not scaffolding. The line count is the reason this is an attic
branch rather than a deletion: `git log` showing a deletion commit does not tell a future reader
that 3,099 lines of coherent rebalancing logic exists, and what it was trying to be.

## Why it was parked — the evidence

Recorded in full on [#379](https://github.com/hondyman/uisce/issues/379). Summary:

**1. It has never compiled. At no commit in its history.**

- `activities.go` (added `65e6b76a5`, 2026-06-28, "commit missing source files") declares
  `db *Repository` and calls `a.db.GetPortfolio`.
- `type Repository struct` does not exist in git at that commit. The first commit that contains it
  is `235d26db6` (2026-08-05) — the same commit that introduced the collision.
- So the module entered git in a state where it could not build, and the first commit in which it
  *could* have built is the commit that guaranteed it would not.

**2. It has no consumers.**

```
$ git grep -nE "rebalancing/(worker|api)" -- '*.go' '*.ts' '*.tsx' '*.yml' '*.sh' | grep -v ^rebalancing/
```

returns seven lines, all in shell deploy scripts, and each is looking for a *prebuilt binary*
(`rebalancing/worker/rebalancing-worker`) that the module's own Dockerfile cannot produce.
`backend/go.mod` has no dependency on it. The `rebalancing` references inside `backend/` and
`frontend/src/` are a different subsystem — `internal/temporal/workflows/rebalance_workflow.go`,
`cmd/shadow_runner`, `RebalanceDashboard.tsx` — which does not use this module.

**3. It is in no deployment topology.** No `docker-compose*.yml`, no `k8s/` manifest, no workflow
in `.github/workflows/` references it. The three deploy scripts that would have deployed it
(`NAVIGATOR_DEPLOY.sh`, `REBASE_DEPLOY.sh`, `RISK_ALPHA_DEPLOY.sh`) are themselves referenced by
zero CI, compose or k8s files.

**4. Recent commits are automated sweeps, not development.** The 2026-10-02 touch was
`eaaa366b2` ("chore: remove RabbitMQ from infrastructure, CI, tests and dependencies"), authored by
`opencode <opencode@local>`, 59 files changed, of which exactly one was under `rebalancing/` — one
removed line of RabbitMQ config in `rebalancing/docker-compose.yml`. The 2026-08-29 commits are
likewise Hasura-cleanup sweeps.

## What it would take to make it build

Recorded because it is the shape of the work, not because it was done. Doing the "real fix" was
considered and declined: it would give a module with no consumer its first-ever compilable state,
which is only justified if someone claims it, and nobody has.

- **Three** type pairs are declared in both `domain.go` and `repository.go`, not two:
  `Portfolio`, `Holding`, and `RebalancePlan` (missed until the rename was actually applied).
- Renaming the row types does not fix it. It converts the redeclarations into: duplicate-resolution
  fallout (`h.ID undefined`, `plan.TaxSavings undefined`, `plan.PortfolioID` type mismatch,
  `h.PurchaseDate` as `time.Time`), **a row→domain mapper that does not exist anywhere in the
  module** (`activities.go:424` returns `a.db.GetPortfolio(...)` — the row — from a signature
  saying `*Portfolio`, while `AnalyzeDrift` needs `TargetModel map[string]float64` and
  `buildRebalancePrompt` needs `p.Constraints.ESGPreference` and `p.PolicyDocument`), and **a
  defect unrelated to any of it**: `main.go:36`, `NewRepository(pool.Pool)` where `NewPool` returns
  `*pgxpool.Pool`, which has no `Pool` field.

## The unanswered design question

This is the part the parking deliberately does **not** settle.

The module carries **three** models of one entity:

| | `Portfolio` / `Holding` (`domain.go`) | `Portfolio` / `Holding` (`repository.go`) | `PortfolioHolding` (`rebalance_service.go:36`) |
|---|---|---|---|
| identity | `ID string`, `TenantID string` | `ID uuid.UUID`, `TenantID uuid.UUID` | — |
| target model | `map[string]float64` | `json.RawMessage` | — |
| constraints | `RebalanceConstraints` struct | `json.RawMessage` | — |
| extras | `RiskScore`, `Alpha`, `SectorAttribution`, `MitigationAction`, `PolicyDocument` | — | `MarketValue`, `UnrealizedGain`, `DaysHeld`, `CurrentWeight`, `AssetClass` |

The two `Portfolio` types are **layering types, not duplicates** — the repository's is a persistence
row that `GetPortfolio` `Scan`s columns directly into at `repository.go:56`, which is why
`uuid.UUID` and `json.RawMessage` are load-bearing there and cannot be replaced by the domain type.
But why there are three models, whether one is meant to replace the others, and who was meant to
consume them, was never answered by anyone.

**If you are here to answer that, the question is worth more than the code.**

## Resurrection

Do not copy files back into `main` one at a time. This branch is the whole module; merge or
cherry-pick it wholesale, then expect `#398`'s `workspace-modules` job to be the thing that proves
the result — it builds every module in `go.work` and fails on the first one that does not, which is
exactly the check that was absent for this module's entire life.
