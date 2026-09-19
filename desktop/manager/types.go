package manager

// ScreenInfo represents display information exposed to the frontend and window manager.
type ScreenInfo struct {
	Index       int     `json:"index"`
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	IsPrimary   bool    `json:"isPrimary"`
	Scale       float32 `json:"scale"`
	ScaleFactor float32 `json:"scaleFactor"` // alias for frontend compatibility
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}

// WindowSpawnOptions represents parameters from the frontend for spawning or targeting a window.
type WindowSpawnOptions struct {
	WindowID          string `json:"windowId"`
	Route             string `json:"route"`
	Title             string `json:"title"`
	TargetScreenIndex int    `json:"targetScreenIndex"` // 0-based index or -1 for default
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	X                 int    `json:"x"`
	Y                 int    `json:"y"`
	Frameless         bool   `json:"frameless"`
	AuthToken         string `json:"authToken,omitempty"`
	SessionJWT        string `json:"sessionJwt,omitempty"`
}

// WindowBounds represents position and dimensions of a window.
type WindowBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
