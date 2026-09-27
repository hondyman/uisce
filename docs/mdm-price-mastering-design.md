# Price master — time-series mastering design

Status: **decided** (2026-09-27, see *Decisions*); build follows the Security master.
Companion to [mdm-mastering-blueprint.md](mdm-mastering-blueprint.md), which covers record mastering
(Product; security and benchmark reference data follow the same model).

## Why prices need their own mode

| | Record mastering (Product, Security, Benchmark) | Price mastering |
|---|---|---|
| Grain | one golden record per entity | one golden **value** per entity × price type × valuation date |
| Volume | thousands of records, changing occasionally | millions of observations **per day** (instruments × types × sources) |
| Identity | match sources to each other (identifiers, fuzzy) | resolve each quote to an **already-mastered instrument**; no fuzzy matching |
| Versioning | a version per changed record | a version per **restatement** of a date's price; one day's value is history, not a change |
| Survivorship | per attribute, over a handful of sources | per key, over sources ranked by asset class, price type and currency, **validated** against peers, prior day and staleness |
| Timing | whenever a load arrives | valuation-date **cutoffs** per market; provisional → final → restated |
| Exceptions | a few per load | the price-validation suite: tolerance, cross-source variance, stale, missing, challenges |
| Execution | record at a time in Go | **set-based**, per valuation date, in the database |

Mastering prices record-at-a-time — a Go loop and a new golden version per price per day — would be
slow at this volume and would bury real changes under daily noise. Prices need the same configuration
and controls, run set-based over a date.

## What already exists (reuse, don't rebuild)

**crims schema (37 `mdm.price_*` tables, empty but for reference data):**
- `mdm.price` — observations: entity × price type × source × date, with quality tier, fair-value level,
  staleness, official/executable/indicative flags; `uq_price_current` keeps one current observation per
  key and source; `predecessor_price_id` + `mdm.price_history` hold restatements.
- `mdm.price_golden_record` — golden value per entity × date × price type × version, with winning
  source, source count, variance %, confidence, DQ, staleness, status.
- `mdm.price_source_priority` — ranking by entity type, asset class, sub-type, price type and currency,
  with fallback flags and a maximum staleness.
- `mdm.price_variance_threshold` — warning / error / critical thresholds by asset class, sub-type and
  price type.
- `mdm.price_variance_event`, `mdm.price_stale_event`, `mdm.price_challenge`, `mdm.price_quality_event`,
  `mdm.price_reconciliation*` — the control and dispute records.
- Reference data: 19 price types (BID/ASK/MID, LAST, OFFICIAL_CLOSE, EVALUATED, NAV, …), 7 quality tiers
  (L1 exchange-traded … L7 indicative), 15 price sources (Bloomberg, ICE, Markit, WM/Refinitiv, internal
  model, manual, …), plus golden publication / distribution tables.
- Staging: `staging.ff_price_eod` (FactSet EOD landing) and `staging.price_incoming` (canonical).
- Work in progress: `streaming.price_tick` + `cmd/stream-ingest` (Redpanda ticks with watermarks).

**Platform pieces the price master reuses as they are:**
- the mastering profile, staging bindings (one vendor mapping), message catalog, console;
- catalog rules on the one rule VM — validation, and survivorship **selection rules** (peers as a
  collection, so `MEDIAN`, `STDEV_P`, `PERCENTILE`, `COUNT`, `ABS` apply);
- layered survivorship: hierarchy → per-attribute strategy → selection rule → steward;
- the per-entity override / merge policy (approval or direct) and its audit;
- the one scheduler (calendars, external triggers from Tidal / Control-M, run history).

## Design

### 1. A second profile kind: `TIMESERIES`

`mdm.mastering_entity` gains `kind` (`RECORD` | `TIMESERIES`). A PRICE profile declares, as settings:

- **key**: `price_entity_type, price_entity_id, price_type_cd, price_date` — the golden grain;
- **observation table** `mdm.price`, **golden table** `mdm.price_golden_record`, **value column**;
- **instrument resolution**: which master the quotes resolve against, and through which identifiers
  (the security master's identifier table and cross-reference — ISIN, CUSIP, SEDOL, FIGI, vendor symbols
  via `mdm.price_vendor_symbol`);
- **source registry**: `mdm.price_source` (`source_cd`) — the profile names its registry, since price
  sources (ICE, Markit, WM/Refinitiv, …) are not the reference-data vendors in `mdm.source_systems`;
- **hierarchy** `mdm.price_source_priority` and **thresholds** `mdm.price_variance_threshold`, both
  scoped by asset class, sub-type, price type (and currency for priority).

### 2. The pipeline, set-based per valuation date

Same stages as record mastering; each runs as SQL over a date's batch, chunked by entity range.

1. **Land** — EOD files through data pipelines (as today); intraday/official ticks through
   stream-ingest. Both end in staging, bound to the PRICE business object by staging bindings.
2. **Canonicalize** — `INSERT … SELECT` from staging into `staging.price_incoming` through the binding;
   resolve each quote to its instrument by joining the security master's identifiers. An unresolved
   quote is an `UNRESOLVED_INSTRUMENT` exception (in bulk), never a guess. Canonical validation rules run
   set-based (see *Rules at volume*).
3. **Observe** — upsert `mdm.price`: one current observation per entity × type × source × date. A vendor
   resending a different value for the same key is a **restatement**: the old row moves to
   `mdm.price_history` and `predecessor_price_id` links them.
