// One-off regression check: does GenerateDDL still resolve physical
// columns for execution_npv_rollup after execution's driver_table_name
// was repointed from /orm/execution to /oms/execution?
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	svc := analytics.NewPreAggregationService(db, nil, nil)
	ddl, err := svc.GenerateDDL(context.Background(), uuid.MustParse("5217fdac-ae21-461a-8223-7d65bb23d707"), "starrocks")
	if err != nil {
		log.Fatalf("GenerateDDL failed: %v", err)
	}
	fmt.Println(ddl)
}
