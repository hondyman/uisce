package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Notification struct {
	ID              string
	TenantID        string
	DatasourceID    string
	TemplateKey     string
	RecipientUserID string
	Subject         string
	Body            string
	Channel         string
	Status          string
	Priority        string
	SentAt          time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func main() {
	// Connect to database
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=alpha sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	tenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"
	datasourceID := "982aef38-418f-46dc-acd0-35fe8f3b97b0"

	// Lookup User ID for admin@example.com
	var userID string
	err = db.Get(&userID, "SELECT id FROM users WHERE email = $1", "admin@example.com")
	if err != nil {
		log.Printf("Failed to find user 'admin@example.com', falling back to hardcoded ID or creating default user... Error: %v", err)
		// Fallback to a hardcoded ID just in case, or handle error
		userID = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	} else {
		fmt.Printf("found user: 'admin@example.com' with ID: %s\n", userID)
	}

	notifications := []Notification{
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Action Required: Approve Expense Report",
			Body:            "John Smith has submitted an expense report for Q3 that requires your approval. Total amount: $4,250.00. Please review and approve by end of day.",
			Channel:         "email",
			Status:          "sent",
			Priority:        "urgent",
			SentAt:          time.Now().Add(-2 * time.Hour),
			CreatedAt:       time.Now().Add(-2 * time.Hour),
			UpdatedAt:       time.Now().Add(-2 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Critical: Workflow 'Q4 Review' Failed",
			Body:            "Step 'Data Validation' returned a critical error. Immediate action required to prevent delays in the quarterly review process.",
			Channel:         "slack",
			Status:          "sent",
			Priority:        "urgent",
			SentAt:          time.Now().Add(-30 * time.Minute),
			CreatedAt:       time.Now().Add(-30 * time.Minute),
			UpdatedAt:       time.Now().Add(-30 * time.Minute),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "New feature deployed to production",
			Body:            "The new dashboard analytics feature has been successfully deployed. Please review the changes and provide feedback on the new metrics visualization.",
			Channel:         "slack",
			Status:          "sent",
			Priority:        "high",
			SentAt:          time.Now().Add(-8 * time.Hour),
			CreatedAt:       time.Now().Add(-8 * time.Hour),
			UpdatedAt:       time.Now().Add(-8 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Budget Approval Needed",
			Body:            "The Q1 2024 marketing budget proposal is ready for your review. Total requested: $125,000. Deadline: Friday EOD.",
			Channel:         "email",
			Status:          "sent",
			Priority:        "high",
			SentAt:          time.Now().Add(-24 * time.Hour),
			CreatedAt:       time.Now().Add(-24 * time.Hour),
			UpdatedAt:       time.Now().Add(-24 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Security Alert: Unusual Login Activity",
			Body:            "We detected a login from an unrecognized device in San Francisco, CA. If this was not you, please secure your account immediately.",
			Channel:         "sms",
			Status:          "sent",
			Priority:        "high",
			SentAt:          time.Now().Add(-3 * time.Hour),
			CreatedAt:       time.Now().Add(-3 * time.Hour),
			UpdatedAt:       time.Now().Add(-3 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Weekly Team Sync Reminder",
			Body:            "Reminder: The weekly sync is scheduled for tomorrow at 10:00 AM PST. Please add your updates to the agenda document.",
			Channel:         "teams",
			Status:          "sent",
			Priority:        "normal",
			SentAt:          time.Now().Add(-24 * time.Hour),
			CreatedAt:       time.Now().Add(-24 * time.Hour),
			UpdatedAt:       time.Now().Add(-24 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "New Task Assigned: Review User Feedback",
			Body:            "You have been assigned to review user feedback for the mobile app redesign. 47 responses are waiting for your analysis.",
			Channel:         "email",
			Status:          "sent",
			Priority:        "normal",
			SentAt:          time.Now().Add(-48 * time.Hour),
			CreatedAt:       time.Now().Add(-48 * time.Hour),
			UpdatedAt:       time.Now().Add(-48 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Document Shared: Q4 Planning",
			Body:            "Sarah Johnson shared 'Q4 Strategic Planning.pdf' with you. The document contains the roadmap for next quarter's initiatives.",
			Channel:         "email",
			Status:          "sent",
			Priority:        "normal",
			SentAt:          time.Now().Add(-12 * time.Hour),
			CreatedAt:       time.Now().Add(-12 * time.Hour),
			UpdatedAt:       time.Now().Add(-12 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "System Maintenance Scheduled",
			Body:            "Scheduled maintenance will occur this Saturday from 2 AM to 4 AM. Services may be temporarily unavailable during this window.",
			Channel:         "push",
			Status:          "sent",
			Priority:        "low",
			SentAt:          time.Now().Add(-72 * time.Hour),
			CreatedAt:       time.Now().Add(-72 * time.Hour),
			UpdatedAt:       time.Now().Add(-72 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Workflow 'Daily Backup' Completed",
			Body:            "The daily database backup workflow ran successfully. All data has been backed up to the secure storage location.",
			Channel:         "email",
			Status:          "sent",
			Priority:        "low",
			SentAt:          time.Now().Add(-24 * time.Hour),
			CreatedAt:       time.Now().Add(-24 * time.Hour),
			UpdatedAt:       time.Now().Add(-24 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "New Feature Available: Dark Mode",
			Body:            "We've added dark mode to the notification center! Click the sun/moon icon in the top right to try it out.",
			Channel:         "push",
			Status:          "sent",
			Priority:        "low",
			SentAt:          time.Now().Add(-4 * time.Hour),
			CreatedAt:       time.Now().Add(-4 * time.Hour),
			UpdatedAt:       time.Now().Add(-4 * time.Hour),
		},
		{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			DatasourceID:    datasourceID,
			TemplateKey:     "manual_notification",
			RecipientUserID: userID,
			Subject:         "Monthly Report Ready",
			Body:            "Your monthly activity report for December is now available. View insights on your productivity and team collaboration.",
			Channel:         "email",
			Status:          "sent",
			Priority:        "low",
			SentAt:          time.Now().Add(-48 * time.Hour),
			CreatedAt:       time.Now().Add(-48 * time.Hour),
			UpdatedAt:       time.Now().Add(-48 * time.Hour),
		},
	}

	fmt.Println("🌱 Seeding mock notification data...")
	fmt.Printf("Tenant: %s\n", tenantID)
	fmt.Printf("User: %s\n\n", userID)

	query := `INSERT INTO notification_logs (
		id, tenant_id, datasource_id, template_key, recipient_user_id,
		subject, body, channel, status, priority, sent_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	for i, notif := range notifications {
		_, err := db.Exec(query,
			notif.ID, notif.TenantID, notif.DatasourceID, notif.TemplateKey,
			notif.RecipientUserID, notif.Subject, notif.Body, notif.Channel,
			notif.Status, notif.Priority, notif.SentAt, notif.CreatedAt, notif.UpdatedAt)

		if err != nil {
			log.Printf("Failed to insert notification %d: %v", i+1, err)
		} else {
			fmt.Printf("✓ Created: %s (%s - %s)\n", notif.Subject, notif.Priority, notif.Channel)
		}
	}

	fmt.Println("\n✅ Successfully created 12 mock notifications!")
	fmt.Println("\nTo view them, open: http://localhost:5173/core/notifications")
	fmt.Println("\nTest features:")
	fmt.Println("  • Toggle dark mode (sun/moon icon)")
	fmt.Println("  • Open filters and try search")
	fmt.Println("  • Select notifications with checkboxes")
	fmt.Println("  • Click 'Approve' or 'Reject' buttons")
	fmt.Println("  • Mark notifications as read")
}
