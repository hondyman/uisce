// migrate is the canonical, standalone entry point for the migration runner
// in backend/internal/migrations. It exists so migrations can be applied,
// inspected, or verified without booting the full server (which currently
// runs ApplyMigrations itself on every start, via internal/api/api.go).
//
// Usage:
//
//	DATABASE_URL=... migrate up      # apply pending migrations (same as server boot)
//	DATABASE_URL=... migrate status  # list every migration file: applied/pending
//	DATABASE_URL=... migrate verify  # recompute checksums of applied files, report drift
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/migrations"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	switch os.Args[1] {
	case "up":
		runUp(db)
	case "status":
		runStatus(db)
	case "verify":
		runVerify(db)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate <up|status|verify>")
}

func runUp(db *sql.DB) {
	if err := migrations.ApplyMigrations(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
}

func runStatus(db *sql.DB) {
	entries, err := migrations.Status(db)
	if err != nil {
		log.Fatalf("status failed: %v", err)
	}

	pending := 0
	for _, e := range entries {
		if e.Applied {
			fmt.Printf("APPLIED  %s  (%s)\n", e.Filename, e.AppliedAt.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("PENDING  %s\n", e.Filename)
			pending++
		}
	}
	fmt.Printf("\n%d applied, %d pending\n", len(entries)-pending, pending)
}

func runVerify(db *sql.DB) {
	drift, err := migrations.Verify(db)
	if err != nil {
		log.Fatalf("verify failed: %v", err)
	}

	if len(drift) == 0 {
		fmt.Println("no drift: every applied migration's on-disk content matches its recorded checksum")
		return
	}

	for _, d := range drift {
		if d.CurrentSHA == "" {
			fmt.Printf("MISSING  %s  applied %s, recorded sha256=%s, file no longer on disk\n",
				d.Filename, d.AppliedAt.Format("2006-01-02 15:04:05"), d.RecordedSHA)
		} else {
			fmt.Printf("DRIFT    %s  applied %s, recorded sha256=%s, current sha256=%s\n",
				d.Filename, d.AppliedAt.Format("2006-01-02 15:04:05"), d.RecordedSHA, d.CurrentSHA)
		}
	}
	fmt.Fprintf(os.Stderr, "\n%d migration(s) with drift\n", len(drift))
	os.Exit(1)
}
