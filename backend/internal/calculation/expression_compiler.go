package calculation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type ExpressionCompileRequest struct {
	Formula  string `json:"formula"`
	BOName   string `json:"boName"`
	TenantID string `json:"tenantId"`
}

type ExpressionCompileResponse struct {
	Valid          bool     `json:"valid"`
	ExtractedFields []string `json:"extractedFields"`
	CompiledSQL    string   `json:"compiledSql"`
	Error          string   `json:"error,omitempty"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CompileExpression(ctx context.Context, req ExpressionCompileRequest) (ExpressionCompileResponse, error) {
	if strings.TrimSpace(req.Formula) == "" {
		return ExpressionCompileResponse{Valid: false, Error: "Formula cannot be empty"}, nil
	}

	// Extract field tokens inside square brackets [...]
	re := regexp.MustCompile(`\[([^\]]+)\]`)
	matches := re.FindAllStringSubmatch(req.Formula, -1)

	var fields []string
	for _, m := range matches {
		if len(m) > 1 {
			fields = append(fields, m[1])
		}
	}

	// Replace bracket syntax [Account.market_value] with physical SQL projection
	sqlExpr := req.Formula
	for _, f := range fields {
		parts := strings.Split(f, ".")
		colName := f
		if len(parts) > 1 {
			colName = parts[1]
		}
		sqlExpr = strings.ReplaceAll(sqlExpr, fmt.Sprintf("[%s]", f), colName)
	}

	return ExpressionCompileResponse{
		Valid:           true,
		ExtractedFields: fields,
		CompiledSQL:     sqlExpr,
	}, nil
}

func (s *Service) CompileExpressionHandler(w http.ResponseWriter, r *http.Request) {
	var req ExpressionCompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		req.TenantID = claims.TenantID
	}

	res, err := s.CompileExpression(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to compile expression: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
