package reports_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reports"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a test folder with automatic cleanup
func createTestFolder(t *testing.T, repo *reports.Repository, db *sql.DB, tenantID uuid.UUID, userID string, parentID *uuid.UUID, name string) *reports.ReportFolder {
	t.Helper()
	folder := &reports.ReportFolder{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   userID,
		ParentID: parentID,
		Name:     name,
	}
	err := repo.CreateFolder(context.Background(), folder)
	require.NoError(t, err, "failed to create test folder")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM public.report_folders WHERE id = $1`, folder.ID)
	})

	return folder
}

func TestFolderRepository_SiblingNameCollision(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)

	// 1. Create root folder "Investments"
	f1 := createTestFolder(t, repo, db, tenantID, userA, nil, "Investments")
	assert.NotEqual(t, uuid.Nil, f1.ID)

	// 2. Attempt duplicate root folder with different casing "investments"
	dupRoot := &reports.ReportFolder{
		TenantID: tenantID,
		UserID:   userA,
		ParentID: nil,
		Name:     "investments",
	}
	err := repo.CreateFolder(ctx, dupRoot)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrConflict), "expected ErrConflict on duplicate root sibling, got: %v", err)

	// 3. Create child folder "Equities" under f1
	c1 := createTestFolder(t, repo, db, tenantID, userA, &f1.ID, "Equities")
	assert.NotEqual(t, uuid.Nil, c1.ID)

	// 4. Attempt duplicate child folder with different casing "EQUITIES" under f1
	dupChild := &reports.ReportFolder{
		TenantID: tenantID,
		UserID:   userA,
		ParentID: &f1.ID,
		Name:     "EQUITIES",
	}
	err = repo.CreateFolder(ctx, dupChild)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrConflict), "expected ErrConflict on duplicate child sibling, got: %v", err)

	// 5. Creating child "Equities" under another parent or root is allowed
	f2 := createTestFolder(t, repo, db, tenantID, userA, nil, "Operations")
	c2 := createTestFolder(t, repo, db, tenantID, userA, &f2.ID, "Equities")
	assert.NotEqual(t, uuid.Nil, c2.ID)

	// 6. Rename collision: attempting to rename f2 to "investments" fails
	err = repo.RenameFolder(ctx, tenantID, userA, f2.ID, "investments")
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrConflict), "expected ErrConflict on rename collision, got: %v", err)
}

func TestFolderRepository_UserIsolation(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)
	userB := createTestUser(t, db, tenantID)

	// User A creates a folder
	folderA := createTestFolder(t, repo, db, tenantID, userA, nil, "User A Confidential")

	// User B listing folders must not see User A's folder
	foldersB, err := repo.ListFolders(ctx, tenantID, userB)
	require.NoError(t, err)
	for _, f := range foldersB {
		assert.NotEqual(t, folderA.ID, f.ID, "User B should not see User A's folder in listing")
	}

	// User B attempting to rename User A's folder returns ErrNotFound (404, not revealing existence)
	err = repo.RenameFolder(ctx, tenantID, userB, folderA.ID, "Hijacked Name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound when User B renames User A's folder, got: %v", err)

	// User B attempting to move User A's folder returns ErrNotFound
	err = repo.MoveFolder(ctx, tenantID, userB, folderA.ID, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound when User B moves User A's folder, got: %v", err)

	// User B attempting to delete User A's folder returns ErrNotFound
	err = repo.DeleteFolder(ctx, tenantID, userB, folderA.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound when User B deletes User A's folder, got: %v", err)
}

func TestFolderRepository_TenantIsolation(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := createTestUser(t, db, tenantA)
	userB := createTestUser(t, db, tenantB)

	// Tenant A user creates a folder
	folderA := createTestFolder(t, repo, db, tenantA, userA, nil, "Tenant A Private Folder")

	// Tenant B user listing folders must not see Tenant A's folder
	foldersB, err := repo.ListFolders(ctx, tenantB, userB)
	require.NoError(t, err)
	for _, f := range foldersB {
		assert.NotEqual(t, folderA.ID, f.ID, "Tenant B user should not see Tenant A's folder")
	}

	// Tenant B user attempting to touch Tenant A's folder returns ErrNotFound
	err = repo.RenameFolder(ctx, tenantB, userB, folderA.ID, "Cross-Tenant Name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound))

	err = repo.DeleteFolder(ctx, tenantB, userB, folderA.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound))
}

func TestFolderRepository_GoldCopyCoreFiling_Allowed(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	goldCopyID, err := repo.ResolveGoldCopyTenantID(ctx)
	require.NoError(t, err)

	tenantA := uuid.New()
	userA := createTestUser(t, db, tenantA)

	// Create a Gold Copy Core Report
	coreTmpl := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     goldCopyID,
		TemplateName: "Gold Copy Core Audit Summary " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   false,
	})

	// User A creates a private folder
	folder := createTestFolder(t, repo, db, tenantA, userA, nil, "Core Reviews")

	// User A files the Gold Copy Core Report into their private folder
	err = repo.AddReportToFolder(ctx, tenantA, userA, folder.ID, coreTmpl.ID)
	require.NoError(t, err, "tenant user filing gold copy core report into private folder must succeed")

	// Verify report ID appears in folder listing
	ids, err := repo.ListFolderReportIDs(ctx, tenantA, userA, folder.ID)
	require.NoError(t, err)
	assert.Contains(t, ids, coreTmpl.ID, "folder items must include filed gold copy core report")

	// Verify folder item_count reflects the filed core report
	folders, err := repo.ListFolders(ctx, tenantA, userA)
	require.NoError(t, err)
	var found bool
	for _, f := range folders {
		if f.ID == folder.ID {
			found = true
			assert.Equal(t, 1, f.ItemCount, "item count must reflect filed core report")
		}
	}
	assert.True(t, found)
}

func TestFolderRepository_AddItem_Idempotency(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)

	tmpl := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: "Idempotent Test Template " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   false,
	})

	folder := createTestFolder(t, repo, db, tenantID, userA, nil, "Idempotency Folder")

	// First filing succeeds
	err := repo.AddReportToFolder(ctx, tenantID, userA, folder.ID, tmpl.ID)
	require.NoError(t, err)

	// Second identical filing succeeds idempotently without error (no 404, no 409)
	err = repo.AddReportToFolder(ctx, tenantID, userA, folder.ID, tmpl.ID)
	require.NoError(t, err, "re-filing an already filed report must succeed idempotently")

	// Items listing contains exactly 1 entry for this template
	ids, err := repo.ListFolderReportIDs(ctx, tenantID, userA, folder.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, tmpl.ID, ids[0])
}

func TestFolderRepository_MoveFolder_ForeignParent_Forbidden(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)
	userB := createTestUser(t, db, tenantID)

	folderA := createTestFolder(t, repo, db, tenantID, userA, nil, "User A Folder")
	folderB := createTestFolder(t, repo, db, tenantID, userB, nil, "User B Folder")

	// User A attempts to move folderA under User B's folderB -> must return ErrNotFound
	err := repo.MoveFolder(ctx, tenantID, userA, folderA.ID, &folderB.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound moving under foreign parent, got: %v", err)

	// User A attempts to move folderA under a non-existent UUID -> ErrNotFound
	randomUUID := uuid.New()
	err = repo.MoveFolder(ctx, tenantID, userA, folderA.ID, &randomUUID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound moving under non-existent parent, got: %v", err)
}

func TestFolderRepository_DepthLimit_RejectsSixthLevel(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	// Build a genuine 5-level deep hierarchy:
	// Level 1: Root
	l1 := createTestFolder(t, repo, db, tenantID, userID, nil, "Level 1 Root")
	// Level 2
	l2 := createTestFolder(t, repo, db, tenantID, userID, &l1.ID, "Level 2")
	// Level 3
	l3 := createTestFolder(t, repo, db, tenantID, userID, &l2.ID, "Level 3")
	// Level 4
	l4 := createTestFolder(t, repo, db, tenantID, userID, &l3.ID, "Level 4")
	// Level 5
	l5 := createTestFolder(t, repo, db, tenantID, userID, &l4.ID, "Level 5")

	// Attempting to create Level 6 under Level 5 must fail with ErrDepthLimitExceeded
	l6 := &reports.ReportFolder{
		TenantID: tenantID,
		UserID:   userID,
		ParentID: &l5.ID,
		Name:     "Level 6 Attempt",
	}
	err := repo.CreateFolder(ctx, l6)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrDepthLimitExceeded), "expected ErrDepthLimitExceeded on 6th level creation, got: %v", err)

	// Create a separate 2-level subtree: SubtreeRoot (height=2) -> SubtreeChild (height=1)
	subRoot := createTestFolder(t, repo, db, tenantID, userID, nil, "Subtree Root")
	_ = createTestFolder(t, repo, db, tenantID, userID, &subRoot.ID, "Subtree Child")

	// Attempt to move SubtreeRoot under l4 (depth=4 + height=2 = 6 > 5) -> must fail!
	err = repo.MoveFolder(ctx, tenantID, userID, subRoot.ID, &l4.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrDepthLimitExceeded), "expected ErrDepthLimitExceeded moving 2-level subtree under depth 4 parent, got: %v", err)

	// Moving SubtreeRoot under l3 (depth=3 + height=2 = 5 <= 5) -> must succeed!
	err = repo.MoveFolder(ctx, tenantID, userID, subRoot.ID, &l3.ID)
	require.NoError(t, err, "moving 2-level subtree under depth 3 parent must succeed (total depth 5)")
}

func TestFolderRepository_CycleDetection_RejectsDirectAndIndirectCycle(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	// Setup hierarchy: A -> B -> C
	nodeA := createTestFolder(t, repo, db, tenantID, userID, nil, "Node A")
	nodeB := createTestFolder(t, repo, db, tenantID, userID, &nodeA.ID, "Node B")
	nodeC := createTestFolder(t, repo, db, tenantID, userID, &nodeB.ID, "Node C")

	// 1. Direct cycle: Move A under A -> ErrCycleDetected
	err := repo.MoveFolder(ctx, tenantID, userID, nodeA.ID, &nodeA.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrCycleDetected), "expected ErrCycleDetected on self-parenting, got: %v", err)

	// 2. Direct cycle: Move A under its child B -> ErrCycleDetected
	err = repo.MoveFolder(ctx, tenantID, userID, nodeA.ID, &nodeB.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrCycleDetected), "expected ErrCycleDetected moving A under child B, got: %v", err)

	// 3. Indirect cycle: Move A under its grandchild C -> ErrCycleDetected
	err = repo.MoveFolder(ctx, tenantID, userID, nodeA.ID, &nodeC.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrCycleDetected), "expected ErrCycleDetected moving A under grandchild C, got: %v", err)
}

func TestFolderRepository_ListFolderReportIDs_ExcludesInactiveOrRepersonalized(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userA := createTestUser(t, db, tenantID)

	tmpl := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: "Lockstep Test Report " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   false,
	})

	folder := createTestFolder(t, repo, db, tenantID, userA, nil, "Lockstep Folder")

	// File the active report into folder
	err := repo.AddReportToFolder(ctx, tenantID, userA, folder.ID, tmpl.ID)
	require.NoError(t, err)

	// Assert report ID is returned in folder contents
	ids, err := repo.ListFolderReportIDs(ctx, tenantID, userA, folder.ID)
	require.NoError(t, err)
	assert.Contains(t, ids, tmpl.ID)

	// 1. Deactivate the report (soft delete / active = false)
	_, err = db.Exec(`UPDATE public.report_templates SET is_active = false WHERE id = $1`, tmpl.ID)
	require.NoError(t, err)

	// Read path lockstep: report must immediately disappear from folder contents!
	idsAfterDeactivate, err := repo.ListFolderReportIDs(ctx, tenantID, userA, folder.ID)
	require.NoError(t, err)
	assert.NotContains(t, idsAfterDeactivate, tmpl.ID, "deactivated report must disappear from folder read path")

	// Folder item count in ListFolders must also exclude it
	folders, err := repo.ListFolders(ctx, tenantID, userA)
	require.NoError(t, err)
	for _, f := range folders {
		if f.ID == folder.ID {
			assert.Equal(t, 0, f.ItemCount, "item count must exclude inactive reports")
		}
	}

	// 2. Reactivate the report
	_, err = db.Exec(`UPDATE public.report_templates SET is_active = true WHERE id = $1`, tmpl.ID)
	require.NoError(t, err)

	// Report must reappear in folder contents
	idsAfterReactivate, err := repo.ListFolderReportIDs(ctx, tenantID, userA, folder.ID)
	require.NoError(t, err)
	assert.Contains(t, idsAfterReactivate, tmpl.ID, "reactivated report must reappear in folder contents")
}

func TestFolderRepository_CrossTenantFiling_Forbidden(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := createTestUser(t, db, tenantA)
	userB := createTestUser(t, db, tenantB)

	// Tenant B report
	tmplB := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantB,
		TemplateName: "Tenant B Secret Report " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   false,
	})

	// Tenant A user folder
	folderA := createTestFolder(t, repo, db, tenantA, userA, nil, "User A Investigation")

	// User A attempts to file Tenant B's report into their folder
	err := repo.AddReportToFolder(ctx, tenantA, userA, folderA.ID, tmplB.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound when filing cross-tenant report, got: %v", err)

	// Confirm no junction row was inserted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM public.report_folder_items WHERE folder_id = $1`, folderA.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "zero junction rows must be created on cross-tenant filing attempt")

	// User A attempts to file userB's personal report (even if in same tenant)
	personalTmplB := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantA,
		TemplateName: "User B Personal Report " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   true,
		CreatedByID:  &userB,
	})
	err = repo.AddReportToFolder(ctx, tenantA, userA, folderA.ID, personalTmplB.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, reports.ErrNotFound), "expected ErrNotFound when filing another user's personal report, got: %v", err)
}

