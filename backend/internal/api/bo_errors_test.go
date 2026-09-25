package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// The business-object record handlers answer every error through the
// message catalog: no http.Error, no error text written to a response.
func TestBOHandlers_NoRawErrorResponses(t *testing.T) {
	raw := regexp.MustCompile(`http\.Error\(|writeJSONError\(|writeBOWriteError\(|Error:\s*\w+\.Err(or)?\.Error\(\)`)
	for _, f := range []string{"bo_crud_handler.go", "bo_bulk_handler.go", "bo_relationship_records_handler.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if raw.MatchString(line) {
				t.Errorf("%s:%d answers with a raw error; use h.fail with a catalog message: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// A database error in a bulk row never reaches the client.
func TestBulkRowError_HidesCause(t *testing.T) {
	h := &BOCRUDHandler{}
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	code, msg := h.rowError(req, uuid.MustParse(routingTenant), "ref-1", 0, errors.New(`pq: duplicate key value violates unique constraint "secret_idx"`))
	assert.Equal(t, "1-4", code)
	assert.NotContains(t, msg, "secret_idx")
}
