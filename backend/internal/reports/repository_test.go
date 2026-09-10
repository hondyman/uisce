package reports_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reports"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("UISCE_TEST_DB") == "" {
		t.Skip("Skipping live database integration tests. Set UISCE_TEST_DB=1 to run.")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("UISCE_TEST_DB is set but DATABASE_URL is missing. Please provide a valid database connection string.")
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "failed to open database")

	err = db.Ping()
	require.NoError(t, err, "failed to connect to database")

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

// createTestUser creates an ephemeral app_user record and registers cleanup.
func createTestUser(t *testing.T, db *sql.DB, tenantID uuid.UUID) string {
	t.Helper()
	userID := "test-user-" + uuid.New().String()
	email := fmt.Sprintf("%s@integration-test.internal", userID)

	_, err := db.Exec(`
		INSERT INTO app_user (id, email, tenant_id, username, is_active)
		VALUES ($1, $2, $3, $4, true)
	`, userID, email, tenantID.String(), userID)
	require.NoError(t, err, "failed to insert test app_user")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM app_user WHERE id = $1`, userID)
	})

	return userID
}

// createTestTemplate inserts a template directly and registers cleanup.
func createTestTemplate(t *testing.T, repo *reports.Repository, db *sql.DB, tmpl *reports.ReportTemplate) *reports.ReportTemplate {
	t.Helper()
	ctx := context.Background()
	err := repo.CreateTemplate(ctx, tmpl)
	require.NoError(t, err, "failed to create template")

	t.Cleanup(func() {
		_, err := db.Exec(`DELETE FROM report_templates WHERE id = $1`, tmpl.ID)
		if err != nil {
			t.Logf("cleanup error for template %s: %v", tmpl.ID, err)
		}
	})

	return tmpl
}

func TestRepository_DuplicateNameCaseInsensitive_SameTenant(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	baseName := fmt.Sprintf("Portfolio Summary %s", uuid.New().String()[:8])

	tmpl1 := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: baseName,
		Category:     "executive",
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl1)

	// Attempt creating another template with differing case in the same tenant
	tmpl2 := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: strings.ToUpper(baseName),
		Category:     "executive",
		IsActive:     true,
	}
	err := repo.CreateTemplate(ctx, tmpl2)
	require.Error(t, err, "expected duplicate report name to fail")
	assert.True(t, errors.Is(err, reports.ErrConflict), "expected ErrConflict, got: %v", err)
}

func TestRepository_DuplicateName_DifferentTenant_Allowed(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	baseName := fmt.Sprintf("Cross-Tenant Report %s", uuid.New().String()[:8])

	tmpl1 := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenant1,
		TemplateName: baseName,
		Category:     "operations",
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl1)

	tmpl2 := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenant2,
		TemplateName: baseName,
		Category:     "operations",
		IsActive:     true,
	}
	err := repo.CreateTemplate(ctx, tmpl2)
	require.NoError(t, err, "same report name in a different tenant must succeed")
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM report_templates WHERE id = $1`, tmpl2.ID)
	})
}

