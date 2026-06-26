package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// FolderService provides methods for managing folders.
type FolderService struct {
	db *sqlx.DB
}

// NewFolderService creates a new FolderService.
func NewFolderService(db *sqlx.DB) *FolderService {
	return &FolderService{db: db}
}

// ListFolders retrieves folders and their contents for a user.
func (s *FolderService) ListFolders(ctx context.Context, userID string) ([]models.FullFolder, error) {
	// This is simplified for demonstration. A real implementation would check ACLs.
	var folders []models.Folder
	query := `SELECT * FROM explorer_folder WHERE owner_user_id = $1 ORDER BY name`
	if err := s.db.SelectContext(ctx, &folders, query, userID); err != nil {
		return nil, err
	}

	var fullFolders []models.FullFolder
	for _, f := range folders {
		var items []models.FolderItemDetail
		itemsQuery := `
            SELECT
                fi.item_type,
                fi.item_id,
                fi.position,
                COALESCE(sq.name, wb.name) as name
            FROM explorer_folder_item fi
            LEFT JOIN explorer_saved_query sq ON fi.item_type = 'query' AND fi.item_id = sq.id
            LEFT JOIN explorer_workbook wb ON fi.item_type = 'workbook' AND fi.item_id = wb.id
            WHERE fi.folder_id = $1
            ORDER BY fi.position
        `
		if err := s.db.SelectContext(ctx, &items, itemsQuery, f.ID); err != nil {
			logging.GetLogger().Sugar().Warnf("WARN: could not fetch items for folder %s: %v", f.Name, err)
			// Continue with empty items
		}
		fullFolders = append(fullFolders, models.FullFolder{
			Folder: f,
			Items:  items,
		})
	}
	return fullFolders, nil
}

// AddItemToFolder adds a query or workbook to a folder.
func (s *FolderService) AddItemToFolder(ctx context.Context, folderID, itemID, itemType, userID string) error {
	// Check if user has write access to the folder
	hasAccess, err := s.checkFolderAccess(ctx, userID, folderID, "write")
	if err != nil {
		return fmt.Errorf("failed to check folder access: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("user does not have write access to folder %s", folderID)
	}

	var maxPos sql.NullInt64
	posQuery := `SELECT MAX(position) FROM explorer_folder_item WHERE folder_id = $1`
	_ = s.db.GetContext(ctx, &maxPos, posQuery, folderID)

	newPos := 0
	if maxPos.Valid {
		newPos = int(maxPos.Int64) + 1
	}

	query := `INSERT INTO explorer_folder_item (id, folder_id, item_type, item_id, position) VALUES (gen_random_uuid(), $1, $2, $3, $4) ON CONFLICT (folder_id, item_type, item_id) DO NOTHING`
	_, err = s.db.ExecContext(ctx, query, folderID, itemType, itemID, newPos)
	return err
}

// GetFolderDiff compares folder contents between two points in time.
// NOTE: This is a mocked implementation for demonstration.
func (s *FolderService) GetFolderDiff(ctx context.Context, folderID string, from, to time.Time) (map[string][]models.FolderItemDetail, error) {
	// A real implementation would query an audit log or snapshot table.
	diff := make(map[string][]models.FolderItemDetail)

	// Mock response
	diff["added"] = []models.FolderItemDetail{
		{ItemID: uuid.New(), ItemType: "query", Name: "New Q3 Sales Query", Position: 3},
	}
	diff["removed"] = []models.FolderItemDetail{
		{ItemID: uuid.New(), ItemType: "query", Name: "Old Marketing Spend", Position: 1},
	}
	diff["modified"] = []models.FolderItemDetail{} // More complex to determine
	diff["unchanged"] = []models.FolderItemDetail{}

	return diff, nil
}

// GetFolderAnalytics retrieves usage statistics for a folder.
func (s *FolderService) GetFolderAnalytics(ctx context.Context, folderID string) (*models.FolderAnalyticsSummary, error) {
	// This query would aggregate from explorer_folder_usage in a real implementation.
	// We'll return mock data for now.
	summary := models.FolderAnalyticsSummary{RunCount30d: 152, ExportCount30d: 12, ViewerCount30d: 8, UpdatedAt: time.Now()}
	return &summary, nil
}
