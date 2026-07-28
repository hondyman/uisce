package pgwire

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"go.uber.org/zap"
)

// SQLGenerator defines the subset of BOSQLGenerator required by the proxy.
type SQLGenerator interface {
	GenerateFromSemanticRequest(req boresolver.SemanticSQLGenerationRequest) (string, []interface{}, error)
}

// SecurityInterceptor defines the security boundary for ABAC queries.
type SecurityInterceptor interface {
	InterceptSemanticRequest(ctx context.Context, tenantID, userID string, req *boresolver.SemanticSQLGenerationRequest) error
}

// Config configures the Postgres Wire Server.
type Config struct {
	Addr                string
	DefaultTenantID     string
	SQLGenerator        SQLGenerator
	SecurityInterceptor SecurityInterceptor
	Logger              *zap.Logger
}

// PGWireServer represents a headless Postgres Wire Protocol proxy server.
type PGWireServer struct {
	cfg      Config
	listener net.Listener
	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
}

// NewPGWireServer creates a new PGWireServer instance.
func NewPGWireServer(cfg Config) *PGWireServer {
	if cfg.Addr == "" {
		cfg.Addr = ":5433"
	}
	if cfg.DefaultTenantID == "" {
		cfg.DefaultTenantID = "core"
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &PGWireServer{
		cfg: cfg,
	}
}

// Start begins listening and serving standard PostgreSQL protocol clients.
func (s *PGWireServer) Start(parentCtx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("pgwire server already running")
	}

	listener, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.Addr, err)
	}
	s.listener = listener
	s.running = true

	ctx, cancel := context.WithCancel(parentCtx)
	s.cancel = cancel
	s.mu.Unlock()

	s.cfg.Logger.Info("Postgres Wire Protocol Proxy started", zap.String("addr", s.cfg.Addr))

	go s.acceptLoop(ctx)
	return nil
}

// Stop gracefully shuts down the Postgres wire proxy.
func (s *PGWireServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.cfg.Logger.Info("Postgres Wire Protocol Proxy stopped")
	return nil
}

func (s *PGWireServer) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				s.cfg.Logger.Error("Error accepting pgwire connection", zap.Error(err))
				continue
			}
		}

		go s.handleConnection(ctx, conn)
	}
}

func (s *PGWireServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	handler := &SessionHandler{
		server:   s,
		tenantID: s.cfg.DefaultTenantID,
		userID:   "pgwire_user",
	}

	s.cfg.Logger.Debug("Handled pgwire connection established", zap.String("remote_addr", conn.RemoteAddr().String()))
	handler.ServeConn(ctx, conn)
}

// TranslateSimpleQuery parses incoming SQL from BI tools into a SemanticSQLGenerationRequest AST.
func TranslateSimpleQuery(query string, tenantID string) (*boresolver.SemanticSQLGenerationRequest, error) {
	cleanQuery := strings.TrimSpace(query)
	cleanQuery = strings.TrimSuffix(cleanQuery, ";")

	// Intercept standard metadata reflection queries issued by Tableau/Excel
	if strings.Contains(strings.ToLower(cleanQuery), "information_schema") ||
		strings.Contains(strings.ToLower(cleanQuery), "pg_catalog") {
		return nil, fmt.Errorf("METADATA_QUERY_HANDLED")
	}

	// Basic AST SQL parser for standard "SELECT col1, col2 FROM \"BusinessObject\" WHERE ..."
	upperQuery := strings.ToUpper(cleanQuery)
	fromIdx := strings.Index(upperQuery, " FROM ")
	if !strings.HasPrefix(upperQuery, "SELECT ") || fromIdx == -1 {
		return nil, fmt.Errorf("unsupported SQL query format for semantic AST conversion: %s", query)
	}

	selectPart := cleanQuery[7:fromIdx]
	restPart := cleanQuery[fromIdx+6:]

	whereIdx := strings.Index(strings.ToUpper(restPart), " WHERE ")
	limitIdx := strings.Index(strings.ToUpper(restPart), " LIMIT ")

	var datasource string
	var wherePart string
	limitVal := 100

	if whereIdx != -1 {
		datasource = strings.TrimSpace(restPart[:whereIdx])
		if limitIdx != -1 && limitIdx > whereIdx {
			wherePart = strings.TrimSpace(restPart[whereIdx+7 : limitIdx])
		} else {
			wherePart = strings.TrimSpace(restPart[whereIdx+7:])
		}
	} else if limitIdx != -1 {
		datasource = strings.TrimSpace(restPart[:limitIdx])
	} else {
		datasource = strings.TrimSpace(restPart)
	}

	// Clean double quotes around business object / table name
	datasource = strings.Trim(datasource, `"`)

	// Extract select terms
	termTokens := strings.Split(selectPart, ",")
	var selectTerms []boresolver.SemanticField
	for _, tok := range termTokens {
		t := strings.TrimSpace(tok)
		t = strings.Trim(t, `"`)
		if t != "" && t != "*" {
			selectTerms = append(selectTerms, boresolver.SemanticField{
				Term:  t,
				Label: t,
			})
		}
	}

	var filters []boresolver.SemanticFilter
	if wherePart != "" {
		// Simple filter extraction
		parts := strings.Split(wherePart, "=")
		if len(parts) == 2 {
			term := strings.Trim(strings.TrimSpace(parts[0]), `"`)
			val := strings.Trim(strings.TrimSpace(parts[1]), "'")
			filters = append(filters, boresolver.SemanticFilter{
				Term:  term,
				Op:    "=",
				Value: val,
			})
		}
	}

	req := &boresolver.SemanticSQLGenerationRequest{
		Datasource: datasource,
		Select:     selectTerms,
		Filters:    filters,
		Limit:      limitVal,
		TenantID:   tenantID,
	}

	return req, nil
}
