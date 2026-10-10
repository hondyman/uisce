// Command uisce-tenant creates a tenant from a few parameters and prompts for what is missing:
//
//	uisce-tenant create --name "XYZ Investments" --region "US East" --product orm --label ABC
//
// That registers the tenant in the region, registers the ORM product, and creates its database as
// <label>_<product> (abc_orm) on the region's Postgres cluster, builds its structure from the gold
// copy and seeds its reference rows. Anything left out is asked for, with the allowed values; the
// plan is shown and confirmed before anything is created. Everything is decided by the server
// (POST /api/system/tenants/provision/describe), so the CLI and any other client check the same
// rules.
//
//	uisce-tenant create [--name N] [--code C] [--region R] [--product P] [--label L]
//	                    [--instance I] [--yes] [--no-prompt] [--no-wait] [--timeout 60m]
//	uisce-tenant status --workflow <id>
//
// Exit codes: 0 done, 1 provisioning failed (and was rolled back), 2 the request is incomplete or
// invalid, 3 timed out waiting, 4 not authorized, 5 bad usage or configuration, 6 other API error.
//
// It authenticates as a Keycloak service account that is a global administrator (client credentials):
//
//	UISCE_URL            e.g. https://uisce.example.com
//	UISCE_TOKEN_URL      e.g. https://keycloak.example.com/realms/uisce/protocol/openid-connect/token
//	UISCE_CLIENT_ID
//	UISCE_CLIENT_SECRET  or UISCE_CLIENT_SECRET_FILE (preferred: a file only the operator can read)
//
// Creating the tenant again with the same parameters finds the run already started; it never starts
// a second one.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/provisioning"
)

const (
	exitOK = iota
	exitFailed
	exitInvalid
	exitTimeout
	exitUnauthorized
	exitUsage
	exitAPI
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr))
}

type config struct {
	baseURL, tokenURL, clientID, clientSecret string
	http                                      *http.Client
	poll                                      time.Duration
}

func run(ctx context.Context, args []string, env func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "create" && args[0] != "status") {
		fmt.Fprintln(stderr, "usage: uisce-tenant create [--name N] [--region R] [--product P] [--label L] [--yes] [--no-prompt]\n       uisce-tenant status --workflow <id>")
		return exitUsage
	}
	cmd := args[0]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "the tenant's name")
	code := fs.String("code", "", "the tenant's code (derived from the name when empty)")
	region := fs.String("region", "", `the region: a code ("us-east-1") or a name ("US East")`)
	product := fs.String("product", "", "the product to register (orm)")
	label := fs.String("label", "", "the label that names the product's database: <label>_<product>")
	instance := fs.String("instance", "", "the instance name (default primary)")
	yes := fs.Bool("yes", false, "create without asking for confirmation")
	noPrompt := fs.Bool("no-prompt", false, "never ask: fail when anything is missing")
	noWait := fs.Bool("no-wait", false, "return once the run has started")
	timeout := fs.Duration("timeout", 60*time.Minute, "how long to wait for the run")
	poll := fs.Duration("poll", 3*time.Second, "how often to read the run's status")
	workflow := fs.String("workflow", "", "status: the workflow id")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	cfg, err := configFrom(env)
	if err != nil {
		fmt.Fprintln(stderr, "uisce-tenant:", err)
		return exitUsage
	}
	cfg.poll = *poll
	c := &client{cfg: cfg}

	if cmd == "status" {
		if *workflow == "" {
			fmt.Fprintln(stderr, "uisce-tenant: --workflow is required")
			return exitUsage
		}
		st, err := c.status(ctx, *workflow)
		if err != nil {
			return c.fail(stderr, err)
		}
		printStatus(stdout, st)
		return exitForStatus(st)
	}

	req := provisioning.ProvisionTenantRequest{TenantName: *name, TenantCode: *code, Region: *region, InstanceName: *instance}
	if *product != "" || *label != "" {
		req.Products = []provisioning.ProductRequest{{Product: *product, Label: *label}}
	}
	p := &prompter{in: bufio.NewReader(stdin), out: stdout, disabled: *noPrompt}

	plan, norm, code2 := c.complete(ctx, &req, p, stdout, stderr)
	if code2 != exitOK {
		return code2
	}
	printPlan(stdout, plan)
	if !*yes {
		if *noPrompt {
			fmt.Fprintln(stderr, "uisce-tenant: --no-prompt needs --yes to create")
			return exitUsage
		}
		ok, err := p.confirm("Create this tenant?")
		if err != nil {
			fmt.Fprintln(stderr, "uisce-tenant:", err)
			return exitUsage
		}
		if !ok {
			fmt.Fprintln(stdout, "Nothing was created.")
			return exitOK
		}
	}

	resp, err := c.provision(ctx, norm)
	if err != nil {
		var inv *invalidError
		if errors.As(err, &inv) {
			printIssues(stderr, inv.resp)
			return exitInvalid
		}
		return c.fail(stderr, err)
	}
	fmt.Fprintf(stdout, "Started %s (tenant %s, database %s)\n", resp.WorkflowID, resp.TenantID, resp.DatabaseName)
	if *noWait {
		return exitOK
	}
	return c.follow(ctx, resp.WorkflowID, *timeout, stdout, stderr)
}

