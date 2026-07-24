package viewmodel

// View represents a generated or user-authored view over cubes.
type View struct {
	Name         string         `json:"name"`
	Extends      string         `json:"extends,omitempty"`
	Title        string         `json:"title,omitempty"`
	Description  string         `json:"description,omitempty"`
	Public       any            `json:"public,omitempty"` // bool or expression string
	Meta         map[string]any `json:"meta,omitempty"`
	Cubes        []ViewCube     `json:"cubes"`
	Folders      []Folder       `json:"folders,omitempty"`
	AccessPolicy *AccessPolicy  `json:"access_policy,omitempty"`
}

type ViewCube struct {
	JoinPath string        `json:"join_path"`
	Prefix   bool          `json:"prefix,omitempty"`
	Alias    string        `json:"alias,omitempty"`
	Includes []IncludeItem `json:"includes"`
	Excludes []string      `json:"excludes,omitempty"`
}

type IncludeItem struct {
	Name        string         `json:"name"`
	Alias       string         `json:"alias,omitempty"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Format      string         `json:"format,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}

type Folder struct {
	Name     string       `json:"name"`
	Includes []FolderItem `json:"includes"`
}

type FolderItem struct {
	Member string  `json:"member,omitempty"`
	Nested *Folder `json:"nested,omitempty"`
}

type AccessPolicy struct {
	Rules any `json:"rules,omitempty"`
}

// ViewExtension represents a user-authored extension for a generated view.
// It mirrors View but requires Extends to point at a generated base and
// omits fields not typically overridden.
type ViewExtension struct {
	Name        string         `json:"name"`
	Extends     string         `json:"extends"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Public      any            `json:"public,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
	Cubes       []ViewCube     `json:"cubes,omitempty"`
	Folders     []Folder       `json:"folders,omitempty"`
}
