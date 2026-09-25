package datapipeline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// FileEngine is the DataFusion engine's file API (datafusion-engine/src/files.rs).
// Paths are relative to the engine's file root; the pipeline confines every
// path to the tenant's folder (see tenantPath).
type FileEngine interface {
	Profile(ctx context.Context, spec FileSpec, sampleRows int, countRows bool) (*FileProfile, error)
	// Read streams the file as NDJSON (CSV values arrive as strings).
	Read(ctx context.Context, spec FileSpec) (io.ReadCloser, error)
	// Write converts an NDJSON body to the target file, published atomically.
	Write(ctx context.Context, uri, format, delimiter string, ndjson io.Reader) (rows int, err error)
	// Upload stores a file as-is, atomically.
	Upload(ctx context.Context, uri string, body io.Reader) (bytes int64, err error)
	// List returns files under a folder.
	List(ctx context.Context, prefix string) ([]FileEntry, error)
}

// FileEntry is one stored file; Path is relative to the engine root.
type FileEntry struct {
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Modified int64  `json:"modified,omitempty"`
}

// FileSpec is the engine's file descriptor.
type FileSpec struct {
	URI       string   `json:"uri"`
	Format    string   `json:"format"`
	Delimiter string   `json:"delimiter,omitempty"`
	HasHeader bool     `json:"has_header"`
	Columns   []string `json:"columns,omitempty"`
}

// FileProfile is what an analyst sees before building a contract.
type FileProfile struct {
	Columns []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Nullable bool   `json:"nullable"`
	} `json:"columns"`
	Sample   []map[string]any `json:"sample"`
	RowCount *int             `json:"row_count"`
}

// tenantPath confines a user-supplied URI to the tenant's folder under the
// engine root. The engine itself rejects absolute paths and "..".
func tenantPath(tenantID, uri string) (string, error) {
	rel := strings.TrimPrefix(uri, "file://")
	if strings.Contains(uri, "://") && !strings.HasPrefix(uri, "file://") {
		return "", fmt.Errorf("only uploaded files (file://) are supported; got %q", uri)
	}
	rel = strings.TrimLeft(rel, "/")
	if rel == "" {
		return "", fmt.Errorf("file path is required")
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." || seg == "." {
			return "", fmt.Errorf("file path must not contain '.' or '..'")
		}
	}
	if tenantID == "" {
		return "", fmt.Errorf("tenant is required")
	}
	return "tenants/" + tenantID + "/" + rel, nil
}

// HTTPFileEngine calls the DataFusion engine over HTTP.
type HTTPFileEngine struct {
	BaseURL string
	Client  *http.Client
}

func (e *HTTPFileEngine) client() *http.Client {
	if e.Client != nil {
		return e.Client
	}
	return &http.Client{Timeout: 30 * time.Minute}
}

func (e *HTTPFileEngine) post(ctx context.Context, path string, body io.Reader, ctype string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.BaseURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ctype)
	resp, err := e.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("file engine unavailable: %w", err)
	}
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("file engine: %s", strings.TrimSpace(string(msg)))
	}
	return resp, nil
}

func (e *HTTPFileEngine) Profile(ctx context.Context, spec FileSpec, sampleRows int, countRows bool) (*FileProfile, error) {
	b, _ := json.Marshal(struct {
		FileSpec
		SampleRows int  `json:"sample_rows"`
		CountRows  bool `json:"count_rows"`
	}{spec, sampleRows, countRows})
	resp, err := e.post(ctx, "/files/profile", bytes.NewReader(b), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var p FileProfile
	return &p, json.NewDecoder(resp.Body).Decode(&p)
}

func (e *HTTPFileEngine) Read(ctx context.Context, spec FileSpec) (io.ReadCloser, error) {
	b, _ := json.Marshal(spec)
	resp, err := e.post(ctx, "/files/read", bytes.NewReader(b), "application/json")
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (e *HTTPFileEngine) Write(ctx context.Context, uri, format, delimiter string, ndjson io.Reader) (int, error) {
	q := url.Values{"uri": {uri}, "format": {format}}
	if delimiter != "" {
		q.Set("delimiter", delimiter)
	}
	resp, err := e.post(ctx, "/files/write?"+q.Encode(), ndjson, "application/x-ndjson")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Rows int `json:"rows"`
	}
	return out.Rows, json.NewDecoder(resp.Body).Decode(&out)
}

func (e *HTTPFileEngine) Upload(ctx context.Context, uri string, body io.Reader) (int64, error) {
	resp, err := e.post(ctx, "/files/upload?"+url.Values{"uri": {uri}}.Encode(), body, "application/octet-stream")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Bytes int64 `json:"bytes"`
	}
	return out.Bytes, json.NewDecoder(resp.Body).Decode(&out)
}

