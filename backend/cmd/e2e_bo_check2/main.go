package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("failed to open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Printf("db ping failed: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000000"
	parentID := uuid.NewString()
	childID := uuid.NewString()

	res, err := db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, NOW())`, parentID, tenantID, "e2e_parent_"+parentID, "E2E Parent", "E2E Parent", "e2e_parent")
	if err != nil {
		fmt.Printf("failed to insert parent: %v\n", err)
		os.Exit(1)
	}
	_ = res
	res, err = db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, parent_id, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::uuid, NOW())`, childID, tenantID, "e2e_child_"+childID, "E2E Child", "E2E Child", "e2e_child", parentID)
	if err != nil {
		fmt.Printf("failed to insert child: %v\n", err)
		_, _ = db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", parentID)
		os.Exit(1)
	}
	// Insert an example field for the child (old schema: bo_id, field_name, display_label, field_type, display_order)
	_, err = db.ExecContext(ctx, `INSERT INTO bo_fields (id, tenant_id, bo_id, field_name, display_label, field_type, display_order) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)`, uuid.NewString(), tenantID, childID, "f1", "Field 1", "string", 1)
	if err != nil {
		fmt.Printf("failed to insert child field: %v\n", err)
		_, _ = db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", childID)
		_, _ = db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", parentID)
		os.Exit(1)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", childID)
		_, _ = db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", parentID)
	}()

	svc := metadata.NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{TenantID: tenantID}
	bo, err := svc.GetBusinessObject(ctx, secCtx, parentID)
	if err != nil {
		fmt.Printf("GetBusinessObject failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("BO ID=%s Key=%s Subtypes=%d\n", bo.ID, bo.Key, len(bo.Subtypes))
	for k, s := range bo.Subtypes {
		fmt.Printf(" subtype key=%s id=%s name=%s fields=%d\n", k, s.ID, s.Name, len(s.SubtypeFields))
	}
}
