package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// AbacService provides methods for attribute-based access control.
type AbacService struct {
	db *sqlx.DB
}

// NewAbacService creates a new AbacService.
func NewAbacService(db *sqlx.DB) *AbacService {
	return &AbacService{db: db}
}

// EvaluateAccess is the core ABAC evaluation engine.
// It checks if a subject's action on a resource is permitted based on active policies and environmental context.
func (s *AbacService) EvaluateAccess(ctx context.Context, subjectAttrs map[string]any, action string, resourceAttrs map[string]any, envAttrs map[string]any) (bool, string, error) {
	// Fetch active policies, ordered by priority for deterministic conflict resolution.
	var policies []models.Policy
	err := s.db.SelectContext(ctx, &policies, `SELECT * FROM policies WHERE active = true ORDER BY priority DESC`)
	if err != nil {
		return false, "Error fetching policies", err
	}

	var matchedPolicy *models.Policy
	var decision bool
	reason := "Access denied: No matching policy found."

	for _, p := range policies {
		var rules map[string]any
		if err := json.Unmarshal(p.Rules, &rules); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to unmarshal rules for policy %s: %v", p.ID, err)
			continue
		}

		// Rule matching logic
		if subjectOk, _ := matchesAttrs(subjectAttrs, rules["subject"]); subjectOk {
			if actionOk, _ := matchesAttrs(map[string]any{"action": action}, rules); actionOk {
				if resourceOk, _ := matchesAttrs(resourceAttrs, rules["resource"]); resourceOk {
					// Temporal and location checks
					if !s.isWithinTemporal(p, time.Now()) {
						continue
					}
					if !s.isWithinLocation(p, envAttrs) {
						continue
					}

					// First matching policy by priority determines the outcome.
					matchedPolicy = &p
					if effect, ok := rules["effect"].(string); ok && effect == "allow" {
						decision = true
						reason = fmt.Sprintf("Access allowed: Matched policy '%s' (ID: %s)", p.Name, p.ID)
					} else {
						decision = false
						reason = fmt.Sprintf("Access explicitly denied: Matched policy '%s' (ID: %s)", p.Name, p.ID)
					}
					break // Stop at the first matching policy due to priority ordering.
				}
			}
		}
	}

	// Log the audit event asynchronously.
	go s.logAudit(ctx, subjectAttrs["user_id"], "policy_eval", map[string]any{
		"decision":       decision,
		"reason":         reason,
		"subject":        subjectAttrs,
		"action":         action,
		"resource":       resourceAttrs,
		"env":            envAttrs,
		"matched_policy": matchedPolicy,
	})

	return decision, reason, nil
}

// matchesAttrs compares attribute maps. It returns true if all attributes in `required`
// are present and satisfied in `given`.
func matchesAttrs(given, required any) (bool, error) {
	requiredMap, ok := required.(map[string]any)
	if !ok || len(requiredMap) == 0 {
		// If there are no required attributes, it's a match.
		return true, nil
	}

	givenMap, ok := given.(map[string]any)
	if !ok {
		// If given attributes are not a map, but we have requirements, it's a mismatch.
		return false, nil
	}

	for key, requiredVal := range requiredMap {
		givenVal, exists := givenMap[key]
		if !exists {
			return false, nil // Required attribute is missing.
		}

		// Helper to check slice/array contains element
		contains := func(slice []any, item any) bool {
			for _, val := range slice {
				if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", item) {
					return true
				}
			}
			return false
		}

		// Convert given value to a slice of interfaces
		var givenSlice []any
		switch g := givenVal.(type) {
		case []any:
			givenSlice = g
		case []string:
			for _, val := range g {
				givenSlice = append(givenSlice, val)
			}
		default:
			givenSlice = []any{g}
		}

		// Convert required value to a slice of interfaces
		var requiredSlice []any
		switch r := requiredVal.(type) {
		case []any:
			requiredSlice = r
		case []string:
			for _, val := range r {
				requiredSlice = append(requiredSlice, val)
			}
		default:
			requiredSlice = []any{r}
		}

		// Check if there is any intersection/match between given and required values
		matchFound := false
		for _, reqItem := range requiredSlice {
			if contains(givenSlice, reqItem) {
				matchFound = true
				break
			}
		}

		if !matchFound {
			return false, nil
		}
	}

	return true, nil
}