func (e *HTTPFileEngine) List(ctx context.Context, prefix string) ([]FileEntry, error) {
	b, _ := json.Marshal(map[string]string{"prefix": prefix})
	resp, err := e.post(ctx, "/files/list", bytes.NewReader(b), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []FileEntry
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// TenantPath exposes tenantPath for the API layer.
func TenantPath(tenantID, uri string) (string, error) { return tenantPath(tenantID, uri) }

// TenantRelative strips the tenant folder from an engine path, so analysts
// only ever see their own relative paths.
func TenantRelative(tenantID, p string) string {
	return strings.TrimPrefix(p, "tenants/"+tenantID+"/")
}

// --- file_source ---------------------------------------------------------

type fileSource struct {
	cfg    FileSourceConfig
	engine FileEngine
}

func newFileSource(n Node, e FileEngine) (Source, error) {
	var c FileSourceConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("the file engine is not configured for this environment")
	}
	if len(c.Columns) == 0 {
		return nil, fmt.Errorf("define the file's columns before running (use Preview to infer them)")
	}
	return &fileSource{cfg: c, engine: e}, nil
}

func (s *fileSource) spec(tenantID string) (FileSpec, error) {
	p, err := tenantPath(tenantID, s.cfg.URI)
	if err != nil {
		return FileSpec{}, err
	}
	hasHeader := s.cfg.HasHeader == nil || *s.cfg.HasHeader
	fs := FileSpec{URI: p, Format: s.cfg.Format, Delimiter: s.cfg.Delimiter, HasHeader: hasHeader}
	if !hasHeader {
		for _, c := range s.cfg.Columns {
			fs.Columns = append(fs.Columns, c.Name)
		}
	}
	return fs, nil
}

func (s *fileSource) Stream(ctx context.Context, rc *RunContext, batchSize int, emit func([]Row, []Reject) error) error {
	fs, err := s.spec(rc.TenantID)
	if err != nil {
		return err
	}
	body, err := s.engine.Read(ctx, fs)
	if err != nil {
		return err
	}
	defer body.Close()

	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	var rows []Row
	var rejects []Reject
	flush := func() error {
		if len(rows) == 0 && len(rejects) == 0 {
			return nil
		}
		err := emit(rows, rejects)
		rows, rejects = nil, nil
		return err
	}
	num := 0
	checked := false
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		num++
		raw := map[string]any{}
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()
		if err := dec.Decode(&raw); err != nil {
			return fmt.Errorf("row %d: engine returned invalid JSON: %w", num, err)
		}
		if !checked {
			if err := s.checkHeader(raw); err != nil {
				return err
			}
			checked = true
		}
		data, field, reason := applyContract(s.cfg.Columns, raw)
		r := Row{Num: num, Data: data}
		if reason != "" {
			rejects = append(rejects, Reject{Row: Row{Num: num, Data: raw}, Field: field, Reason: reason})
		} else {
			rows = append(rows, r)
		}
		if len(rows)+len(rejects) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return flush()
}

// checkHeader fails the run when the file lacks a contract column - a
// structural problem, not a bad row. The engine writes explicit nulls, so
// every column of the file is present on every line, whatever the format.
func (s *fileSource) checkHeader(first map[string]any) error {
	var missing []string
	for _, c := range s.cfg.Columns {
		if _, ok := first[c.Name]; !ok {
			missing = append(missing, c.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("the file is missing column(s) %s", strings.Join(missing, ", "))
	}
	return nil
}

// applyContract keeps the contract's columns, typed. It returns the failing
// column and a plain-language reason for the first value that does not fit.
func applyContract(cols []Column, raw map[string]any) (map[string]any, string, string) {
	out := make(map[string]any, len(cols))
	for _, c := range cols {
		v := raw[c.Name]
		if isBlank(v) {
			if c.Nullable != nil && !*c.Nullable {
				return nil, c.Name, fmt.Sprintf("%s is required", c.Name)
			}
			out[c.Name] = nil
			continue
		}
		tv, err := coerce(c.Type, v)
		if err != nil {
			return nil, c.Name, fmt.Sprintf("%s: %v", c.Name, err)
		}
		if len(c.Enum) > 0 && !inEnum(c.Enum, fmt.Sprint(tv)) {
			return nil, c.Name, fmt.Sprintf("%s: %q is not one of %s", c.Name, fmt.Sprint(tv), strings.Join(c.Enum, ", "))
		}
		out[c.Name] = tv
	}
	return out, "", ""
}

func inEnum(enum []string, v string) bool {
	for _, e := range enum {
		if e == v {
			return true
		}
	}
	return false
}

var timestampLayouts = []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"}

func coerce(typ string, v any) (any, error) {
	s := strings.TrimSpace(fmt.Sprint(v))
	switch typ {
	case "string":
		if str, ok := v.(string); ok {
			return str, nil
		}
		return s, nil
	case "int":
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a whole number", s)
		}
		return n, nil
	case "float":
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", s)
		}
		return f, nil
	case "decimal":
		// Kept as text so no precision is lost on the way to NUMERIC columns.
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return nil, fmt.Errorf("%q is not a number", s)
		}
		return s, nil
	case "bool":
		switch strings.ToLower(s) {
		case "true", "t", "yes", "y", "1":
			return true, nil
		case "false", "f", "no", "n", "0":
			return false, nil
		}
		return nil, fmt.Errorf("%q is not true/false", s)
	case "date":
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return nil, fmt.Errorf("%q is not a date (YYYY-MM-DD)", s)
		}
		return s, nil
	case "timestamp":
		for _, l := range timestampLayouts {
			if t, err := time.Parse(l, s); err == nil {
				return t.UTC().Format(time.RFC3339Nano), nil
			}
		}
		return nil, fmt.Errorf("%q is not a timestamp", s)
	}
	return nil, fmt.Errorf("unknown column type %q", typ)
}

