package chat_history

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureSession(
	ctx context.Context,
	tenantID, conversationID uuid.UUID,
	agentID, userID string,
	userEmail *string,
	viewType string,
	embedded bool,
	embedSurface *string,
) (uuid.UUID, error) {
	return s.repo.EnsureSession(ctx, tenantID, conversationID, agentID, userID, userEmail, viewType, embedded, embedSurface)
}

func (s *Service) AppendMessage(ctx context.Context, tenantID, sessionID uuid.UUID, msg MessageRecord) (*MessageRecord, error) {
	if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" && msg.Role != "tool" {
		return nil, ErrInvalidRole
	}
	return s.repo.AppendMessage(ctx, tenantID, sessionID, msg)
}

func (s *Service) AppendMessageWithTrace(ctx context.Context, tenantID, sessionID uuid.UUID, msg MessageRecord, traceID, spanID string) (*MessageRecord, error) {
	if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" && msg.Role != "tool" {
		return nil, ErrInvalidRole
	}
	return s.repo.RecordMessageDirect(ctx, tenantID, sessionID, msg, traceID, spanID)
}

func (s *Service) SubmitFeedback(ctx context.Context, tenantID, sessionID uuid.UUID, userID string, isGlobalAdmin bool, score int16, comment *string) error {
	if score != 1 && score != -1 {
		return ErrInvalidFeedback
	}
	return s.repo.SetFeedback(ctx, tenantID, sessionID, userID, isGlobalAdmin, score, comment)
}

func (s *Service) EndSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	return s.repo.EndSession(ctx, tenantID, sessionID)
}

func (s *Service) ListSessions(ctx context.Context, f ListFilters) ([]SessionRecord, int, error) {
	return s.repo.ListSessions(ctx, f)
}

func (s *Service) GetSessionDetail(ctx context.Context, tenantID, sessionID uuid.UUID, isGlobalAdmin bool) (*SessionDetail, error) {
	return s.repo.GetSession(ctx, tenantID, sessionID, isGlobalAdmin)
}

func (s *Service) GetMessagesOnly(ctx context.Context, tenantID, sessionID uuid.UUID) ([]MessageRecord, error) {
	return s.repo.GetMessages(ctx, tenantID, sessionID)
}

func (s *Service) StampSessionTrace(ctx context.Context, tenantID, sessionID uuid.UUID, traceID string) error {
	return s.repo.StampSessionTrace(ctx, tenantID, sessionID, traceID)
}