func TestRepository_DuplicateName_InactiveReport_Allowed(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	baseName := fmt.Sprintf("Decommissioned Report %s", uuid.New().String()[:8])

	// Create an inactive template
	tmplInactive := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: baseName,
		Category:     "legacy",
		IsActive:     false,
	}
	createTestTemplate(t, repo, db, tmplInactive)

	// Create an active template with the same name (case-insensitive)
	tmplActive := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: strings.ToLower(baseName),
		Category:     "modern",
		IsActive:     true,
	}
	err := repo.CreateTemplate(ctx, tmplActive)
	require.NoError(t, err, "reusing name of inactive report in same tenant must succeed")
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM report_templates WHERE id = $1`, tmplActive.ID)
	})
}

func TestRepository_PersonalReportVisibility_UserIsolation(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)
	userB := createTestUser(t, db, tenantID)

	// 1. User A creates a personal report
	personalTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("User A Personal %s", uuid.New().String()[:8]),
		IsPersonal:   true,
		CreatedByID:  &userA,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, personalTmpl)

	// 2. Tenant-wide report (is_personal = false)
	sharedTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Tenant Shared %s", uuid.New().String()[:8]),
		IsPersonal:   false,
		CreatedByID:  &userA,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, sharedTmpl)

	// User A listing: should see BOTH personal and shared
	listA, err := repo.ListTemplatesScoped(ctx, tenantID, userA)
	require.NoError(t, err)
	foundPersonalA := false
	foundSharedA := false
	for _, item := range listA {
		if item.ID == personalTmpl.ID {
			foundPersonalA = true
		}
		if item.ID == sharedTmpl.ID {
			foundSharedA = true
		}
	}
	assert.True(t, foundPersonalA, "User A must see their own personal report")
	assert.True(t, foundSharedA, "User A must see tenant shared report")

	// User B listing: should see shared report, but NOT User A's personal report
	listB, err := repo.ListTemplatesScoped(ctx, tenantID, userB)
	require.NoError(t, err)
	foundPersonalB := false
	foundSharedB := false
	for _, item := range listB {
		if item.ID == personalTmpl.ID {
			foundPersonalB = true
		}
		if item.ID == sharedTmpl.ID {
			foundSharedB = true
		}
	}
	assert.False(t, foundPersonalB, "User B must NOT see User A's personal report")
	assert.True(t, foundSharedB, "User B must see tenant shared report")
}

func TestRepository_FavoritesIsolation_PerUser(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)
	userB := createTestUser(t, db, tenantID)

	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Shared Favorited %s", uuid.New().String()[:8]),
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl)

	// User A favorites the report
	err := repo.SetFavorite(ctx, tenantID, userA, tmpl.ID)
	require.NoError(t, err)

	// Query as User A: is_favorite should be true
	listA, err := repo.ListTemplatesScoped(ctx, tenantID, userA)
	require.NoError(t, err)
	var foundA *reports.ReportTemplate
	for i := range listA {
		if listA[i].ID == tmpl.ID {
			foundA = &listA[i]
			break
		}
	}
	require.NotNil(t, foundA)
	assert.True(t, foundA.IsFavorite, "User A should see report favorited")

	// Query as User B: is_favorite should be false
	listB, err := repo.ListTemplatesScoped(ctx, tenantID, userB)
	require.NoError(t, err)
	var foundB *reports.ReportTemplate
	for i := range listB {
		if listB[i].ID == tmpl.ID {
			foundB = &listB[i]
			break
		}
	}
	require.NotNil(t, foundB)
	assert.False(t, foundB.IsFavorite, "User B should NOT see report favorited")
}

func TestRepository_FavoriteIdempotency(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	user := createTestUser(t, db, tenantID)

	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Idempotency Test %s", uuid.New().String()[:8]),
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl)

	// Calling SetFavorite multiple times should not error (ON CONFLICT DO NOTHING)
	require.NoError(t, repo.SetFavorite(ctx, tenantID, user, tmpl.ID))
	require.NoError(t, repo.SetFavorite(ctx, tenantID, user, tmpl.ID))

	// Calling RemoveFavorite multiple times should not error
	require.NoError(t, repo.RemoveFavorite(ctx, tenantID, user, tmpl.ID))
	require.NoError(t, repo.RemoveFavorite(ctx, tenantID, user, tmpl.ID))
}

func TestRepository_DeleteTemplate_CascadesFavorites(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	user := createTestUser(t, db, tenantID)

	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Cascade Test %s", uuid.New().String()[:8]),
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl)

	err := repo.SetFavorite(ctx, tenantID, user, tmpl.ID)
	require.NoError(t, err)

	// Confirm favorite exists in DB
	var favCount int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM report_favorites WHERE template_id = $1`, tmpl.ID).Scan(&favCount)
	require.NoError(t, err)
	assert.Equal(t, 1, favCount)

	// Delete the template
	err = repo.DeleteTemplate(ctx, tmpl.ID)
	require.NoError(t, err)

	// Confirm foreign key cascade removed the favorite row
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM report_favorites WHERE template_id = $1`, tmpl.ID).Scan(&favCount)
	require.NoError(t, err)
	assert.Equal(t, 0, favCount, "report_favorites row must cascade on template deletion")
}

func TestRepository_CrossTenantUpdate_Forbidden(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenant1,
		TemplateName: fmt.Sprintf("Tenant 1 Report %s", uuid.New().String()[:8]),
		Description:  "Original description",
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl)

	// Attacker from tenant2 attempts to modify tenant1's template
	attackerUpdate := &reports.ReportTemplate{
		ID:           tmpl.ID,
		TenantID:     tenant2, // spoofed/unauthorized tenant
		TemplateName: "Hijacked Report",
		Description:  "Attacker payload",
		IsActive:     true,
	}

	err := repo.UpdateTemplate(ctx, attackerUpdate)
	require.Error(t, err, "cross-tenant update must fail")
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound, got: %v", err)

	// Verify original template is untouched
	unmodified, err := repo.GetTemplate(ctx, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, tmpl.TemplateName, unmodified.TemplateName)
	assert.Equal(t, tmpl.Description, unmodified.Description)
}

func TestRepository_CoreInheritanceFromGoldCopy_PersonalIsolated(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	goldCopyID, err := repo.ResolveGoldCopyTenantID(ctx)
	require.NoError(t, err, "must be able to resolve gold_copy master tenant")

	clientTenantID := uuid.New()
	goldCopyUser := createTestUser(t, db, goldCopyID)
	clientUser := createTestUser(t, db, clientTenantID)

	// 1. Core report in gold_copy tenant (is_personal = false)
	coreTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     goldCopyID,
		TemplateName: fmt.Sprintf("Gold Copy Core %s", uuid.New().String()[:8]),
		IsPersonal:   false,
		CreatedByID:  &goldCopyUser,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, coreTmpl)

	// 2. Personal report in gold_copy tenant (is_personal = true)
	goldPersonalTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     goldCopyID,
		TemplateName: fmt.Sprintf("Gold Copy Personal %s", uuid.New().String()[:8]),
		IsPersonal:   true,
		CreatedByID:  &goldCopyUser,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, goldPersonalTmpl)

	// Query as client tenant user
	clientList, err := repo.ListTemplatesScoped(ctx, clientTenantID, clientUser)
	require.NoError(t, err)

	foundCore := false
	foundGoldPersonal := false
	for _, item := range clientList {
		if item.ID == coreTmpl.ID {
			foundCore = true
		}
		if item.ID == goldPersonalTmpl.ID {
			foundGoldPersonal = true
		}
	}

	assert.True(t, foundCore, "Client tenant user MUST inherit non-personal core reports from gold_copy tenant")
	assert.False(t, foundGoldPersonal, "Gold-copy personal reports must NEVER leak into client tenant listings")
}

func TestRepository_CrossTenantFavoriteInjection_Forbidden(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := createTestUser(t, db, tenantA)

	// Tenant B owns this template
	tmplB := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantB,
		TemplateName: fmt.Sprintf("Tenant B Template %s", uuid.New().String()[:8]),
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmplB)

	// User A in Tenant A attempts to favorite Tenant B's template
	err := repo.SetFavorite(ctx, tenantA, userA, tmplB.ID)
	require.Error(t, err, "User in Tenant A cannot favorite a template belonging to Tenant B")
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound, got: %v", err)

	// Verify no row was inserted into report_favorites
	var count int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM report_favorites WHERE template_id = $1`, tmplB.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "No report_favorites row should exist for cross-tenant injection attempt")
}