// complete asks the server what is missing and prompts for it until the request is complete.
func (c *client) complete(ctx context.Context, req *provisioning.ProvisionTenantRequest, p *prompter, stdout, stderr io.Writer) (*provisioning.Plan, provisioning.ProvisionTenantRequest, int) {
	for round := 0; round < 12; round++ {
		d, err := c.describe(ctx, *req)
		if err != nil {
			return nil, *req, c.fail(stderr, err)
		}
		if d.Complete {
			return d.Plan, *d.Normalized, exitOK
		}
		issues := append(append([]provisioning.Issue{}, d.Missing...), d.Invalid...)
		if p.disabled {
			printIssues(stderr, d)
			return nil, *req, exitInvalid
		}
		progressed := false
		for _, is := range issues {
			if is.Message != "" && containsIssue(d.Invalid, is) {
				fmt.Fprintf(stdout, "%s: %s\n", is.Field, is.Message)
			}
			ok, err := fill(req, is, p)
			if err != nil {
				fmt.Fprintln(stderr, "uisce-tenant:", err)
				return nil, *req, exitUsage
			}
			progressed = progressed || ok
		}
		if !progressed {
			printIssues(stderr, d)
			return nil, *req, exitInvalid
		}
	}
	fmt.Fprintln(stderr, "uisce-tenant: the request is still not valid; giving up")
	return nil, *req, exitInvalid
}

func containsIssue(list []provisioning.Issue, is provisioning.Issue) bool {
	for _, x := range list {
		if x.Field == is.Field && x.Message == is.Message {
			return true
		}
	}
	return false
}

// fill asks for the field an issue names and stores the answer. It reports false for a field it has
// no question for (the caller then shows the issue instead of looping on it).
func fill(req *provisioning.ProvisionTenantRequest, is provisioning.Issue, p *prompter) (bool, error) {
	ensureProduct := func() {
		if len(req.Products) == 0 {
			req.Products = []provisioning.ProductRequest{{}}
		}
	}
	ask := func(prompt string) (string, error) { return p.ask(prompt, is.Allowed) }
	var (
		v   string
		err error
	)
	switch is.Field {
	case "tenant_name":
		v, err = ask("Tenant name")
		req.TenantName = v
	case "tenant_code":
		v, err = ask("Tenant code (lowercase letters, digits, underscores)")
		req.TenantCode = v
	case "region":
		v, err = ask("Region")
		req.Region = v
	case "products", "products[0].product":
		ensureProduct()
		v, err = ask("Product")
		req.Products[0].Product = v
	case "products[0].label":
		ensureProduct()
		v, err = ask("Label for the " + req.Products[0].Product + " database")
		req.Products[0].Label = v
	default:
		return false, nil
	}
	return err == nil && v != "", err
}

// --- prompting --------------------------------------------------------------

type prompter struct {
	in       *bufio.Reader
	out      io.Writer
	disabled bool
}

var errInputEnded = errors.New("input ended before every answer was given")

