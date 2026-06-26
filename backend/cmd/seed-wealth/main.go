package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PhysicalMapping struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type SemanticTermProperties struct {
	Type            string           `json:"type"`      // "physical"
	DataType        string           `json:"data_type"` // "number", "string", "boolean"
	DisplayName     string           `json:"display_name"`
	PhysicalMapping *PhysicalMapping `json:"physical_mapping,omitempty"`
	Tags            []string         `json:"tags"`
}

type WealthEntity struct {
	Name        string
	TableName   string
	Description string
	Columns     []WealthColumn
	Category    string
}

type WealthColumn struct {
	Name         string // physical column name
	DisplayName  string // semantic term display name
	DataType     string
	Description  string
	IsPrimaryKey bool
	IsForeignKey bool
}

type WealthRelationship struct {
	SourceTable    string
	TargetTable    string
	SourceColumn   string
	TargetColumn   string
	ConstraintName string
}

var wealthEntities = []WealthEntity{
	{
		Name:        "ABS Master",
		TableName:   "wealth.abs_structures",
		Description: "Asset-Backed Security Structure Master",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "name", DisplayName: "ABS Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the ABS structure"},
			{Name: "overcollateralization_pct", DisplayName: "OC %", DataType: "number", Description: "Overcollateralization percentage"},
			{Name: "subordination_level", DisplayName: "Subordination Level", DataType: "number", Description: "Level of subordination"},
			{Name: "naic_designation", DisplayName: "NAIC Designation", DataType: "number", Description: "NAIC Risk Designation"},
		},
	},
	{
		Name:        "ABS Collateral",
		TableName:   "wealth.abs_collateral",
		Description: "Underlying collateral for ABS",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "asset_class", DisplayName: "Asset Class", IsForeignKey: true, DataType: "string", Description: "Type of collateral asset"},
			{Name: "geographic_region", DisplayName: "Geographic Region", DataType: "string", Description: "Region of the collateral"},
			{Name: "amount", DisplayName: "Collateral Amount", DataType: "number", Description: "Principal value of the collateral"},
		},
	},
	{
		Name:        "CLO Tranche Master",
		TableName:   "wealth.clo_tranches",
		Description: "CLO Tranche Master Definition",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "name", DisplayName: "Tranche Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the tranche"},
			{Name: "subordination_percentage", DisplayName: "Subordination %", DataType: "number", Description: "Subordination percentage"},
			{Name: "rating", DisplayName: "Rating", DataType: "string", Description: "Credit rating"},
			{Name: "recovery_rate_assumption", DisplayName: "Recovery Rate Assumption", DataType: "number", Description: "Assumed recovery rate"},
		},
	},
	{
		Name:        "CLO Stress Test",
		TableName:   "wealth.clo_stress_tests",
		Description: "Stress test results for CLO tranches",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "scenario_name", DisplayName: "Scenario Name", DataType: "string", Description: "Name of the stress scenario"},
			{Name: "liquidity_coverage_ratio", DisplayName: "LCR", DataType: "number", Description: "Liquidity Coverage Ratio"},
			{Name: "test_date", DisplayName: "Test Date", DataType: "date", Description: "Date of the test"},
		},
	},
	{
		Name:        "CLO Underlying Loan",
		TableName:   "wealth.clo_underlying_loans",
		Description: "Individual loans within a CLO tranche",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "industry", DisplayName: "Industry", DataType: "string", Description: "Obligor industry"},
			{Name: "obligor_name", DisplayName: "Obligor Name", IsForeignKey: true, DataType: "string", Description: "Name of the loan obligor"},
			{Name: "loan_amount", DisplayName: "Loan Amount", DataType: "number", Description: "Principal amount of the loan"},
		},
	},
	{
		Name:        "CMBS Pool Master",
		TableName:   "wealth.cmbs_pools",
		Description: "CMBS Pool Master Definition",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "name", DisplayName: "Pool Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the CMBS pool"},
			{Name: "naic_designation", DisplayName: "NAIC Designation", DataType: "number", Description: "NAIC Risk Designation"},
		},
	},
	{
		Name:        "CMBS Loan",
		TableName:   "wealth.cmbs_loans",
		Description: "Commercial mortgage loans in CMBS pools",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "property_type", DisplayName: "Property Type", DataType: "string", Description: "Commercial property type (Office, Retail, etc.)"},
			{Name: "ltv_percentage", DisplayName: "LTV %", DataType: "number", Description: "Loan-to-Value ratio"},
			{Name: "dscr_ratio", DisplayName: "DSCR", DataType: "number", Description: "Debt Service Coverage Ratio"},
			{Name: "loan_amount", DisplayName: "Loan Amount", DataType: "number", Description: "Principal of the commercial loan"},
		},
	},
	{
		Name:        "ETN",
		TableName:   "wealth.etns",
		Description: "Exchange-Traded Note Definition",
		Category:    "Funds",
		Columns: []WealthColumn{
			{Name: "ticker", DisplayName: "Ticker", IsPrimaryKey: true, DataType: "string", Description: "ETN Ticker symbol"},
			{Name: "name", DisplayName: "ETN Name", DataType: "string", Description: "Full name of the ETN"},
			{Name: "credit_rating", DisplayName: "Credit Rating", DataType: "string", Description: "Issuer credit rating"},
			{Name: "issuer_name", DisplayName: "Issuer Name", DataType: "string", Description: "Name of the ETN issuer"},
			{Name: "counterparty_exposure_pct", DisplayName: "Counterparty Exposure %", DataType: "number", Description: "Counterparty risk exposure"},
		},
	},
	{
		Name:        "Hedge Fund",
		TableName:   "wealth.hedge_funds",
		Description: "Hedge Fund Master Definition",
		Category:    "Alternatives",
		Columns: []WealthColumn{
			{Name: "name", DisplayName: "Hedge Fund Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the hedge fund"},
			{Name: "gross_leverage", DisplayName: "Gross Leverage", DataType: "number", Description: "Gross leverage ratio"},
			{Name: "net_leverage", DisplayName: "Net Leverage", DataType: "number", Description: "Net leverage ratio"},
			{Name: "lockup_period_months", DisplayName: "Lockup Period (Months)", DataType: "number", Description: "Investor lockup duration"},
			{Name: "gate_trigger_pct", DisplayName: "Gate Trigger %", DataType: "number", Description: "Redemption gate trigger percentage"},
		},
	},
	{
		Name:        "MBS Pool Master",
		TableName:   "wealth.mbs_pools",
		Description: "MBS Pool Master Definition",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "name", DisplayName: "MBS Pool Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the MBS pool"},
			{Name: "issuer", DisplayName: "Issuer", DataType: "string", Description: "MBS Issuer (Ginnie Mae, Fannie Mae, etc.)"},
			{Name: "privacy_compliant", DisplayName: "Privacy Compliant", DataType: "boolean", Description: "Whether pool data meets privacy standards"},
			{Name: "single_originator_pct", DisplayName: "Single Originator %", DataType: "number", Description: "Maximum single originator concentration"},
			{Name: "geographic_exposure_max", DisplayName: "Max Geographic Exp", DataType: "number", Description: "Maximum concentration in a single region"},
			{Name: "naic_designation", DisplayName: "NAIC Designation", DataType: "number", Description: "NAIC Risk Designation"},
		},
	},
	{
		Name:        "MBS Loan",
		TableName:   "wealth.mbs_loans",
		Description: "Mortgage loans within MBS pools",
		Category:    "Structured Products",
		Columns: []WealthColumn{
			{Name: "zip_code_masked", DisplayName: "Masked Zip", DataType: "string", Description: "Masked zip code of borrower"},
			{Name: "credit_score_bucket", DisplayName: "Credit Score Bucket", DataType: "string", Description: "Anonymized credit score range"},
			{Name: "upb", DisplayName: "UPB", DataType: "number", Description: "Unpaid Principal Balance"},
		},
	},
	{
		Name:        "Money Market Fund Master",
		TableName:   "wealth.money_market_funds",
		Description: "Money Market Fund Master Definition",
		Category:    "Funds",
		Columns: []WealthColumn{
			{Name: "name", DisplayName: "MMF Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the MMF"},
			{Name: "weekly_liquid_assets_pct", DisplayName: "Weekly Liquid Assets %", DataType: "number", Description: "Percentage of assets liquid within a week"},
			{Name: "liquidity_fee_imposed", DisplayName: "Liquidity Fee Active", DataType: "boolean", Description: "Whether a liquidity fee is currently active"},
			{Name: "fee_percentage", DisplayName: "Fee %", DataType: "number", Description: "Active liquidity fee percentage"},
		},
	},
	{
		Name:        "MMF Stress Test",
		TableName:   "wealth.mmf_stress_tests",
		Description: "Stress test results for Money Market Funds",
		Category:    "Funds",
		Columns: []WealthColumn{
			{Name: "scenario", DisplayName: "Scenario", DataType: "string", Description: "Stress scenario name"},
			{Name: "result_score", DisplayName: "Result Score", DataType: "number", Description: "Numerical score of the test result"},
			{Name: "passed", DisplayName: "Passed", DataType: "boolean", Description: "Whether the fund passed the stress test"},
		},
	},
	{
		Name:        "Venture Debt Agreement Master",
		TableName:   "wealth.venture_debt_agreements",
		Description: "Venture Debt Master Definition",
		Category:    "Private Markets",
		Columns: []WealthColumn{
			{Name: "borrower_name", DisplayName: "Borrower Name", IsPrimaryKey: true, DataType: "string", Description: "Name of the venture debt borrower"},
			{Name: "valuation_method", DisplayName: "Valuation Method", DataType: "string", Description: "Fair value methodology"},
			{Name: "discount_rate", DisplayName: "Discount Rate", DataType: "number", Description: "Rate used for valuation PV"},
		},
	},
	{
		Name:        "Venture Debt Covenant",
		TableName:   "wealth.venture_debt_covenants",
		Description: "Covenants and thresholds for venture debt",
		Category:    "Private Markets",
		Columns: []WealthColumn{
			{Name: "covenant_type", DisplayName: "Covenant Type", DataType: "string", Description: "Type of covenant (Min Cash, Max Leverage, etc.)"},
			{Name: "threshold_value", DisplayName: "Threshold", DataType: "number", Description: "Covenant threshold value"},
			{Name: "current_value", DisplayName: "Current Value", DataType: "number", Description: "Last reported covenant value"},
			{Name: "status", DisplayName: "Status", DataType: "string", Description: "Compliance status (Compliant, Waived, Breach)"},
		},
	},
}