func TestRepository_FavoriteGoldCopyCoreReport_Allowed(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	goldCopyID, err := repo.ResolveGoldCopyTenantID(ctx)
	require.NoError(t, err, "must be able to resolve gold_copy master tenant")

	clientTenantID := uuid.New()
	userA := createTestUser(t, db, clientTenantID)
	userB := createTestUser(t, db, clientTenantID)

	// Create a core report in gold-copy tenant
	coreTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     goldCopyID,
		TemplateName: fmt.Sprintf("Gold Copy Favorite Core %s", uuid.New().String()[:8]),
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, coreTmpl)

	// User A in client tenant favorites the inherited core report
	err = repo.SetFavorite(ctx, clientTenantID, userA, coreTmpl.ID)
	require.NoError(t, err, "tenant user MUST be able to favorite an inherited gold-copy core report")

	// Verify User A listing shows is_favorite = true
	listA, err := repo.ListTemplatesScoped(ctx, clientTenantID, userA)
	require.NoError(t, err)
	var foundA *reports.ReportTemplate
	for i := range listA {
		if listA[i].ID == coreTmpl.ID {
			foundA = &listA[i]
			break
		}
	}
	require.NotNil(t, foundA, "core report should appear in User A listing")
	assert.True(t, foundA.IsFavorite, "User A should see core report as favorite")

	// Verify User B listing shows is_favorite = false
	listB, err := repo.ListTemplatesScoped(ctx, clientTenantID, userB)
	require.NoError(t, err)
	var foundB *reports.ReportTemplate
	for i := range listB {
		if listB[i].ID == coreTmpl.ID {
			foundB = &listB[i]
			break
		}
	}
	require.NotNil(t, foundB, "core report should appear in User B listing")
	assert.False(t, foundB.IsFavorite, "User B should NOT see core report as favorite")
}

