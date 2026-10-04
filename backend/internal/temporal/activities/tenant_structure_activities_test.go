package activities_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/internal/tenantschema"
	"github.com/hondyman/uisce/backend/models"
	"github.com/stretchr/testify/require"
)

// These drive the real saga activities against the real cluster rig, with the template served by a fake store so the
// scan content is exactly what each test says. The structure is the shape of the gold copy's: two schemas that refer to
// each other, a partitioned table with a default partition, an array, a routine and a trigger.

type structStore struct {
	owner string
	gold  bool
	nodes []*models.CatalogNode
	reads int
}

func (s *structStore) Owner(context.Context, string) (string, error)    { return s.owner, nil }
func (s *structStore) IsGoldCopy(context.Context, string) (bool, error) { return s.gold, nil }
func (s *structStore) Read(context.Context, string, string) (string, []*models.CatalogNode, error) {
	s.reads++
	return "st_core,st_ref", s.nodes, nil
}

func js(v interface{}) json.RawMessage { b, _ := json.Marshal(v); return b }

func structureNodes() []*models.CatalogNode {
	var n []*models.CatalogNode
	schema := func(name string, p map[string]interface{}) {
		m := map[string]interface{}{"definitions_captured": true, "definitions_version": scanner.DefinitionsVersion, "scan_id": "scan-7"}
		for k, v := range p {
			m[k] = v
		}
		n = append(n, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_SCHEMA, NodeName: name, QualifiedPath: "/" + name, Properties: js(m)})
	}
	table := func(sc, name string, p map[string]interface{}) {
		m := map[string]interface{}{"schema": sc, "scan_id": "scan-7"}
		for k, v := range p {
			m[k] = v
		}
		n = append(n, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_TABLE, NodeName: name, QualifiedPath: "/" + sc + "/" + name, Properties: js(m)})
	}
	col := func(sc, tb, name, ft string, ord int, nullable bool, def string) {
		m := map[string]interface{}{"is_physical_column": true, "format_type": ft, "ordinal_position": ord, "is_nullable": nullable, "scan_id": "scan-7"}
		if def != "" {
			m["default_value"] = def
		}
		n = append(n, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_COLUMN, NodeName: name, QualifiedPath: "/" + sc + "/" + tb + "/" + name, Properties: js(m)})
	}
	cons := func(name, typ, def string) map[string]interface{} {
		return map[string]interface{}{"name": name, "type": typ, "definition": def}
	}
	schema("st_core", map[string]interface{}{
		"extensions": []interface{}{map[string]interface{}{"name": "uuid-ossp", "version": "1.1", "schema": "public"}},
		"routines":   []interface{}{map[string]interface{}{"name": "touch", "arguments": "", "definition": "CREATE OR REPLACE FUNCTION st_core.touch() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.note := 'x'; RETURN NEW; END $$"}},
	})
	schema("st_ref", nil)
	table("st_core", "account", map[string]interface{}{"constraints": []interface{}{cons("account_pkey", "p", "PRIMARY KEY (id)")}})
	col("st_core", "account", "id", "uuid", 1, false, "public.uuid_generate_v4()")
	col("st_core", "account", "name", "text", 2, false, "")
	table("st_ref", "currency", map[string]interface{}{"constraints": []interface{}{
		cons("currency_pkey", "p", "PRIMARY KEY (code)"),
		cons("fk_currency_owner", "f", "FOREIGN KEY (owner) REFERENCES st_core.account(id) ON DELETE CASCADE"),
	}})
	col("st_ref", "currency", "code", "character(3)", 1, false, "")
	col("st_ref", "currency", "owner", "uuid", 2, true, "")
	table("st_core", "quote", map[string]interface{}{
		"partition": map[string]interface{}{"key": "RANGE (quote_time)"},
		"constraints": []interface{}{cons("quote_pkey", "p", "PRIMARY KEY (id, quote_time)"), cons("chk_bid", "c", "CHECK ((bid >= (0)::numeric))"),
			cons("fk_quote_ccy", "f", "FOREIGN KEY (ccy) REFERENCES st_ref.currency(code)")},
		"indexes":  []interface{}{map[string]interface{}{"name": "idx_quote_time", "definition": "CREATE INDEX idx_quote_time ON ONLY st_core.quote USING btree (quote_time DESC)"}},
		"triggers": []interface{}{map[string]interface{}{"name": "trg_quote", "definition": "CREATE TRIGGER trg_quote BEFORE INSERT ON st_core.quote FOR EACH ROW EXECUTE FUNCTION st_core.touch()"}},
	})
	col("st_core", "quote", "id", "uuid", 1, false, "")
	col("st_core", "quote", "quote_time", "timestamp with time zone", 2, false, "")
	col("st_core", "quote", "bid", "numeric(18,9)", 3, true, "")
	col("st_core", "quote", "ccy", "character(3)", 4, true, "")
	col("st_core", "quote", "tags", "text[]", 5, true, "")
	col("st_core", "quote", "note", "text", 6, true, "")
	table("st_core", "quote_default", map[string]interface{}{"partition": map[string]interface{}{"parent": "st_core.quote", "bound": "DEFAULT"}})
	col("st_core", "quote_default", "id", "uuid", 1, false, "")
	return n
}

