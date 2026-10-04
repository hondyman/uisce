package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/models"
	"github.com/stretchr/testify/require"
)

// Profiling a scan's data is optional. These pin how the request says so, and that a scan started without an opinion
// is told nothing, so the datasource's own setting (or the default) decides.

type optionsFake struct {
	mu   sync.Mutex
	opts []metadata.ScanOptions
}

func (f *optionsFake) record(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, metadata.ScanOptionsFrom(ctx))
}

func (f *optionsFake) ScanDatasources(ctx context.Context, _ *uuid.UUID) ([]metadata.ScanResult, error) {
	f.record(ctx)
	return []metadata.ScanResult{{DatasourceID: uuid.New(), Name: "ds", Success: true}}, nil
}

func (f *optionsFake) ScanWithProgress(ctx context.Context, _ *uuid.UUID, _ chan<- models.ScanProgress) ([]metadata.ScanResult, error) {
	f.record(ctx)
	return nil, nil
}

func scanReq(body, query string) *http.Request {
	req := newTestScanRequest(http.MethodPost, "/api/catalog/scan"+query)
	req.Body = io.NopCloser(strings.NewReader(body))
	return req
}

func TestHandleCatalogScan_ProfileDataIsOptionalAndCarriedToTheScan(t *testing.T) {
	f, t2 := false, true
	for name, tc := range map[string]struct {
		body, query string
		want        *bool
	}{
		"no preference: the scan is told nothing": {"", "", nil},
		"body, profile off":                       {`{"profile_data":false}`, "", &f},
		"body, profile on":                        {`{"profile_data":true}`, "", &t2},
		"hasura input, profile off":               {`{"input":{"profile_data":false}}`, "", &f},
		"query, profile off":                      {"", "?profile_data=false", &f},
		"query beats the body":                    {`{"profile_data":true}`, "?profile_data=false", &f},
		"hasura input beats the body":             {`{"profile_data":true,"input":{"profile_data":false}}`, "", &f},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &optionsFake{}
			h := NewCatalogScanHandler(fake, SecurityContextDeps{Resolver: &mockCatalogScanDatasourceResolver{}})
			w := httptest.NewRecorder()
			h.HandleCatalogScan(w, scanReq(tc.body, tc.query))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Len(t, fake.opts, 1)
			if tc.want == nil {
				require.Nil(t, fake.opts[0].ProfileData)
			} else {
				require.NotNil(t, fake.opts[0].ProfileData)
				require.Equal(t, *tc.want, *fake.opts[0].ProfileData)
			}
		})
	}
}

func TestHandleCatalogScan_ABadProfileDataIsRefusedBeforeAnythingIsScanned(t *testing.T) {
	for name, tc := range map[string]struct{ body, query string }{
		"query not a boolean": {"", "?profile_data=maybe"},
		"body not a boolean":  {`{"profile_data":"no"}`, ""},
		"input not a boolean": {`{"input":{"profile_data":1}}`, ""},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &optionsFake{}
			h := NewCatalogScanHandler(fake, SecurityContextDeps{Resolver: &mockCatalogScanDatasourceResolver{}})
			w := httptest.NewRecorder()
			h.HandleCatalogScan(w, scanReq(tc.body, tc.query))
			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Empty(t, fake.opts, "no scan is started on input that cannot be understood")
		})
	}
}

func TestHandleScanStream_ProfileDataIsCarriedAndABadValueIsAPlain400(t *testing.T) {
	ds := uuid.NewString()
	fake := &optionsFake{}
	h := NewCatalogScanHandler(fake, SecurityContextDeps{Resolver: &mockCatalogScanDatasourceResolver{}})

	w := httptest.NewRecorder()
	h.HandleScanStream(w, httptest.NewRequest(http.MethodGet, "/api/catalog/scan/stream?datasource_id="+ds+"&profile_data=false", nil))
	require.Len(t, fake.opts, 1)
	require.NotNil(t, fake.opts[0].ProfileData)
	require.False(t, *fake.opts[0].ProfileData)

	w = httptest.NewRecorder()
	h.HandleScanStream(w, httptest.NewRequest(http.MethodGet, "/api/catalog/scan/stream?datasource_id="+ds+"&profile_data=nope", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"), "refused before the event stream opened")
	require.Len(t, fake.opts, 1)
}
