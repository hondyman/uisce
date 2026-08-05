package cube

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	baseURL    string
	apiSecret  string
	httpClient *http.Client
}

func NewClient(baseURL, apiSecret string) *Client {
	return &Client{
		baseURL:   baseURL,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type Query struct {
	Measures       []string          `json:"measures,omitempty"`
	Dimensions     []string          `json:"dimensions,omitempty"`
	Filters        []Filter          `json:"filters,omitempty"`
	TimeDimensions []TimeDimension   `json:"timeDimensions,omitempty"`
	Order          map[string]string `json:"order,omitempty"`
	Limit          int               `json:"limit,omitempty"`
	Offset         int               `json:"offset,omitempty"`
	Segments       []string          `json:"segments,omitempty"`
	Timezone       string            `json:"timezone,omitempty"`
	RenewQuery     bool              `json:"renewQuery,omitempty"`
}

type Filter struct {
	Member   string   `json:"member"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type TimeDimension struct {
	Dimension   string   `json:"dimension"`
	DateRange   []string `json:"dateRange,omitempty"`
	Granularity string   `json:"granularity,omitempty"`
}

type QueryResult struct {
	Data       []map[string]interface{} `json:"data"`
	Annotation *Annotation              `json:"annotation,omitempty"`
	Query      *Query                   `json:"query,omitempty"`
}

type Annotation struct {
	Measures       map[string]MemberMeta `json:"measures"`
	Dimensions     map[string]MemberMeta `json:"dimensions"`
	TimeDimensions map[string]MemberMeta `json:"timeDimensions"`
}

type MemberMeta struct {
	Title        string   `json:"title"`
	ShortTitle   string   `json:"shortTitle"`
	Type         string   `json:"type"`
	Format       string   `json:"format,omitempty"`
	DrillMembers []string `json:"drillMembers,omitempty"`
}

type TenantContext struct {
	TenantID     uuid.UUID
	DatasourceID uuid.UUID
	UserID       string
}

func (c *Client) ExecuteQuery(ctx context.Context, query *Query, tenantCtx TenantContext) (*QueryResult, error) {
	return &QueryResult{}, nil
}

func (c *Client) GetMeta(ctx context.Context, tenantCtx TenantContext) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (c *Client) PreAggregationStatus(ctx context.Context, tenantCtx TenantContext) ([]map[string]interface{}, error) {
	return nil, nil
}

func (c *Client) DryRun(ctx context.Context, query *Query, tenantCtx TenantContext) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func BuildQuery(cube string, measures, dimensions []string, filters []Filter) *Query {
	return &Query{
		Measures:   measures,
		Dimensions: dimensions,
		Filters:    filters,
		Timezone:   "UTC",
	}
}

func BuildTimeDimension(dimension, granularity string, dateRange []string) TimeDimension {
	return TimeDimension{
		Dimension:   dimension,
		Granularity: granularity,
		DateRange:   dateRange,
	}
}
