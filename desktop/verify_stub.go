//go:build !verify

package main

import (
	"github.com/hondyman/uisce/desktop/manager"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// IsVerifyMode always returns false in production binaries.
func IsVerifyMode() bool {
	return false
}

// RunDesktopVerificationSuite is a no-op stub in production builds.
func RunDesktopVerificationSuite(
	app *application.App,
	deskManager *DeskWindowManager,
	vault *manager.TokenVault,
	spaHandler *SPAHandler,
	mainWin *application.WebviewWindow,
) {
	// No verification suite in production builds
}