4. **Survive** (per key) — the layered model:
   - ranking: `price_source_priority` for the key's asset class / price type / currency (a window
     function), with fallback sources;
   - staleness per source and quality tier (`max_staleness_minutes`);
   - **selection rule** — e.g. the consensus check, now per asset class **without a conditional in the
     rule language**: the key's threshold row is part of the context, so
     `ABS(value - MEDIAN(peers.value)) <= threshold.error_threshold / 100 * MEDIAN(peers.value)`
     reads the right threshold for equities, government bonds or private credit;
   - ENFORCE / FLAG, HOLD / ALLOW, minimum peers — as for records.
5. **Validate & control** — the market-standard price-validation suite, as configuration:
   - **day-over-day tolerance** against the prior date's golden value, per `price_variance_threshold`
     (warning / error / critical);
   - **cross-source variance** → `mdm.price_variance_event` (the pair, the variance in %/bps, severity);
   - **stale** (unchanged for N days, or older than the tier allows) → `mdm.price_stale_event`;
   - **missing** — the expected universe (e.g. held instruments × required price types) with no golden
     price by cutoff;
   - **vendor challenge** — `mdm.price_challenge` for disputing a vendor's price, with its response.
   Error / critical outcomes follow a per-control setting: publish with a flag, or hold in REVIEW.
6. **Publish** — `mdm.price_golden_record`: a new version only when the value or winner changes.
   - `effective_date` = valuation date and `knowledge_timestamp` = when known make it **bitemporal**:
     "what did we believe the 30-Sep close was on 2-Oct?" is a query, not a reconstruction;
   - **compact provenance** on the golden row (winning source, the rule and version, and every
     candidate's source / value / age / tier / selected in `golden_attributes`), rather than per-field
     and per-decision log rows, which at price volume would be many rows per price. Full
     `survivorship_log` rows only for held, overridden or challenged prices.
7. **Distribute** — `price_golden_publication` / `_distribution`, and a "golden price published /
   restated" event on Redpanda for OMS, performance, NAV and risk.

### 3. Timing: provisional, final, restated

- Each source has a cutoff (`price_source.cutoff_time`, `timezone`); each market a business calendar.
- Before cutoff a golden price is **provisional** (early sources only); at cutoff it becomes **final**
  when every expected source has reported, or the cutoff passes.
- After final, a changed input is a **restatement**: version N+1 with the reason, a restatement event,
  and optional approval.
- Cutoffs run on the one scheduler: per-market schedules on business calendars, triggered by Tidal
  where the tenant uses it.

### 4. Overrides and challenges

- A steward override of a golden price for a date uses the entity's policy (approval or direct).
  It is recorded as an observation from the `MANUAL` price source, with the approvers in its provenance,
  so it survives, validates and publishes like any other price.
- A price challenge (disputing a vendor) is its own workflow: open → sent to vendor → vendor response →
  resolved, with the final price feeding back as that vendor's restatement.

### 5. Rules at volume

The rule VM compiles expressions to SQL (`CompileToSQL`, `CompileConditionSQL`), but only for the
StarRocks dialect today. Two options for set-based rule checks in crims (Postgres):
- **add a Postgres dialect** to the VM function library (one `SQLEmit` entry per function; the library
  was designed for exactly this), so validation and selection rules run inside the batch query; or
- evaluate in Go over chunks (simple, slower; fine for EOD, not for intraday).
Recommendation: Postgres dialect for the functions price rules use (arithmetic, `ABS`, `COUNT`,
`MEDIAN` → `percentile_cont(0.5)`, `AVG`, `STDEV_P`), Go evaluation as the fallback.

### 6. Volume and storage

- An illustrative scale: 200k instruments × 5 price types × 3–6 sources ≈ 3–6M observations and ~1M
  golden prices per valuation date.
- `mdm.price`, `mdm.price_golden_record`, `mdm.price_history` and `mdm.price_variance_event` are plain
  tables today: **partition by `price_date`** (monthly range), with indexes on the key.
- Batches run per valuation date, chunked by entity range, each chunk one transaction — a failed chunk
  retries alone, and the run is idempotent per (date, chunk, load).

### 7. Intraday (later)

`streaming.price_tick` → `mdm.price_intraday` windows (already modelled) give provisional intraday
golden prices; they are not versioned per tick, and the EOD official price supersedes them.

## Slices

1. **Profile kind** `TIMESERIES`, the PRICE profile, per-profile source registry.
2. **Canonicalize + instrument resolution** set-based, `UNRESOLVED_INSTRUMENT` exceptions.
3. **Observe / survive / publish** set-based: scoped hierarchy, staleness by tier, selection rules with
   threshold context, bitemporal golden prices with compact provenance.
4. **Controls**: tolerance, variance events, stale, missing; the console's price views (by date and
   instrument, variance queue, stale, missing).
5. **Cutoffs, finalization, restatements**, distribution events.
6. **Overrides and vendor challenges.**
7. **Intraday** from the stream.
Enablers alongside: date partitioning; the Postgres dialect for rule pushdown.

## Decisions (2026-09-27)

1. **Instrument identity** — a **self-contained Security master first** (its own golden records,
   identifiers and cross-reference under the generic engine); prices resolve to security golden records.
2. **One vendor registry** — a vendor is one source whatever it supplies: `BLOOMBERG` is the same source
   for securities, prices, products and benchmarks (no `BBRG_PRICE` / `BBRG_SEC`). `mdm.source_systems`
   is the registry; price-specific facts (cutoff time, time zone, quality tier, delivery) become price
   settings of that source, and `mdm.price_source` is re-keyed to it.
3. **End of day first**; intraday later.
4. **Finals only** for end of day; provisional golden prices come with intraday.
5. **Compact provenance** on the golden row; full log rows only for held, overridden or challenged
   prices.
6. **Postgres (crims) set-based**, with a Postgres dialect added to the rule VM's function library for
   pushdown; Go evaluation over chunks as the fallback.
