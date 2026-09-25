package datapipeline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeEngine serves /files/read from a fixed NDJSON body and records
// /files/write calls.
func fakeEngine(t *testing.T, ndjson string) (*HTTPFileEngine, *[]string, *[]string) {
	var readURIs, written []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/files/read":
			var fs FileSpec
			_ = json.NewDecoder(r.Body).Decode(&fs)
			readURIs = append(readURIs, fs.URI)
			io.WriteString(w, ndjson)
		case "/files/write":
			b, _ := io.ReadAll(r.Body)
			written = append(written, r.URL.Query().Get("uri")+"|"+string(b))
			json.NewEncoder(w).Encode(map[string]int{"rows": strings.Count(string(b), "\n")})
		default:
			http.Error(w, "nope", 404)
		}
	}))
	t.Cleanup(srv.Close)
	return &HTTPFileEngine{BaseURL: srv.URL, Token: "test-token"}, &readURIs, &written
}

func notNull() *bool { f := false; return &f }

func TestFileSourceAppliesContract(t *testing.T) {
	eng, reads, _ := fakeEngine(t,
		`{"FSYM_ID":"AB12","AUM":"1000.5","INCEPTION":"2020-01-15","EXTRA":"x"}`+"\n"+
			`{"FSYM_ID":"","AUM":"5","INCEPTION":"2020-01-15"}`+"\n"+
			`{"FSYM_ID":"CD34","AUM":"lots","INCEPTION":"2020-01-15"}`+"\n"+
			`{"FSYM_ID":"EF56","AUM":"","INCEPTION":"15/01/2020"}`+"\n")
	n := Node{ID: "f", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "uploads/fs.txt", Format: "csv", Delimiter: "|", Columns: []Column{
		{Name: "FSYM_ID", Type: "string", Nullable: notNull()},
		{Name: "AUM", Type: "float"},
		{Name: "INCEPTION", Type: "date"},
	}})}
	src, err := newFileSource(n, eng)
	if err != nil {
		t.Fatal(err)
	}
	var rows []Row
	var rej []Reject
	err = src.Stream(context.Background(), &RunContext{TenantID: "t1"}, 2, func(rs []Row, rj []Reject) error {
		rows, rej = append(rows, rs...), append(rej, rj...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if (*reads)[0] != "tenants/t1/uploads/fs.txt" {
		t.Errorf("path not tenant-confined: %s", (*reads)[0])
	}
	if len(rows) != 1 || rows[0].Data["AUM"] != 1000.5 || rows[0].Data["EXTRA"] != nil {
		t.Fatalf("typed good row wrong: %+v", rows)
	}
	want := []string{"FSYM_ID is required", `AUM: "lots" is not a number`, `INCEPTION: "15/01/2020" is not a date (YYYY-MM-DD)`}
	if len(rej) != 3 {
		t.Fatalf("rejects: %+v", rej)
	}
	for i, w := range want {
		if rej[i].Reason != w || rej[i].Row.Num != i+2 {
			t.Errorf("reject %d = %d %q, want row %d %q", i, rej[i].Row.Num, rej[i].Reason, i+2, w)
		}
	}
}

func TestFileSourceMissingColumnFailsRun(t *testing.T) {
	eng, _, _ := fakeEngine(t, `{"A":"1"}`+"\n")
	src, _ := newFileSource(Node{ID: "f", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "a.csv", Format: "csv",
		Columns: []Column{{Name: "A", Type: "int"}, {Name: "B", Type: "string"}}})}, eng)
	err := src.Stream(context.Background(), &RunContext{TenantID: "t1"}, 10, func([]Row, []Reject) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "missing column(s) B") {
		t.Fatalf("got %v", err)
	}
	if _, err := newFileSource(Node{ID: "f", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "a.csv", Format: "csv"})}, eng); err == nil {
		t.Error("a run without a contract must be refused")
	}
}

func TestTenantPath(t *testing.T) {
	for _, bad := range []string{"../t2/x.csv", "a/../../b", "s3://bucket/x", "", "./x"} {
		if _, err := tenantPath("t1", bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	if p, _ := tenantPath("t1", "file:///uploads/x.csv"); p != "tenants/t1/uploads/x.csv" {
		t.Errorf("got %s", p)
	}
}

func TestFileSinkPublishesOnlyOnSuccess(t *testing.T) {
	eng, _, written := fakeEngine(t, "")
	n := Node{ID: "o", Type: NodeFileSink, Config: cfg(FileSinkConfig{URI: "exports/out.parquet", Format: "parquet"})}
	for _, runErr := range []error{io.ErrUnexpectedEOF, nil} {
		p, _ := newFileSink(n, eng)
		ctx := context.Background()
		if err := p.Open(ctx, &RunContext{TenantID: "t1"}); err != nil {
			t.Fatal(err)
		}
		p.Process(ctx, []Row{{Num: 1, Data: map[string]any{"a": 1}}, {Num: 2, Data: map[string]any{"a": 2}}})
		if err := p.Close(ctx, runErr); err != nil {
			t.Fatal(err)
		}
	}
	if len(*written) != 1 || !strings.HasPrefix((*written)[0], "tenants/t1/exports/out.parquet|{\"a\":1}\n{\"a\":2}\n") {
		t.Fatalf("writes: %q", *written)
	}
}