// ask reads one line. When allowed values are offered they are listed, and a number picks one.
func (p *prompter) ask(label string, allowed []string) (string, error) {
	if len(allowed) > 0 {
		fmt.Fprintf(p.out, "%s:\n", label)
		for i, a := range allowed {
			fmt.Fprintf(p.out, "  %d) %s\n", i+1, a)
		}
		fmt.Fprint(p.out, "> ")
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	line, err := p.in.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && line == "" {
		return "", errInputEnded
	}
	if n := 0; len(allowed) > 0 {
		if _, serr := fmt.Sscanf(line, "%d", &n); serr == nil && n >= 1 && n <= len(allowed) {
			// "us-east-1 (US East (N. Virginia))": the code is what the server takes.
			return strings.Fields(allowed[n-1])[0], nil
		}
	}
	return line, nil
}

func (p *prompter) confirm(question string) (bool, error) {
	fmt.Fprintf(p.out, "%s [y/N] ", question)
	line, err := p.in.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return false, errInputEnded
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

// --- output -----------------------------------------------------------------

func printPlan(w io.Writer, p *provisioning.Plan) {
	fmt.Fprintf(w, "\nTenant   %s (%s)\nRegion   %s (%s), cluster %s:%d\n", p.TenantName, p.TenantCode, p.Region, p.RegionName, p.Host, p.Port)
	for _, d := range p.Databases {
		fmt.Fprintf(w, "Product  %s, label %s: database %s, role %s\n", d.Product, d.Label, d.Database, d.Role)
	}
	fmt.Fprintln(w)
}

func printIssues(w io.Writer, d provisioning.DescribeResponse) {
	for _, is := range d.Missing {
		fmt.Fprintf(w, "missing  %s: %s%s\n", is.Field, is.Message, allowedSuffix(is.Allowed))
	}
	for _, is := range d.Invalid {
		fmt.Fprintf(w, "invalid  %s: %s%s\n", is.Field, is.Message, allowedSuffix(is.Allowed))
	}
}

func allowedSuffix(a []string) string {
	if len(a) == 0 {
		return ""
	}
	return " (one of: " + strings.Join(a, ", ") + ")"
}

func printStatus(w io.Writer, st provisioning.ProvisioningStatus) {
	fmt.Fprintf(w, "workflow=%s status=%s", st.WorkflowID, st.Status)
	if st.Step != "" {
		fmt.Fprintf(w, " step=%s", st.Step)
	}
	if st.DatabaseName != "" {
		fmt.Fprintf(w, " database=%s", st.DatabaseName)
	}
	if st.TenantID != "" {
		fmt.Fprintf(w, " tenant=%s", st.TenantID)
	}
	if st.Error != "" {
		fmt.Fprintf(w, " error=%q", st.Error)
	}
	fmt.Fprintln(w)
}

func exitForStatus(st provisioning.ProvisioningStatus) int {
	switch st.Status {
	case "completed":
		return exitOK
	case "failed", "canceled", "terminated":
		return exitFailed
	}
	return exitOK
}

// follow prints each step as the run reaches it, and returns when it ends.
func (c *client) follow(ctx context.Context, id string, timeout time.Duration, stdout, stderr io.Writer) int {
	deadline := time.Now().Add(timeout)
	last := ""
	for {
		st, err := c.status(ctx, id)
		if err != nil {
			return c.fail(stderr, err)
		}
		if st.Step != "" && st.Step != last {
			fmt.Fprintf(stdout, "  %s\n", st.Step)
			last = st.Step
		}
		if st.Status != "provisioning" && st.Status != "unknown" {
			printStatus(stdout, st)
			if st.Status == "completed" {
				fmt.Fprintf(stdout, "Tenant %s is active; database %s is ready.\n", st.TenantID, st.DatabaseName)
			}
			return exitForStatus(st)
		}
		if time.Now().After(deadline) {
			fmt.Fprintf(stderr, "uisce-tenant: still provisioning after %s; check with: uisce-tenant status --workflow %s\n", timeout, id)
			return exitTimeout
		}
		select {
		case <-ctx.Done():
			return exitTimeout
		case <-time.After(c.cfg.poll):
		}
	}
}

// --- API client -------------------------------------------------------------

type apiError struct {
	status  int
	message string
}

func (e *apiError) Error() string { return fmt.Sprintf("%s (HTTP %d)", e.message, e.status) }

// invalidError is a 400 that carries the server's list of what is wrong.
type invalidError struct{ resp provisioning.DescribeResponse }

func (e *invalidError) Error() string { return "the request is not valid" }

type client struct {
	cfg     config
	token   string
	expires time.Time
}

func configFrom(env func(string) string) (config, error) {
	c := config{
		baseURL: strings.TrimRight(env("UISCE_URL"), "/"), tokenURL: env("UISCE_TOKEN_URL"),
		clientID: env("UISCE_CLIENT_ID"), clientSecret: env("UISCE_CLIENT_SECRET"),
		http: &http.Client{Timeout: 2 * time.Minute},
	}
	if f := env("UISCE_CLIENT_SECRET_FILE"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return c, fmt.Errorf("reading UISCE_CLIENT_SECRET_FILE: %w", err)
		}
		c.clientSecret = strings.TrimSpace(string(b))
	}
	var missing []string
	for _, kv := range [][2]string{{"UISCE_URL", c.baseURL}, {"UISCE_TOKEN_URL", c.tokenURL}, {"UISCE_CLIENT_ID", c.clientID}, {"UISCE_CLIENT_SECRET(_FILE)", c.clientSecret}} {
		if kv[1] == "" {
			missing = append(missing, kv[0])
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("set %s", strings.Join(missing, ", "))
	}
	return c, nil
}

func (c *client) fail(w io.Writer, err error) int {
	fmt.Fprintln(w, "uisce-tenant:", err)
	var ae *apiError
	if errors.As(err, &ae) && (ae.status == http.StatusUnauthorized || ae.status == http.StatusForbidden) {
		return exitUnauthorized
	}
	return exitAPI
}

func (c *client) accessToken(ctx context.Context) (string, error) {
	if c.token != "" && time.Until(c.expires) > 30*time.Second {
		return c.token, nil
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.cfg.clientID}, "client_secret": {c.cfg.clientSecret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.cfg.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting a token: %w", err)
	}
	defer res.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error_description"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	if res.StatusCode != http.StatusOK || body.AccessToken == "" {
		return "", &apiError{status: http.StatusUnauthorized, message: "the token endpoint refused the client credentials: " + body.Error}
	}
	c.token, c.expires = body.AccessToken, time.Now().Add(time.Duration(body.ExpiresIn)*time.Second)
	return c.token, nil
}

// do sends one request. Reads are retried on a gateway error; the create is not, because its
// idempotency lives in the server and a retry that raced the first would only find the same run.
func (c *client) do(ctx context.Context, method, path string, in, out any) error {
	attempts := 1
	if method == http.MethodGet {
		attempts = 4
	}
	var last error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i*i) * time.Second):
			}
		}
		err := c.once(ctx, method, path, in, out)
		var ae *apiError
		if err == nil || !errors.As(err, &ae) || (ae.status != http.StatusBadGateway && ae.status != http.StatusServiceUnavailable && ae.status != http.StatusGatewayTimeout) {
			return err
		}
		last = err
	}
	return last
}

