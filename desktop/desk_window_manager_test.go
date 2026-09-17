package main

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/desktop/manager"
)

func TestDeskWindowManager_ExchangeToken(t *testing.T) {
	vault := manager.NewTokenVault(1 * time.Second)
	defer vault.Stop()

	mgr := NewDeskWindowManager(nil, vault)

	// Pre-populate a token
	tok, err := vault.GenerateToken("mock_jwt_session_token")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// 1. First exchange succeeds
	payload, err := mgr.ExchangeToken(tok)
	if err != nil {
		t.Fatalf("ExchangeToken failed: %v", err)
	}
	if payload != "mock_jwt_session_token" {
		t.Fatalf("unexpected payload: %s", payload)
	}

	// 2. Second exchange fails (single-use enforced)
	_, err2 := mgr.ExchangeToken(tok)
	if err2 == nil {
		t.Fatalf("expected error on second exchange of same token, got nil")
	}

	// 3. Empty token fails
	_, err3 := mgr.ExchangeToken("")
	if err3 == nil {
		t.Fatalf("expected error on empty token, got nil")
	}
}

func TestDeskWindowManager_GetMonitorsFallback(t *testing.T) {
	// Without live app, GetMonitors falls back cleanly to default screen
	mgr := NewDeskWindowManager(nil, nil)
	screens := mgr.GetMonitors()

	if len(screens) != 1 {
		t.Fatalf("expected 1 fallback screen, got %d", len(screens))
	}
	if !screens[0].IsPrimary {
		t.Fatalf("expected primary display")
	}
	if screens[0].Width != 1920 || screens[0].Height != 1080 {
		t.Fatalf("expected 1920x1080, got %dx%d", screens[0].Width, screens[0].Height)
	}

	// GetScreens alias
	aliasScreens := mgr.GetScreens()
	if len(aliasScreens) != 1 {
		t.Fatalf("expected 1 alias screen, got %d", len(aliasScreens))
	}
}

func TestDeskWindowManager_SpawnWindowNilApp(t *testing.T) {
	mgr := NewDeskWindowManager(nil, nil)

	_, err := mgr.SpawnWindow(manager.WindowSpawnOptions{
		WindowID: "test_win_1",
		Route:    "/view/rebalancer",
	})
	if err == nil {
		t.Fatalf("expected error when app is nil, got nil")
	}
}

func TestDeskWindowManager_CloseWindowNotFound(t *testing.T) {
	mgr := NewDeskWindowManager(nil, nil)

	err := mgr.CloseWindow("non_existent_win")
	if err == nil {
		t.Fatalf("expected error when closing non-existent window, got nil")
	}
}

func TestDeskWindowManager_ReclampOrphanedWindows(t *testing.T) {
	mgr := NewDeskWindowManager(nil, nil)
	// Without live windows, reclamp returns 0 safely without panic
	count := mgr.ReclampOrphanedWindows()
	if count != 0 {
		t.Fatalf("expected 0 reclamped windows, got %d", count)
	}
}
