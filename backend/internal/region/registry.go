package region

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

// regionCodePattern is the allowed shape of a region code (e.g. us-east-1).
// Region codes are used as config keys and in log lines only, but they are
// still checked so a typo or hostile value fails at load time.
var regionCodePattern = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]+$`)

// maxRegionCodeLen bounds region codes; real codes are well under this.
const maxRegionCodeLen = 32

// Endpoints are the infrastructure addresses a tenant instance in one region
// talks to. Each region is defined once in the boot-time config file.
type Endpoints struct {
	PostgresHost       string `yaml:"postgres_host"`
	PostgresPort       int    `yaml:"postgres_port"`
	LakekeeperURL      string `yaml:"lakekeeper_url"`
	StarRocksFEHost    string `yaml:"starrocks_fe_host"`
	DebeziumConnectURL string `yaml:"debezium_connect_url"`
}

// Registry maps region codes to their endpoints. It is immutable after load.
type Registry struct {
	regions map[string]Endpoints
}

type registryFile struct {
	Regions map[string]Endpoints `yaml:"regions"`
}

// LoadRegistry reads and validates the region config file. It refuses to
// start with an invalid file, so a bad region never reaches provisioning.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read region config: %w", err)
	}
	return ParseRegistry(raw)
}

// ParseRegistry validates region config bytes. Split out from LoadRegistry so
// tests can exercise validation without touching the filesystem.
func ParseRegistry(raw []byte) (*Registry, error) {
	var f registryFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse region config: %w", err)
	}
	if len(f.Regions) == 0 {
		return nil, fmt.Errorf("region config defines no regions")
	}
	for code, ep := range f.Regions {
		if len(code) > maxRegionCodeLen || !regionCodePattern.MatchString(code) {
			return nil, fmt.Errorf("region code %q is not a valid code", code)
		}
		if err := validateEndpoints(code, ep); err != nil {
			return nil, err
		}
	}
	return &Registry{regions: f.Regions}, nil
}

func validateEndpoints(code string, ep Endpoints) error {
	if ep.PostgresHost == "" {
		return fmt.Errorf("region %s: postgres_host is required", code)
	}
	if ep.PostgresPort < 1 || ep.PostgresPort > 65535 {
		return fmt.Errorf("region %s: postgres_port %d is out of range", code, ep.PostgresPort)
	}
	if err := validateURL(code, "lakekeeper_url", ep.LakekeeperURL); err != nil {
		return err
	}
	if err := validateURL(code, "debezium_connect_url", ep.DebeziumConnectURL); err != nil {
		return err
	}
	if ep.StarRocksFEHost == "" {
		return fmt.Errorf("region %s: starrocks_fe_host is required", code)
	}
	return nil
}

func validateURL(code, field, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || raw == "" {
		return fmt.Errorf("region %s: %s is not a valid URL", code, field)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("region %s: %s must use http or https", code, field)
	}
	if u.Host == "" {
		return fmt.Errorf("region %s: %s has no host", code, field)
	}
	return nil
}

// Resolve returns the endpoints for a region. An unknown region is an error:
// provisioning must never guess a region or fall back to a default.
func (r *Registry) Resolve(code string) (Endpoints, error) {
	ep, ok := r.regions[code]
	if !ok {
		return Endpoints{}, fmt.Errorf("unknown region %q", code)
	}
	return ep, nil
}

// Codes returns the configured region codes in sorted order.
func (r *Registry) Codes() []string {
	codes := make([]string, 0, len(r.regions))
	for code := range r.regions {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}
