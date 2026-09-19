package manager

import (
	"testing"
)

func mockScreens() []ScreenInfo {
	return []ScreenInfo{
		{
			Index:       0,
			ID:          "screen-0",
			Name:        "Display 1 (Primary)",
			IsPrimary:   true,
			Scale:       2.0,
			ScaleFactor: 2.0,
			X:           0,
			Y:           0,
			Width:       1728,
			Height:      1117,
		},
		{
			Index:       1,
			ID:          "screen-1",
			Name:        "Display 2 (External 4K)",
			IsPrimary:   false,
			Scale:       1.0,
			ScaleFactor: 1.0,
			X:           1728,
			Y:           0,
			Width:       3840,
			Height:      2160,
		},
		{
			Index:       2,
			ID:          "screen-2",
			Name:        "Display 3 (Vertical 1440p)",
			IsPrimary:   false,
			Scale:       1.0,
			ScaleFactor: 1.0,
			X:           -1440,
			Y:           0,
			Width:       1440,
			Height:      2560,
		},
	}
}

func TestSelectTargetScreen_ValidIndex(t *testing.T) {
	screens := mockScreens()

	// Select second screen
	s1, idx1 := SelectTargetScreen(screens, 1)
	if idx1 != 1 || s1.ID != "screen-1" {
		t.Fatalf("expected screen-1 at index 1, got %s at %d", s1.ID, idx1)
	}

	// Select third screen (negative X offset)
	s2, idx2 := SelectTargetScreen(screens, 2)
	if idx2 != 2 || s2.ID != "screen-2" {
		t.Fatalf("expected screen-2 at index 2, got %s at %d", s2.ID, idx2)
	}
}

func TestSelectTargetScreen_OutOfRangeFallbackToPrimary(t *testing.T) {
	screens := mockScreens()

	// Out of range index 99 -> falls back to primary screen 0
	s, idx := SelectTargetScreen(screens, 99)
	if idx != 0 || !s.IsPrimary {
		t.Fatalf("expected fallback to primary display 0, got %s at %d", s.ID, idx)
	}

	// Negative index -1 -> falls back to primary screen 0
	sNeg, idxNeg := SelectTargetScreen(screens, -1)
	if idxNeg != 0 || !sNeg.IsPrimary {
		t.Fatalf("expected fallback to primary display 0, got %s at %d", sNeg.ID, idxNeg)
	}
}

func TestSelectTargetScreen_EmptyList(t *testing.T) {
	s, idx := SelectTargetScreen(nil, 0)
	if idx != 0 || s.Width != 1920 || s.Height != 1080 {
		t.Fatalf("expected default 1080p fallback screen, got %+v", s)
	}
}

func TestComputeWindowBounds_Centering(t *testing.T) {
	screens := mockScreens()
	extScreen := screens[1] // 4K screen: X=1728, Y=0, W=3840, H=2160

	// Request 1280x800 with no explicit coordinates (reqX=0, reqY=0)
	bounds := ComputeWindowBounds(extScreen, 1280, 800, 0, 0)

	expectedX := 1728 + (3840-1280)/2 // 1728 + 1280 = 3008
	expectedY := 0 + (2160-800)/2    // 680

	if bounds.Width != 1280 || bounds.Height != 800 {
		t.Fatalf("expected 1280x800, got %dx%d", bounds.Width, bounds.Height)
	}
	if bounds.X != expectedX || bounds.Y != expectedY {
		t.Fatalf("expected centered at (%d, %d), got (%d, %d)", expectedX, expectedY, bounds.X, bounds.Y)
	}
}

func TestComputeWindowBounds_ClampingToScreen(t *testing.T) {
	screens := mockScreens()
	primary := screens[0] // W=1728, H=1117

	// Request oversized window 5000x5000
	bounds := ComputeWindowBounds(primary, 5000, 5000, 0, 0)

	if bounds.Width != primary.Width {
		t.Fatalf("expected width clamped to %d, got %d", primary.Width, bounds.Width)
	}
	if bounds.Height != primary.Height {
		t.Fatalf("expected height clamped to %d, got %d", primary.Height, bounds.Height)
	}
}

