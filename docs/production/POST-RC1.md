# Uisce Semantic OS — Post-RC1 Aspirational Roadmap

The following features are scoped out of RC1 but are required for a fully operational G-SIFI production deployment. This document serves as the authoritative backlog for the next build cycle.

---

## Priority 1: Redis Idempotency Caching

**Problem:** `external_compliance_handler.go` accepts a `redisClient` but never uses it for response caching. The stress test's cache hit-rate assertion fails.

**Fix:**
1. In `HandleEvaluateExternal`, check `X-Idempotency-Key` header
2. `redisClient.Get(ctx, "idem:"+key)` — if HIT, return cached result with `X-Cache-Status: HIT` header
3. On MISS, evaluate rule, `redisClient.Set(ctx, "idem:"+key, response, 24h)`, return with `X-Cache-Status: MISS`
4. Assert in stress test: Phase B cache hit rate > 80%

**Estimated:** 4-6 hours

---

## Priority 2: Arrow Flight SQL Handler (Port 8090)

**Problem:** `flight.NewFlightServer(8090)` starts in `server.go` but has no gRPC query handler registered.

**Fix:**
1. Implement `FlightSQLHandler` that translates Arrow Flight SQL RPCs to `vm.Run()`
2. Register the handler: `flightServer.RegisterFlightSqlHandler(handler)`
3. Add `STARROCKS_HOST` env var consumption to backend
4. Document in Helm README as "Active" (not "aspirational")

**Estimated:** 2-3 days

---

## Priority 3: QuickFIX Acceptor (Port 8980)

**Problem:** `fix.NewServer(fixAdapter, "")` starts but has no FIX session configuration or tickerplant integration.

**Fix:**
1. Load FIX session config from `config/fix_sessions.yaml`
2. Implement tickerplant adapter: subscribe to `shadow_replay_diffs` CDC events
3. Wire `fixServer.AddSession(sessionConfig)` with proper heartbeat interval
4. Add `FIX_ACCEPTOR_PORT` consumption to `server.go`

**Estimated:** 3-5 days

---

## Priority 4: StarRocks Connector

**Problem:** `STARROCKS_HOST` is passed to the backend but never used.

**Fix:**
1. Add `starrocks` import and `starrocks.NewClient(STARROCKS_HOST)`
2. In `HandleEvaluateExternal`, optionally route analytical queries to StarRocks
3. Add StarRocks DDL migration for `shadow_replay_summary` materialized view

**Estimated:** 2 days

---

## Priority 5: eBPF XDP Integration

**Problem:** eBPF is disabled in RC1 (`EBPF_ENABLED=false`).

**Requirements:**
- Linux kernel >= 5.8 with BTF support
- `CAP_NET_ADMIN` capability
- eBPF object file (`.o`) pre-compiled and loaded via `bpffs`
- Host network mode in Docker/K8s

**Fix:**
1. Compile `ebpf/ingress_socket_filter.o` via `clang -target bpf`
2. Add `hostNetwork: true` + `securityContext.capabilities.add: [NET_ADMIN]` to Helm values
3. Implement `ebpf.Receive()` path feeding into `FastRecord` pool
4. Add e2e test verifying packet-level latency vs. HTTP-level latency

**Estimated:** 5-10 days (includes kernel testing)

---

## Priority 6: RBAC Manifests

**Problem:** No Kubernetes RBAC resources in the Helm chart.

**Fix:**
Add to `deploy/helm/templates/rbac.yaml`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: uisce-operator
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: uisce-operator-binding
subjects:
  - kind: ServiceAccount
    name: uisce-sa
    namespace: uisce
roleRef:
  kind: Role
  name: uisce-operator
  apiGroup: rbac.authorization.k8s.io
```

**Estimated:** 1 day

---

## Priority 7: Prometheus Custom Metrics for HPA

**Problem:** HPA uses CPU-based scaling only; VM eval p99 latency is a better signal.

**Fix:**
1. Add `prometheus/client_golang` to expose `uisce_vm_eval_duration_seconds` histogram
2. Add `prometheus-adapter` ConfigMap with:
   ```yaml
   - seriesQuery: 'uisce_vm_eval_duration_seconds'
     resources:
       overrides:
         namespace: { resource: "namespace" }
     name:
       matches: "uisce_vm_eval_duration_seconds"
       as: "uisce_vm_eval_duration_seconds_p99"
   ```
3. Wire HPA to use custom metric

**Estimated:** 2 days

---

## Priority 8: Temporal Regulatory Pack Workflow (RC2 Path 3)

**Problem:** Manual overnight SEC Form 13F / PF pack generation is not automated.

**Fix:**
1. Implement `workflows/regulatory_pack.go` with Temporal workflow
2. Steps:
   - `FederateTrades(ctx)` — query StarRocks + Iceberg for 500M trades
   - `CertifyData(ctx, certifications)` — apply data quality gates
   - `Compile13F(ctx)` — generate Form 13F PDF package
   - `ArchiveToS3(ctx)` — immutable S3 storage with Glacier lifecycle
3. Add `workflows/regulatory_pack_test.go` with mock data
4. Cron trigger: `0 0 * * *` via Temporal schedule

**Estimated:** 5-8 days

---

## Priority 9: Frontend Governance HUDs (RC2 Path 2)

**Components:**
- `ShadowImpactStudio.tsx` — live 24-hour shadow simulation diffs
- `RulePerformanceHUD.tsx` — p50/p95/p99 ring buffer latencies
- `MergeConflictResolver.tsx` — visual 3-way merge conflict UI

**Estimated:** 8-15 days

---

## Open Questions

| Item | Question | Owner |
|------|----------|-------|
| Redis TTL | What TTL for idempotency keys? (24h default proposed) | Compliance |
| FIX session auth | Username/password or certificate-based? | FIX specialist |
| StarRocks sizing | FE-only vs. FE+BE cluster? | Infrastructure |
| eBPF kernel matrix | Which kernels to test? (Ubuntu 22.04 LTS min) | DevOps |
