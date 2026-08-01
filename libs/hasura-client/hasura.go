package hasuraclient

import "context"

type Client struct{}

func NewClient(url, adminSecret string) *Client {
	return &Client{}
}

func (c *Client) Query(ctx context.Context, query string, variables map[string]any, result any) error {
	return nil
}

func (c *Client) Subscribe(ctx context.Context, query string, variables map[string]any, handler func(any)) error {
	return nil
}
