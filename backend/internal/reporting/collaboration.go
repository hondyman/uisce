package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// REAL-TIME COLLABORATION
// ============================================================================

// CollaborationSession represents an active editing session
type CollaborationSession struct {
	ID           uuid.UUID                `json:"id"`
	TenantID     uuid.UUID                `json:"tenant_id"`
	ReportID     uuid.UUID                `json:"report_id"`
	CreatedBy    uuid.UUID                `json:"created_by"`
	CreatedAt    time.Time                `json:"created_at"`
	Participants []*Participant           `json:"participants"`
	State        *DocumentState           `json:"state"`
	Cursors      map[uuid.UUID]*Cursor    `json:"cursors"`
	Selections   map[uuid.UUID]*Selection `json:"selections"`
	Version      int64                    `json:"version"`
	LastActivity time.Time                `json:"last_activity"`
}

// Participant represents a user in a collaboration session
type Participant struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Color       string    `json:"color"` // Assigned color for cursors/highlights
	JoinedAt    time.Time `json:"joined_at"`
	LastSeen    time.Time `json:"last_seen"`
	IsActive    bool      `json:"is_active"`
	Role        string    `json:"role"` // owner, editor, viewer
}

// Cursor represents a user's cursor position
type Cursor struct {
	UserID    uuid.UUID `json:"user_id"`
	Position  Position  `json:"position"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Selection represents a user's selection
type Selection struct {
	UserID uuid.UUID `json:"user_id"`
	Start  Position  `json:"start"`
	End    Position  `json:"end"`
}

// Position in the document
type Position struct {
	Section   string `json:"section"` // header, body, footer, parameter
	ElementID string `json:"element_id,omitempty"`
	Offset    int    `json:"offset"`
}

// DocumentState represents the current document state
type DocumentState struct {
	Content   json.RawMessage `json:"content"`
	Version   int64           `json:"version"`
	UpdatedAt time.Time       `json:"updated_at"`
	UpdatedBy uuid.UUID       `json:"updated_by"`
}

// ============================================================================
// OPERATIONAL TRANSFORMATION
// ============================================================================

// Operation represents an edit operation
type Operation struct {
	ID        uuid.UUID       `json:"id"`
	Type      OpType          `json:"type"`
	UserID    uuid.UUID       `json:"user_id"`
	Position  Position        `json:"position"`
	Payload   json.RawMessage `json:"payload"`
	Version   int64           `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
}

// OpType defines operation types
type OpType string

const (
	OpInsert  OpType = "insert"
	OpDelete  OpType = "delete"
	OpReplace OpType = "replace"
	OpMove    OpType = "move"
	OpStyle   OpType = "style"
	OpComment OpType = "comment"
)

// OperationalTransform transforms operations for concurrent editing
type OperationalTransform struct {
	mutex sync.RWMutex
}

// Transform adjusts an operation based on concurrent operations
func (ot *OperationalTransform) Transform(op *Operation, against []*Operation) *Operation {
	ot.mutex.RLock()
	defer ot.mutex.RUnlock()

	result := *op

	for _, other := range against {
		if other.Version >= op.Version && other.UserID != op.UserID {
			result = *ot.transformPair(&result, other)
		}
	}

	return &result
}

func (ot *OperationalTransform) transformPair(op *Operation, other *Operation) *Operation {
	result := *op

	// Position adjustment based on operation type
	if op.Position.Section == other.Position.Section {
		switch other.Type {
		case OpInsert:
			if op.Position.Offset >= other.Position.Offset {
				result.Position.Offset++
			}
		case OpDelete:
			if op.Position.Offset > other.Position.Offset {
				result.Position.Offset--
			}
		}
	}

	return &result
}

// ============================================================================
// COLLABORATION HUB
// ============================================================================