// isWithinTemporal checks if the current time is within the policy's defined time constraints.
func (s *AbacService) isWithinTemporal(p models.Policy, now time.Time) bool {
	if p.StartDate != nil && now.Before(*p.StartDate) {
		return false
	}
	if p.EndDate != nil && now.After(*p.EndDate) {
		return false
	}
	// Implement schedule logic (days of week, time windows)
	if len(p.Schedule) > 0 { // Simplified from `if p.Schedule != nil && len(p.Schedule) > 0 {`
		var schedule map[string]interface{}
		if err := json.Unmarshal(p.Schedule, &schedule); err != nil {
			return false // Invalid schedule format
		}

		currentTime := time.Now()
		currentDay := currentTime.Weekday().String()
		currentHour := currentTime.Hour()

		// Check if schedule has day restrictions
		if days, ok := schedule["allowed_days"].([]interface{}); ok {
			dayAllowed := false
			for _, day := range days {
				if dayStr, ok := day.(string); ok && dayStr == currentDay {
					dayAllowed = true
					break
				}
			}
			if !dayAllowed {
				return false // Not allowed on this day
			}
		}

		// Check time windows
		if startHour, ok := schedule["start_hour"].(float64); ok {
			if endHour, ok := schedule["end_hour"].(float64); ok {
				if currentHour < int(startHour) || currentHour >= int(endHour) {
					return false // Outside allowed time window
				}
			}
		}
	}
	return true
}

// isWithinLocation checks if the request's environment attributes match the policy's location rules.
func (s *AbacService) isWithinLocation(p models.Policy, env map[string]any) bool {
	if p.LocationRules == nil {
		return true
	}
	var rules map[string]any
	if err := json.Unmarshal(p.LocationRules, &rules); err != nil {
		return false // Invalid location rule format
	}

	if ipRange, ok := rules["ip_range"].(string); ok {
		if ipStr, ipOk := env["ip"].(string); ipOk {
			ip := net.ParseIP(ipStr)
			_, cidr, err := net.ParseCIDR(ipRange)
			if err != nil || !cidr.Contains(ip) {
				return false
			}
		}
	}
	// Implement geofence logic from rules
	if geofences, ok := rules["geofences"].([]interface{}); ok && len(geofences) > 0 {
		// Extract user location from environment
		userLat, hasLat := env["user_latitude"].(float64)
		userLon, hasLon := env["user_longitude"].(float64)

		if hasLat && hasLon {
			// Check if user is within any allowed geofence
			inGeofence := false
			for _, fence := range geofences {
				if fenceMap, ok := fence.(map[string]interface{}); ok {
					fenceLat, _ := fenceMap["latitude"].(float64)
					fenceLon, _ := fenceMap["longitude"].(float64)
					radiusKm, _ := fenceMap["radius_km"].(float64)

					// Simple distance calculation (Haversine formula)
					distance := calculateDistance(userLat, userLon, fenceLat, fenceLon)
					if distance <= radiusKm {
						inGeofence = true
						break
					}
				}
			}

			if !inGeofence {
				return false // User outside all allowed geofences
			}
		}
	}
	return true
}