var wealthRelationships = []WealthRelationship{
	{
		SourceTable:    "wealth.abs_collateral",
		TargetTable:    "wealth.abs_structures",
		SourceColumn:   "asset_class",
		TargetColumn:   "name",
		ConstraintName: "fk_abs_collateral_structure",
	},
	{
		SourceTable:    "wealth.clo_underlying_loans",
		TargetTable:    "wealth.clo_tranches",
		SourceColumn:   "obligor_name",
		TargetColumn:   "name",
		ConstraintName: "fk_clo_loans_tranche",
	},
}

func main() {
	connStr := "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	if env := os.Getenv("DATABASE_URL"); env != "" {
		connStr = env
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to Alpha DB.")

	// Set valid Tenant ID for Northwinds
	tenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"
	log.Printf("Using Northwinds Tenant: %s", tenantID)

	// Hardcoded Node Types from previous inspection
	tableTypeID := "49a50271-ae58-4d3e-ae1c-2f5b89d89192"        // table
	columnTypeID := "a64c1011-16e8-4ddf-b447-363bf8e15c9a"       // column
	semanticTermTypeID := "820b942a-9c9e-4abc-acdc-84616db33098" // semantic_term

	namespace := uuid.NameSpaceURL

	// Valid Tenant Datasource UUID for Wealth
	tenantDatasourceID := "a2b1c3d4-e5f6-4a5b-9c8d-7e6f5a4b3c2d"

	// Valid User UUID
	userID := "5d974525-f2e5-46e8-a4c0-fbd3d9da9a20"

	// Ensure dependencies: Valid Tenant Instance, Product, Datasource
	if err := ensureDependencies(db, tenantID, tenantDatasourceID, namespace); err != nil {
		log.Fatalf("Failed to ensure dependencies: %v", err)
	}

	for _, entity := range wealthEntities {
		log.Printf("Processing Entity: %s", entity.Name)

		// 1. Scan/Register Table Node
		tableNodeID := uuid.NewSHA1(namespace, []byte("table:"+entity.TableName)).String()
		tableProps := map[string]interface{}{"table_name": entity.TableName}
		tablePropsJson, _ := json.Marshal(tableProps)
		upsertNode(db, tableNodeID, tenantID, entity.TableName, entity.Description, tableTypeID, tablePropsJson, entity.TableName, tenantDatasourceID, "")

		// 2. Scan Columns and Semantic Terms
		var boFields []map[string]interface{}

		for i, col := range entity.Columns {
			// Register Column Node
			colQualifiedPath := entity.TableName + "." + col.Name
			columnNodeID := uuid.NewSHA1(namespace, []byte("column:"+colQualifiedPath)).String()
			colProps := map[string]interface{}{
				"column_name":    col.Name,
				"data_type":      col.DataType,
				"is_primary_key": col.IsPrimaryKey,
				"is_foreign_key": col.IsForeignKey,
			}
			colPropsJson, _ := json.Marshal(colProps)
			upsertNode(db, columnNodeID, tenantID, col.Name, col.Description, columnTypeID, colPropsJson, colQualifiedPath, tenantDatasourceID, tableNodeID)

			// Register Semantic Term
			boSlug := strings.ToLower(strings.ReplaceAll(entity.Name, " ", "_"))
			termSlug := boSlug + "." + col.Name
			termNodeID := uuid.NewSHA1(namespace, []byte("term:"+termSlug)).String()

			props := SemanticTermProperties{
				Type:        "physical",
				DataType:    col.DataType,
				DisplayName: col.DisplayName,
				PhysicalMapping: &PhysicalMapping{
					Table:  entity.TableName,
					Column: col.Name,
				},
				Tags: []string{"Wealth", "Compliance", entity.Name},
			}
			propsJson, _ := json.Marshal(props)

			qualifiedPath := "semantic_term/" + termSlug
			upsertNode(db, termNodeID, tenantID, termSlug, col.Description, semanticTermTypeID, propsJson, qualifiedPath, tenantDatasourceID, "")

			// Add as BO Field
			boFields = append(boFields, map[string]interface{}{
				"key":            col.Name,
				"name":           col.DisplayName,
				"type":           col.DataType,
				"sequence":       i,
				"description":    col.Description,
				"semanticTermId": termNodeID,
				"technicalName":  col.Name,
			})
		}

		// 3. Create Business Object in business_objects table
		// Use Check-then-Insert logic
		boKey := strings.ToLower(strings.ReplaceAll(entity.Name, " ", "_"))
		boID := uuid.NewSHA1(namespace, []byte("bo:"+boKey)).String()

		// config with is_core
		config := map[string]interface{}{"is_core": false}
		configJson, _ := json.Marshal(config)
		fieldsJson, _ := json.Marshal(boFields)

		// Check availability
		var existsCount int
		err := db.QueryRow("SELECT COUNT(*) FROM business_objects WHERE tenant_id = $1 AND key = $2", tenantID, boKey).Scan(&existsCount)
		if err != nil {
			log.Printf("Error checking for existing BO %s: %v", entity.Name, err)
			continue
		}

		if existsCount > 0 {
			// Update
			updateQuery := `
                UPDATE business_objects
                SET display_name = $1, description = $2, driver_table_id = $3, fields = $4, datasource_id = $5::uuid, last_modified_at = NOW()
                WHERE tenant_id = $6 AND key = $7
             `
			_, err = db.Exec(updateQuery, entity.Name, entity.Description, tableNodeID, fieldsJson, tenantDatasourceID, tenantID, boKey)
			if err != nil {
				log.Printf("Error updating BO %s: %v", entity.Name, err)
			} else {
				log.Printf("Updated BO %s", entity.Name)
			}

		} else {
			// Insert
			boQuery := `
                INSERT INTO business_objects (
                    id, tenant_id, key, name, display_name, technical_name,
                    description, icon, is_core, category, driver_table_id, driver_table_name,
                    created_at, created_by, last_modified_at, last_modified_by, is_active,
                    config, fields, datasource_id
                ) VALUES (
                    $1, $2, $3, $4, $5, $6,
                    $7, $8, $9, $10, $11, $12,
                    NOW(), $15, NOW(), $15, true,
                    $13, $14, $16::uuid
                )
            `
			_, err := db.Exec(boQuery,
				boID, tenantID, boKey, entity.Name, entity.Name, boKey,
				entity.Description, "database", false, entity.Category, tableNodeID, entity.TableName,
				configJson, fieldsJson, userID, tenantDatasourceID,
			)
			if err != nil {
				log.Printf("Error inserting BO %s: %v", entity.Name, err)
			} else {
				log.Printf("Inserted BO %s", entity.Name)
			}
		}
	}
	// 4. Create Sample Edges
	edgeTypeID := "f21b4a8f-05af-43b9-92cd-061265ed54e0" // foreign_key
	for _, rel := range wealthRelationships {
		sourceTableID := uuid.NewSHA1(namespace, []byte("table:"+rel.SourceTable)).String()
		targetTableID := uuid.NewSHA1(namespace, []byte("table:"+rel.TargetTable)).String()
		sourceColID := uuid.NewSHA1(namespace, []byte("column:"+rel.SourceTable+"."+rel.SourceColumn)).String()
		targetColID := uuid.NewSHA1(namespace, []byte("column:"+rel.TargetTable+"."+rel.TargetColumn)).String()

		edgeID := uuid.NewSHA1(namespace, []byte("edge:"+rel.ConstraintName)).String()

		props := map[string]interface{}{
			"primary_constraint_name": rel.ConstraintName,
			"columns": []map[string]interface{}{
				{
					"source_column":    rel.SourceColumn,
					"source_column_id": sourceColID,
					"target_column":    rel.TargetColumn,
					"target_column_id": targetColID,
				},
			},
		}
		propsJSON, _ := json.Marshal(props)

		upsertEdge(db, edgeID, tenantID, tenantDatasourceID, sourceTableID, targetTableID, edgeTypeID, propsJSON)
		log.Printf("Created Edge: %s -> %s", rel.SourceTable, rel.TargetTable)
	}

	log.Println("Wealth Seeding Complete.")

	// 5. Trigger Chart Refresh
	refreshURL := "http://localhost:8080/api/charts/" + tenantDatasourceID + "/refresh"
	resp, err := http.Post(refreshURL, "application/json", nil)
	if err != nil {
		log.Printf("Failed to trigger chart refresh: %v", err)
	} else {
		defer resp.Body.Close()
		log.Printf("Triggered chart refresh: %s", resp.Status)
	}
}

