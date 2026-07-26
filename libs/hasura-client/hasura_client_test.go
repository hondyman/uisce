package hasuraclient

import "testing"

func TestNewHasuraClientStoresConfig(t *testing.T) {
	cfg := &HasuraConfig{Endpoint: "http://localhost:8080/v1/graphql", AdminSecret: "s3cr3t"}
	client := NewHasuraClient(cfg)
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.cfg == nil {
		t.Fatalf("expected cfg to be stored on client")
	}
	if client.cfg.AdminSecret != "s3cr3t" {
		t.Fatalf("expected admin secret to be stored, got %v", client.cfg.AdminSecret)
	}
}