func (r *sagaRig) withTemplate(store *structStore) {
	r.acts.Templates = &tenantschema.Loader{Store: store}
}

func goldStore() *structStore {
	return &structStore{owner: uuid.NewString(), gold: true, nodes: structureNodes()}
}

func TestStructure_PlanCreatesNothingAndReportsWhatItWouldBuild(t *testing.T) {
	r := newSagaRig(t)
	r.withTemplate(goldStore())
	in := r.provisioned(t)
	in.TemplateDatasourceID = uuid.NewString()
	plan, err := r.acts.PlanTenantStructure(context.Background(), in)
	require.NoError(t, err)
	require.Len(t, plan.Hash, 64)
	require.Equal(t, 4, plan.Tables)
	require.Equal(t, []string{"st_core", "st_ref"}, plan.Schemas)
	require.Greater(t, plan.Statements, 10)

	db, err := r.cluster.Open(r.database)
	require.NoError(t, err)
	defer db.Close()
	var n int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_namespace WHERE nspname IN ('st_core','st_ref')`).Scan(&n))
	require.Zero(t, n, "planning creates nothing")

	again, err := r.acts.PlanTenantStructure(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, plan.Hash, again.Hash, "the same scan always plans the same structure")
}

func TestStructure_PlanRefusesWhatCannotBeDeployedBeforeAnythingIsCreated(t *testing.T) {
	t.Run("a datasource that is not the gold copy's, and nothing of theirs is read", func(t *testing.T) {
		r := newSagaRig(t)
		s := goldStore()
		s.gold = false
		r.withTemplate(s)
		in := r.in()
		in.TemplateDatasourceID = uuid.NewString()
		_, err := r.acts.PlanTenantStructure(context.Background(), in)
		require.True(t, isNonRetryableOf(err, "TenantStructureRefused"), "%v", err)
		require.ErrorIs(t, err, tenantschema.ErrNotGoldCopy)
		require.Zero(t, s.reads)
	})
	t.Run("a scan that did not record everything", func(t *testing.T) {
		r := newSagaRig(t)
		s := goldStore()
		s.nodes[0].Properties = js(map[string]interface{}{"definitions_captured": false, "scan_id": "scan-7"})
		r.withTemplate(s)
		in := r.in()
		in.TemplateDatasourceID = uuid.NewString()
		_, err := r.acts.PlanTenantStructure(context.Background(), in)
		require.True(t, isNonRetryableOf(err, "TenantStructureRefused"), "%v", err)
		require.ErrorIs(t, err, tenantschema.ErrScanNotComplete)
	})
	t.Run("no template named", func(t *testing.T) {
		r := newSagaRig(t)
		r.withTemplate(goldStore())
		_, err := r.acts.PlanTenantStructure(context.Background(), r.in())
		require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)
	})
	t.Run("a worker with no loader fails closed", func(t *testing.T) {
		r := newSagaRig(t)
		r.acts.Templates = nil
		in := r.in()
		in.TemplateDatasourceID = uuid.NewString()
		_, err := r.acts.PlanTenantStructure(context.Background(), in)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseNotConfigured"), "%v", err)
	})
	t.Run("an infrastructure error is retried, not refused", func(t *testing.T) {
		r := newSagaRig(t)
		r.acts.Templates = &tenantschema.Loader{Store: errStore{}}
		in := r.in()
		in.TemplateDatasourceID = uuid.NewString()
		_, err := r.acts.PlanTenantStructure(context.Background(), in)
		require.Error(t, err)
		require.False(t, isNonRetryableOf(err, ""), "a database blip must not fail the run for good: %v", err)
	})
}

type errStore struct{}

func (errStore) Owner(context.Context, string) (string, error) {
	return "", errors.New("alpha unavailable")
}
func (errStore) IsGoldCopy(context.Context, string) (bool, error) {
	return false, nil
}
func (errStore) Read(context.Context, string, string) (string, []*models.CatalogNode, error) {
	return "", nil, nil
}

func TestStructure_ApplyBuildsTheStructureAndGivesTheRoleTheAccessItNeeds(t *testing.T) {
	r := newSagaRig(t)
	r.withTemplate(goldStore())
	in := r.provisioned(t)
	in.TemplateDatasourceID = uuid.NewString()
	plan, err := r.acts.PlanTenantStructure(context.Background(), in)
	require.NoError(t, err)
	in.StructureHash = plan.Hash

	rep, err := r.acts.ApplyTenantStructure(context.Background(), in)
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, []string{"0001_structure.up.sql"}, rep.Ran)
	require.Contains(t, rep.Target, ":structure")

	adm, err := r.cluster.Open(r.database)
	require.NoError(t, err)
	defer adm.Close()
	var tables, parts int
	require.NoError(t, adm.QueryRow(`SELECT count(*) FROM pg_class c WHERE c.relkind IN ('r','p') AND c.relnamespace IN ('st_core'::regnamespace,'st_ref'::regnamespace)`).Scan(&tables))
	require.Equal(t, plan.Tables, tables)
	require.NoError(t, adm.QueryRow(`SELECT count(*) FROM pg_class WHERE relispartition AND relnamespace = 'st_core'::regnamespace AND relkind = 'r'`).Scan(&parts))
	require.Equal(t, 1, parts)

	// The role was created before the structure existed and granted on public only. It must now reach every schema of the structure.
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)
	stored, err := r.sec.GetMap(context.Background(), path)
	require.NoError(t, err)
	role := r.asRole(t, stored[dscreds.KeyPassword])
	_, err = role.Exec(`INSERT INTO st_core.account (name) VALUES ('a')`)
	require.NoError(t, err, "the role writes the tables of a schema that is not public, and the uuid default from the extension works")
	_, err = role.Exec(`INSERT INTO st_core.quote (id, quote_time, tags) VALUES (gen_random_uuid(), now(), ARRAY['x','y'])`)
	require.NoError(t, err, "a row routes to the default partition")
	var n int
	require.NoError(t, role.QueryRow(`SELECT count(*) FROM st_core.quote_default`).Scan(&n))
	require.Equal(t, 1, n)
	_, err = role.Exec(`CREATE TABLE st_core.sneaky (id int)`)
	require.Error(t, err, "the role gets no DDL in the structure")
	_, err = role.Exec(`DROP TABLE st_core.account CASCADE`)
	require.Error(t, err)
	_, err = role.Exec(`ALTER TABLE st_ref.currency ADD COLUMN x int`)
	require.Error(t, err)
	// A table created later, by the administrator, is usable too.
	_, err = adm.Exec(`CREATE TABLE st_ref.later (id serial PRIMARY KEY)`)
	require.NoError(t, err)
	_, err = role.Exec(`INSERT INTO st_ref.later DEFAULT VALUES`)
	require.NoError(t, err, "default privileges cover what is created after provisioning")

	// The structure's own rules hold: a check and a cascading foreign key.
	_, err = role.Exec(`INSERT INTO st_core.quote (id, quote_time, bid) VALUES (gen_random_uuid(), now(), -1)`)
	require.Error(t, err, "the recorded check constraint is enforced")
	require.NoError(t, r.acts.ProbeTenantDatabase(context.Background(), in), "and the tenant connects the way production will")
}

func TestStructure_ApplyIsIdempotentAndRefusesAChangedTemplate(t *testing.T) {
	r := newSagaRig(t)
	store := goldStore()
	r.withTemplate(store)
	in := r.provisioned(t)
	in.TemplateDatasourceID = uuid.NewString()
	plan, err := r.acts.PlanTenantStructure(context.Background(), in)
	require.NoError(t, err)
	in.StructureHash = plan.Hash
	_, err = r.acts.ApplyTenantStructure(context.Background(), in)
	require.NoError(t, err)

	again, err := r.acts.ApplyTenantStructure(context.Background(), in)
	require.NoError(t, err, "a retried run converges")
	require.True(t, again.Done)
	require.Empty(t, again.Ran, "and applies nothing twice")

	// The gold copy is rescanned between planning and applying: the run is refused, not silently applied.
	store.nodes = append(store.nodes, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_COLUMN, NodeName: "extra", QualifiedPath: "/st_core/account/extra",
		Properties: js(map[string]interface{}{"is_physical_column": true, "format_type": "integer", "ordinal_position": 3, "is_nullable": true, "scan_id": "scan-7"})})
	_, err = r.acts.ApplyTenantStructure(context.Background(), in)
	require.True(t, isNonRetryableOf(err, "TenantStructureChanged"), "%v", err)

	// A tenant built from the old structure is refused, never silently mixed, even when the new plan is the planned one.
	newPlan, err := r.acts.PlanTenantStructure(context.Background(), in)
	require.NoError(t, err)
	require.NotEqual(t, plan.Hash, newPlan.Hash)
	in.StructureHash = newPlan.Hash
	_, err = r.acts.ApplyTenantStructure(context.Background(), in)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseMigrationFailed"), "drift: %v", err)
	adm, err := r.cluster.Open(r.database)
	require.NoError(t, err)
	defer adm.Close()
	var c int
	require.NoError(t, adm.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_schema='st_core' AND table_name='account' AND column_name='extra'`).Scan(&c))
	require.Zero(t, c, "nothing of the newer structure was applied to the older tenant")
}