func ensureDependencies(db *sql.DB, tenantID, datasourceID string, namespace uuid.UUID) error {
	// 1. Ensure Tenant Instance
	instanceID := uuid.NewSHA1(namespace, []byte("instance:default")).String()
	_, err := db.Exec(`
		INSERT INTO tenant_instance (id, tenant_id, instance_name, config, display_name, status)
		VALUES ($1, $2, 'default', '{}', 'Default Instance', 'active')
		ON CONFLICT (tenant_id, instance_name) DO NOTHING
	`, instanceID, tenantID)
	if err != nil {
		return fmt.Errorf("upserting tenant_instance: %w", err)
	}

	// 2. Ensure Alpha Product (Wealth)
	productID := uuid.NewSHA1(namespace, []byte("product:wealth")).String()
	_, err = db.Exec(`
		INSERT INTO alpha_product (id, product_name, product_code, is_active)
		VALUES ($1, 'Wealth', 'wealth', true)
		ON CONFLICT (product_name) DO UPDATE SET is_active = EXCLUDED.is_active
	`, productID)
	if err != nil {
		return fmt.Errorf("upserting alpha_product: %w", err)
	}

	// 3. Ensure Alpha Datasource (Warehouse)
	alphaDSID := uuid.NewSHA1(namespace, []byte("datasource:warehouse")).String()
	_, err = db.Exec(`
		INSERT INTO alpha_datasource (id, datasource_name, datasource_code, is_active)
		VALUES ($1, 'Warehouse', 'warehouse', true)
		ON CONFLICT (datasource_code) DO UPDATE SET is_active = EXCLUDED.is_active
	`, alphaDSID)
	if err != nil {
		return fmt.Errorf("upserting alpha_datasource: %w", err)
	}

	// 4. Ensure Tenant Product
	tenantProductID := uuid.NewSHA1(namespace, []byte("tenant_product:wealth")).String()
	_, err = db.Exec(`
		INSERT INTO tenant_product (id, datasource_id, alpha_product_id, version, is_active, tenant_id)
		VALUES ($1, $2, $3, 1.0, true, $4)
		ON CONFLICT (datasource_id, alpha_product_id) DO NOTHING
	`, tenantProductID, instanceID, productID, tenantID)
	if err != nil {
		return fmt.Errorf("upserting tenant_product: %w", err)
	}

	// 5. Ensure Tenant Product Datasource (The Target)
	// We use the passed datasourceID (which is expected to be a2b1c3d4-e5f6-4a5b-9c8d-7e6f5a4b3c2d)
	_, err = db.Exec(`
		INSERT INTO tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, is_active, source_name)
		VALUES ($1, $2, $3, true, 'warehouse')
		ON CONFLICT (id) DO UPDATE SET is_active = EXCLUDED.is_active
	`, datasourceID, tenantProductID, alphaDSID)
	if err != nil {
		return fmt.Errorf("upserting tenant_product_datasource: %w", err)
	}
	// Also ensure unique constraint on source_name if needed, but ID is primary.

	log.Println("Dependencies ensured.")
	return nil
}

