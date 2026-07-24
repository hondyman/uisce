package help

// FlagHelp describes a single parameter for an endpoint.
type FlagHelp struct {
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description"`
}

// EndpointHelp provides structured documentation for an API endpoint.
type EndpointHelp struct {
	Description string              `json:"description"`
	Flags       map[string]FlagHelp `json:"flags"`
}

// Registry stores help information for all registered endpoints.
type Registry struct {
	endpoints map[string]EndpointHelp
}

// NewRegistry creates a new help registry.
func NewRegistry() *Registry {
	return &Registry{
		endpoints: make(map[string]EndpointHelp),
	}
}

// Register adds help documentation for a specific endpoint path.
func (r *Registry) Register(path string, help EndpointHelp) {
	r.endpoints[path] = help
}

// GetHelp returns all registered help information.
func (r *Registry) GetHelp() map[string]EndpointHelp {
	return r.endpoints
}