func TestStructure_ApplyRequiresThePlanAndRefusesBadInput(t *testing.T) {
	r := newSagaRig(t)
	r.withTemplate(goldStore())
	in := r.provisioned(t)
	_, err := r.acts.ApplyTenantStructure(context.Background(), in)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "no template and no planned hash: %v", err)
	in.TemplateDatasourceID = uuid.NewString()
	_, err = r.acts.ApplyTenantStructure(context.Background(), in)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "no planned hash: %v", err)
	in.StructureHash = "deadbeef"
	_, err = r.acts.ApplyTenantStructure(context.Background(), in)
	require.True(t, isNonRetryableOf(err, "TenantStructureChanged"), "a hash that is not the template's: %v", err)
	bad := in
	bad.DatabaseName = `x"; DROP DATABASE postgres; --`
	_, err = r.acts.ApplyTenantStructure(context.Background(), bad)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)
}

// ---- the role group: one pg_hba.conf line admits every tenant role ----------------------------------------

func (r *sagaRig) groupMembers(t *testing.T, group string) []string {
	t.Helper()
	adm, err := r.cluster.Open("postgres")
	require.NoError(t, err)
	defer adm.Close()
	rows, err := adm.Query(`SELECT m.rolname FROM pg_auth_members am JOIN pg_roles g ON g.oid = am.roleid JOIN pg_roles m ON m.oid = am.member WHERE g.rolname = $1 ORDER BY 1`, group)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		out = append(out, n)
	}
	return out
}

