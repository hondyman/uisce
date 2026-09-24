package manager

import (
	"fmt"
	"net/url"
	"strings"
)

// SelectTargetScreen chooses the appropriate screen based on the requested
// target index, falling back to the primary display if out of range.
func SelectTargetScreen(screens []ScreenInfo, targetIndex int) (ScreenInfo, int) {
	if len(screens) == 0 {
		return ScreenInfo{
			Index:       0,
			ID:          "screen-default",
			Name:        "Default Display",
			IsPrimary:   true,
			Scale:       1.0,
			ScaleFactor: 1.0,
			X:           0,
			Y:           0,
			Width:       1920,
			Height:      1080,
		}, 0
	}

	if targetIndex >= 0 && targetIndex < len(screens) {
		return screens[targetIndex], targetIndex
	}

	// Fallback to primary screen
	for i, s := range screens {
		if s.IsPrimary {
			return s, i
		}
	}

	// If no screen is marked primary, return the first
	return screens[0], 0
}

// ComputeWindowBounds calculates clamped dimensions and coordinates on the target
// screen. If explicit (x, y) coordinates are not specified (both <= 0), the window
// is centered on the target screen.
func ComputeWindowBounds(screen ScreenInfo, requestedW, requestedH, reqX, reqY int) WindowBounds {
	width := requestedW
	height := requestedH

	// Sane defaults if width/height not specified
	if width <= 0 {
		width = 1280
		if width > screen.Width-80 {
			width = screen.Width - 80
		}
		if width < 400 {
			width = screen.Width
		}
	}
	if height <= 0 {
		height = 800
		if height > screen.Height-80 {
			height = screen.Height - 80
		}
		if height < 300 {
			height = screen.Height
		}
	}

	// Clamp to screen dimensions
	if width > screen.Width {
		width = screen.Width
	}
	if height > screen.Height {
		height = screen.Height
	}

	var x, y int
	if reqX <= 0 && reqY <= 0 {
		// Center window on target display
		x = screen.X + (screen.Width-width)/2
		y = screen.Y + (screen.Height-height)/2
	} else {
		// Adjust for monitor offset and ensure the window caption stays visible
		minX := screen.X
		maxX := screen.X + screen.Width - 100
		minY := screen.Y
		maxY := screen.Y + screen.Height - 100

		x = reqX
		if x < minX {
			x = minX
		}
		if x > maxX {
			x = maxX
		}

		y = reqY
		if y < minY {
			y = minY
		}
		if y > maxY {
			y = maxY
		}
	}

	return WindowBounds{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

// AppendTokenToRoute safely appends the ephemeral init_token parameter to the route.
func AppendTokenToRoute(route, token string) string {
	if token == "" {
		return route
	}

	// If route contains query params, join with &, else ?
	delimiter := "?"
	if strings.Contains(route, "?") {
		delimiter = "&"
	}

	return fmt.Sprintf("%s%sinit_token=%s", route, delimiter, url.QueryEscape(token))
}

// IsWindowOnAnyScreen returns true if at least a minimal titlebar grab area
// (100x40px) of the window lies within any currently connected display.
func IsWindowOnAnyScreen(win WindowBounds, screens []ScreenInfo) bool {
	if len(screens) == 0 {
		return false
	}

	checkW := 100
	if checkW > win.Width {
		checkW = win.Width
	}
	checkH := 40
	if checkH > win.Height {
		checkH = win.Height
	}

	for _, s := range screens {
		overlapX := win.X < s.X+s.Width && win.X+checkW > s.X
		overlapY := win.Y < s.Y+s.Height && win.Y+checkH > s.Y
		if overlapX && overlapY {
			return true
		}
	}

	return false
}

// ReclampWindowToScreen moves an orphaned window onto targetScreen and clamps
// its dimensions so it is comfortably visible and centered.
func ReclampWindowToScreen(win WindowBounds, targetScreen ScreenInfo) WindowBounds {
	width := win.Width
	height := win.Height

	if width <= 0 || width > targetScreen.Width {
		width = targetScreen.Width - 80
		if width < 400 {
			width = targetScreen.Width
		}
	}
	if height <= 0 || height > targetScreen.Height {
		height = targetScreen.Height - 80
		if height < 300 {
			height = targetScreen.Height
		}
	}

	x := targetScreen.X + (targetScreen.Width-width)/2
	y := targetScreen.Y + (targetScreen.Height-height)/2

	return WindowBounds{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}
