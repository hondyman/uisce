package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/config"
	_ "github.com/lib/pq"
)

func verifyEngagementNotifications() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database successfully")
	fmt.Println("Verifying engagement notifications tables...")

	// Check if tables exist
	tables := []string{
		"engagement_notifications",
		"notification_templates",
		"user_notification_preferences",
		"notification_campaigns",
		"notification_analytics",
		"user_engagement_profiles",
	}

	for _, table := range tables {
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '%s')", table)
		err := db.QueryRow(query).Scan(&exists)
		if err != nil {
			log.Printf("Error checking table %s: %v", table, err)
			continue
		}

		if exists {
			fmt.Printf("✅ Table '%s' exists\n", table)
		} else {
			fmt.Printf("❌ Table '%s' does not exist\n", table)
		}
	}

	// Check indexes
	fmt.Println("\nVerifying indexes...")
	indexes := []string{
		"idx_engagement_notifications_user_id",
		"idx_engagement_notifications_status",
		"idx_engagement_notifications_type",
		"idx_notification_templates_type",
		"idx_notification_campaigns_status",
		"idx_user_engagement_profiles_segment",
	}

	for _, index := range indexes {
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = '%s')", index)
		err := db.QueryRow(query).Scan(&exists)
		if err != nil {
			log.Printf("Error checking index %s: %v", index, err)
			continue
		}

		if exists {
			fmt.Printf("✅ Index '%s' exists\n", index)
		} else {
			fmt.Printf("❌ Index '%s' does not exist\n", index)
		}
	}

	// Check triggers
	fmt.Println("\nVerifying triggers...")
	triggers := []string{
		"update_engagement_notifications_updated_at",
		"update_notification_templates_updated_at",
		"update_user_notification_preferences_updated_at",
		"update_notification_campaigns_updated_at",
		"update_user_engagement_profiles_updated_at",
	}

	for _, trigger := range triggers {
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = '%s')", trigger)
		err := db.QueryRow(query).Scan(&exists)
		if err != nil {
			log.Printf("Error checking trigger %s: %v", trigger, err)
			continue
		}

		if exists {
			fmt.Printf("✅ Trigger '%s' exists\n", trigger)
		} else {
			fmt.Printf("❌ Trigger '%s' does not exist\n", trigger)
		}
	}

	fmt.Println("\nEngagement notifications schema verification completed!")
}

func RunVerifyEngagementNotifications() {
	verifyEngagementNotifications()
}
