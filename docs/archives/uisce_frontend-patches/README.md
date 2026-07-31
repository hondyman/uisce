# `uisce_frontend` — Preserved commit history

This directory preserves the **9 unpushed commits** that lived in the
`uisce_frontend` nested clone (`/Users/eganpj/GitHub/uisce/uisce_frontend`)
before it was consolidated into the monorepo on **2026-07-30**.

The nested clone's remote (`git@github.com:hondyman/uisce_frontend.git`) was
unreachable at consolidation time (404 from GitHub's public API), so these
patches are the only preserved record of those commits. They were generated
directly from the nested clone's git history; metadata (author, dates,
messages) is intact.

## Contents

| File | Description |
|---|---|
| `0001-…0009-*.patch` | `git format-patch` output — apply individually with `git am`. |
| `uisce_frontend_unpushed.bundle` | `git bundle` of the same 9 commits, restorable as a full branch. |

## Commits preserved

1. `feat(frontend): Page Designer Smart Linking + ECharts BY/Visual modes`
2. `feat(frontend): Time Travel — LookbackToolbar + LookbackManager`
3. `feat(ai): AI Copilot lookback intent detection + BO_* binding`
4. `feat(frontend): BOBindingConfigPanel for polyglot CRUD bindings`
5. `feat(frontend): FinancialSuperpowersConsole UI for EDM, Compliance, IBOR/ABOR, Household & MCP`
6. `feat(frontend): ScenarioToolbar for What-If scenario simulation in PageRenderer`
7. `feat(frontend): AgentApprovalInbox UI for Maker-Checker Four-Eyes authorization`
8. `feat(ui): add AIRecommendationBar with two-stage feedback pattern and page designer integration`
9. `MUI v7 migration: fix lucide-react imports, duplicate sx props, Grid/TableChart/Description misuse`

## How to inspect / apply

### Read-only inspection

```bash
cd /tmp && mkdir inspect && cd inspect
git init
git bundle verify /path/to/uisce_frontend_unpushed.bundle
git fetch /path/to/uisce_frontend_unpushed.bundle HEAD:unpushed
git log --oneline unpushed
```

### Apply on top of current `main` (recreate the lost branch)

```bash
# Inside this repo:
git am < docs/archives/uisce_frontend-patches/0001-*.patch
# … repeat for 0002 through 0009
```

Or in one shot:

```bash
git am docs/archives/uisce_frontend-patches/000[1-9]-*.patch
```

> **Warning:** these commits targeted the old `uisce_frontend` repo's
> `frontend/` tree at a different point in history. After consolidation,
> `frontend/` is the canonical path inside this monorepo. Applying them
> verbatim will likely produce many textual conflicts. Use them as a
> reference for the *intent* of each feature, not as a drop-in replay.

## Why this exists

We deliberately kept these patches **inside the repo** (not in `/tmp` or
a separate archive) so the history is recoverable without an out-of-band
backup step. If the upstream `hondyman/uisce_frontend` repo becomes
reachable again later, you can `git am` these on a fresh clone of that
remote to restore the exact branch state.