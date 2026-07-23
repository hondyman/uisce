package services

import (
	"context"

	"github.com/google/uuid"
)

// checkFolderAccess checks if a user has specific access to a folder
func (s *FolderService) checkFolderAccess(ctx context.Context, userID, folderID, accessType string) (bool, error) {
	// Parse UUIDs
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}

	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		return false, err
	}

	// Query folder permissions
	query := `
		SELECT COUNT(*) > 0
		FROM folder_permissions
		WHERE folder_id = $1 
		  AND user_id = $2 
		  AND permission_type IN ('owner', 'write', 'read')
		  AND (
		    permission_type = 'owner' OR
		    (permission_type = 'write' AND $3 IN ('write', 'read')) OR
		    (permission_type = 'read' AND $3 = 'read')
		  )
	`

	var hasAccess bool
	err = s.db.QueryRowContext(ctx, query, folderUUID, userUUID, accessType).Scan(&hasAccess)
	if err != nil {
		return false, err
	}

	return hasAccess, nil
}
