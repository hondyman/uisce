package chat_history

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

func (s *Service) StreamCSV(ctx context.Context, w io.Writer, f ListFilters) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := []string{
		"session_id", "started_at", "duration_seconds", "tenant_id", "user_id",
		"user_email", "agent_id", "view_type", "embedded", "embed_surface",
		"feedback_score", "feedback_comment", "message_count", "first_message", "last_message",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	f.Limit = 100
	f.Offset = 0
	f.SkipCount = true

	truncate := func(s *string, limit int) string {
		if s == nil {
			return ""
		}
		r := []rune(*s)
		if len(r) <= limit {
			return *s
		}
		if limit > 3 {
			return string(r[:limit-3]) + "..."
		}
		return string(r[:limit])
	}

	for {
		sessions, _, err := s.repo.ListSessions(ctx, f)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			break
		}

		for _, sess := range sessions {
			durationSec := ""
			if sess.EndedAt != nil {
				durationSec = fmt.Sprintf("%.0f", sess.EndedAt.Sub(sess.StartedAt).Seconds())
			}
			userEmail := ""
			if sess.UserEmail != nil {
				userEmail = *sess.UserEmail
			}
			embedSurface := ""
			if sess.EmbedSurface != nil {
				embedSurface = *sess.EmbedSurface
			}
			feedbackScore := ""
			if sess.FeedbackScore != nil {
				feedbackScore = fmt.Sprintf("%d", *sess.FeedbackScore)
			}
			feedbackComment := ""
			if sess.FeedbackComment != nil {
				feedbackComment = *sess.FeedbackComment
			}

			row := []string{
				sess.ID.String(),
				sess.StartedAt.Format(time.RFC3339),
				durationSec,
				sess.TenantID.String(),
				sess.UserID,
				userEmail,
				sess.AgentID,
				sess.ViewType,
				fmt.Sprintf("%t", sess.Embedded),
				embedSurface,
				feedbackScore,
				feedbackComment,
				fmt.Sprintf("%d", sess.MessageCount),
				truncate(sess.FirstMessage, 200),
				truncate(sess.LastMessage, 200),
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			return err
		}

		if len(sessions) < f.Limit {
			break
		}
		f.Offset += f.Limit
	}

	return nil
}