func TestFolderRepository_DeleteFolder_CascadesItemsPreservesReports(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	tmpl1 := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: "Preserved Report 1 " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   false,
	})
	tmpl2 := createTestTemplate(t, repo, db, &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: "Preserved Report 2 " + uuid.New().String()[:8],
		IsActive:     true,
		IsPersonal:   false,
	})

	parentFolder := createTestFolder(t, repo, db, tenantID, userID, nil, "Parent Folder")
	childFolder := createTestFolder(t, repo, db, tenantID, userID, &parentFolder.ID, "Child Folder")

	// File reports into parent and child folders
	err := repo.AddReportToFolder(ctx, tenantID, userID, parentFolder.ID, tmpl1.ID)
	require.NoError(t, err)
	err = repo.AddReportToFolder(ctx, tenantID, userID, childFolder.ID, tmpl2.ID)
	require.NoError(t, err)

	// Delete parent folder
	err = repo.DeleteFolder(ctx, tenantID, userID, parentFolder.ID)
	require.NoError(t, err)

	// 1. Verify parent folder is deleted
	var parentCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM public.report_folders WHERE id = $1`, parentFolder.ID).Scan(&parentCount)
	require.NoError(t, err)
	assert.Equal(t, 0, parentCount)

	// 2. Verify child folder was cascade-deleted
	var childCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM public.report_folders WHERE id = $1`, childFolder.ID).Scan(&childCount)
	require.NoError(t, err)
	assert.Equal(t, 0, childCount)

	// 3. Verify junction rows for both folders were cascade-deleted
	var junctionCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM public.report_folder_items WHERE folder_id IN ($1, $2)`, parentFolder.ID, childFolder.ID).Scan(&junctionCount)
	require.NoError(t, err)
	assert.Equal(t, 0, junctionCount)

	// 4. CRITICAL: Verify underlying report templates in public.report_templates are 100% INTACT!
	var reportCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM public.report_templates WHERE id IN ($1, $2)`, tmpl1.ID, tmpl2.ID).Scan(&reportCount)
	require.NoError(t, err)
	assert.Equal(t, 2, reportCount, "report_templates records must remain intact after folder deletion")
}

func TestFolderRepository_RenameFolder_LeavesParentUntouched(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := createTestUser(t, db, tenantID)

	parent := createTestFolder(t, repo, db, tenantID, userID, nil, "Parent Container")
	child := createTestFolder(t, repo, db, tenantID, userID, &parent.ID, "Original Child Name")

	// Rename child folder
	err := repo.RenameFolder(ctx, tenantID, userID, child.ID, "Updated Child Name")
	require.NoError(t, err)

	// Query child folder directly from database
	var currentName string
	var currentParentID *uuid.UUID
	err = db.QueryRow(`SELECT name, parent_id FROM public.report_folders WHERE id = $1`, child.ID).Scan(&currentName, &currentParentID)
	require.NoError(t, err)

	assert.Equal(t, "Updated Child Name", currentName)
	require.NotNil(t, currentParentID, "parent_id must not be moved to root")
	assert.Equal(t, parent.ID, *currentParentID, "parent_id must remain unchanged after rename")
}
