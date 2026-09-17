package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hondyman/uisce/desktop/manager"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// DeskWindowManager manages desktop windows, multi-monitor targeting,
// FDC3 IPC message relay, and zero-trust single-use token exchanges.
type DeskWindowManager struct {
	mu      sync.RWMutex
	app     *application.App
	vault   *manager.TokenVault
	windows map[string]*application.WebviewWindow
}

// NewDeskWindowManager creates a new DeskWindowManager instance.
func NewDeskWindowManager(app *application.App, vault *manager.TokenVault) *DeskWindowManager {
	if vault == nil {
		vault = manager.NewTokenVault(60 * time.Second)
	}

	return &DeskWindowManager{
		app:     app,
		vault:   vault,
		windows: make(map[string]*application.WebviewWindow),
	}
}

// GetMonitors returns all physical displays detected by the operating system.
func (m *DeskWindowManager) GetMonitors() []manager.ScreenInfo {
	if m.app == nil || m.app.Screen == nil {
		return []manager.ScreenInfo{
			{
				Index:       0,
				ID:          "screen-0",
				Name:        "Primary Display",
				IsPrimary:   true,
				Scale:       1.0,
				ScaleFactor: 1.0,
				X:           0,
				Y:           0,
				Width:       1920,
				Height:      1080,
			},
		}
	}

	appScreens := m.app.Screen.GetAll()
	result := make([]manager.ScreenInfo, len(appScreens))

	for i, s := range appScreens {
		width := s.Size.Width
		height := s.Size.Height
		if width <= 0 && s.Bounds.Width > 0 {
			width = s.Bounds.Width
		}
		if height <= 0 && s.Bounds.Height > 0 {
			height = s.Bounds.Height
		}

		result[i] = manager.ScreenInfo{
			Index:       i,
			ID:          s.ID,
			Name:        s.Name,
			IsPrimary:   s.IsPrimary,
			Scale:       s.ScaleFactor,
			ScaleFactor: s.ScaleFactor,
			X:           s.X,
			Y:           s.Y,
			Width:       width,
			Height:      height,
		}
	}

	return result
}

// GetScreens is an alias for GetMonitors to ensure universal API compatibility.
func (m *DeskWindowManager) GetScreens() []manager.ScreenInfo {
	return m.GetMonitors()
}

// SpawnWindow opens a new workstation window or focuses an existing window.
// It handles monitor assignment, bounds clamping, and security token injection.
func (m *DeskWindowManager) SpawnWindow(opts manager.WindowSpawnOptions) (string, error) {
	if opts.WindowID == "" {
		opts.WindowID = fmt.Sprintf("win_%d", time.Now().UnixNano())
	}

	// 1. Deduplication: Focus existing window if already open
	m.mu.Lock()
	if existingWin, exists := m.windows[opts.WindowID]; exists && existingWin != nil {
		m.mu.Unlock()
		existingWin.Focus()
		existingWin.UnMinimise()
		return opts.WindowID, nil
	}
	m.mu.Unlock()

	// 2. Token Vault: Issue single-use token if auth token is supplied
	route := opts.Route
	authToken := opts.AuthToken
	if authToken == "" {
		authToken = opts.SessionJWT
	}
	if authToken != "" {
		token, err := m.vault.GenerateToken(authToken)
		if err == nil {
			route = manager.AppendTokenToRoute(route, token)
		} else {
			log.Printf("[DeskWindowManager] Warning: Failed to generate ephemeral token: %v", err)
		}
	}

	// 3. Multi-monitor target and bounds computation
	screens := m.GetMonitors()
	targetScreenInfo, selectedIdx := manager.SelectTargetScreen(screens, opts.TargetScreenIndex)
	bounds := manager.ComputeWindowBounds(targetScreenInfo, opts.Width, opts.Height, opts.X, opts.Y)

	var targetAppScreen *application.Screen
	if m.app != nil && m.app.Screen != nil {
		appScreens := m.app.Screen.GetAll()
		if selectedIdx >= 0 && selectedIdx < len(appScreens) {
			targetAppScreen = appScreens[selectedIdx]
		}
	}

	// 4. Create WebviewWindow
	title := opts.Title
	if title == "" {
		title = "Uisce Workstation"
	}

	winOpts := application.WebviewWindowOptions{
		Name:      opts.WindowID,
		Title:     title,
		URL:       route,
		Width:     bounds.Width,
		Height:    bounds.Height,
		X:         bounds.X,
		Y:         bounds.Y,
		Frameless: opts.Frameless,
		Screen:    targetAppScreen,
	}

	if m.app == nil {
		return "", errors.New("wails application not initialized")
	}

	win := m.app.Window.NewWithOptions(winOpts)
	if win == nil {
		return "", fmt.Errorf("failed to create window %q", opts.WindowID)
	}

	// 5. Register hook to clean up map when window closes
	winID := opts.WindowID
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		m.mu.Lock()
		delete(m.windows, winID)
		m.mu.Unlock()
		log.Printf("[DeskWindowManager] Window closed and deregistered: %s", winID)
		m.notifyWindowClosed(winID)
	})

	m.mu.Lock()
	m.windows[winID] = win
	m.mu.Unlock()

	return winID, nil
}

