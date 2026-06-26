package api

import (
	"net/http"

	"calendar-service/internal/config"

	"github.com/hasura/go-graphql-client"
)

type HasuraClient struct {
	*graphql.Client
}

func NewHasuraClient(cfg *config.Config) (*HasuraClient, error) {
	httpClient := &http.Client{}

	if cfg.HasuraAdminSecret != "" {
		httpClient = &http.Client{
			Transport: &authedTransport{
				wrapped:     http.DefaultTransport,
				adminSecret: cfg.HasuraAdminSecret,
			},
		}
	}

	client := graphql.NewClient(cfg.HasuraEndpoint, httpClient)
	return &HasuraClient{client}, nil
}

type authedTransport struct {
	wrapped     http.RoundTripper
	adminSecret string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Hasura-Admin-Secret", t.adminSecret)
	return t.wrapped.RoundTrip(req)
}
