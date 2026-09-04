package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	metadata "github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required (hardcoded fallback removed post-mTLS hardening of 100.84.50.65)")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to alpha DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Resolve gold_copy tenant at runtime — no hardcoded UUID
	var tenantID string
	err = db.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE gold_copy = true LIMIT 1`)
	if err != nil || tenantID == "" {
		err = db.GetContext(ctx, &tenantID, `SELECT id FROM tenants LIMIT 1`)
		if err != nil {
			log.Fatalf("Failed to find tenant: %v", err)
		}
		log.Printf("WARNING: No gold_copy tenant found, falling back to first tenant: %s\n", tenantID)
	} else {
		log.Printf("Resolved gold_copy tenant: %s\n", tenantID)
	}

	// Resolve ORM Suite datasource (tenant_product_datasource.id) for the gold_copy tenant
	var datasourceID string
	err = db.GetContext(ctx, &datasourceID, `
		SELECT tpd.id
		FROM tenant_product_datasource tpd
		JOIN tenant_product tp ON tp.id = tpd.tenant_product_id
		WHERE tp.tenant_id = $1
		  AND tpd.source_name = 'orm_suite_primary'
		LIMIT 1
	`, tenantID)
	if err != nil {
		log.Printf("WARNING: ORM Suite tenant_product_datasource not found for tenant %s: %v\n", tenantID, err)
		log.Printf("BOs will be created without datasource binding. Run Phase 1 migration first.\n")
		datasourceID = ""
	} else {
		log.Printf("Resolved ORM Suite datasource: %s\n", datasourceID)
	}

	// Resolve CRIMS ORM alpha_datasource.id for binding records
	var alphaDatasourceID string
	err = db.GetContext(ctx, &alphaDatasourceID, `
		SELECT id FROM alpha_datasource WHERE datasource_code = 'crims_orm' LIMIT 1
	`)
	if err != nil {
		log.Printf("WARNING: CRIMS ORM alpha_datasource not found: %v\n", err)
		alphaDatasourceID = ""
	} else {
		log.Printf("Resolved CRIMS ORM alpha_datasource: %s\n", alphaDatasourceID)
	}

	seederUUID := "113d0169-4819-42ff-968b-778f72af79e9"
	secCtx := &security.Context{
		TenantID:      tenantID,
		Region:        "us-west",
		IsGlobalAdmin: true,
	}

	ormBOs := []struct {
		req           models.CreateBusinessObjectRequest
		bindingName   string
		physicalTable string
	}{
		{
			req: models.CreateBusinessObjectRequest{
				Name:            "Trade Order",
				BOKey:           "trade_order",
				DisplayName:     "Trade Order",
				Description:     "Front-office trade order from CRIMS ORM — parent order with full specification and execution state",
				Icon:            "file-text",
				Category:        "Trading",
				TechnicalName:   "trade_order",
				DriverTableName: "oms.orders",
				EnableHistory:   true,
				HistoryMode:     "SCD2",
			},
			bindingName:   "oms_orders_primary",
			physicalTable: "oms.orders",
		},
		{
			req: models.CreateBusinessObjectRequest{
				Name:            "Trade Execution Fill",
				BOKey:           "trade_execution",
				DisplayName:     "Trade Execution Fill",
				Description:     "Venue execution fill — immutable record of one market print matched on a slice",
				Icon:            "check-circle",
				Category:        "Trading",
				TechnicalName:   "trade_execution",
				DriverTableName: "oms.execution",
				EnableHistory:   false,
			},
			bindingName:   "oms_execution_primary",
			physicalTable: "oms.execution",
		},
		{
			req: models.CreateBusinessObjectRequest{
				Name:            "Portfolio Position",
				BOKey:           "portfolio_position",
				DisplayName:     "Portfolio Position",
				Description:     "Position lot — per-account, per-security holding with cost basis and unrealised PnL from CRIMS ORM",
				Icon:            "pie-chart",
				Category:        "Portfolio Management",
				TechnicalName:   "portfolio_position",
				DriverTableName: "oms.position_lots",
				EnableHistory:   true,
				HistoryMode:     "SCD2",
			},
			bindingName:   "oms_position_lots_primary",
			physicalTable: "oms.position_lots",
		},
		{
			req: models.CreateBusinessObjectRequest{
				Name:            "Financial Security",
				BOKey:           "financial_security",
				DisplayName:     "Financial Security",
				Description:     "Security master — equities, fixed income, derivatives, and fund instruments from CRIMS ORM",
				Icon:            "shield",
				Category:        "Market Data",
				TechnicalName:   "financial_security",
				DriverTableName: "mds.security_master",
				EnableHistory:   true,
				HistoryMode:     "SCD2",
			},
			bindingName:   "mds_security_master_primary",
			physicalTable: "mds.security_master",
		},
		{
			req: models.CreateBusinessObjectRequest{
				Name:            "Trading Account",
				BOKey:           "trading_account",
				DisplayName:     "Trading Account",
				Description:     "Investment account — cash, margin, and segregated account registry from CRIMS ORM",
				Icon:            "credit-card",
				Category:        "Account Management",
				TechnicalName:   "trading_account",
				DriverTableName: "mds.account",
				EnableHistory:   true,
				HistoryMode:     "SCD2",
			},
			bindingName:   "mds_account_primary",
			physicalTable: "mds.account",
		},
	}

	boService := metadata.NewBusinessObjectService(db, nil, nil, nil)
	bindingService := metadata.NewBindingService(db)

	for _, boDef := range ormBOs {
		var existsCount int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM business_objects
			WHERE tenant_id = $1 AND key = $2
		`, tenantID, boDef.req.BOKey).Scan(&existsCount)
		if err != nil {
			log.Printf("Failed to check existence of BO '%s': %v\n", boDef.req.Name, err)
			continue
		}

		var boID string
		if existsCount > 0 {
			// Get existing BO ID
			err = db.QueryRow(`
				SELECT id FROM business_objects WHERE tenant_id = $1 AND key = $2
			`, tenantID, boDef.req.BOKey).Scan(&boID)
			if err != nil {
				log.Printf("Failed to get existing BO ID for '%s': %v\n", boDef.req.Name, err)
				continue
			}
			log.Printf("BO '%s' already exists (id=%s), skipping creation but updating binding.\n", boDef.req.Name, boID)
		} else {
			// Create new BO with DatasourceID set
			boDef.req.DatasourceID = datasourceID
			bo, err := boService.CreateBusinessObject(ctx, secCtx, boDef.req, seederUUID)
			if err != nil {
				log.Printf("Failed to create BO '%s': %v\n", boDef.req.Name, err)
				continue
			}
			boID = bo.ID
			log.Printf("Created BO '%s' (%s) with ID %s\n", bo.DisplayName, bo.Name, bo.ID)
		}

		// Create/update binding for this BO
		if alphaDatasourceID != "" {
			binding := metadata.BusinessObjectBinding{
				TenantID:          tenantID,
				BOID:              boID,
				AlphaDatasourceID: sql.NullString{String: alphaDatasourceID, Valid: true},
				IsDefault:         true,
			}
			err = bindingService.SaveBinding(ctx, binding)
			if err != nil {
				log.Printf("WARNING: Failed to save binding for BO '%s': %v\n", boDef.req.Name, err)
			} else {
				log.Printf("  Binding '%s' -> %s saved for BO %s\n", boDef.bindingName, boDef.physicalTable, boDef.req.Name)
			}
		}
	}

	fmt.Println("\nORM Business Objects seeding finished successfully.")
}
