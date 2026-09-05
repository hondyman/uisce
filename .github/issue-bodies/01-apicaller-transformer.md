## Problem

`APICallerTransformer` in `backend/internal/datapipeline/transforms.go` stamps `{"verified": true}` without making any HTTP call. It is registered in the transform palette. A transformer that claims to call an API but doesn't is a silent correctness failure.

## Production Impact

Pipelines using `api_caller` produce outputs claiming verification was performed when no API was contacted. Downstream decisions are based on fabricated data.

## Fix Direction

1. Bind to the server-side API Studio endpoint registry (no arbitrary URLs — SSRF prevention)
2. Support service-held auth tokens (OAuth2 client credentials, API keys)
3. Cap responses at 1MB
4. Propagate real HTTP errors through the pipeline failure path

## Acceptance Criteria

- [ ] APICallerTransformer makes real HTTP call to registered endpoint
- [ ] SSRF: arbitrary URL rejected, only registry endpoints accepted
- [ ] Service-held auth tokens used (not caller-supplied credentials)
- [ ] Response >1MB rejected with appropriate error
- [ ] HTTP errors propagate through pipeline failure path
- [ ] go test -count=1 ./internal/datapipeline/... passes

Ref: STATUS_AUDIT_BACKLOG.md → `APICallerTransformer-Unimplemented`