// RegisterWindow adds an externally created window (like the primary win_main)
// into the managed window registry with automatic closing hooks.
func (m *DeskWindowManager) RegisterWindow(windowID string, win *application.WebviewWindow) {
	if win == nil || windowID == "" {
		return
	}

	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		m.mu.Lock()
		delete(m.windows, windowID)
		m.mu.Unlock()
		log.Printf("[DeskWindowManager] Window closed and deregistered: %s", windowID)
		m.notifyWindowClosed(windowID)
	})

	m.mu.Lock()
	m.windows[windowID] = win
	m.mu.Unlock()
}

// GetWindowCount returns the number of active windows currently registered.
func (m *DeskWindowManager) GetWindowCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.windows)
}

// GetOpenWindowIDs returns the IDs of all currently open and registered windows.
func (m *DeskWindowManager) GetOpenWindowIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.windows))
	for id := range m.windows {
		ids = append(ids, id)
	}
	return ids
}

// notifyWindowClosed emits Wails and DOM events to remaining active windows.
func (m *DeskWindowManager) notifyWindowClosed(winID string) {
	if m.app != nil && m.app.Event != nil {
		m.app.Event.Emit("desktop:window-closed", winID)
	}

	m.mu.RLock()
	activeWindows := make([]*application.WebviewWindow, 0, len(m.windows))
	for _, win := range m.windows {
		if win != nil {
			activeWindows = append(activeWindows, win)
		}
	}
	m.mu.RUnlock()

	jsScript := fmt.Sprintf(`(function() {
		try {
			window.dispatchEvent(new CustomEvent('desktop:window-closed', { detail: { windowId: %q } }));
		} catch(e) {}
	})();`, winID)

	for _, win := range activeWindows {
		win.ExecJS(jsScript)
	}
}

// HasWindow checks whether a window with the given ID is currently registered.
func (m *DeskWindowManager) HasWindow(windowID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.windows[windowID]
	return exists
}

// CloseWindow closes the specified window if it is currently open.
func (m *DeskWindowManager) CloseWindow(windowID string) error {
	m.mu.Lock()
	win, exists := m.windows[windowID]
	if exists {
		delete(m.windows, windowID)
	}
	m.mu.Unlock()

	if !exists || win == nil {
		return fmt.Errorf("window %q not found", windowID)
	}

	m.notifyWindowClosed(windowID)
	win.Close()
	return nil
}

// RelayMessage broadcasts an FDC3 message across all open windows.
func (m *DeskWindowManager) RelayMessage(channel string, payload string) error {
	if m.app == nil {
		return errors.New("app not initialized")
	}

	// 1. Emit via Wails Event system
	if m.app.Event != nil {
		m.app.Event.Emit("fdc3_relay_message", payload)
	}

	// 2. Direct DOM dispatch via ExecJS on every active window
	m.mu.RLock()
	activeWindows := make([]*application.WebviewWindow, 0, len(m.windows))
	for _, win := range m.windows {
		if win != nil {
			activeWindows = append(activeWindows, win)
		}
	}
	m.mu.RUnlock()

	quotedPayload, err := json.Marshal(payload)
	if err != nil {
		quotedPayload = []byte(fmt.Sprintf("%q", payload))
	}

	jsScript := fmt.Sprintf(`(function() {
		try {
			var raw = %s;
			var data = null;
			try { data = JSON.parse(raw); } catch (e) { data = raw; }
			window.dispatchEvent(new CustomEvent('fdc3_relay_message', { detail: { channelId: %q, envelope: data } }));
			if (typeof window.__onFdc3RelayMessage === 'function') {
				window.__onFdc3RelayMessage(data);
			}
		} catch (err) {
			console.error('[GoRelay] Dispatch error:', err);
		}
	})();`, string(quotedPayload), channel)

	for _, win := range activeWindows {
		win.ExecJS(jsScript)
	}

	return nil
}

// ExchangeToken retrieves and atomically deletes a single-use initialization token.
func (m *DeskWindowManager) ExchangeToken(initToken string) (string, error) {
	if initToken == "" {
		return "", errors.New("missing initialization token")
	}

	payload, ok := m.vault.ConsumeToken(initToken)
	if !ok {
		return "", errors.New("invalid or expired initialization token")
	}

	return payload, nil
}

// ReclampOrphanedWindows inspects all open windows and moves any window that is
// off-screen (due to a monitor disconnection or display topology change) back onto
// the primary display.
func (m *DeskWindowManager) ReclampOrphanedWindows() int {
	screens := m.GetMonitors()
	if len(screens) == 0 {
		return 0
	}

	primaryScreen, _ := manager.SelectTargetScreen(screens, -1)

	m.mu.RLock()
	type winItem struct {
		id  string
		win *application.WebviewWindow
	}
	windowList := make([]winItem, 0, len(m.windows))
	for id, win := range m.windows {
		if win != nil {
			windowList = append(windowList, winItem{id: id, win: win})
		}
	}
	m.mu.RUnlock()

	reclampedCount := 0
	for _, item := range windowList {
		x, y := item.win.Position()
		w, h := item.win.Size()

		bounds := manager.WindowBounds{
			X:      x,
			Y:      y,
			Width:  w,
			Height: h,
		}

		if !manager.IsWindowOnAnyScreen(bounds, screens) {
			newBounds := manager.ReclampWindowToScreen(bounds, primaryScreen)
			item.win.SetPosition(newBounds.X, newBounds.Y)
			item.win.SetSize(newBounds.Width, newBounds.Height)
			log.Printf("[DeskWindowManager] Reclamped orphaned window %q to primary display at (%d, %d)", item.id, newBounds.X, newBounds.Y)
			reclampedCount++
		}
	}

	return reclampedCount
}
