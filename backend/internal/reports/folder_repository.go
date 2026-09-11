package reports

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CheckFolderSiblingCollision checks if a folder with the given case-insensitive name
// already exists under the specified parent (or root if parentID is nil) for the given user.
func (r *Repository) CheckFolderSiblingCollision(ctx context.Context, tenantID uuid.UUID, userID string, parentID *uuid.UUID, name string, excludeFolderID uuid.UUID) error {
	var count int
	var err error

	if parentID == nil {
		query := `
			SELECT COUNT(*)
			FROM public.report_folders
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND parent_id IS NULL
			  AND LOWER(name) = LOWER($3)
			  AND id != $4
		`
		err = r.db.QueryRowContext(ctx, query, tenantID, userID, name, excludeFolderID).Scan(&count)
	} else {
		query := `
			SELECT COUNT(*)
			FROM public.report_folders
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND parent_id = $3
			  AND LOWER(name) = LOWER($4)
			  AND id != $5
		`
		err = r.db.QueryRowContext(ctx, query, tenantID, userID, *parentID, name, excludeFolderID).Scan(&count)
	}

	if err != nil {
		return fmt.Errorf("failed to check sibling folder collision: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: folder with name '%s' already exists at this level", ErrConflict, name)
	}
	return nil
}

// CreateFolder creates a new folder for the user, checking sibling uniqueness, parent ownership, and depth limit <= 5.
func (r *Repository) CreateFolder(ctx context.Context, folder *ReportFolder) error {
	if folder.ID == uuid.Nil {
		folder.ID = uuid.New()
	}

	// 1. Sibling name uniqueness pre-check
	if err := r.CheckFolderSiblingCollision(ctx, folder.TenantID, folder.UserID, folder.ParentID, folder.Name, folder.ID); err != nil {
		return err
	}

	// 2. If attaching under a parent, verify parent ownership and depth
	if folder.ParentID != nil {
		parentDepthQuery := `
			WITH RECURSIVE parent_chain AS (
				SELECT id, parent_id, 1 AS depth
				FROM public.report_folders
				WHERE id = $1 AND tenant_id = $2 AND user_id = $3
				UNION ALL
				SELECT f.id, f.parent_id, pc.depth + 1
				FROM public.report_folders f
				INNER JOIN parent_chain pc ON f.id = pc.parent_id
			)
			SELECT COUNT(*), COALESCE(MAX(depth), 0)
			FROM parent_chain;
		`
		var matchCount, parentDepth int
		err := r.db.QueryRowContext(ctx, parentDepthQuery, *folder.ParentID, folder.TenantID, folder.UserID).Scan(&matchCount, &parentDepth)
		if err != nil {
			return fmt.Errorf("failed to verify parent depth: %w", err)
		}
		if matchCount == 0 {
			return fmt.Errorf("%w: parent folder %s not found", ErrNotFound, *folder.ParentID)
		}
		// Since a new folder has subtree height 1, parentDepth + 1 must be <= 5
		if parentDepth+1 > 5 {
			return fmt.Errorf("%w: folder hierarchy exceeds maximum depth limit of 5 (parent depth %d + 1)", ErrDepthLimitExceeded, parentDepth)
		}
	}

	// 3. Insert folder
	query := `
		INSERT INTO public.report_folders (id, tenant_id, user_id, parent_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at;
	`
	err := r.db.QueryRowContext(ctx, query,
		folder.ID,
		folder.TenantID,
		folder.UserID,
		folder.ParentID,
		folder.Name,
	).Scan(&folder.CreatedAt, &folder.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("%w: sibling folder name conflict: %v", ErrConflict, err)
		}
		return fmt.Errorf("failed to insert folder: %w", err)
	}

	return nil
}