// --- file_sink -----------------------------------------------------------

// fileSink spools accepted rows as NDJSON to a local temp file and hands the
// whole file to the engine on a successful Close, so a failed run never
// publishes a partial export.
type fileSink struct {
	cfg    FileSinkConfig
	engine FileEngine
	tenant string
	spool  *os.File
	w      *bufio.Writer
	rows   int
}

func newFileSink(n Node, e FileEngine) (Processor, error) {
	var c FileSinkConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("the file engine is not configured for this environment")
	}
	return &fileSink{cfg: c, engine: e}, nil
}

func (s *fileSink) Open(_ context.Context, rc *RunContext) error {
	if _, err := tenantPath(rc.TenantID, s.cfg.URI); err != nil {
		return err
	}
	s.tenant = rc.TenantID
	if rc.DryRun {
		return nil // spool stays nil: Process only counts, Close publishes nothing
	}
	f, err := os.CreateTemp("", "pipeline-export-*.ndjson")
	if err != nil {
		return err
	}
	s.spool, s.w = f, bufio.NewWriter(f)
	return nil
}

func (s *fileSink) Process(_ context.Context, rows []Row) (Result, error) {
	if s.spool == nil {
		return Result{Out: rows}, nil
	}
	for _, r := range rows {
		b, err := json.Marshal(r.Data)
		if err != nil {
			return Result{}, fmt.Errorf("row %d: %w", r.Num, err)
		}
		s.w.Write(b)
		s.w.WriteByte('\n')
		s.rows++
	}
	return Result{Out: rows}, nil
}

func (s *fileSink) Close(ctx context.Context, runErr error) error {
	if s.spool == nil {
		return nil
	}
	defer os.Remove(s.spool.Name())
	defer s.spool.Close()
	if runErr != nil || s.rows == 0 {
		return nil
	}
	if err := s.w.Flush(); err != nil {
		return err
	}
	if _, err := s.spool.Seek(0, io.SeekStart); err != nil {
		return err
	}
	p, _ := tenantPath(s.tenant, s.cfg.URI)
	n, err := s.engine.Write(ctx, p, s.cfg.Format, s.cfg.Delimiter, s.spool)
	if err != nil {
		return fmt.Errorf("export to %s: %w", s.cfg.URI, err)
	}
	if n != s.rows {
		return fmt.Errorf("export to %s wrote %d rows, expected %d", s.cfg.URI, n, s.rows)
	}
	return nil
}