// -----------------------------------------------------------------------------
// Full-Text Search (FTS) Integration Tests
// -----------------------------------------------------------------------------

// TestSearch_visibilityPredicate verifies that personal reports owned by another user,
// reports owned by a different tenant, and inactive reports are strictly invisible to search
// even when the search query matches their names exactly.
func TestSearch_visibilityPredicate(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := createTestUser(t, db, tenantA)
	userB := createTestUser(t, db, tenantA)

	// 1. User A creates personal report in Tenant A matching term
	personalTerm := fmt.Sprintf("SecretAlpha%s", uuid.New().String()[:8])
	tmplPersonalA := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantA,
		TemplateName: fmt.Sprintf("Report %s", personalTerm),
		IsPersonal:   true,
		CreatedByID:  &userA,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmplPersonalA)

	// User A searching for personalTerm MUST see it
	resA, err := repo.SearchTemplatesScoped(ctx, tenantA, userA, personalTerm)
	require.NoError(t, err)
	require.Len(t, resA, 1, "User A must see their own personal report in search")
	assert.Equal(t, tmplPersonalA.ID, resA[0].ID)

	// User B searching for personalTerm MUST NOT see it (personal isolation)
	resB, err := repo.SearchTemplatesScoped(ctx, tenantA, userB, personalTerm)
	require.NoError(t, err)
	assert.Empty(t, resB, "User B must NEVER see User A's personal report in search results")

	// 2. Cross-tenant isolation: Tenant B creates public report matching a distinct term
	crossTerm := fmt.Sprintf("CrossTenant%s", uuid.New().String()[:8])
	tmplTenantB := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantB,
		TemplateName: fmt.Sprintf("Report %s", crossTerm),
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmplTenantB)

	// User A in Tenant A searching for crossTerm MUST NOT see Tenant B's report
	resCross, err := repo.SearchTemplatesScoped(ctx, tenantA, userA, crossTerm)
	require.NoError(t, err)
	assert.Empty(t, resCross, "User in Tenant A must NEVER see Tenant B's reports in search")

	// 3. Inactive report isolation: Inactive report matching a distinct term
	inactiveTerm := fmt.Sprintf("InactiveOnly%s", uuid.New().String()[:8])
	tmplInactive := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantA,
		TemplateName: fmt.Sprintf("Report %s", inactiveTerm),
		IsPersonal:   false,
		IsActive:     false,
	}
	createTestTemplate(t, repo, db, tmplInactive)

	// User A searching for inactiveTerm MUST NOT see it
	resInactive, err := repo.SearchTemplatesScoped(ctx, tenantA, userA, inactiveTerm)
	require.NoError(t, err)
	assert.Empty(t, resInactive, "Inactive reports must NEVER appear in search results")
}

// TestSearch_emptyQueryEqualsListing verifies that empty or whitespace queries produce
// the exact same results and ordering as ListTemplatesScoped.
func TestSearch_emptyQueryEqualsListing(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	for i := 0; i < 3; i++ {
		tmpl := &reports.ReportTemplate{
			ID:           uuid.New(),
			TenantID:     tenantID,
			TemplateName: fmt.Sprintf("Report %c %s", 'A'+i, uuid.New().String()[:8]),
			IsPersonal:   false,
			IsActive:     true,
		}
		createTestTemplate(t, repo, db, tmpl)
	}

	listResults, err := repo.ListTemplatesScoped(ctx, tenantID, userID)
	require.NoError(t, err)

	emptySearchResults, err := repo.SearchTemplatesScoped(ctx, tenantID, userID, "")
	require.NoError(t, err)
	assert.Equal(t, listResults, emptySearchResults, "empty query must match ListTemplatesScoped exactly")

	whitespaceSearchResults, err := repo.SearchTemplatesScoped(ctx, tenantID, userID, "   ")
	require.NoError(t, err)
	assert.Equal(t, listResults, whitespaceSearchResults, "whitespace query must match ListTemplatesScoped exactly")
}

