// Command migrate-datasource-creds moves datasource credentials out of the
// metadata DB and into the secrets store (Infisical), leaving only a
// secret_path reference behind. See internal/dscreds for the layout.
//
// It never prints a credential value: output is ids, names, key names and
// booleans only. Secret values travel DB -> secrets store in-process.
//
//	-mode plan   (default) read-only report; touches nothing
//	-mode copy   write each row's inline credentials to its canonical secrets
//	             folder, read them back, then add the secret_path reference.
//	             Inline values are left in place (readers ignore them once a
//	             reference exists), so rollback is removing secret_path.
//	-mode scrub  for rows whose reference resolves to exactly the inline
//	             values, remove the inline copies from the DB.
//
// copy and scrub require -yes. -id limits any mode to one owning row.
//
// Env: DATABASE_URL, plus the SECRETS_PROVIDER / INFISICAL_* variables read by
// dscreds.ProviderFromEnv. Run with a role that can read every tenant's rows.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/secrets"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type owner struct {
	kind   dscreds.Kind
	id     string
	tenant string
	name   string
	ref    string
	inline map[string]string
	err    error // extraction problem (conflict, bad JSON); row is skipped
}

func (o owner) canonical() string {
	p, err := dscreds.CanonicalPath(o.kind, o.tenant, o.id)
	if err != nil {
		return ""
	}
	return p
}

func main() {
	mode := flag.String("mode", "plan", "plan | copy | scrub")
	only := flag.String("id", "", "limit to one tenant_product_datasource or connections id")
	yes := flag.Bool("yes", false, "confirm copy/scrub")
	flag.Parse()

	if *mode != "plan" && *mode != "copy" && *mode != "scrub" {
		fail("unknown -mode %q", *mode)
	}
	if *mode != "plan" && !*yes {
		fail("-mode %s writes to the secrets store and/or the DB; re-run with -yes", *mode)
	}

	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fail("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fail("open database: %v", redact(err, dsn))
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		fail("connect database: %v", redact(err, dsn))
	}

	provider, err := dscreds.ProviderFromEnv()
	if err != nil {
		fail("%v", err)
	}
	if provider == nil && *mode != "plan" {
		fail("SECRETS_PROVIDER is not set; copy/scrub need the secrets store")
	}

	owners, err := loadOwners(ctx, db, *only)
	if err != nil {
		fail("%v", err)
	}

	switch *mode {
	case "plan":
		plan(ctx, db, provider, owners)
	case "copy":
		failures := 0
		for _, o := range owners {
			if err := copyOwner(ctx, db, provider, o); err != nil {
				failures++
				fmt.Printf("FAIL  %s %s: %v\n", o.kind, o.id, err)
			}
		}
		exitOn(failures)
	case "scrub":
		failures := 0
		for _, o := range owners {
			if err := scrubOwner(ctx, db, provider, o); err != nil {
				failures++
				fmt.Printf("FAIL  %s %s: %v\n", o.kind, o.id, err)
			}
		}
		exitOn(failures)
	}
}

