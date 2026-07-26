package hasuraclient

import (
	"context"
	"net/http"
	"time"

	"github.com/machinebox/graphql"
)

// HasuraConfig holds configuration for the Hasura client
type HasuraConfig struct {
	Endpoint    string
	AdminSecret string
	Headers     map[string]string
}

// QueryOptions holds options for GraphQL queries/mutations
type QueryOptions struct {
	Headers map[string]string
}

// HasuraClient provides typed GraphQL operations for data access
type HasuraClient struct {
	client *graphql.Client
	cfg    *HasuraConfig
}

// NewHasuraClient creates a new Hasura GraphQL client
func NewHasuraClient(config *HasuraConfig) *HasuraClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client := graphql.NewClient(config.Endpoint, graphql.WithHTTPClient(httpClient))

	// Add default headers
	if config.AdminSecret != "" {
		client.Log = func(s string) {} // Disable logging
	}

	return &HasuraClient{
		client: client,
		cfg:    config,
	}
}

// Query executes a GraphQL query
func (c *HasuraClient) Query(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	req := graphql.NewRequest(query)

	// Add default headers from config
	if c.cfg != nil {
		if c.cfg.AdminSecret != "" {
			req.Header.Set("x-hasura-admin-secret", c.cfg.AdminSecret)
		}
		for k, v := range c.cfg.Headers {
			req.Header.Set(k, v)
		}
	}

	// Add variables if provided
	for key, value := range variables {
		req.Var(key, value)
	}

	var resp map[string]interface{}
	err := c.client.Run(context.Background(), req, &resp)
	return resp, err
}

// Mutate executes a GraphQL mutation
func (c *HasuraClient) Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
	req := graphql.NewRequest(mutation)

	// Add default headers from config
	if c.cfg != nil {
		if c.cfg.AdminSecret != "" {
			req.Header.Set("x-hasura-admin-secret", c.cfg.AdminSecret)
		}
		for k, v := range c.cfg.Headers {
			req.Header.Set(k, v)
		}
	}

	// Add variables if provided
	for key, value := range variables {
		req.Var(key, value)
	}

	var resp map[string]interface{}
	err := c.client.Run(context.Background(), req, &resp)
	return resp, err
}
