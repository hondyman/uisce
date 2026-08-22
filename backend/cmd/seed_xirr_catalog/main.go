package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to alpha database.")

	// 1. Fetch Master / Gold Copy Tenant
	var masterTenantID string
	err = db.QueryRow("SELECT id FROM tenants WHERE gold_copy = true LIMIT 1").Scan(&masterTenantID)
	if err != nil {
		log.Fatalf("Fatal: Master tenant (gold_copy = true) not found: %v", err)
	}
	log.Printf("Master Tenant (Core): %s", masterTenantID)

	// Fetch all tenants
	type TenantRow struct {
		ID       string
		Name     string
		GoldCopy bool
	}
	var tenants []TenantRow
	tRows, err := db.Query("SELECT id, name, gold_copy FROM tenants")
	if err != nil {
		log.Fatalf("Failed to query tenants: %v", err)
	}
	defer tRows.Close()
	for tRows.Next() {
		var tr TenantRow
		if err := tRows.Scan(&tr.ID, &tr.Name, &tr.GoldCopy); err == nil {
			tenants = append(tenants, tr)
		}
	}

	// 2. Fetch Node Type IDs & Edge Type IDs
	var boNodeTypeID, stNodeTypeID, boFieldNodeTypeID string
	_ = db.QueryRow("SELECT id FROM catalog_node_types WHERE catalog_type_name = 'business_object' LIMIT 1").Scan(&boNodeTypeID)
	_ = db.QueryRow("SELECT id FROM catalog_node_types WHERE catalog_type_name = 'semantic_term' LIMIT 1").Scan(&stNodeTypeID)
	_ = db.QueryRow("SELECT id FROM catalog_node_types WHERE catalog_type_name = 'bo_field' LIMIT 1").Scan(&boFieldNodeTypeID)

	var hasFieldEdgeTypeID, backedByTermEdgeTypeID, usesTermEdgeTypeID string
	_ = db.QueryRow("SELECT id FROM catalog_edge_types WHERE edge_type_name = 'HAS_FIELD' LIMIT 1").Scan(&hasFieldEdgeTypeID)
	_ = db.QueryRow("SELECT id FROM catalog_edge_types WHERE edge_type_name = 'BACKED_BY_TERM' LIMIT 1").Scan(&backedByTermEdgeTypeID)
	_ = db.QueryRow("SELECT id FROM catalog_edge_types WHERE edge_type_name = 'USES_SEMANTIC_TERM' LIMIT 1").Scan(&usesTermEdgeTypeID)

	log.Printf("Node Types: BO=%s, ST=%s, BOField=%s", boNodeTypeID, stNodeTypeID, boFieldNodeTypeID)
	log.Printf("Edge Types: HAS_FIELD=%s, BACKED_BY_TERM=%s, USES_TERM=%s", hasFieldEdgeTypeID, backedByTermEdgeTypeID, usesTermEdgeTypeID)

	ns := uuid.NameSpaceURL

	for _, tenant := range tenants {
		log.Printf("--- Seeding Catalog for Tenant: %s (GoldCopy: %v) ---", tenant.Name, tenant.GoldCopy)

		// 3A. Semantic Terms
		terms := []struct {
			Key         string
			Name        string
			Description string
			Category    string
			Subcategory string
			Formula     string
			DataType    string
			ReturnType  string
		}{
			{
				Key:         "investment_xirr",
				Name:        "Investment XIRR",
				Description: "Exact date-weighted internal rate of return for irregular private market cash flows.",
				Category:    "Private Markets",
				Subcategory: "Performance",
				Formula:     "{{ xirr(ARRAY_AGG(${pre_agg_name}.cash_flow), ARRAY_AGG(${pre_agg_name}.transaction_date)) }}",
				DataType:    "number",
				ReturnType:  "percent",
			},
			{
				Key:         "excel_xirr",
				Name:        "Excel XIRR",
				Description: "Excel compatible XIRR calculation function for uneven cash flow streams.",
				Category:    "Private Markets",
				Subcategory: "IRR",
				Formula:     "{{ excel_formula('=XIRR({cash_flows}, {dates})') }}",
				DataType:    "number",
				ReturnType:  "percent",
			},
			{
				Key:         "current_nav",
				Name:        "Current NAV",
				Description: "Latest reported Net Asset Value of alternative investment holding.",
				Category:    "Private Markets",
				Subcategory: "Valuation",
				Formula:     "current_nav",
				DataType:    "number",
				ReturnType:  "currency",
			},
			{
				Key:         "total_capital_called",
				Name:        "Total Capital Called",
				Description: "Cumulative paid-in capital called by general partner across all drawdowns.",
				Category:    "Private Markets",
				Subcategory: "Cash Flows",
				Formula:     "SUM(amount_funded)",
				DataType:    "number",
				ReturnType:  "currency",
			},
			{
				Key:         "total_distributions",
				Name:        "Total Distributions",
				Description: "Cumulative capital distributions returned to investors.",
				Category:    "Private Markets",
				Subcategory: "Cash Flows",
				Formula:     "SUM(amount)",
				DataType:    "number",
				ReturnType:  "currency",
			},
		}

		termNodeIDs := make(map[string]string)

		for _, t := range terms {
			termNodeID := uuid.NewSHA1(ns, []byte(fmt.Sprintf("term:%s:%s", tenant.ID, t.Key))).String()
			qPath := fmt.Sprintf("semantic_term/%s.%s", t.Category, t.Key)

			// Look up if already exists by (tenant_id, node_type_id, qualified_path)
			var existingID string
			err := db.QueryRow("SELECT id FROM catalog_node WHERE tenant_id = $1 AND node_type_id = $2 AND qualified_path = $3", tenant.ID, stNodeTypeID, qPath).Scan(&existingID)
			if err == nil && existingID != "" {
				termNodeID = existingID
			}
			termNodeIDs[t.Key] = termNodeID

			props, _ := json.Marshal(map[string]interface{}{
				"type":         "calculated",
				"term_type":    "calculated",
				"data_type":    t.DataType,
				"expression":   t.Formula,
				"formula":      t.Formula,
				"display_name": t.Name,
				"tags":         []string{t.Category, t.Subcategory},
				"return_type":  t.ReturnType,
			})

			_, err = db.Exec(`
				INSERT INTO catalog_node (id, tenant_id, node_name, description, node_type_id, properties, qualified_path, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
				ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET
					node_name = EXCLUDED.node_name,
					description = EXCLUDED.description,
					properties = EXCLUDED.properties,
					updated_at = NOW()
			`, termNodeID, tenant.ID, t.Name, t.Description, stNodeTypeID, props, qPath)
			if err != nil {
				log.Printf("  Error inserting term %s: %v", t.Key, err)
			} else {
				log.Printf("  ✓ Upserted Semantic Term: %s (%s)", t.Name, termNodeID)
			}
		}

		// 3B. Business Object: Alternative Investment
		boKey := "alternative_investment"
		boNodeID := uuid.NewSHA1(ns, []byte(fmt.Sprintf("bo:%s:%s", tenant.ID, boKey))).String()
		boQPath := fmt.Sprintf("bo:%s", boKey)

		var existingBOID string
		err = db.QueryRow("SELECT id FROM catalog_node WHERE tenant_id = $1 AND node_type_id = $2 AND qualified_path = $3", tenant.ID, boNodeTypeID, boQPath).Scan(&existingBOID)
		if err == nil && existingBOID != "" {
			boNodeID = existingBOID
		}

		boProps, _ := json.Marshal(map[string]interface{}{
			"bo_key":            boKey,
			"display_name":      "Alternative Investment",
			"description":       "Alternative investment master holding, tracking capital commitments, calls, distributions and on-the-fly XIRR performance.",
			"driver_table_name": "alternative_investments",
			"category":          "Private Markets",
		})

		_, err = db.Exec(`
			INSERT INTO catalog_node (id, tenant_id, node_name, description, node_type_id, properties, qualified_path, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				updated_at = NOW()
		`, boNodeID, tenant.ID, "Alternative Investment", "Alternative investment entity with automated XIRR calculation", boNodeTypeID, boProps, boQPath)
		if err != nil {
			log.Printf("  Error inserting BO node: %v", err)
		} else {
			log.Printf("  ✓ Upserted Business Object Node: Alternative Investment (%s)", boNodeID)
		}

		// 3C. Business Object Fields (bo_field nodes)
		type BOFieldDef struct {
			Key         string
			Name        string
			Role        string
			TermKey     string
		}

		boFields := []BOFieldDef{
			{Key: "irr_since_inception", Name: "Inception XIRR", Role: "MEASURE", TermKey: "investment_xirr"},
			{Key: "current_nav", Name: "Current Valuation (NAV)", Role: "MEASURE", TermKey: "current_nav"},
			{Key: "total_capital_called", Name: "Capital Called", Role: "MEASURE", TermKey: "total_capital_called"},
			{Key: "total_distributions", Name: "Capital Distributed", Role: "MEASURE", TermKey: "total_distributions"},
		}

		var boFieldJsonList []map[string]interface{}

		for _, f := range boFields {
			fNodeID := uuid.NewSHA1(ns, []byte(fmt.Sprintf("bofield:%s:%s:%s", tenant.ID, boKey, f.Key))).String()
			fQPath := fmt.Sprintf("bo_field:%s:%s", boKey, f.Key)
			termID := termNodeIDs[f.TermKey]

			var existingFID string
			err = db.QueryRow("SELECT id FROM catalog_node WHERE tenant_id = $1 AND node_type_id = $2 AND qualified_path = $3", tenant.ID, boFieldNodeTypeID, fQPath).Scan(&existingFID)
			if err == nil && existingFID != "" {
				fNodeID = existingFID
			}

			fProps, _ := json.Marshal(map[string]interface{}{
				"field_role":             f.Role,
				"is_exposed":             true,
				"binding_requirement":    "REQUIRED",
				"semantic_term_node_id":  termID,
			})

			_, err = db.Exec(`
				INSERT INTO catalog_node (id, tenant_id, node_name, description, node_type_id, properties, qualified_path, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
				ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET
					node_name = EXCLUDED.node_name,
					properties = EXCLUDED.properties,
					updated_at = NOW()
			`, fNodeID, tenant.ID, f.Name, fmt.Sprintf("%s field for %s", f.Name, boKey), boFieldNodeTypeID, fProps, fQPath)
			if err != nil {
				log.Printf("  Error inserting bo_field %s: %v", f.Key, err)
			}

			// Graph Edges:
			// 1. BO --HAS_FIELD--> BOField
			if hasFieldEdgeTypeID != "" {
				edgeID := uuid.NewSHA1(ns, []byte(fmt.Sprintf("edge:hasfield:%s:%s:%s", tenant.ID, boNodeID, fNodeID))).String()
				_, _ = db.Exec(`
					INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, '{}', NOW(), NOW())
					ON CONFLICT DO NOTHING
				`, edgeID, tenant.ID, boNodeID, fNodeID, hasFieldEdgeTypeID)
			}

			// 2. BOField --BACKED_BY_TERM--> SemanticTerm
			if backedByTermEdgeTypeID != "" && termID != "" {
				edgeID := uuid.NewSHA1(ns, []byte(fmt.Sprintf("edge:backedby:%s:%s:%s", tenant.ID, fNodeID, termID))).String()
				_, _ = db.Exec(`
					INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, '{}', NOW(), NOW())
					ON CONFLICT DO NOTHING
				`, edgeID, tenant.ID, fNodeID, termID, backedByTermEdgeTypeID)
			}

			// 3. BO --USES_SEMANTIC_TERM--> SemanticTerm
			if usesTermEdgeTypeID != "" && termID != "" {
				edgeID := uuid.NewSHA1(ns, []byte(fmt.Sprintf("edge:usesterm:%s:%s:%s", tenant.ID, boNodeID, termID))).String()
				_, _ = db.Exec(`
					INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, '{}', NOW(), NOW())
					ON CONFLICT DO NOTHING
				`, edgeID, tenant.ID, boNodeID, termID, usesTermEdgeTypeID)
			}

			boFieldJsonList = append(boFieldJsonList, map[string]interface{}{
				"id":             fNodeID,
				"name":           f.Key,
				"displayName":    f.Name,
				"semanticTermId": termID,
				"dataType":       "number",
			})
		}

		// 3D. Register in structural `business_objects` table
		fieldsJson, _ := json.Marshal(boFieldJsonList)
		configJson, _ := json.Marshal(map[string]interface{}{"is_core": tenant.GoldCopy})

		var boExists int
		_ = db.QueryRow("SELECT COUNT(*) FROM business_objects WHERE tenant_id = $1 AND key = $2", tenant.ID, boKey).Scan(&boExists)
		if boExists > 0 {
			_, err = db.Exec(`
				UPDATE business_objects
				SET name = $1, display_name = $2, description = $3, fields = $4, config = $5, last_modified_at = NOW()
				WHERE tenant_id = $6 AND key = $7
			`, "Alternative Investment", "Alternative Investment", "Alternative investment master holding with live XIRR calculation engine",
				fieldsJson, configJson, tenant.ID, boKey)
		} else {
			_, err = db.Exec(`
				INSERT INTO business_objects (
					id, tenant_id, key, name, display_name, technical_name,
					description, icon, is_core, category, driver_table_name,
					created_at, last_modified_at, is_active, config, fields
				) VALUES (
					$1, $2, $3, $4, $5, $6,
					$7, $8, $9, $10, $11,
					NOW(), NOW(), true, $12, $13
				)
			`, boNodeID, tenant.ID, boKey, "Alternative Investment", "Alternative Investment", "alternative_investments",
				"Alternative investment master holding with live XIRR calculation engine", "briefcase", tenant.GoldCopy, "Private Markets", "alternative_investments",
				configJson, fieldsJson)
		}

		if err != nil {
			log.Printf("  Error upserting business_objects row: %v", err)
		} else {
			log.Printf("  ✓ Upserted business_objects structural row: %s", boKey)
		}
	}

	// 4. Seed Live Demo Alternative Investment Data (Fund, Capital Calls, Distributions)
	log.Println("--- Seeding Live Demo Investment Data ---")

	// Ensure Client exists
	clientID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	_, err = db.Exec(`
		INSERT INTO clients (id, client_name, client_code, client_type, is_active)
		VALUES ($1, 'Apex Family Office', 'APEX-001', 'family_office', true)
		ON CONFLICT (id) DO NOTHING
	`, clientID)
	if err != nil {
		log.Printf("Warning inserting client: %v", err)
	}

	demoInvID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	vintageYear := 2021
	generalPartner := "Blackstone Alternative Asset Management"
	valSource := "GP_REPORTED"

	_, err = db.Exec(`
		INSERT INTO alternative_investments (
			investment_id, client_id, investment_type, fund_name, general_partner,
			vintage_year, total_commitment_amount, unfunded_commitment, total_capital_called,
			total_distributions, current_nav, nav_date, valuation_source, created_at, updated_at
		) VALUES (
			$1, $2, 'PRIVATE_EQUITY', 'Blackstone Real Estate Partners X', $3,
			$4, 5000000.00, 2000000.00, 3000000.00,
			950000.00, 3850000.00, '2024-12-31', $5, NOW(), NOW()
		)
		ON CONFLICT (investment_id) DO UPDATE SET
			fund_name = EXCLUDED.fund_name,
			current_nav = EXCLUDED.current_nav,
			nav_date = EXCLUDED.nav_date,
			valuation_source = EXCLUDED.valuation_source,
			updated_at = NOW()
	`, demoInvID, clientID, generalPartner, vintageYear, valSource)
	if err != nil {
		log.Fatalf("Failed to insert demo investment: %v", err)
	}
	log.Printf("✓ Seeded Alternative Investment: Blackstone Real Estate Partners X (%s)", demoInvID)

	// Clean existing test calls/distributions for idempotency
	_, _ = db.Exec("DELETE FROM capital_calls WHERE investment_id = $1", demoInvID)
	_, _ = db.Exec("DELETE FROM distributions WHERE investment_id = $1", demoInvID)

	// Insert realistic capital calls
	calls := []struct {
		NoticeDate string
		DueDate    string
		Amount     float64
		Status     string
	}{
		{"2021-03-15", "2021-03-31", 1500000.00, "FUNDED"},
		{"2021-10-01", "2021-10-15", 1000000.00, "FUNDED"},
		{"2022-06-01", "2022-06-15", 500000.00, "FUNDED"},
	}

	for _, c := range calls {
		callID := uuid.New()
		noticeDate, _ := time.Parse("2006-01-02", c.NoticeDate)
		dueDate, _ := time.Parse("2006-01-02", c.DueDate)

		_, err := db.Exec(`
			INSERT INTO capital_calls (
				call_id, investment_id, notice_date, due_date, amount_requested,
				amount_funded, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $5, $6, NOW(), NOW())
		`, callID, demoInvID, noticeDate, dueDate, c.Amount, c.Status)
		if err != nil {
			log.Printf("Error inserting capital call: %v", err)
		} else {
			log.Printf("  ✓ Capital Call: %s for $%.2f", c.DueDate, c.Amount)
		}
	}

	// Insert realistic distributions
	distributions := []struct {
		Date   string
		Amount float64
		Type   string
	}{
		{"2023-03-31", 350000.00, "INCOME"},
		{"2023-11-15", 600000.00, "CAPITAL_GAIN"},
	}

	for _, d := range distributions {
		distID := uuid.New()
		distDate, _ := time.Parse("2006-01-02", d.Date)

		_, err := db.Exec(`
			INSERT INTO distributions (
				distribution_id, investment_id, distribution_date, amount,
				distribution_type, reinvested, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, false, NOW(), NOW())
		`, distID, demoInvID, distDate, d.Amount, d.Type)
		if err != nil {
			log.Printf("Error inserting distribution: %v", err)
		} else {
			log.Printf("  ✓ Distribution: %s for $%.2f", d.Date, d.Amount)
		}
	}

	log.Println("\n=======================================================")
	log.Println("✓ XIRR Catalog Seeding and Demo Setup Completed Successfully!")
	log.Println("=======================================================")
}