func (c *client) once(ctx context.Context, method, path string, in, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.cfg.http.Do(req)
	if err != nil {
		return &apiError{status: http.StatusServiceUnavailable, message: err.Error()}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		if res.StatusCode == http.StatusBadRequest {
			var d provisioning.DescribeResponse
			if json.Unmarshal(raw, &d) == nil && (len(d.Missing) > 0 || len(d.Invalid) > 0) {
				return &invalidError{resp: d}
			}
		}
		return &apiError{status: res.StatusCode, message: strings.TrimSpace(string(raw))}
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (c *client) describe(ctx context.Context, req provisioning.ProvisionTenantRequest) (provisioning.DescribeResponse, error) {
	var d provisioning.DescribeResponse
	err := c.do(ctx, http.MethodPost, "/api/system/tenants/provision/describe", req, &d)
	return d, err
}

func (c *client) provision(ctx context.Context, req provisioning.ProvisionTenantRequest) (provisioning.ProvisionTenantResponse, error) {
	var r provisioning.ProvisionTenantResponse
	err := c.do(ctx, http.MethodPost, "/api/system/tenants/provision", req, &r)
	return r, err
}

func (c *client) status(ctx context.Context, workflowID string) (provisioning.ProvisioningStatus, error) {
	var st provisioning.ProvisioningStatus
	// The tenant id in the path is not used by the server; the workflow id names the run.
	err := c.do(ctx, http.MethodGet, "/api/system/tenants/-/provision/"+url.PathEscape(workflowID), nil, &st)
	return st, err
}