// CollaborationHub manages all collaboration sessions
type CollaborationHub struct {
	sessions map[uuid.UUID]*CollaborationSession // reportID -> session
	clients  map[uuid.UUID][]*CollabClient       // sessionID -> clients
	ot       *OperationalTransform
	mutex    sync.RWMutex

	// Event channels
	broadcast  chan *CollabEvent
	register   chan *CollabClient
	unregister chan *CollabClient

	stopCh chan struct{}
}

// CollabClient represents a connected client
type CollabClient struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	Send      chan *CollabEvent
}

// CollabEvent represents a collaboration event
type CollabEvent struct {
	Type      CollabEventType        `json:"type"`
	SessionID uuid.UUID              `json:"session_id"`
	UserID    uuid.UUID              `json:"user_id"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// CollabEventType defines event types
type CollabEventType string

const (
	CollabEventJoin      CollabEventType = "join"
	CollabEventLeave     CollabEventType = "leave"
	CollabEventCursor    CollabEventType = "cursor"
	CollabEventSelection CollabEventType = "selection"
	CollabEventOperation CollabEventType = "operation"
	CollabEventSync      CollabEventType = "sync"
	CollabEventComment   CollabEventType = "comment"
	CollabEventLock      CollabEventType = "lock"
	CollabEventUnlock    CollabEventType = "unlock"
	CollabEventPresence  CollabEventType = "presence"
)

// NewCollaborationHub creates a collaboration hub
func NewCollaborationHub() *CollaborationHub {
	hub := &CollaborationHub{
		sessions:   make(map[uuid.UUID]*CollaborationSession),
		clients:    make(map[uuid.UUID][]*CollabClient),
		ot:         &OperationalTransform{},
		broadcast:  make(chan *CollabEvent, 1000),
		register:   make(chan *CollabClient),
		unregister: make(chan *CollabClient),
		stopCh:     make(chan struct{}),
	}
	go hub.run()
	return hub
}

func (hub *CollaborationHub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.addClient(client)
		case client := <-hub.unregister:
			hub.removeClient(client)
		case event := <-hub.broadcast:
			hub.broadcastEvent(event)
		case <-hub.stopCh:
			return
		}
	}
}

func (hub *CollaborationHub) addClient(client *CollabClient) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	hub.clients[client.SessionID] = append(hub.clients[client.SessionID], client)

	// Update participant status
	if session, ok := hub.sessions[client.SessionID]; ok {
		for _, p := range session.Participants {
			if p.UserID == client.UserID {
				p.IsActive = true
				p.LastSeen = time.Now()
				break
			}
		}
	}
}

func (hub *CollaborationHub) removeClient(client *CollabClient) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	clients := hub.clients[client.SessionID]
	for i, c := range clients {
		if c == client {
			hub.clients[client.SessionID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	close(client.Send)
}

func (hub *CollaborationHub) broadcastEvent(event *CollabEvent) {
	hub.mutex.RLock()
	clients := hub.clients[event.SessionID]
	hub.mutex.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- event:
		default:
			// Client buffer full, skip
		}
	}
}

// JoinSession joins or creates a collaboration session
func (hub *CollaborationHub) JoinSession(
	ctx context.Context,
	tenantID uuid.UUID,
	reportID uuid.UUID,
	userID uuid.UUID,
	displayName string,
) (*CollaborationSession, *CollabClient, error) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	// Find or create session
	session, ok := hub.sessions[reportID]
	if !ok {
		session = &CollaborationSession{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ReportID:     reportID,
			CreatedBy:    userID,
			CreatedAt:    time.Now(),
			Participants: make([]*Participant, 0),
			Cursors:      make(map[uuid.UUID]*Cursor),
			Selections:   make(map[uuid.UUID]*Selection),
			Version:      0,
			LastActivity: time.Now(),
		}
		hub.sessions[reportID] = session
	}

	// Add participant
	participant := &Participant{
		UserID:      userID,
		DisplayName: displayName,
		Color:       hub.assignColor(len(session.Participants)),
		JoinedAt:    time.Now(),
		LastSeen:    time.Now(),
		IsActive:    true,
		Role:        "editor",
	}
	session.Participants = append(session.Participants, participant)

	// Create client
	client := &CollabClient{
		SessionID: session.ID,
		UserID:    userID,
		Send:      make(chan *CollabEvent, 256),
	}

	// Register client
	hub.register <- client

	// Broadcast join event
	hub.broadcast <- &CollabEvent{
		Type:      CollabEventJoin,
		SessionID: session.ID,
		UserID:    userID,
		Payload: map[string]interface{}{
			"participant": participant,
		},
		Timestamp: time.Now(),
	}

	return session, client, nil
}

// LeaveSession leaves a collaboration session
func (hub *CollaborationHub) LeaveSession(client *CollabClient) {
	hub.unregister <- client

	hub.broadcast <- &CollabEvent{
		Type:      CollabEventLeave,
		SessionID: client.SessionID,
		UserID:    client.UserID,
		Payload:   nil,
		Timestamp: time.Now(),
	}
}

// UpdateCursor updates a user's cursor position
func (hub *CollaborationHub) UpdateCursor(sessionID uuid.UUID, userID uuid.UUID, position Position) {
	hub.mutex.Lock()

	var foundSession *CollaborationSession
	for _, session := range hub.sessions {
		if session.ID == sessionID {
			session.Cursors[userID] = &Cursor{
				UserID:    userID,
				Position:  position,
				UpdatedAt: time.Now(),
			}
			foundSession = session
			break
		}
	}
	hub.mutex.Unlock()

	if foundSession != nil {
		hub.broadcast <- &CollabEvent{
			Type:      CollabEventCursor,
			SessionID: sessionID,
			UserID:    userID,
			Payload: map[string]interface{}{
				"position": position,
			},
			Timestamp: time.Now(),
		}
	}
}

// ApplyOperation applies an operation and broadcasts it
func (hub *CollaborationHub) ApplyOperation(sessionID uuid.UUID, op *Operation) error {
	hub.mutex.Lock()

	var session *CollaborationSession
	for _, s := range hub.sessions {
		if s.ID == sessionID {
			session = s
			break
		}
	}

	if session == nil {
		hub.mutex.Unlock()
		return fmt.Errorf("session not found")
	}

	// Transform against concurrent operations (simplified)
	op.Version = session.Version + 1
	session.Version = op.Version
	session.LastActivity = time.Now()

	hub.mutex.Unlock()

	hub.broadcast <- &CollabEvent{
		Type:      CollabEventOperation,
		SessionID: sessionID,
		UserID:    op.UserID,
		Payload: map[string]interface{}{
			"operation": op,
		},
		Timestamp: time.Now(),
	}

	return nil
}

func (hub *CollaborationHub) assignColor(index int) string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
		"#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
		"#BB8FCE", "#85C1E9", "#F8B500", "#00CED1",
	}
	return colors[index%len(colors)]
}

// GetSession retrieves a session
func (hub *CollaborationHub) GetSession(reportID uuid.UUID) *CollaborationSession {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()
	return hub.sessions[reportID]
}

// Stop shuts down the hub
func (hub *CollaborationHub) Stop() {
	close(hub.stopCh)
}

// ============================================================================
// COMMENTS & ANNOTATIONS
// ============================================================================

// Comment represents a comment on a report
type Comment struct {
	ID       uuid.UUID  `json:"id"`
	TenantID uuid.UUID  `json:"tenant_id"`
	ReportID uuid.UUID  `json:"report_id"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"` // For threads
	UserID   uuid.UUID  `json:"user_id"`
	UserName string     `json:"user_name"`
	Content  string     `json:"content"`

	// Anchor information
	Anchor *CommentAnchor `json:"anchor,omitempty"`

	// Status
	Status     CommentStatus `json:"status"`
	ResolvedBy *uuid.UUID    `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`

	// Mentions
	Mentions []uuid.UUID `json:"mentions,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Nested replies (for API responses)
	Replies []*Comment `json:"replies,omitempty"`
}

// CommentAnchor specifies where a comment is anchored
type CommentAnchor struct {
	ElementID   string     `json:"element_id"`
	ElementType string     `json:"element_type"`
	Selection   *Selection `json:"selection,omitempty"`
}

// CommentStatus represents comment state
type CommentStatus string

const (
	CommentStatusOpen     CommentStatus = "open"
	CommentStatusResolved CommentStatus = "resolved"
	CommentStatusArchived CommentStatus = "archived"
)

// CommentService manages comments
type CommentService struct {
	comments map[uuid.UUID][]*Comment // reportID -> comments
	mutex    sync.RWMutex
}

// NewCommentService creates a comment service
func NewCommentService() *CommentService {
	return &CommentService{
		comments: make(map[uuid.UUID][]*Comment),
	}
}

// AddComment adds a comment
func (cs *CommentService) AddComment(comment *Comment) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	comment.ID = uuid.New()
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	comment.Status = CommentStatusOpen

	cs.comments[comment.ReportID] = append(cs.comments[comment.ReportID], comment)
	return nil
}

// GetComments retrieves comments for a report
func (cs *CommentService) GetComments(reportID uuid.UUID) []*Comment {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	comments := cs.comments[reportID]

	// Build thread structure
	rootComments := make([]*Comment, 0)
	replyMap := make(map[uuid.UUID][]*Comment)

	for _, c := range comments {
		if c.ParentID == nil {
			rootComments = append(rootComments, c)
		} else {
			replyMap[*c.ParentID] = append(replyMap[*c.ParentID], c)
		}
	}

	// Attach replies
	for _, root := range rootComments {
		root.Replies = replyMap[root.ID]
	}

	return rootComments
}

// ResolveComment marks a comment as resolved
func (cs *CommentService) ResolveComment(commentID uuid.UUID, resolvedBy uuid.UUID) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for _, comments := range cs.comments {
		for _, c := range comments {
			if c.ID == commentID {
				c.Status = CommentStatusResolved
				c.ResolvedBy = &resolvedBy
				now := time.Now()
				c.ResolvedAt = &now
				c.UpdatedAt = now
				return nil
			}
		}
	}

	return fmt.Errorf("comment not found")
}

// ============================================================================
// SHARING & PERMISSIONS
// ============================================================================

// ShareConfig defines sharing settings
type ShareConfig struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	ReportID uuid.UUID `json:"report_id"`
	SharedBy uuid.UUID `json:"shared_by"`

	// Share type
	ShareType ShareType `json:"share_type"`

	// Recipient (for direct shares)
	RecipientID   *uuid.UUID `json:"recipient_id,omitempty"`
	RecipientType string     `json:"recipient_type,omitempty"` // user, team, role

	// Link sharing
	ShareLink  string     `json:"share_link,omitempty"`
	LinkExpiry *time.Time `json:"link_expiry,omitempty"`
	Password   *string    `json:"-"` // Hashed password

	// Permissions
	Permission SharePermission `json:"permission"`

	// Restrictions
	AllowExport bool `json:"allow_export"`
	AllowPrint  bool `json:"allow_print"`
	Watermark   bool `json:"watermark"`

	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ShareType defines how something is shared
type ShareType string

const (
	ShareTypeDirect ShareType = "direct" // Direct to user/team
	ShareTypeLink   ShareType = "link"   // Anyone with link
	ShareTypePublic ShareType = "public" // Public access
)

// SharePermission defines what recipients can do
type SharePermission string

const (
	SharePermissionView    SharePermission = "view"
	SharePermissionComment SharePermission = "comment"
	SharePermissionEdit    SharePermission = "edit"
	SharePermissionAdmin   SharePermission = "admin"
)

// SharingService manages report sharing
type SharingService struct {
	shares map[uuid.UUID][]*ShareConfig // reportID -> shares
	mutex  sync.RWMutex
}

// NewSharingService creates a sharing service
func NewSharingService() *SharingService {
	return &SharingService{
		shares: make(map[uuid.UUID][]*ShareConfig),
	}
}

// Share creates a share
func (ss *SharingService) Share(config *ShareConfig) (*ShareConfig, error) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	config.ID = uuid.New()
	config.CreatedAt = time.Now()

	if config.ShareType == ShareTypeLink {
		config.ShareLink = ss.generateShareLink()
	}

	ss.shares[config.ReportID] = append(ss.shares[config.ReportID], config)
	return config, nil
}

// GetShares retrieves shares for a report
func (ss *SharingService) GetShares(reportID uuid.UUID) []*ShareConfig {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	return ss.shares[reportID]
}

// GetUserPermission checks what permission a user has
func (ss *SharingService) GetUserPermission(reportID uuid.UUID, userID uuid.UUID) SharePermission {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	for _, share := range ss.shares[reportID] {
		if share.RecipientID != nil && *share.RecipientID == userID {
			// Check expiry
			if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
				continue
			}
			return share.Permission
		}
	}

	return "" // No permission
}

// Revoke removes a share
func (ss *SharingService) Revoke(shareID uuid.UUID) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	for reportID, shares := range ss.shares {
		for i, share := range shares {
			if share.ID == shareID {
				ss.shares[reportID] = append(shares[:i], shares[i+1:]...)
				return nil
			}
		}
	}

	return fmt.Errorf("share not found")
}

func (ss *SharingService) generateShareLink() string {
	return fmt.Sprintf("share_%s", uuid.New().String()[:8])
}

// ============================================================================
// VERSION HISTORY
// ============================================================================

// ReportVersion represents a version of a report
type ReportVersion struct {
	ID            uuid.UUID       `json:"id"`
	ReportID      uuid.UUID       `json:"report_id"`
	Version       int             `json:"version"`
	Content       json.RawMessage `json:"content"`
	ChangeType    string          `json:"change_type"` // create, update, restore
	ChangeSummary string          `json:"change_summary"`
	CreatedBy     uuid.UUID       `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`

	// For comparison
	Diff *VersionDiff `json:"diff,omitempty"`
}

// VersionDiff represents changes between versions
type VersionDiff struct {
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// VersionService manages report versions
type VersionService struct {
	versions map[uuid.UUID][]*ReportVersion // reportID -> versions
	mutex    sync.RWMutex
}

// NewVersionService creates a version service
func NewVersionService() *VersionService {
	return &VersionService{
		versions: make(map[uuid.UUID][]*ReportVersion),
	}
}

// SaveVersion saves a new version
func (vs *VersionService) SaveVersion(version *ReportVersion) error {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	versions := vs.versions[version.ReportID]
	version.ID = uuid.New()
	version.Version = len(versions) + 1
	version.CreatedAt = time.Now()

	vs.versions[version.ReportID] = append(versions, version)
	return nil
}

// GetVersions retrieves version history
func (vs *VersionService) GetVersions(reportID uuid.UUID, limit int) []*ReportVersion {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	versions := vs.versions[reportID]
	if limit > 0 && limit < len(versions) {
		return versions[len(versions)-limit:]
	}
	return versions
}

// GetVersion retrieves a specific version
func (vs *VersionService) GetVersion(reportID uuid.UUID, version int) *ReportVersion {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	for _, v := range vs.versions[reportID] {
		if v.Version == version {
			return v
		}
	}
	return nil
}

// CompareVersions compares two versions
func (vs *VersionService) CompareVersions(reportID uuid.UUID, v1, v2 int) (*VersionDiff, error) {
	version1 := vs.GetVersion(reportID, v1)
	version2 := vs.GetVersion(reportID, v2)

	if version1 == nil || version2 == nil {
		return nil, fmt.Errorf("version not found")
	}

	// Simplified diff - actual implementation would do deep comparison
	diff := &VersionDiff{
		Added:    []string{},
		Removed:  []string{},
		Modified: []string{},
	}

	return diff, nil
}