func TestComputeWindowBounds_ExplicitCoordinatesPreservedAndClamped(t *testing.T) {
	screens := mockScreens()
	extScreen := screens[1] // X=1728, Y=0, W=3840, H=2160

	// Valid explicit coordinate within screen
	bounds1 := ComputeWindowBounds(extScreen, 1000, 700, 2000, 300)
	if bounds1.X != 2000 || bounds1.Y != 300 {
		t.Fatalf("expected coordinates (2000, 300) preserved, got (%d, %d)", bounds1.X, bounds1.Y)
	}

	// Coordinate way off to the left -> clamped to minX (1728)
	bounds2 := ComputeWindowBounds(extScreen, 1000, 700, 100, 300)
	if bounds2.X != 1728 {
		t.Fatalf("expected X clamped to minX 1728, got %d", bounds2.X)
	}
}

func TestAppendTokenToRoute(t *testing.T) {
	// Simple route without existing query
	r1 := AppendTokenToRoute("/view/rebalancer", "secret123")
	if r1 != "/view/rebalancer?init_token=secret123" {
		t.Fatalf("unexpected route: %s", r1)
	}

	// Route with existing query
	r2 := AppendTokenToRoute("/view/page/orders?tab=active", "secret123")
	if r2 != "/view/page/orders?tab=active&init_token=secret123" {
		t.Fatalf("unexpected route with query: %s", r2)
	}

	// Empty token leaves route unchanged
	r3 := AppendTokenToRoute("/view/rebalancer", "")
	if r3 != "/view/rebalancer" {
		t.Fatalf("unexpected route: %s", r3)
	}
}

func TestIsWindowOnAnyScreen(t *testing.T) {
	screens := mockScreens() // Screen 0: [0, 1728), Screen 1: [1728, 5568), Screen 2: [-1440, 0)

	// Window on primary screen
	w1 := WindowBounds{X: 100, Y: 100, Width: 1000, Height: 600}
	if !IsWindowOnAnyScreen(w1, screens) {
		t.Fatalf("expected w1 to be detected on primary screen")
	}

	// Window on external 4K screen
	w2 := WindowBounds{X: 2000, Y: 200, Width: 1200, Height: 800}
	if !IsWindowOnAnyScreen(w2, screens) {
		t.Fatalf("expected w2 to be detected on external screen")
	}

	// Orphaned window: display was unplugged, now at X=9000
	wOrphan := WindowBounds{X: 9000, Y: 500, Width: 1200, Height: 800}
	if IsWindowOnAnyScreen(wOrphan, screens) {
		t.Fatalf("expected wOrphan to be reported NOT on any screen")
	}

	// Empty screens list returns false
	if IsWindowOnAnyScreen(w1, nil) {
		t.Fatalf("expected false on nil screens")
	}
}

func TestReclampWindowToScreen(t *testing.T) {
	screens := mockScreens()
	primary := screens[0] // X=0, Y=0, W=1728, H=1117

	// Orphaned window at (9000, 500)
	wOrphan := WindowBounds{X: 9000, Y: 500, Width: 1200, Height: 800}
	reclamped := ReclampWindowToScreen(wOrphan, primary)

	// Should be centered on primary screen
	expectedX := 0 + (1728-1200)/2 // 264
	expectedY := 0 + (1117-800)/2  // 158

	if reclamped.X != expectedX || reclamped.Y != expectedY {
		t.Fatalf("expected reclamped at (%d, %d), got (%d, %d)", expectedX, expectedY, reclamped.X, reclamped.Y)
	}
	if reclamped.Width != 1200 || reclamped.Height != 800 {
		t.Fatalf("expected dimensions preserved, got %dx%d", reclamped.Width, reclamped.Height)
	}

	// Verify that the newly reclamped window is now on screen
	if !IsWindowOnAnyScreen(reclamped, screens) {
		t.Fatalf("expected reclamped window to be on screen")
	}
}
