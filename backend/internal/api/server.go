package api

import (
	"log"
	"net/http"
	"os"

	"github.com/hondyman/semlayer/backend/internal/services"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func StartServer() {
	log.Println("Initializing schema validator...")
	// TODO: Add schema validation initialization
	// if err := validate.Init(); err != nil {
	//     log.Fatalf("FATAL: Failed to initialize schema validator: %v", err)
	// }

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Println("WARN: POSTGRES_DSN not set, using default.")
		dsn = "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}

	// Initialize temporal client with retry logic
	temporalC, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("FATAL: Failed to create temporal client: %v", err)
	}
	defer temporalC.Close()

	// Initialize QoSManager
	qosManager := services.NewQoSManager(db)

	router := SetupRouter(db.DB, nil, nil, temporalC, qosManager, nil, nil, nil, nil)
	log.Println("Server listening on :8080")
	http.ListenAndServe(":8080", router)
}
