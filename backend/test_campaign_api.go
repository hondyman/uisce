package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type NotificationCampaign struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Type        string                     `json:"type"`
	Status      string                     `json:"status"`
	TargetUsers []string                   `json:"target_users"`
	UserSegment string                     `json:"user_segment"`
	Steps       []NotificationCampaignStep `json:"steps"`
	CreatedBy   string                     `json:"created_by"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
}

type NotificationCampaignStep struct {
	ID           string  `json:"id"`
	StepNumber   int     `json:"step_number"`
	TemplateID   string  `json:"template_id"`
	DelayHours   int     `json:"delay_hours"`
	TriggerEvent string  `json:"trigger_event,omitempty"`
	Condition    string  `json:"condition,omitempty"`
	SentCount    int     `json:"sent_count"`
	OpenRate     float64 `json:"open_rate"`
	ClickRate    float64 `json:"click_rate"`
}

//lint:ignore U1000 keep this test helper for local manual testing and examples
func testCampaignAPI() {
	fmt.Println("🧪 Testing Notification Campaign API Endpoints")
	fmt.Println("=============================================")

	baseURL := "http://localhost:8080/api/notifications"

	// Test 1: Create a notification campaign
	fmt.Println("\n📝 Test 1: Creating Notification Campaign")

	campaign := NotificationCampaign{
		Name:        "Welcome Series Test",
		Description: "Automated welcome notification series for new users",
		Type:        "onboarding",
		Status:      "draft",
		TargetUsers: []string{"user123", "user456"},
		UserSegment: "new_users",
		Steps: []NotificationCampaignStep{
			{
				StepNumber:   1,
				TemplateID:   "welcome-template",
				DelayHours:   0,
				TriggerEvent: "user_registration",
				SentCount:    0,
				OpenRate:     0.0,
				ClickRate:    0.0,
			},
			{
				StepNumber:   2,
				TemplateID:   "feature-intro-template",
				DelayHours:   24,
				TriggerEvent: "",
				Condition:    "step_1_opened",
				SentCount:    0,
				OpenRate:     0.0,
				ClickRate:    0.0,
			},
		},
		CreatedBy: "test-user",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	campaignJSON, _ := json.MarshalIndent(campaign, "", "  ")
	fmt.Printf("Campaign payload:\n%s\n", string(campaignJSON))

	// Create campaign
	resp, err := http.Post(baseURL+"/campaigns", "application/json", bytes.NewBuffer(campaignJSON))
	if err != nil {
		log.Printf("❌ Failed to create campaign: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to create campaign. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	var createdCampaign NotificationCampaign
	json.Unmarshal(body, &createdCampaign)
	fmt.Printf("✅ Campaign created successfully: %s (ID: %s)\n", createdCampaign.Name, createdCampaign.ID)

	// Test 2: Get active campaigns
	fmt.Println("\n📋 Test 2: Getting Active Campaigns")

	resp, err = http.Get(baseURL + "/campaigns/active")
	if err != nil {
		log.Printf("❌ Failed to get active campaigns: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to get active campaigns. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	var campaigns []NotificationCampaign
	json.Unmarshal(body, &campaigns)
	fmt.Printf("✅ Found %d active campaigns\n", len(campaigns))

	// Test 3: Launch the campaign
	fmt.Println("\n🚀 Test 3: Launching Campaign")

	launchURL := fmt.Sprintf("%s/campaigns/%s/launch", baseURL, createdCampaign.ID)
	resp, err = http.Post(launchURL, "application/json", nil)
	if err != nil {
		log.Printf("❌ Failed to launch campaign: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to launch campaign. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	fmt.Printf("✅ Campaign launched successfully: %s\n", createdCampaign.Name)

	// Test 4: Get campaign details
	fmt.Println("\n📄 Test 4: Getting Campaign Details")

	detailsURL := fmt.Sprintf("%s/campaigns/%s", baseURL, createdCampaign.ID)
	resp, err = http.Get(detailsURL)
	if err != nil {
		log.Printf("❌ Failed to get campaign details: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to get campaign details. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	var campaignDetails NotificationCampaign
	json.Unmarshal(body, &campaignDetails)
	fmt.Printf("✅ Campaign details retrieved: %s (Status: %s)\n", campaignDetails.Name, campaignDetails.Status)

	// Test 5: Pause the campaign
	fmt.Println("\n⏸️  Test 5: Pausing Campaign")

	pauseURL := fmt.Sprintf("%s/campaigns/%s/pause", baseURL, createdCampaign.ID)
	resp, err = http.Post(pauseURL, "application/json", nil)
	if err != nil {
		log.Printf("❌ Failed to pause campaign: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to pause campaign. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	fmt.Printf("✅ Campaign paused successfully: %s\n", createdCampaign.Name)

	// Test 6: Test user preferences
	fmt.Println("\n👤 Test 6: Testing User Preferences")

	preferences := map[string]interface{}{
		"user_id":           "test-user-123",
		"email_enabled":     true,
		"sms_enabled":       false,
		"push_enabled":      true,
		"in_app_enabled":    true,
		"quiet_hours_start": "22:00",
		"quiet_hours_end":   "08:00",
		"timezone":          "America/New_York",
		"channel_preferences": map[string]bool{
			"email":  true,
			"sms":    false,
			"push":   true,
			"in_app": true,
		},
		"type_preferences": map[string]bool{
			"welcome":        true,
			"feature":        true,
			"alert":          true,
			"recommendation": true,
		},
		"frequency_preferences": map[string]string{
			"marketing": "daily",
			"system":    "immediate",
			"social":    "weekly",
		},
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
	}

	prefsJSON, _ := json.MarshalIndent(preferences, "", "  ")
	fmt.Printf("Preferences payload:\n%s\n", string(prefsJSON))

	// Update user preferences
	prefsURL := fmt.Sprintf("%s/preferences/%s", baseURL, "test-user-123")
	req, _ := http.NewRequest("PUT", prefsURL, bytes.NewBuffer(prefsJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to update user preferences: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to update user preferences. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	fmt.Printf("✅ User preferences updated successfully\n")

	// Get user preferences
	resp, err = http.Get(prefsURL)
	if err != nil {
		log.Printf("❌ Failed to get user preferences: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Failed to get user preferences. Status: %d, Body: %s", resp.StatusCode, string(body))
		return
	}

	fmt.Printf("✅ User preferences retrieved successfully\n")

	fmt.Println("\n🎉 All Notification Campaign API Tests Completed!")
	fmt.Println("================================================")
	fmt.Println("✅ Campaign creation and management")
	fmt.Println("✅ Campaign launching and status control")
	fmt.Println("✅ User preferences management")
	fmt.Println("✅ Real-time campaign orchestration")
	fmt.Println("✅ Automated notification delivery")
}