func TestRoleGroup_ATenantsRoleJoinsTheConfiguredGroupAndNothingElseDoes(t *testing.T) {
	r := newSagaRig(t)
	adm, err := r.cluster.Open("postgres")
	require.NoError(t, err)
	defer adm.Close()
	group := "ivy_tenant_apps_t" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	_, err = adm.Exec(`CREATE ROLE ` + group + ` NOLOGIN`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = adm.Exec(`DROP ROLE IF EXISTS ` + group) })

	r.acts.RoleGroup = group
	in := r.bound(t)
	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(context.Background(), in))
	require.Equal(t, []string{r.database + "_app"}, r.groupMembers(t, group), "the tenant's role is a member, so a single `+group` pg_hba line admits it")

	// repeatable: a retried step converges and adds nothing
	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(context.Background(), in))
	require.Equal(t, []string{r.database + "_app"}, r.groupMembers(t, group))

	// a group member gets no privilege the tenant role did not already have: it is still not a superuser and has no DDL
	var super bool
	require.NoError(t, adm.QueryRow(`SELECT rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls FROM pg_roles WHERE rolname = $1`, r.database+"_app").Scan(&super))
	require.False(t, super)
}

func TestRoleGroup_AGroupThatDoesNotExistFailsClosedAndSaysWhatToCreate(t *testing.T) {
	r := newSagaRig(t)
	r.acts.RoleGroup = "ivy_no_such_group_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	err := r.acts.ProvisionTenantDatabaseAccess(context.Background(), r.bound(t))
	require.Error(t, err)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseNotConfigured"), "a missing group is not fixed by retrying: %v", err)
	require.ErrorContains(t, err, "CREATE ROLE")
	require.ErrorContains(t, err, "docs/runbooks/tenant-database-access.md")
}

