package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/security"
)

// A stored fragment as the API would hold it: canonical content and its hash.
func stored(t *testing.T, slug string, version int, content string) *PageFragment {
	t.Helper()
	canon, _, err := canonicalFragment([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return &PageFragment{Slug: slug, Version: version, Name: slug, Content: canon, ContentHash: fragmentHash(canon)}
}

func fragStore(fs ...*PageFragment) fragmentGetter {
	return func(slug string, v int) (*PageFragment, error) {
		for _, f := range fs {
			if f.Slug == slug && f.Version == v {
				return f, nil
			}
		}
		return nil, nil
	}
}

func TestCollectFragments_ClosureIsOrderedAndComplete(t *testing.T) {
	leaf := stored(t, "leaf", 1, `{}`)
	mid := stored(t, "mid", 2, `{}`)
	top := stored(t, "alpha", 1, `{"uses":[{"fragment":"mid","version":2},{"fragment":"leaf","version":1}]}`)
	// The page names alpha and leaf; leaf is also reached through alpha.
	got, err := collectFragments([]fragmentRef{{"alpha", 1}, {"leaf", 1}}, fragStore(leaf, mid, top))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range got {
		names = append(names, f.Slug)
	}
	if strings.Join(names, ",") != "alpha,leaf,mid" { // leaf once, though reached twice
		t.Fatalf("got %v", names)
	}
}

func TestCollectFragments_RefusesMissingAndTooDeep(t *testing.T) {
	if _, err := collectFragments([]fragmentRef{{"nope", 1}}, fragStore()); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing: %v", err)
	}
	c := stored(t, "c", 1, `{}`)
	b := stored(t, "b", 1, `{"uses":[{"fragment":"c","version":1}]}`)
	a := stored(t, "a", 1, `{"uses":[{"fragment":"b","version":1}]}`)
	if _, err := collectFragments([]fragmentRef{{"a", 1}}, fragStore(a, b, c)); err == nil || !strings.Contains(err.Error(), "deeper") {
		t.Fatalf("deep: %v", err)
	}
}

func bundleOf(t *testing.T, page string, fs ...*PageFragment) PageBundle {
	b := PageBundle{Format: pageBundleFormat, Page: json.RawMessage(page)}
	for _, f := range fs {
		b.Fragments = append(b.Fragments, BundleFragment{Slug: f.Slug, Version: f.Version, Name: f.Name, ContentHash: f.ContentHash, Content: f.Content})
	}
	return b
}

const pageUsing = `{"name":"P","app":{"fragments":[{"fragment":"orders","version":1}]}}`

func TestVerifyBundle(t *testing.T) {
	orders := stored(t, "orders", 1, `{"components":{"a":{"id":"a","type":"TextBlock"}}}`)
	if err := verifyBundle(bundleOf(t, pageUsing, orders)); err != nil {
		t.Fatalf("a good bundle: %v", err)
	}
	tampered := bundleOf(t, pageUsing, orders)
	tampered.Fragments[0].Content = json.RawMessage(`{"components":{"a":{"id":"a","type":"KeyValue"}}}`)
	if err := verifyBundle(tampered); err == nil || !strings.Contains(err.Error(), "hash") {
		t.Fatalf("tampered: %v", err)
	}
	if err := verifyBundle(bundleOf(t, pageUsing)); err == nil || !strings.Contains(err.Error(), "missing fragment") {
		t.Fatalf("page reference unresolved: %v", err)
	}
	inner := stored(t, "inner", 1, `{}`)
	outer := stored(t, "orders", 1, `{"uses":[{"fragment":"inner","version":1}]}`)
	if err := verifyBundle(bundleOf(t, pageUsing, outer)); err == nil || !strings.Contains(err.Error(), `"inner"`) {
		t.Fatalf("fragment reference unresolved: %v", err)
	}
	_ = inner
	wrong := bundleOf(t, pageUsing, orders)
	wrong.Format = "other/9"
	if err := verifyBundle(wrong); err == nil {
		t.Fatal("unknown format accepted")
	}
}

func TestPlanImport_CreateReuseConflict(t *testing.T) {
	a, b, c := stored(t, "a", 1, `{}`), stored(t, "b", 1, `{"queries":[]}`), stored(t, "c", 1, `{"root":"x"}`)
	have := map[string]string{"b@1": b.ContentHash, "c@1": "different"}
	plan, err := planImport(bundleOf(t, `{}`, a, b, c), func(slug string, v int) (string, bool, error) {
		h, ok := have[slug+"@1"]
		return h, ok, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Create) != 1 || plan.Create[0].Slug != "a" || len(plan.Reuse) != 1 || len(plan.Conflicts) != 1 || !strings.Contains(plan.Conflicts[0], `"c"`) {
		t.Fatalf("plan %+v", plan)
	}
}

func TestImportBundle_ConflictWritesNothing_DryRunWritesNothing_ApplyWritesAll(t *testing.T) {
	h, mock := newFragHandler(t)
	a := stored(t, "a", 1, `{}`)
	body, _ := json.Marshal(bundleOf(t, `{"name":"P"}`, a))
	post := func(q string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.importBundle(w, tenantRequest(http.MethodPost, "/page-studio/bundles/import"+q, goldTenant.String(), security.AuthInfo{UserID: "u"}, string(body)))
		return w
	}
	absent := func() {
		mock.ExpectQuery(`SELECT content_hash FROM page_fragments`).WillReturnRows(sqlmock.NewRows([]string{"content_hash"}))
	}

	mock.ExpectQuery(`SELECT content_hash FROM page_fragments`).WillReturnRows(sqlmock.NewRows([]string{"content_hash"}).AddRow("other"))
	if w := post(""); w.Code != http.StatusConflict {
		t.Fatalf("conflict: %d %s", w.Code, w.Body.String())
	}
	absent()
	if w := post("?dryRun=true"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"applied":false`) {
		t.Fatalf("dry run: %d %s", w.Code, w.Body.String())
	}
	absent()
	mock.ExpectBegin()
	// goldCopyID + ApplyTenantGUCs (R3 wave1): gold resolve, then current/app/gold GUCs
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	mock.ExpectExec("SELECT set_config").WithArgs(goldTenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config").WithArgs(goldTenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config").WithArgs(goldTenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO page_fragments`).WithArgs(goldTenant, "a", 1, "a", "", sqlmock.AnyArg(), a.ContentHash, "u").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if w := post(""); w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"applied":true`) {
		t.Fatalf("apply: %d %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
