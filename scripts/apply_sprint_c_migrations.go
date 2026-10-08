package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

func getAlphaDSN() string {
	dsn := os.Getenv("ALPHA_DSN")
	if dsn == "" {
		home, _ := os.UserHomeDir()
		caPath := filepath.Join(home, ".uisce/certs/ca.crt")
		certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
		keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

		if _, err := os.Stat(caPath); err == nil {
			dsn = fmt.Sprintf("host=100.84.50.65 port=5432 user=postgres password=postgres dbname=alpha sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", caPath, certPath, keyPath)
		}
	}
	return dsn
}

func main() {
	dsn := getAlphaDSN()
	if dsn == "" {
		log.Fatal("ALPHA_DSN not configured and certs not found")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tabsJSON := `[{"id":"library","label":"Core Rule Library (94 Rules)","layout":{"root":"lib_root","nodes":{"lib_root":{"id":"lib_root","type":"Column","children":["lib_widget"],"style":{"gap":"16px"}}}}},{"id":"matrix","label":"Tenant Rule Matrix","layout":{"root":"matrix_root","nodes":{"matrix_root":{"id":"matrix_root","type":"Column","children":["matrix_widget"],"style":{"gap":"16px"}}}}},{"id":"blotter","label":"Pre-Trade Blotter","layout":{"root":"blotter_root","nodes":{"blotter_root":{"id":"blotter_root","type":"Column","children":["blotter_widget"],"style":{"gap":"16px"}}}}},{"id":"surveillance","label":"Surveillance & Exceptions","layout":{"root":"surv_root","nodes":{"surv_root":{"id":"surv_root","type":"Column","children":["surv_widget"],"style":{"gap":"16px"}}}}},{"id":"regulatory","label":"Regulatory Triage","layout":{"root":"reg_root","nodes":{"reg_root":{"id":"reg_root","type":"Column","children":["reg_widget"],"style":{"gap":"16px"}}}}},{"id":"calendar","label":"Compliance Calendar","layout":{"root":"cal_root","nodes":{"cal_root":{"id":"cal_root","type":"Column","children":["cal_widget"],"style":{"gap":"16px"}}}}},{"id":"limits","label":"Limit Utilization","layout":{"root":"limits_root","nodes":{"limits_root":{"id":"limits_root","type":"Column","children":["limits_widget"],"style":{"gap":"16px"}}}}}]`
	componentsJSON := `{"hdr":{"id":"hdr","type":"PageHeader","props":{"icon":"security","title":"Compliance & Exception Governance","subtitle":"Unified console: 94-rule core library, tenant activation matrix, pre-trade blotter, and surveillance finding queues"},"style":{"flex":"1 1 320px"}},"lib_widget":{"id":"lib_widget","type":"DomainComponent","props":{"component":"compliance.RuleLibraryExplorer","inputs":{}}},"matrix_widget":{"id":"matrix_widget","type":"DomainComponent","props":{"component":"compliance.RuleActivationMatrix","inputs":{}}},"blotter_widget":{"id":"blotter_widget","type":"DomainComponent","props":{"component":"compliance.DecisionBlotter","inputs":{}}},"surv_widget":{"id":"surv_widget","type":"DomainComponent","props":{"component":"compliance.SurveillanceQueue","inputs":{}}},"reg_widget":{"id":"reg_widget","type":"DomainComponent","props":{"component":"compliance.RegulatoryQueue","inputs":{}}},"cal_widget":{"id":"cal_widget","type":"DomainComponent","props":{"component":"compliance.Calendar","inputs":{}}},"limits_widget":{"id":"limits_widget","type":"DomainComponent","props":{"component":"compliance.LimitDashboard","inputs":{}}}}`
	appModelJSON := `{"chrome":"none","surface":{"maxWidth":1600,"padding":3},"tabVariable":"tab","variables":[{"name":"tab","default":"library","url":true}],"queries":[]}`

	_, err = db.ExecContext(ctx, `
		INSERT INTO public.page_definitions (
			id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
			presentation_events, filter_bar, app_model, version, is_core, status, updated_at
		) VALUES (
			'018f9d02-0001-7000-8000-000000000500',
			'99e99e99-99e9-49e9-89e9-99e99e99e999',
			'Compliance & Exception Governance',
			'compliance-hub',
			'Unified compliance console: 94 core rules, tenant overrides, pre-trade blotter, post-trade exception surveillance, regulatory change queue, compliance calendar, and limit utilization.',
			'{"root":"root","nodes":{"root":{"id":"root","type":"Column","children":["hdr"],"style":{"gap":"16px"}}}}'::jsonb,
			$1::jsonb,
			$2::jsonb,
			'[]'::jsonb,
			'[]'::jsonb,
			'{}'::jsonb,
			$3::jsonb,
			1,
			true,
			'published',
			NOW()
		)
		ON CONFLICT (tenant_id, slug) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			layout = EXCLUDED.layout,
			tabs = EXCLUDED.tabs,
			components = EXCLUDED.components,
			data_sources = EXCLUDED.data_sources,
			presentation_events = EXCLUDED.presentation_events,
			filter_bar = EXCLUDED.filter_bar,
			app_model = EXCLUDED.app_model,
			version = EXCLUDED.version,
			is_core = EXCLUDED.is_core,
			status = EXCLUDED.status,
			updated_at = NOW();
	`, tabsJSON, componentsJSON, appModelJSON)
	content, err := os.ReadFile("backend/db/migrations/20261224_022_compliance_statutory_calendar_schedule.up.sql")
	if err != nil {
		log.Fatalf("Read file 022 failed: %v", err)
	}
	_, err = db.ExecContext(ctx, string(content))
	if err != nil {
		log.Fatalf("Exec migration 022 failed: %v", err)
	}
	fmt.Println("Successfully applied migration 20261224_022_compliance_statutory_calendar_schedule.up.sql!")
}
