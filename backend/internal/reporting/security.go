package reporting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// AUDIT LOGGING
// ============================================================================

// AuditEventType defines types of audit events
type AuditEventType string

const (
	AuditEventReportCreated    AuditEventType = "report.created"
	AuditEventReportUpdated    AuditEventType = "report.updated"
	AuditEventReportDeleted    AuditEventType = "report.deleted"
	AuditEventReportViewed     AuditEventType = "report.viewed"
	AuditEventReportRendered   AuditEventType = "report.rendered"
	AuditEventReportExported   AuditEventType = "report.exported"
	AuditEventReportShared     AuditEventType = "report.shared"
	AuditEventScheduleCreated  AuditEventType = "schedule.created"
	AuditEventScheduleUpdated  AuditEventType = "schedule.updated"
	AuditEventScheduleDeleted  AuditEventType = "schedule.deleted"
	AuditEventScheduleExecuted AuditEventType = "schedule.executed"
	AuditEventAccessGranted    AuditEventType = "access.granted"
	AuditEventAccessRevoked    AuditEventType = "access.revoked"
	AuditEventDataAccessed     AuditEventType = "data.accessed"
	AuditEventConfigChanged    AuditEventType = "config.changed"
)

// AuditEvent represents an auditable action
type AuditEvent struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     uuid.UUID              `json:"tenant_id"`
	UserID       uuid.UUID              `json:"user_id"`
	EventType    AuditEventType         `json:"event_type"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   uuid.UUID              `json:"resource_id"`
	Action       string                 `json:"action"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	SessionID    string                 `json:"session_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Outcome      string                 `json:"outcome"` // success, failure, denied
	ErrorMessage string                 `json:"error_message,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`

	// For compliance
	DataClassification string `json:"data_classification,omitempty"`
	RetentionPolicy    string `json:"retention_policy,omitempty"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	events chan *AuditEvent
	stopCh chan struct{}
	wg     sync.WaitGroup

	// Pluggable backends
	backends []AuditBackend
}

// AuditBackend defines an audit log destination
type AuditBackend interface {
	Write(ctx context.Context, event *AuditEvent) error
	Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error)
	Close() error
}

// AuditFilter for querying audit logs
type AuditFilter struct {
	TenantID     *uuid.UUID
	UserID       *uuid.UUID
	EventTypes   []AuditEventType
	ResourceType string
	ResourceID   *uuid.UUID
	StartTime    time.Time
	EndTime      time.Time
	Outcome      string
	Limit        int
	Offset       int
}

// NewAuditLogger creates an audit logger
func NewAuditLogger(backends ...AuditBackend) *AuditLogger {
	al := &AuditLogger{
		events:   make(chan *AuditEvent, 10000),
		stopCh:   make(chan struct{}),
		backends: backends,
	}
	al.start()
	return al
}

func (al *AuditLogger) start() {
	al.wg.Add(1)
	go al.processEvents()
}

func (al *AuditLogger) processEvents() {
	defer al.wg.Done()

	for {
		select {
		case event := <-al.events:
			ctx := context.Background()
			for _, backend := range al.backends {
				if err := backend.Write(ctx, event); err != nil {
					// Log error but don't fail - audit should not block operations
					fmt.Printf("audit backend error: %v\n", err)
				}
			}
		case <-al.stopCh:
			// Drain remaining events
			for {
				select {
				case event := <-al.events:
					ctx := context.Background()
					for _, backend := range al.backends {
						_ = backend.Write(ctx, event)
					}
				default:
					return
				}
			}
		}
	}
}

// Log records an audit event
func (al *AuditLogger) Log(event *AuditEvent) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	select {
	case al.events <- event:
	default:
		// Channel full - log synchronously to prevent data loss
		ctx := context.Background()
		for _, backend := range al.backends {
			_ = backend.Write(ctx, event)
		}
	}
}

// LogReportAction logs a report-related action
func (al *AuditLogger) LogReportAction(
	ctx context.Context,
	tenantID uuid.UUID,
	userID uuid.UUID,
	eventType AuditEventType,
	reportID uuid.UUID,
	details map[string]interface{},
	outcome string,
) {
	al.Log(&AuditEvent{
		TenantID:     tenantID,
		UserID:       userID,
		EventType:    eventType,
		ResourceType: "report",
		ResourceID:   reportID,
		Action:       string(eventType),
		Details:      details,
		Outcome:      outcome,
	})
}

// Query searches audit logs
func (al *AuditLogger) Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error) {
	// Use first backend that supports querying
	for _, backend := range al.backends {
		events, err := backend.Query(ctx, filter)
		if err == nil {
			return events, nil
		}
	}
	return nil, fmt.Errorf("no queryable audit backend available")
}

// Stop gracefully stops the audit logger
func (al *AuditLogger) Stop() {
	close(al.stopCh)
	al.wg.Wait()
	for _, backend := range al.backends {
		_ = backend.Close()
	}
}

// ============================================================================
// DATA MASKING & PII PROTECTION
// ============================================================================

// DataMaskingRule defines how to mask sensitive data
type DataMaskingRule struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	FieldPattern string    `json:"field_pattern"` // Regex to match field names
	MaskType     MaskType  `json:"mask_type"`

	// For custom masks
	CustomPattern string `json:"custom_pattern,omitempty"`
	Replacement   string `json:"replacement,omitempty"`

	// Conditions
	DataClassifications []string `json:"data_classifications,omitempty"`
	ApplyToRoles        []string `json:"apply_to_roles,omitempty"`
	ExcludeRoles        []string `json:"exclude_roles,omitempty"`
}

// MaskType defines masking strategies
type MaskType string

const (
	MaskTypeRedact     MaskType = "redact"      // Replace with ***
	MaskTypePartial    MaskType = "partial"     // Show first/last chars
	MaskTypeHash       MaskType = "hash"        // SHA256 hash
	MaskTypeTokenize   MaskType = "tokenize"    // Replace with token
	MaskTypeEncrypt    MaskType = "encrypt"     // Encrypt value
	MaskTypeNull       MaskType = "null"        // Replace with null
	MaskTypeCustom     MaskType = "custom"      // Custom pattern
	MaskTypeEmail      MaskType = "email"       // j***@example.com
	MaskTypePhone      MaskType = "phone"       // ***-***-1234
	MaskTypeCreditCard MaskType = "credit_card" // ****-****-****-1234
	MaskTypeSSN        MaskType = "ssn"         // ***-**-1234
)

// DataMasker handles PII and sensitive data masking
type DataMasker struct {
	rules         []*DataMaskingRule
	compiledRules map[uuid.UUID]*regexp.Regexp
	mutex         sync.RWMutex
}

// NewDataMasker creates a data masker with default rules
func NewDataMasker() *DataMasker {
	dm := &DataMasker{
		rules:         make([]*DataMaskingRule, 0),
		compiledRules: make(map[uuid.UUID]*regexp.Regexp),
	}
	dm.loadDefaultRules()
	return dm
}

func (dm *DataMasker) loadDefaultRules() {
	defaultRules := []*DataMaskingRule{
		{
			ID:           uuid.New(),
			Name:         "Email Addresses",
			FieldPattern: "(?i)(email|e_mail|e-mail|email_address)",
			MaskType:     MaskTypeEmail,
		},
		{
			ID:           uuid.New(),
			Name:         "Phone Numbers",
			FieldPattern: "(?i)(phone|telephone|mobile|cell|fax)",
			MaskType:     MaskTypePhone,
		},
		{
			ID:           uuid.New(),
			Name:         "Credit Cards",
			FieldPattern: "(?i)(credit_card|card_number|cc_num|ccn)",
			MaskType:     MaskTypeCreditCard,
		},
		{
			ID:           uuid.New(),
			Name:         "Social Security Numbers",
			FieldPattern: "(?i)(ssn|social_security|sin|national_id)",
			MaskType:     MaskTypeSSN,
		},
		{
			ID:           uuid.New(),
			Name:         "Passwords",
			FieldPattern: "(?i)(password|passwd|pwd|secret|api_key|token)",
			MaskType:     MaskTypeRedact,
		},
		{
			ID:           uuid.New(),
			Name:         "Bank Accounts",
			FieldPattern: "(?i)(bank_account|account_number|routing_number|iban|swift)",
			MaskType:     MaskTypePartial,
		},
	}

	for _, rule := range defaultRules {
		dm.AddRule(rule)
	}
}

// AddRule adds a masking rule
func (dm *DataMasker) AddRule(rule *DataMaskingRule) error {
	compiled, err := regexp.Compile(rule.FieldPattern)
	if err != nil {
		return fmt.Errorf("invalid field pattern: %w", err)
	}

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.rules = append(dm.rules, rule)
	dm.compiledRules[rule.ID] = compiled
	return nil
}

// MaskData masks sensitive data in a map
func (dm *DataMasker) MaskData(data map[string]interface{}, userRoles []string) map[string]interface{} {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	result := make(map[string]interface{})
	for key, value := range data {
		result[key] = dm.maskValue(key, value, userRoles)
	}
	return result
}

func (dm *DataMasker) maskValue(fieldName string, value interface{}, userRoles []string) interface{} {
	// Handle nested structures
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = dm.maskValue(key, val, userRoles)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = dm.maskValue(fieldName, item, userRoles)
		}
		return result
	}

	// Find matching rule
	for _, rule := range dm.rules {
		compiled := dm.compiledRules[rule.ID]
		if compiled.MatchString(fieldName) {
			// Check if user is excluded
			if dm.userExcluded(userRoles, rule) {
				continue
			}
			return dm.applyMask(value, rule.MaskType)
		}
	}

	return value
}

func (dm *DataMasker) userExcluded(userRoles []string, rule *DataMaskingRule) bool {
	for _, excludeRole := range rule.ExcludeRoles {
		for _, userRole := range userRoles {
			if excludeRole == userRole {
				return true
			}
		}
	}
	return false
}

func (dm *DataMasker) applyMask(value interface{}, maskType MaskType) interface{} {
	str, ok := value.(string)
	if !ok {
		// Try to convert to string for masking
		str = fmt.Sprintf("%v", value)
	}

	switch maskType {
	case MaskTypeRedact:
		return "***REDACTED***"
	case MaskTypePartial:
		return dm.partialMask(str)
	case MaskTypeHash:
		hash := sha256.Sum256([]byte(str))
		return hex.EncodeToString(hash[:])
	case MaskTypeNull:
		return nil
	case MaskTypeEmail:
		return dm.maskEmail(str)
	case MaskTypePhone:
		return dm.maskPhone(str)
	case MaskTypeCreditCard:
		return dm.maskCreditCard(str)
	case MaskTypeSSN:
		return dm.maskSSN(str)
	default:
		return "***"
	}
}

func (dm *DataMasker) partialMask(s string) string {
	if len(s) <= 4 {
		return "***"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func (dm *DataMasker) maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***.***"
	}
	local := parts[0]
	if len(local) > 1 {
		local = local[:1] + "***"
	}
	return local + "@" + parts[1]
}

func (dm *DataMasker) maskPhone(phone string) string {
	// Keep last 4 digits
	digits := regexp.MustCompile(`\d`).FindAllString(phone, -1)
	if len(digits) < 4 {
		return "***-***-****"
	}
	return "***-***-" + strings.Join(digits[len(digits)-4:], "")
}

func (dm *DataMasker) maskCreditCard(cc string) string {
	digits := regexp.MustCompile(`\d`).FindAllString(cc, -1)
	if len(digits) < 4 {
		return "****-****-****-****"
	}
	return "****-****-****-" + strings.Join(digits[len(digits)-4:], "")
}

func (dm *DataMasker) maskSSN(ssn string) string {
	digits := regexp.MustCompile(`\d`).FindAllString(ssn, -1)
	if len(digits) < 4 {
		return "***-**-****"
	}
	return "***-**-" + strings.Join(digits[len(digits)-4:], "")
}

// ============================================================================
// ROW-LEVEL SECURITY
// ============================================================================

// RLSPolicy defines row-level security rules
type RLSPolicy struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DataSource  string    `json:"data_source"`
	TableName   string    `json:"table_name"`

	// Policy conditions
	Conditions []RLSCondition `json:"conditions"`

	// Applies to
	Roles []string    `json:"roles"`
	Users []uuid.UUID `json:"users,omitempty"`

	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RLSCondition defines a row filter condition
type RLSCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // =, !=, in, not_in, >, <, >=, <=, like, contains
	Value    string `json:"value"`    // Can reference user attributes with {{user.attribute}}
}

// RLSEngine enforces row-level security
type RLSEngine struct {
	policies map[uuid.UUID][]*RLSPolicy // tenantID -> policies
	mutex    sync.RWMutex
}

// NewRLSEngine creates an RLS engine
func NewRLSEngine() *RLSEngine {
	return &RLSEngine{
		policies: make(map[uuid.UUID][]*RLSPolicy),
	}
}

// AddPolicy adds an RLS policy
func (rls *RLSEngine) AddPolicy(policy *RLSPolicy) {
	rls.mutex.Lock()
	defer rls.mutex.Unlock()

	policies := rls.policies[policy.TenantID]
	rls.policies[policy.TenantID] = append(policies, policy)
}

// GetApplicablePolicies returns policies that apply to a user
func (rls *RLSEngine) GetApplicablePolicies(
	tenantID uuid.UUID,
	userID uuid.UUID,
	userRoles []string,
	dataSource string,
	tableName string,
) []*RLSPolicy {
	rls.mutex.RLock()
	defer rls.mutex.RUnlock()

	var applicable []*RLSPolicy

	for _, policy := range rls.policies[tenantID] {
		if !policy.Enabled {
			continue
		}
		if policy.DataSource != dataSource || policy.TableName != tableName {
			continue
		}

		// Check if policy applies to user
		if rls.policyApplies(policy, userID, userRoles) {
			applicable = append(applicable, policy)
		}
	}

	return applicable
}

func (rls *RLSEngine) policyApplies(policy *RLSPolicy, userID uuid.UUID, userRoles []string) bool {
	// Check user-specific assignment
	for _, uid := range policy.Users {
		if uid == userID {
			return true
		}
	}

	// Check role-based assignment
	for _, policyRole := range policy.Roles {
		for _, userRole := range userRoles {
			if policyRole == userRole {
				return true
			}
		}
	}

	return false
}

// BuildWhereClause generates SQL WHERE clause from policies
func (rls *RLSEngine) BuildWhereClause(
	policies []*RLSPolicy,
	userContext map[string]interface{},
) (string, []interface{}) {
	if len(policies) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}
	argIndex := 1

	for _, policy := range policies {
		for _, cond := range policy.Conditions {
			clause, condArgs, newIndex := rls.buildCondition(cond, userContext, argIndex)
			if clause != "" {
				clauses = append(clauses, clause)
				args = append(args, condArgs...)
				argIndex = newIndex
			}
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " AND "), args
}

func (rls *RLSEngine) buildCondition(
	cond RLSCondition,
	userContext map[string]interface{},
	argIndex int,
) (string, []interface{}, int) {
	// Resolve user context placeholders
	value := rls.resolveValue(cond.Value, userContext)

	switch cond.Operator {
	case "=", "==":
		return fmt.Sprintf("%s = $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case "!=", "<>":
		return fmt.Sprintf("%s != $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case ">":
		return fmt.Sprintf("%s > $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case "<":
		return fmt.Sprintf("%s < $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case ">=":
		return fmt.Sprintf("%s >= $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case "<=":
		return fmt.Sprintf("%s <= $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case "in":
		return fmt.Sprintf("%s = ANY($%d)", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case "like":
		return fmt.Sprintf("%s LIKE $%d", cond.Field, argIndex), []interface{}{value}, argIndex + 1
	case "contains":
		return fmt.Sprintf("%s LIKE $%d", cond.Field, argIndex), []interface{}{"%" + value.(string) + "%"}, argIndex + 1
	default:
		return "", nil, argIndex
	}
}

func (rls *RLSEngine) resolveValue(value string, userContext map[string]interface{}) interface{} {
	// Handle {{user.attribute}} placeholders
	re := regexp.MustCompile(`\{\{user\.(\w+)\}\}`)
	matches := re.FindStringSubmatch(value)
	if len(matches) == 2 {
		attr := matches[1]
		if val, ok := userContext[attr]; ok {
			return val
		}
	}
	return value
}

// ============================================================================
// ENCRYPTION SERVICE
// ============================================================================

// EncryptionService handles data encryption
type EncryptionService struct {
	// Key management would integrate with HSM/KMS in production
	tenantKeys map[uuid.UUID][]byte
	mutex      sync.RWMutex
}

// NewEncryptionService creates an encryption service
func NewEncryptionService() *EncryptionService {
	return &EncryptionService{
		tenantKeys: make(map[uuid.UUID][]byte),
	}
}

// EncryptedField represents an encrypted value
type EncryptedField struct {
	Ciphertext string `json:"ciphertext"`
	KeyID      string `json:"key_id"`
	Algorithm  string `json:"algorithm"`
	Version    int    `json:"version"`
}

// Note: Actual encryption implementation would use AES-GCM or similar
// This is a placeholder for the encryption interface

// ============================================================================
// SECURITY CONTEXT
// ============================================================================

// SecurityContext carries security information through requests
type SecurityContext struct {
	TenantID           uuid.UUID
	UserID             uuid.UUID
	Roles              []string
	Permissions        []string
	DataClassification string
	IPAddress          string
	UserAgent          string
	SessionID          string
	RequestID          string
	Authenticated      bool
	MFAVerified        bool
	TokenExpiry        time.Time

	// Additional attributes for ABAC
	Attributes map[string]interface{}
}

// ToJSON serializes the security context
func (sc *SecurityContext) ToJSON() string {
	data, _ := json.Marshal(sc)
	return string(data)
}

// HasPermission checks if user has a specific permission
func (sc *SecurityContext) HasPermission(permission string) bool {
	for _, p := range sc.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// HasRole checks if user has a specific role
func (sc *SecurityContext) HasRole(role string) bool {
	for _, r := range sc.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if user has any of the specified roles
func (sc *SecurityContext) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if sc.HasRole(role) {
			return true
		}
	}
	return false
}