// TestSearch_weightedRelevance verifies ranking order:
// Name match (Weight A) > Description match (Weight B) > Category match (Weight C)
func TestSearch_weightedRelevance(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	keyword := fmt.Sprintf("relevancekw%s", uuid.New().String()[:6])

	// Category match only
	catTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Arbitrary Title C %s", uuid.New().String()[:6]),
		Description:  "General operations report with no keyword",
		Category:     keyword,
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, catTmpl)

	// Description match only
	descTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Arbitrary Title B %s", uuid.New().String()[:6]),
		Description:  fmt.Sprintf("Specialized ledger detailing %s performance metrics", keyword),
		Category:     "finance",
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, descTmpl)

	// Name match
	nameTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("%s Core Report", keyword),
		Description:  "Standard accounting overview",
		Category:     "accounting",
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, nameTmpl)

	results, err := repo.SearchTemplatesScoped(ctx, tenantID, userID, keyword)
	require.NoError(t, err)
	require.Len(t, results, 3, "All three matches should be found")

	assert.Equal(t, nameTmpl.ID, results[0].ID, "Name match (Weight A) should rank first")
	assert.Equal(t, descTmpl.ID, results[1].ID, "Description match (Weight B) should rank second")
	assert.Equal(t, catTmpl.ID, results[2].ID, "Category match (Weight C) should rank third")
}

// TestSearch_typoTolerance verifies typo tolerance using word_similarity.
// A search for "Portfolo" must match a multi-word template like "Portfolio Summary ...".
func TestSearch_typoTolerance(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	targetTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Portfolio Summary %s", uuid.New().String()[:8]),
		Description:  "Multi-asset consolidated portfolio holdings and valuation breakdown",
		Category:     "wealth",
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, targetTmpl)

	// Typo query: "Portfolo" (missing 'i')
	results, err := repo.SearchTemplatesScoped(ctx, tenantID, userID, "Portfolo")
	require.NoError(t, err)

	var matched bool
	for _, res := range results {
		if res.ID == targetTmpl.ID {
			matched = true
			break
		}
	}
	assert.True(t, matched, "Typo 'Portfolo' must find 'Portfolio Summary' via word_similarity")
}

// TestSearch_coreReportInheritance verifies that master gold-copy core reports are searchable
// by client tenants.
func TestSearch_coreReportInheritance(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	goldCopyID, err := repo.ResolveGoldCopyTenantID(ctx)
	require.NoError(t, err)

	clientTenantID := uuid.New()
	user := createTestUser(t, db, clientTenantID)

	coreKeyword := fmt.Sprintf("GlobalCoreLiquidity%s", uuid.New().String()[:6])
	coreTmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     goldCopyID,
		TemplateName: fmt.Sprintf("Federal Reserve %s Analysis", coreKeyword),
		Description:  "Central bank liquidity analysis and balance sheet trends",
		Category:     "macro",
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, coreTmpl)

	results, err := repo.SearchTemplatesScoped(ctx, clientTenantID, user, coreKeyword)
	require.NoError(t, err)
	require.Len(t, results, 1, "Client tenant must find inherited core report via search")
	assert.Equal(t, coreTmpl.ID, results[0].ID)
}

// TestSearch_favoriteJoinPreserved verifies that the LEFT JOIN report_favorites
// correctly sets is_favorite = true in search results for the caller.
func TestSearch_favoriteJoinPreserved(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)
	userB := createTestUser(t, db, tenantID)

	favKeyword := fmt.Sprintf("FavSearch%s", uuid.New().String()[:6])
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: fmt.Sprintf("Fixed Income %s Benchmark", favKeyword),
		IsPersonal:   false,
		IsActive:     true,
	}
	createTestTemplate(t, repo, db, tmpl)

	// User A favorites the report
	err := repo.SetFavorite(ctx, tenantID, userA, tmpl.ID)
	require.NoError(t, err)

	// User A searches -> is_favorite should be true
	resA, err := repo.SearchTemplatesScoped(ctx, tenantID, userA, favKeyword)
	require.NoError(t, err)
	require.Len(t, resA, 1)
	assert.True(t, resA[0].IsFavorite, "User A should see is_favorite=true in search results")

	// User B searches -> is_favorite should be false
	resB, err := repo.SearchTemplatesScoped(ctx, tenantID, userB, favKeyword)
	require.NoError(t, err)
	require.Len(t, resB, 1)
	assert.False(t, resB[0].IsFavorite, "User B should see is_favorite=false in search results")
}