func loadOwners(ctx context.Context, db *sql.DB, only string) ([]owner, error) {
	var out []owner

	rows, err := db.QueryContext(ctx, `
		SELECT tpd.id::text, COALESCE(tpd.tenant_id::text, ''), COALESCE(tpd.source_name, ''),
		       COALESCE(tpd.config, '{}'::jsonb)::text
		FROM public.tenant_product_datasource tpd
		WHERE ($1 = '' OR tpd.id::text = $1)
		ORDER BY tpd.id`, only)
	if err != nil {
		return nil, fmt.Errorf("read tenant_product_datasource: %w", err)
	}
	for rows.Next() {
		o := owner{kind: dscreds.KindDatasource}
		var cfgText string
		if err := rows.Scan(&o.id, &o.tenant, &o.name, &cfgText); err != nil {
			rows.Close()
			return nil, err
		}
		cfg := map[string]any{}
		if err := json.Unmarshal([]byte(cfgText), &cfg); err != nil {
			o.err = fmt.Errorf("config is not a JSON object")
		} else {
			o.ref, _ = cfg[dscreds.RefKey].(string)
			o.inline, o.err = dscreds.ExtractInline(cfg)
		}
		out = append(out, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.QueryContext(ctx, `
		SELECT c.id::text, COALESCE(c.tenant_id::text, ''), COALESCE(c.name, ''),
		       COALESCE(c.username, ''), COALESCE(c.password, ''), COALESCE(c.api_key, ''),
		       COALESCE(c.secret_path, ''), COALESCE(c.metadata, '{}'::jsonb)::text
		FROM public.connections c
		WHERE ($1 = '' OR c.id::text = $1)
		ORDER BY c.id`, only)
	if err != nil {
		return nil, fmt.Errorf("read connections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		o := owner{kind: dscreds.KindConnection}
		var user, pass, apiKey, metaText string
		if err := rows.Scan(&o.id, &o.tenant, &o.name, &user, &pass, &apiKey, &o.ref, &metaText); err != nil {
			return nil, err
		}
		meta := map[string]any{}
		if err := json.Unmarshal([]byte(metaText), &meta); err != nil {
			o.err = fmt.Errorf("metadata is not a JSON object")
			out = append(out, o)
			continue
		}
		// Columns and metadata both hold credentials; they must agree.
		o.inline, o.err = dscreds.ExtractInline(meta)
		if o.err == nil {
			cols, _ := dscreds.ExtractInline(map[string]any{"username": user, "password": pass, "api_key": apiKey})
			for k, v := range cols {
				if prev, ok := o.inline[k]; ok && prev != v {
					o.err = fmt.Errorf("%s: column and metadata differ: %w", k, dscreds.ErrConflictingCredentials)
					break
				}
				o.inline[k] = v
			}
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func plan(ctx context.Context, db *sql.DB, provider secrets.Provider, owners []owner) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tID\tTENANT\tNAME\tINLINE KEYS\tREFERENCE\tSTORE\tNEXT")
	for _, o := range owners {
		refState := "none"
		switch {
		case o.ref == "":
		case o.ref == o.canonical():
			refState = "canonical"
		default:
			refState = "MISMATCH"
		}
		store := "-"
		matches := false
		if provider != nil && o.canonical() != "" {
			vals, err := provider.GetMap(ctx, o.canonical())
			switch {
			case err != nil || len(vals) == 0:
				store = "absent"
			default:
				store = strings.Join(keys(vals), ",")
				matches = sameCredentials(o.inline, vals)
			}
		}
		next := "-"
		switch {
		case o.err != nil:
			next = "RESOLVE: " + o.err.Error()
		case refState == "MISMATCH":
			next = "RESOLVE: reference is not this row's canonical path"
		case len(o.inline) > 0 && refState == "none":
			next = "copy"
		case len(o.inline) > 0 && refState == "canonical" && provider == nil:
			next = "scrub (store not checked)"
		case len(o.inline) > 0 && refState == "canonical" && matches:
			next = "scrub"
		case len(o.inline) > 0 && refState == "canonical":
			next = "RESOLVE: inline differs from store"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", o.kind, o.id, o.tenant, o.name,
			strings.Join(keys(o.inline), ","), refState, store, next)
	}
	w.Flush()

	// Stores with no reader in the backend: report only; clean up by hand.
	var tdPw int
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM public.tenant_datasources WHERE COALESCE(password, '') <> ''`).Scan(&tdPw)
	var tcDSN int
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM public.tenant_connections WHERE COALESCE(dsn, '') ~ '(:[^:@/]+@|password=)'`).Scan(&tcDSN)
	fmt.Printf("\npublic.tenant_datasources rows with a password: %d\npublic.tenant_connections rows with a password in dsn: %d\n", tdPw, tcDSN)
}

func copyOwner(ctx context.Context, db *sql.DB, provider secrets.Provider, o owner) error {
	if o.err != nil {
		return o.err
	}
	path := o.canonical()
	if path == "" {
		return fmt.Errorf("no canonical path (tenant %q)", o.tenant)
	}
	if o.ref != "" {
		if o.ref != path {
			return fmt.Errorf("existing reference is not the canonical path %s", path)
		}
		fmt.Printf("skip  %s %s: already references %s\n", o.kind, o.id, path)
		return nil
	}
	if o.inline[dscreds.KeyPassword] == "" && o.inline[dscreds.KeyPrivateKey] == "" && o.inline[dscreds.KeyAPIKey] == "" {
		fmt.Printf("skip  %s %s: no inline credentials\n", o.kind, o.id)
		return nil
	}

	existing, err := provider.GetMap(ctx, path)
	if err == nil && len(existing) > 0 {
		if !sameCredentials(o.inline, existing) {
			return fmt.Errorf("secrets store already has different values at %s; refusing to overwrite", path)
		}
	} else if err := provider.PutMap(ctx, path, o.inline); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	back, err := provider.GetMap(ctx, path)
	if err != nil || !sameCredentials(o.inline, back) {
		return fmt.Errorf("read-back of %s does not match; DB left unchanged", path)
	}

	var res sql.Result
	switch o.kind {
	case dscreds.KindDatasource:
		res, err = db.ExecContext(ctx, `
			UPDATE public.tenant_product_datasource
			SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{secret_path}', to_jsonb($2::text)), updated_at = NOW()
			WHERE id = $1::uuid AND tenant_id = $3::uuid AND NOT (COALESCE(config, '{}'::jsonb) ? 'secret_path')`,
			o.id, path, o.tenant)
	case dscreds.KindConnection:
		res, err = db.ExecContext(ctx, `
			UPDATE public.connections SET secret_path = $2, updated_at = NOW()
			WHERE id = $1::uuid AND tenant_id = $3::uuid AND secret_path IS NULL`,
			o.id, path, o.tenant)
	}
	if err != nil {
		return fmt.Errorf("set reference: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return fmt.Errorf("set reference: %d rows updated (row changed concurrently?)", n)
	}
	fmt.Printf("ok    %s %s: stored %s at %s and set reference\n", o.kind, o.id, strings.Join(keys(o.inline), ","), path)
	return nil
}

func scrubOwner(ctx context.Context, db *sql.DB, provider secrets.Provider, o owner) error {
	if o.err != nil {
		return o.err
	}
	path := o.canonical()
	if o.ref == "" || o.ref != path {
		if len(o.inline) > 0 {
			return fmt.Errorf("no canonical reference yet; run -mode copy first")
		}
		return nil
	}
	if len(o.inline) == 0 {
		fmt.Printf("skip  %s %s: nothing inline\n", o.kind, o.id)
		return nil
	}
	vals, err := provider.GetMap(ctx, path)
	if err != nil || !sameCredentials(o.inline, vals) {
		return fmt.Errorf("secrets store at %s does not hold exactly the inline values; not scrubbing", path)
	}

	var res sql.Result
	switch o.kind {
	case dscreds.KindDatasource:
		res, err = db.ExecContext(ctx, `
			UPDATE public.tenant_product_datasource
			SET config = (config #- '{auth,basic,password}') - 'password' - 'private_key' - 'api_key'
			             - 'client_secret' - 'dsn' - 'connection_string' - 'database_url',
			    updated_at = NOW()
			WHERE id = $1::uuid AND tenant_id = $2::uuid AND config->>'secret_path' = $3`,
			o.id, o.tenant, path)
	case dscreds.KindConnection:
		res, err = db.ExecContext(ctx, `
			UPDATE public.connections
			SET password = NULL, api_key = NULL,
			    metadata = (COALESCE(metadata, '{}'::jsonb) #- '{auth,basic,password}') - 'password' - 'private_key' - 'api_key'
			               - 'client_secret' - 'dsn' - 'connection_string' - 'database_url',
			    updated_at = NOW()
			WHERE id = $1::uuid AND tenant_id = $2::uuid AND secret_path = $3`,
			o.id, o.tenant, path)
	}
	if err != nil {
		return fmt.Errorf("scrub: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return fmt.Errorf("scrub: %d rows updated", n)
	}
	fmt.Printf("ok    %s %s: removed inline %s\n", o.kind, o.id, strings.Join(keys(o.inline), ","))
	return nil
}

// sameCredentials reports whether the store holds every inline credential
// with the same value. The store may hold extra keys (e.g. USERNAME only there).
func sameCredentials(inline, store map[string]string) bool {
	for k, v := range inline {
		if store[k] != v {
			return false
		}
	}
	return true
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func redact(err error, dsn string) string {
	return strings.ReplaceAll(err.Error(), dsn, "<DATABASE_URL>")
}

func exitOn(failures int) {
	if failures > 0 {
		fmt.Printf("%d row(s) failed; nothing was changed for them\n", failures)
		os.Exit(1)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "migrate-datasource-creds: "+format+"\n", args...)
	os.Exit(2)
}