func upsertEdge(db *sql.DB, id, tenantID, datasourceID, sourceID, targetID, typeID string, props []byte) {
	query := `
		INSERT INTO catalog_edge (
			id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, edge_type_id, properties, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		ON CONFLICT (tenant_datasource_id, source_node_id, target_node_id, edge_type_id)
		DO UPDATE SET
			properties = EXCLUDED.properties,
			updated_at = NOW()
	`
	_, err := db.Exec(query, id, tenantID, datasourceID, sourceID, targetID, typeID, props)
	if err != nil {
		log.Printf("Error upserting edge: %v", err)
	}
}

func upsertNode(db *sql.DB, id, tenantID, name, description, typeID string, props []byte, path, tenantDataSourceID, parentID string) {
	var parentVal sql.NullString
	if parentID != "" {
		parentVal = sql.NullString{String: parentID, Valid: true}
	}

	query := `
		INSERT INTO catalog_node (
			id, tenant_id, node_name, description, node_type_id, properties, qualified_path, tenant_datasource_id, parent_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		)
		ON CONFLICT (id) 
		DO UPDATE SET
			node_name = EXCLUDED.node_name,
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			qualified_path = EXCLUDED.qualified_path,
			tenant_datasource_id = EXCLUDED.tenant_datasource_id,
			parent_id = EXCLUDED.parent_id,
			updated_at = NOW()
	`
	_, err := db.Exec(query, id, tenantID, name, description, typeID, props, path, tenantDataSourceID, parentVal)
	if err != nil {
		log.Printf("Error upserting node %s: %v", name, err)
	}
}
