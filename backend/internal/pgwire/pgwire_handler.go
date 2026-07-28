package pgwire

import (
	"context"
	"fmt"
	"net"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"go.uber.org/zap"
)

// SessionHandler processes individual client connections over PostgreSQL protocol format.
type SessionHandler struct {
	server   *PGWireServer
	tenantID string
	userID   string
}

// ServeConn handles wire protocol handshakes and mock query execution.
func (h *SessionHandler) ServeConn(ctx context.Context, conn net.Conn) {
	// Simple mock mock session execution loop for testing wire protocol translation
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return
	}

	// Respond with simple ready-for-query signal or process query if text command
	queryStr := string(buf[:n])
	req, err := TranslateSimpleQuery(queryStr, h.tenantID)
	if err != nil {
		h.server.cfg.Logger.Debug("Query translation note", zap.Error(err))
		return
	}

	if h.server.cfg.SecurityInterceptor != nil {
		if err := h.server.cfg.SecurityInterceptor.InterceptSemanticRequest(ctx, h.tenantID, h.userID, req); err != nil {
			h.server.cfg.Logger.Warn("ABAC Interceptor blocked pgwire query", zap.Error(err))
			return
		}
	}

	if h.server.cfg.SQLGenerator != nil {
		compiledSQL, args, err := h.server.cfg.SQLGenerator.GenerateFromSemanticRequest(*req)
		if err != nil {
			h.server.cfg.Logger.Error("BOSQLGenerator failed to compile pgwire query", zap.Error(err))
			return
		}
		h.server.cfg.Logger.Info("pgwire query converged into BOSQLGenerator", zap.String("sql", compiledSQL), zap.Any("args", args))
	}
}

// ExecuteSemanticAST runs a pre-parsed SemanticSQLGenerationRequest through the core generator.
func (h *SessionHandler) ExecuteSemanticAST(ctx context.Context, req boresolver.SemanticSQLGenerationRequest) (string, []interface{}, error) {
	if h.server.cfg.SecurityInterceptor != nil {
		if err := h.server.cfg.SecurityInterceptor.InterceptSemanticRequest(ctx, h.tenantID, h.userID, &req); err != nil {
			return "", nil, fmt.Errorf("security policy violation: %w", err)
		}
	}

	if h.server.cfg.SQLGenerator == nil {
		return "", nil, fmt.Errorf("SQL generator uninitialized")
	}

	return h.server.cfg.SQLGenerator.GenerateFromSemanticRequest(req)
}
