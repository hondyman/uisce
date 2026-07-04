# `scripts/` Go scratch debug scripts

This tree collects throwaway debug/main scripts that used to pile up at
the repo root as `test_db*.go`, `db_query*.go`, `gen_hash.go`,
`test_req.go`, `db_schema.go`, `fix_compilation.go`. Each declares
`package main` and would have collided with every other if left at the
repo root.

## Layout

| Dir | Origin | Purpose |
|---|---|---|
| `db_inspect/<name>/` | `test_db*.go`, `test_dbs.go`, `db_query.go`, `db_schema.go` | One-off Postgres catalog inspection queries against `Northwinds` / `northwind` gold-copy datasources |
| `req/test_req/` | `test_req.go` | One-off `http.Get` against `/api/catalog/nodes` to bypass auth |
| `hashgen/gen_hash/` | `gen_hash.go` | `bcrypt.GenerateFromPassword` for seeding demo users |
| `fix_compilation/` | `fix_compilation.go` | Was tagged `//go:build ignore` already; preserved as-is |

Every file (except those that already had it) now starts with:

```go
//go:build ignore
// +build ignore

package main
```

so they are excluded from the default Go build.

## Running a script

```sh
go run -tags ignore ./scripts/db_inspect/test_db3
go run -tags ignore ./scripts/hashgen/gen_hash
```

## Re-introducing one to the build (rare)

Delete the first three lines (`//go:build ignore`, `// +build ignore`,
and the blank line) and remove the matching patterns from `.gitignore`
under "Root-level Go scratch / debug scripts".

## Hardcoded values

The original scripts hardcoded:

* Postgres DSN: `postgres://postgres:postgres@100.84.50.65:5432/postgres?sslmode=disable`
* Tenant UUID: `99e99e99-99e9-49e9-89e9-99e99e99e999` (`Northwinds`)
* Datasource UUID: `25b5dce3-27d9-4773-933e-6ee29a42871f` (northwind gold-copy)

If your local Postgres differs, edit the constants in the script you want
to run, or set `DATABASE_URL` in your shell and adapt the script to use
it (none of the originals do).