// RenameFolder updates only the folder's name, leaving parent_id untouched.
// Guarantees that a rename operation can never move a folder to root.
func (r *Repository) RenameFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, newName string) error {
	// 1. Fetch current parent_id to check sibling collision under current parent
	var currentParentID *uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT parent_id
		FROM public.report_folders
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3
	`, folderID, tenantID, userID).Scan(&currentParentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: folder not found: %s", ErrNotFound, folderID)
		}
		return fmt.Errorf("failed to get folder parent: %w", err)
	}

	// 2. Sibling collision pre-check
	if err := r.CheckFolderSiblingCollision(ctx, tenantID, userID, currentParentID, newName, folderID); err != nil {
		return err
	}

	// 3. Update name only (parent_id is never modified by RenameFolder)
	query := `
		UPDATE public.report_folders
		SET name = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND user_id = $4
	`
	result, err := r.db.ExecContext(ctx, query, newName, folderID, tenantID, userID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("%w: sibling folder name conflict: %v", ErrConflict, err)
		}
		return fmt.Errorf("failed to rename folder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update count: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: folder not found: %s", ErrNotFound, folderID)
	}

	return nil
}

// MoveFolder moves a folder under a new parent (or to root if newParentID is nil).
// Performs bidirectional depth validation (upward parent depth + downward subtree height <= 5),
// cycle detection (both direct and indirect), and ownership verification.
func (r *Repository) MoveFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, newParentID *uuid.UUID) error {
	// 1. Verify moving folder exists and belongs to caller
	var currentName string
	var currentParentID *uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT name, parent_id
		FROM public.report_folders
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3
	`, folderID, tenantID, userID).Scan(&currentName, &currentParentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: folder not found: %s", ErrNotFound, folderID)
		}
		return fmt.Errorf("failed to check moving folder: %w", err)
	}

	// If destination is already the current parent, no-op success
	if (newParentID == nil && currentParentID == nil) ||
		(newParentID != nil && currentParentID != nil && *newParentID == *currentParentID) {
		return nil
	}

	// 2. Direct cycle check: cannot move folder under itself
	if newParentID != nil && *newParentID == folderID {
		return fmt.Errorf("%w: cannot move folder under itself", ErrCycleDetected)
	}

	var parentDepth int
	if newParentID != nil {
		// 3. Upward CTE: verify target parent exists, belongs to caller, compute parent depth, and check cycles
		parentChainQuery := `
			WITH RECURSIVE parent_chain AS (
				SELECT id, parent_id, 1 AS depth
				FROM public.report_folders
				WHERE id = $1 AND tenant_id = $2 AND user_id = $3
				UNION ALL
				SELECT f.id, f.parent_id, pc.depth + 1
				FROM public.report_folders f
				INNER JOIN parent_chain pc ON f.id = pc.parent_id
			)
			SELECT 
				COUNT(*),
				COALESCE(MAX(depth), 0),
				COALESCE(BOOL_OR(id = $4), false)
			FROM parent_chain;
		`
		var matchCount int
		var isCycle bool
		err = r.db.QueryRowContext(ctx, parentChainQuery, *newParentID, tenantID, userID, folderID).Scan(&matchCount, &parentDepth, &isCycle)
		if err != nil {
			return fmt.Errorf("failed to evaluate parent hierarchy: %w", err)
		}
		// If new_parent_id was specified but parent chain returns 0 rows, target parent is foreign or nonexistent -> 404
		if matchCount == 0 {
			return fmt.Errorf("%w: target parent folder %s not found", ErrNotFound, *newParentID)
		}
		// Indirect cycle check: if moving folder appears in new parent's ancestor chain
		if isCycle {
			return fmt.Errorf("%w: moving folder %s under %s creates a circular dependency", ErrCycleDetected, folderID, *newParentID)
		}
	} else {
		parentDepth = 0
	}

	// 4. Downward CTE: compute subtree height of the moving folder
	subtreeHeightQuery := `
		WITH RECURSIVE subtree AS (
			SELECT id, 1 AS height
			FROM public.report_folders
			WHERE id = $1 AND tenant_id = $2 AND user_id = $3
			UNION ALL
			SELECT f.id, st.height + 1
			FROM public.report_folders f
			INNER JOIN subtree st ON f.parent_id = st.id
		)
		SELECT COALESCE(MAX(height), 1)
		FROM subtree;
	`
	var subtreeHeight int
	err = r.db.QueryRowContext(ctx, subtreeHeightQuery, folderID, tenantID, userID).Scan(&subtreeHeight)
	if err != nil {
		return fmt.Errorf("failed to evaluate subtree height: %w", err)
	}

	// 5. Enforce bidirectional depth limit (parent_depth + subtree_height <= 5)
	if parentDepth+subtreeHeight > 5 {
		return fmt.Errorf("%w: folder hierarchy exceeds maximum depth of 5 (parent depth %d + subtree height %d = %d)",
			ErrDepthLimitExceeded, parentDepth, subtreeHeight, parentDepth+subtreeHeight)
	}

	// 6. Check sibling name collision under new parent
	if err := r.CheckFolderSiblingCollision(ctx, tenantID, userID, newParentID, currentName, folderID); err != nil {
		return err
	}

	// 7. Execute UPDATE with defensive WHERE guard re-verifying ownership of folder and target parent
	updateQuery := `
		UPDATE public.report_folders f
		SET parent_id = $1, updated_at = NOW()
		WHERE f.id = $2 AND f.tenant_id = $3 AND f.user_id = $4
		  AND ($1::uuid IS NULL OR EXISTS (
			  SELECT 1 FROM public.report_folders p
			  WHERE p.id = $1 AND p.tenant_id = $3 AND p.user_id = $4
		  ))
	`
	result, err := r.db.ExecContext(ctx, updateQuery, newParentID, folderID, tenantID, userID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("%w: sibling folder name conflict: %v", ErrConflict, err)
		}
		return fmt.Errorf("failed to move folder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check move count: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: folder or parent not accessible", ErrNotFound)
	}

	return nil
}

