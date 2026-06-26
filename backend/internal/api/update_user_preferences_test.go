package api

import (
	"bytes"
	// "database/sql" - kept for compatibility with sqlmock returns
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	imodels "github.com/hondyman/semlayer/backend/internal/models"
)

func TestUpdateUserPreferences_Success(t *testing.T) {
	srv, mock := newServerWithMockDB(t)
	defer srv.DB.Close()

	userId := "user-1"
	lang := "es"

	// Expect update
	mock.ExpectExec(`UPDATE\s+public\.users\s+SET\s+language`).
		WithArgs(lang, userId).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"language": lang})
	req := httptest.NewRequest("PUT", "/api/users/"+userId+"/preferences", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// attach user in context so auth check passes
	req = req.WithContext(auth.SetUserInContext(req.Context(), imodels.User{ID: userId}))
	// set chi route param so chi.URLParam will return the value
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", userId)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	srv.updateUserPreferences(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}

	// parse response
	var out map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["language"] != lang {
		t.Fatalf("expected language %s got %s", lang, out["language"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateUserPreferences_Unauthorized(t *testing.T) {
	srv, _ := newServerWithMockDB(t)
	defer srv.DB.Close()

	userId := "user-1"
	other := "user-2"
	body, _ := json.Marshal(map[string]string{"language": "fr"})
	req := httptest.NewRequest("PUT", "/api/users/"+userId+"/preferences", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// set context to a different user
	req = req.WithContext(auth.SetUserInContext(req.Context(), imodels.User{ID: other}))
	// set chi route param so chi.URLParam will return the value
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("userId", userId)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx2))

	rr := httptest.NewRecorder()
	srv.updateUserPreferences(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d body=%s", rr.Code, rr.Body.String())
	}
}
