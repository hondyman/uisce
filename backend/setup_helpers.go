package backend

import (
	"log"
	"os"

	"github.com/lib/pq"
)

// CreateDB connects to postgres and creates the alpha database if it doesn't exist.
func CreateDB() error {
	// Connect to postgres database to create alpha
	db, err := GetAppDBConnection("postgres")
	if err != nil {
		return err
	}
	defer db.Close()

	// Create database if it doesn't exist
	_, err = db.Exec("CREATE DATABASE alpha")
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P04" { // 42P04 is duplicate_database
			log.Println("Info: Database 'alpha' already exists, skipping creation.")
			return nil // Not an error in our case
		}
		return err
	}

	log.Println("Database 'alpha' created successfully.")
	return nil
}

// CheckTables lists all tables in the public schema.
func CheckTables() error {
	// Connect to database
	db, err := GetAppDBConnection("alpha")
	if err != nil {
		return err
	}
	defer db.Close()

	// Check what tables exist
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		return err
	}
	defer rows.Close()

	log.Println("Existing tables:")
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		log.Println("-", tableName)
	}
	return rows.Err()
}

// CheckPolicyTables lists all tables related to policies.
func CheckPolicyTables() error {
	// Connect to database
	db, err := GetAppDBConnection("alpha")
	if err != nil {
		return err
	}
	defer db.Close()

	// Check for policy-related tables
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE '%polic%'")
	if err != nil {
		return err
	}
	defer rows.Close()

	log.Println("Policy-related tables:")
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		log.Println("-", tableName)
	}
	return rows.Err()
}

// CreatePoliciesTable creates the policies table in the database.
func CreatePoliciesTable() error {
	// Connect to database
	db, err := GetAppDBConnection("alpha")
	if err != nil {
		return err
	}
	defer db.Close()

	// Read and execute SQL file
	// Note: paths are relative to CWD. If running from root, "migrations/..." works.
	content, err := os.ReadFile("migrations/0004_create_policies.sql")
	if err != nil {
		return err
	}

	// Execute SQL
	if _, err = db.Exec(string(content)); err != nil {
		return err
	}

	log.Println("Policies table created successfully!")
	return nil
}