func TestRoleGroup_WithoutOneNothingChanges(t *testing.T) {
	r := newSagaRig(t)
	require.Empty(t, r.acts.RoleGroup)
	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(context.Background(), r.bound(t)))
}

// ---- binding by the template, not by an application code ---------------------------------------------------

// seedClones adds the tenant's copies of gold datasources, the way CloneGoldCopyInstance leaves them: a core_id that names the
// gold datasource each came from, and no credentials.
func (r *sagaRig) seedClones(t *testing.T, goldIDs ...string) map[string]string {
	t.Helper()
	tx, err := r.app.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, r.tenant)
	require.NoError(t, err)
	var prod, ods string
	require.NoError(t, tx.QueryRow(`SELECT id FROM tenant_product WHERE datasource_id = $1`, r.instance).Scan(&prod))
	require.NoError(t, tx.QueryRow(`SELECT id FROM alpha_datasource WHERE datasource_code = 'orm'`).Scan(&ods))
	out := map[string]string{}
	for _, g := range goldIDs {
		id := uuid.NewString()
		_, err := tx.Exec(`INSERT INTO tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, config, core_id) VALUES ($1, $2, $3, '{}'::jsonb, $4)`, id, prod, ods, g)
		require.NoError(t, err)
		out[g] = id
	}
	require.NoError(t, tx.Commit())
	return out
}

func TestStructureBind_RepointsTheTenantsCopyOfTheTemplateAndNothingElse(t *testing.T) {
	r := newSagaRig(t)
	template, other := uuid.NewString(), uuid.NewString()
	clones := r.seedClones(t, template, other)
	in := r.in()
	in.TemplateDatasourceID = template

	b, err := r.acts.BindTenantDatabase(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, clones[template], b.DatasourceID, "the clone of the template, found by the gold datasource it came from")

	db := func(id string) any {
		return r.one(t, r.tenant, `SELECT config->>'database' FROM tenant_product_datasource WHERE id = $1`, id)[0]
	}
	require.Equal(t, r.database, db(clones[template]), "the template's copy now names the tenant's own database")
	require.Nil(t, db(clones[other]), "another copy of a gold datasource is not repointed: it keeps no database and fails closed")
	require.NotEqual(t, r.database, db(r.dsOrm), "the row the application code `orm` would have chosen is NOT repointed")
	require.Nil(t, db(r.dsOrm), "and because it still named the gold copy's database it loses it, so it fails closed instead of resolving to the gold copy")
}

func TestStructureBind_RefusesWhenThereIsNotExactlyOneCopyOfTheTemplate(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		r := newSagaRig(t)
		in := r.in()
		in.TemplateDatasourceID = uuid.NewString()
		_, err := r.acts.BindTenantDatabase(context.Background(), in)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)
		require.ErrorContains(t, err, "0 datasource(s)")
		require.ErrorContains(t, err, in.TemplateDatasourceID)
		require.Equal(t, r.gold, r.one(t, r.tenant, `SELECT config->>'database' FROM tenant_product_datasource WHERE id = $1`, r.dsOrm)[0], "nothing was repointed")
	})
	t.Run("two", func(t *testing.T) {
		r := newSagaRig(t)
		template := uuid.NewString()
		r.seedClones(t, template, template)
		in := r.in()
		in.TemplateDatasourceID = template
		_, err := r.acts.BindTenantDatabase(context.Background(), in)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)
		require.ErrorContains(t, err, "2 datasource(s)")
	})
}

func TestStructureBind_WithoutATemplateTheAppCodeStillChoosesTheDatasource(t *testing.T) {
	r := newSagaRig(t)
	r.seedClones(t, uuid.NewString()) // a clone with a core_id must not confuse the app-code path
	b, err := r.acts.BindTenantDatabase(context.Background(), r.in())
	require.Error(t, err, "two datasources carry the code orm now, so the app-code path correctly refuses as ambiguous")
	require.Empty(t, b.DatasourceID)
}
