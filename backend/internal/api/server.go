package api

import (
	"log"
	"net/http"
	"os"

	"github.com/hondyman/semlayer/backend/internal/services"
	temporalclientlib "github.com/hondyman/semlayer/libs/temporal-client"
	temporalclient "go.temporal.io/sdk/client"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func StartServer() {
	log.Println("Initializing schema validator...")
	// TODO: Add schema validation initialization
	// if err := validate.Init(); err != nil {
	//     log.Fatalf("FATAL: Failed to initialize schema validator: %v", err)
	// }

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		log.Println("WARN: DATABASE_URL/POSTGRES_DSN not set, using default.")
		dsn = "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}

	// Initialize temporal client with retry logic; allow startup to continue
	// when Temporal is unreachable in local dev.
	var temporalC temporalclient.Client
	temporalC, err = temporalclientlib.NewClientWithRetry()
	if err != nil {
		log.Printf("WARN: Failed to create temporal client: %v. Continuing without Temporal.", err)
		temporalC = nil
	} else {
		defer temporalC.Close()
	}

	// Initialize QoSManager
	qosManager := services.NewQoSManager(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	router := SetupRouter(db.DB, nil, nil, temporalC, qosManager, nil, nil, nil, nil)
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("FATAL: Server failed: %v", err)
	}
}