// DeleteFolder removes a folder and cascades to subfolders and folder-item assignments.
// Underlying reports in public.report_templates remain completely unaffected.
func (r *Repository) DeleteFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM public.report_folders
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3
	`, folderID, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete count: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: folder not found: %s", ErrNotFound, folderID)
	}

	return nil
}

// ListFolders returns all folders owned by the caller within their tenant,
// with item counts calculated via lockstep predicate (excluding inactive or inaccessible reports).
func (r *Repository) ListFolders(ctx context.Context, tenantID uuid.UUID, userID string) ([]ReportFolder, error) {
	goldCopyID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		goldCopyID = tenantID
	}

	query := `
		SELECT 
			f.id,
			f.tenant_id,
			f.user_id,
			f.parent_id,
			f.name,
			f.created_at,
			f.updated_at,
			COUNT(DISTINCT t.id) AS item_count
		FROM public.report_folders f
		LEFT JOIN public.report_folder_items i ON i.folder_id = f.id
		LEFT JOIN public.report_templates t ON t.id = i.template_id
			  AND t.is_active = true
			  AND (t.is_personal = false OR t.created_by_id = $2)
			  AND t.tenant_id IN ($1, $3)
		WHERE f.tenant_id = $1 AND f.user_id = $2
		GROUP BY f.id, f.tenant_id, f.user_id, f.parent_id, f.name, f.created_at, f.updated_at
		ORDER BY f.name ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, userID, goldCopyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []ReportFolder
	for rows.Next() {
		var f ReportFolder
		if err := rows.Scan(
			&f.ID,
			&f.TenantID,
			&f.UserID,
			&f.ParentID,
			&f.Name,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.ItemCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("folder rows iteration error: %w", err)
	}

	return folders, nil
}

