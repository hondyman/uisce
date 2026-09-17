package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hondyman/uisce/desktop/manager"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func resolveAssetFS() fs.FS {
	// 1. Explicit environment variable override
	if p := os.Getenv("UISCE_FRONTEND_DIR"); p != "" {
		if _, err := os.Stat(p); err == nil {
			log.Printf("[Desktop] Using frontend assets from UISCE_FRONTEND_DIR: %s", p)
			return os.DirFS(p)
		}
	}

	// 2. Relative frontend/dist from desktop folder
	relDist := filepath.Join("..", "frontend", "dist")
	if abs, err := filepath.Abs(relDist); err == nil {
		if _, err := os.Stat(abs); err == nil {
			log.Printf("[Desktop] Using frontend assets from: %s", abs)
			return os.DirFS(abs)
		}
	}

	// 3. Local bundled folder if present
	if _, err := os.Stat("frontend_dist"); err == nil {
		log.Printf("[Desktop] Using bundled frontend_dist")
		return os.DirFS("frontend_dist")
	}

	log.Printf("[Desktop] Warning: No frontend assets found. Ensure frontend is built (npm run build).")
	return nil
}

func main() {
	vault := manager.NewTokenVault(60 * time.Second)
	defer vault.Stop()

	deskManager := NewDeskWindowManager(nil, vault)

	assetFS := resolveAssetFS()
	spaHandler := NewSPAHandler(assetFS)

	app := application.New(application.Options{
		Name:        "UisceTradingDesk",
		Description: "Uisce Multi-Monitor Portfolio Workstation",
		Services: []application.Service{
			application.NewService(deskManager),
		},
		Assets: application.AssetOptions{
			Handler: spaHandler,
		},
	})

	// Inject live app reference into window manager
	deskManager.app = app

	// Register clean shutdown handler to stop background vault janitor
	app.OnShutdown(func() {
		log.Println("[Desktop] Application shutting down; stopping TokenVault janitor...")
		vault.Stop()
	})

	// Wire display hotplug & screen topology change listener
	app.Event.OnApplicationEvent(events.Mac.ApplicationDidChangeScreenParameters, func(e *application.ApplicationEvent) {
		log.Println("[Desktop] Display configuration changed; checking for orphaned windows...")
		reclamped := deskManager.ReclampOrphanedWindows()
		if reclamped > 0 {
			log.Printf("[Desktop] Reclamped %d orphaned windows onto primary display", reclamped)
		}
	})

	// Create Primary Workspace Window
	mainWin := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "win_main",
		Title:     "Uisce Multi-Monitor Workstation",
		URL:       "/workspace",
		Width:     1440,
		Height:    900,
		Frameless: false,
	})

	deskManager.RegisterWindow("win_main", mainWin)

	// Launch automated verification suite if requested via --verify or UISCE_DESK_VERIFY=1
	if IsVerifyMode() {
		go RunDesktopVerificationSuite(app, deskManager, vault, spaHandler, mainWin)
	}

	fmt.Println("[Desktop] Starting Uisce Multi-Monitor Workstation...")
	if err := app.Run(); err != nil {
		log.Fatalf("[Desktop] Application error: %v", err)
	}
}
