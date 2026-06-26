//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"calendar-service/internal/api"
	"calendar-service/internal/repository"
	"calendar-service/internal/services"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProfileManagement_E2E tests the complete profile management flow
func TestProfileManagement_E2E(t *testing.T) {
	ctx := context.Background()

	// Setup logging
	logger := logrus.New()
	logEntry := logger.WithField("test", "profile_management")

	// Setup dependencies
	calendarRepo := repository.NewInMemoryCalendarRepository(logEntry)
	repoAdapter := services.NewRepositoryAdapter(calendarRepo, logEntry)
	auditService := services.NewAuditService(logEntry)
	profileService := services.NewProfileService(repoAdapter, auditService, logEntry)
	profileHandler := api.NewProfileHandler(profileService, auditService, logEntry)

	tenantID := "test-tenant-001"
	userID := "test-user-001"

	t.Run("CreateProfile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", nil)
		req.Header.Set("X-Hasura-Tenant-Id", tenantID)
		req.Header.Set("X-Hasura-User-Id", userID)

		reqBody, err := json.Marshal(map[string]interface{}{
			"profile_name":        "US-Core",
			"description":         "US Core Operations Calendar",
			"calendars":           []string{"cal-1", "cal-2"},
			"conflict_resolution": "union",
			"timezone":            "America/New_York",
		})
		require.NoError(t, err)

		req.Body = io.NopCloser(strings.NewReader(string(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		profileHandler.Create(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var profile services.ScheduleProfile
		err = json.NewDecoder(w.Body).Decode(&profile)
		require.NoError(t, err)

		assert.Equal(t, "US-Core", profile.ProfileName)
		assert.Equal(t, "US Core Operations Calendar", profile.Description)
		assert.Equal(t, "America/New_York", profile.Timezone)
		assert.Equal(t, "union", profile.ConflictResolution)
		assert.True(t, profile.Active)
		assert.Nil(t, profile.ValidTo)
		assert.NotEmpty(t, profile.ID)
	})

	t.Run("ListProfiles", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles?limit=10&offset=0", nil)
		req.Header.Set("X-Hasura-Tenant-Id", tenantID)

		w := httptest.NewRecorder()
		profileHandler.List(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var profiles []services.ScheduleProfile
		err := json.NewDecoder(w.Body).Decode(&profiles)
		require.NoError(t, err)

		assert.Greater(t, len(profiles), 0)
	})

	t.Run("GetProfile", func(t *testing.T) {
		// First create a profile to get its ID
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", nil)
		reqCreate.Header.Set("X-Hasura-Tenant-Id", tenantID)
		reqCreate.Header.Set("X-Hasura-User-Id", userID)

		reqBody, _ := json.Marshal(map[string]interface{}{
			"profile_name":        "EU-Finance",
			"calendars":           []string{"cal-1"},
			"conflict_resolution": "intersection",
			"timezone":            "Europe/London",
		})

		reqCreate.Body = io.NopCloser(strings.NewReader(string(reqBody)))
		reqCreate.Header.Set("Content-Type", "application/json")

		wCreate := httptest.NewRecorder()
		profileHandler.Create(wCreate, reqCreate)

		var createdProfile services.ScheduleProfile
		json.NewDecoder(wCreate.Body).Decode(&createdProfile)

		// Now get the profile
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+createdProfile.ID, nil)
		req.Header.Set("X-Hasura-Tenant-Id", tenantID)

		w := httptest.NewRecorder()
		profileHandler.Get(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var profile services.ScheduleProfile
		err := json.NewDecoder(w.Body).Decode(&profile)
		require.NoError(t, err)

		assert.Equal(t, createdProfile.ID, profile.ID)
		assert.Equal(t, "EU-Finance", profile.ProfileName)
	})

	t.Run("UpdateProfile_BitemporalVersioning", func(t *testing.T) {
		// Create profile
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", nil)
		reqCreate.Header.Set("X-Hasura-Tenant-Id", tenantID)
		reqCreate.Header.Set("X-Hasura-User-Id", userID)

		reqBody, _ := json.Marshal(map[string]interface{}{
			"profile_name":        "APAC-Operations",
			"calendars":           []string{"cal-1"},
			"conflict_resolution": "priority",
			"timezone":            "Asia/Tokyo",
		})

		reqCreate.Body = io.NopCloser(strings.NewReader(string(reqBody)))
		reqCreate.Header.Set("Content-Type", "application/json")

		wCreate := httptest.NewRecorder()
		profileHandler.Create(wCreate, reqCreate)

		var createdProfile services.ScheduleProfile
		json.NewDecoder(wCreate.Body).Decode(&createdProfile)

		oldID := createdProfile.ID
		oldTimezone := createdProfile.Timezone

		// Update profile
		reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/"+oldID, nil)
		reqUpdate.Header.Set("X-Hasura-Tenant-Id", tenantID)
		reqUpdate.Header.Set("X-Hasura-User-Id", userID)

		updateBody, _ := json.Marshal(map[string]interface{}{
			"timezone": "Asia/Singapore",
		})

		reqUpdate.Body = io.NopCloser(strings.NewReader(string(updateBody)))
		reqUpdate.Header.Set("Content-Type", "application/json")

		wUpdate := httptest.NewRecorder()
		profileHandler.Update(wUpdate, reqUpdate)

		assert.Equal(t, http.StatusOK, wUpdate.Code)

		var updatedProfile services.ScheduleProfile
		json.NewDecoder(wUpdate.Body).Decode(&updatedProfile)

		// Verify new version
		assert.NotEqual(t, oldID, updatedProfile.ID) // New version has new ID
		assert.NotEqual(t, oldTimezone, updatedProfile.Timezone)
		assert.Equal(t, "Asia/Singapore", updatedProfile.Timezone)
		assert.Nil(t, updatedProfile.ValidTo) // New version is active
	})

	t.Run("DeleteProfile_SoftDelete", func(t *testing.T) {
		// Create profile
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", nil)
		reqCreate.Header.Set("X-Hasura-Tenant-Id", tenantID)
		reqCreate.Header.Set("X-Hasura-User-Id", userID)

		reqBody, _ := json.Marshal(map[string]interface{}{
			"profile_name":        "Temp-Profile",
			"calendars":           []string{"cal-1"},
			"conflict_resolution": "union",
			"timezone":            "UTC",
		})

		reqCreate.Body = io.NopCloser(strings.NewReader(string(reqBody)))
		reqCreate.Header.Set("Content-Type", "application/json")

		wCreate := httptest.NewRecorder()
		profileHandler.Create(wCreate, reqCreate)

		var createdProfile services.ScheduleProfile
		json.NewDecoder(wCreate.Body).Decode(&createdProfile)

		// Delete profile
		reqDelete := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/"+createdProfile.ID, nil)
		reqDelete.Header.Set("X-Hasura-Tenant-Id", tenantID)
		reqDelete.Header.Set("X-Hasura-User-Id", userID)

		wDelete := httptest.NewRecorder()
		profileHandler.Delete(wDelete, reqDelete)

		assert.Equal(t, http.StatusNoContent, wDelete.Code)

		// Verify deletion
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+createdProfile.ID, nil)
		reqGet.Header.Set("X-Hasura-Tenant-Id", tenantID)

		wGet := httptest.NewRecorder()
		profileHandler.Get(wGet, reqGet)

		// Should return 404 since profile was soft-deleted
		assert.Equal(t, http.StatusNotFound, wGet.Code)
	})

	t.Run("ListVersions", func(t *testing.T) {
		// Create and update profile multiple times
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", nil)
		reqCreate.Header.Set("X-Hasura-Tenant-Id", tenantID)
		reqCreate.Header.Set("X-Hasura-User-Id", userID)

		reqBody, _ := json.Marshal(map[string]interface{}{
			"profile_name":        "Multi-Version",
			"calendars":           []string{"cal-1"},
			"conflict_resolution": "union",
			"timezone":            "UTC",
		})

		reqCreate.Body = io.NopCloser(strings.NewReader(string(reqBody)))
		reqCreate.Header.Set("Content-Type", "application/json")

		wCreate := httptest.NewRecorder()
		profileHandler.Create(wCreate, reqCreate)

		var profile services.ScheduleProfile
		json.NewDecoder(wCreate.Body).Decode(&profile)

		profileID := profile.ID

		// Update twice
		for i := 0; i < 2; i++ {
			reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/"+profileID, nil)
			reqUpdate.Header.Set("X-Hasura-Tenant-Id", tenantID)
			reqUpdate.Header.Set("X-Hasura-User-Id", userID)

			updateBody, _ := json.Marshal(map[string]interface{}{
				"description": "Update " + string(rune(i)),
			})

			reqUpdate.Body = io.NopCloser(strings.NewReader(string(updateBody)))
			reqUpdate.Header.Set("Content-Type", "application/json")

			wUpdate := httptest.NewRecorder()
			profileHandler.Update(wUpdate, reqUpdate)

			json.NewDecoder(wUpdate.Body).Decode(&profile)
			profileID = profile.ID             // Use new version ID
			time.Sleep(100 * time.Millisecond) // Small delay for distinct timestamps
		}

		// List versions
		reqVersions := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/Multi-Version/versions", nil)
		reqVersions.Header.Set("X-Hasura-Tenant-Id", tenantID)

		wVersions := httptest.NewRecorder()
		profileHandler.ListVersions(wVersions, reqVersions)

		assert.Equal(t, http.StatusOK, wVersions.Code)

		var versions []services.ScheduleProfile
		err := json.NewDecoder(wVersions.Body).Decode(&versions)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(versions), 1)
	})

	t.Run("TenantIsolation", func(t *testing.T) {
		// Create profile in tenant 1
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", nil)
		reqCreate.Header.Set("X-Hasura-Tenant-Id", "tenant-1")
		reqCreate.Header.Set("X-Hasura-User-Id", userID)

		reqBody, _ := json.Marshal(map[string]interface{}{
			"profile_name":        "Tenant1-Profile",
			"calendars":           []string{"cal-1"},
			"conflict_resolution": "union",
			"timezone":            "UTC",
		})

		reqCreate.Body = io.NopCloser(strings.NewReader(string(reqBody)))
		reqCreate.Header.Set("Content-Type", "application/json")

		wCreate := httptest.NewRecorder()
		profileHandler.Create(wCreate, reqCreate)

		var profile services.ScheduleProfile
		json.NewDecoder(wCreate.Body).Decode(&profile)

		// Try to access with different tenant
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+profile.ID, nil)
		reqGet.Header.Set("X-Hasura-Tenant-Id", "tenant-2")

		wGet := httptest.NewRecorder()
		profileHandler.Get(wGet, reqGet)

		// Should fail with 404 (access denied, not found)
		assert.Equal(t, http.StatusNotFound, wGet.Code)
	})
}

// BenchmarkProfileCreation benchmarks profile creation performance
func BenchmarkProfileCreation(b *testing.B) {
	logger := logrus.New()
	logEntry := logger.WithField("test", "profile_creation")

	calendarRepo := repository.NewInMemoryCalendarRepository(logEntry)
	repoAdapter := services.NewRepositoryAdapter(calendarRepo, logEntry)
	auditService := services.NewAuditService(logEntry)
	profileService := services.NewProfileService(repoAdapter, auditService, logEntry)

	tenantID := "bench-tenant"

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		input := services.CreateProfileInput{
			ProfileName:        "Bench-Profile",
			Calendars:          []string{"cal-1"},
			ConflictResolution: "union",
			Timezone:           "UTC",
			ActorID:            "system",
		}

		_, err := profileService.Create(ctx, tenantID, input)
		if err != nil {
			b.Fatalf("Failed to create profile: %v", err)
		}
	}
}