// calculateDistance calculates distance in km between two lat/lon points using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // Earth radius in kilometers

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// logAudit creates an audit event record in the database.
func (s *AbacService) logAudit(ctx context.Context, userID any, eventType string, details map[string]any) {
	detailsJSON, _ := json.Marshal(details)
	userIDStr := fmt.Sprintf("%v", userID)
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_events (event_type, user_id, details) VALUES ($1, $2, $3)`, eventType, userIDStr, detailsJSON)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to log ABAC audit event: %v", err)
	}
}

// --- Policy CRUD ---

// CreatePolicy creates a new policy in the database.
func (s *AbacService) CreatePolicy(ctx context.Context, policy *models.Policy) (*models.Policy, error) {
	query := `INSERT INTO policies (name, rules, start_date, end_date, schedule, location_rules, priority, active)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *`
	var newPolicy models.Policy

	// Coerce nil JSON fields to empty JSON objects to avoid NULL scan errors
	var rules json.RawMessage
	if len(policy.Rules) == 0 {
		rules = json.RawMessage(`{}`)
	} else {
		rules = policy.Rules
	}
	var schedule json.RawMessage
	if len(policy.Schedule) == 0 {
		schedule = json.RawMessage(`{}`)
	} else {
		schedule = policy.Schedule
	}
	var locationRules json.RawMessage
	if len(policy.LocationRules) == 0 {
		locationRules = json.RawMessage(`{}`)
	} else {
		locationRules = policy.LocationRules
	}

	err := s.db.QueryRowxContext(ctx, query,
		policy.Name, rules, policy.StartDate, policy.EndDate, schedule, locationRules, policy.Priority, policy.Active,
	).StructScan(&newPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}
	return &newPolicy, nil
}

// ListPolicies retrieves all policies from the database.
func (s *AbacService) ListPolicies(ctx context.Context) ([]models.Policy, error) {
	var policies []models.Policy
	err := s.db.SelectContext(ctx, &policies, "SELECT * FROM policies ORDER BY priority DESC, name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	return policies, nil
}

// GetPolicy retrieves a single policy by its ID.
func (s *AbacService) GetPolicy(ctx context.Context, id uuid.UUID) (*models.Policy, error) {
	var policy models.Policy
	err := s.db.GetContext(ctx, &policy, "SELECT * FROM policies WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy %s: %w", id, err)
	}
	return &policy, nil
}

// UpdatePolicy updates an existing policy in the database.
func (s *AbacService) UpdatePolicy(ctx context.Context, policy *models.Policy) error {
	query := `UPDATE policies SET
	            name = :name, rules = :rules, start_date = :start_date, end_date = :end_date,
	            schedule = :schedule, location_rules = :location_rules, priority = :priority, active = :active,
	            updated_at = NOW()
	          WHERE id = :id`
	// Ensure JSON fields are non-nil to avoid driver scan issues when using NamedExec
	if len(policy.Rules) == 0 {
		policy.Rules = json.RawMessage(`{}`)
	}
	if len(policy.Schedule) == 0 {
		policy.Schedule = json.RawMessage(`{}`)
	}
	if len(policy.LocationRules) == 0 {
		policy.LocationRules = json.RawMessage(`{}`)
	}

	result, err := s.db.NamedExecContext(ctx, query, policy)
	if err != nil {
		return fmt.Errorf("failed to update policy %s: %w", policy.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for policy %s: %w", policy.ID, err)
	}
	if rowsAffected == 0 {
		return errors.New("policy not found or no changes made")
	}
	return nil
}

// DeletePolicy removes a policy from the database by its ID.
func (s *AbacService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM policies WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete policy %s: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for policy %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("policy with id %s not found", id)
	}
	return nil
}

// --- Resource CRUD ---

// CreateResource creates a new resource in the database.
func (s *AbacService) CreateResource(ctx context.Context, resource *models.Resource) (*models.Resource, error) {
	query := `INSERT INTO resources (name, attributes) VALUES ($1, $2) RETURNING *`
	var newResource models.Resource
	err := s.db.QueryRowxContext(ctx, query, resource.Name, resource.Attributes).StructScan(&newResource)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return &newResource, nil
}

// ListResources retrieves all resources from the database.
func (s *AbacService) ListResources(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource
	err := s.db.SelectContext(ctx, &resources, "SELECT * FROM resources ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	return resources, nil
}

// GetResource retrieves a single resource by its ID.
func (s *AbacService) GetResource(ctx context.Context, id uuid.UUID) (*models.Resource, error) {
	var resource models.Resource
	err := s.db.GetContext(ctx, &resource, "SELECT * FROM resources WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", id, err)
	}
	return &resource, nil
}

// UpdateResource updates an existing resource in the database.
func (s *AbacService) UpdateResource(ctx context.Context, resource *models.Resource) error {
	query := `UPDATE resources SET name = :name, attributes = :attributes, updated_at = NOW() WHERE id = :id`
	result, err := s.db.NamedExecContext(ctx, query, resource)
	if err != nil {
		return fmt.Errorf("failed to update resource %s: %w", resource.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for resource %s: %w", resource.ID, err)
	}
	if rowsAffected == 0 {
		return errors.New("resource not found or no changes made")
	}
	return nil
}

// DeleteResource removes a resource from the database by its ID.
func (s *AbacService) DeleteResource(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM resources WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete resource %s: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for resource %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("resource with id %s not found", id)
	}
	return nil
}