// AddReportToFolder idempotently files a report into a folder using a two-step sequence:
// 1. Verifies folder ownership and report visibility matching ListTemplatesScoped.
// 2. Executes INSERT ... ON CONFLICT DO NOTHING (idempotent success on re-filing).
func (r *Repository) AddReportToFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, templateID uuid.UUID) error {
	goldCopyID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		goldCopyID = tenantID
	}

	// Step 1: Validate folder ownership and report visibility predicate
	checkQuery := `
		SELECT 
			EXISTS(SELECT 1 FROM public.report_folders WHERE id = $1 AND tenant_id = $2 AND user_id = $3) AS folder_ok,
			EXISTS(
				SELECT 1 FROM public.report_templates t
				WHERE t.id = $4
				  AND t.tenant_id IN ($2, $5)
				  AND t.is_active = true
				  AND (t.is_personal = false OR t.created_by_id = $3)
			) AS template_ok;
	`
	var folderOK, templateOK bool
	err = r.db.QueryRowContext(ctx, checkQuery, folderID, tenantID, userID, templateID, goldCopyID).Scan(&folderOK, &templateOK)
	if err != nil {
		return fmt.Errorf("failed to validate folder item targets: %w", err)
	}
	if !folderOK {
		return fmt.Errorf("%w: folder %s not found or not accessible", ErrNotFound, folderID)
	}
	if !templateOK {
		return fmt.Errorf("%w: report template %s not found or not accessible", ErrNotFound, templateID)
	}

	// Step 2: Idempotent insert into junction table
	insertQuery := `
		INSERT INTO public.report_folder_items (folder_id, template_id, added_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (folder_id, template_id) DO NOTHING;
	`
	_, err = r.db.ExecContext(ctx, insertQuery, folderID, templateID)
	if err != nil {
		return fmt.Errorf("failed to add item to folder: %w", err)
	}

	return nil
}

// RemoveReportFromFolder removes a report from a folder if the caller owns the folder.
func (r *Repository) RemoveReportFromFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, templateID uuid.UUID) error {
	// Check folder ownership first
	var folderExists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM public.report_folders WHERE id = $1 AND tenant_id = $2 AND user_id = $3)
	`, folderID, tenantID, userID).Scan(&folderExists)
	if err != nil {
		return fmt.Errorf("failed to verify folder: %w", err)
	}
	if !folderExists {
		return fmt.Errorf("%w: folder %s not found", ErrNotFound, folderID)
	}

	_, err = r.db.ExecContext(ctx, `
		DELETE FROM public.report_folder_items
		WHERE folder_id = $1 AND template_id = $2
	`, folderID, templateID)
	if err != nil {
		return fmt.Errorf("failed to remove report from folder: %w", err)
	}

	return nil
}

// ListFolderReportIDs returns the IDs of visible reports assigned to a folder.
// Re-enforces the lockstep visibility predicate on the read path: if a filed report
// was deactivated or modified to personal by another user, it is excluded.
func (r *Repository) ListFolderReportIDs(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID) ([]uuid.UUID, error) {
	goldCopyID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		goldCopyID = tenantID
	}

	// 1. Verify folder exists and belongs to caller
	var folderExists bool
	err = r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM public.report_folders WHERE id = $1 AND tenant_id = $2 AND user_id = $3)
	`, folderID, tenantID, userID).Scan(&folderExists)
	if err != nil {
		return nil, fmt.Errorf("failed to verify folder: %w", err)
	}
	if !folderExists {
		return nil, fmt.Errorf("%w: folder %s not found", ErrNotFound, folderID)
	}

	// 2. Select template IDs joining through report_templates with lockstep predicate
	query := `
		SELECT i.template_id
		FROM public.report_folder_items i
		JOIN public.report_templates t ON t.id = i.template_id
		WHERE i.folder_id = $1
		  AND t.is_active = true
		  AND (t.is_personal = false OR t.created_by_id = $2)
		  AND t.tenant_id IN ($3, $4)
		ORDER BY i.added_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, query, folderID, userID, tenantID, goldCopyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folder report IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan template ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("folder item rows iteration error: %w", err)
	}

	return ids, nil
}
