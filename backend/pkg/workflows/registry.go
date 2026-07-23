package workflows

import (
	"sync"
)

// ActivityDefinition holds the metadata for an activity
type ActivityDefinition struct {
	Name         string
	Func         interface{}
	IsClientSafe bool
}

// ActivityRegistry maps string names to activity definitions
type ActivityRegistry struct {
	mu         sync.RWMutex
	activities map[string]ActivityDefinition
}

var (
	// GlobalRegistry is the singleton registry
	GlobalRegistry = &ActivityRegistry{
		activities: make(map[string]ActivityDefinition),
	}
)

// Register adds an activity to the registry (defaults to unsafe/internal)
func (r *ActivityRegistry) Register(name string, activityFunc interface{}, isClientSafe bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activities[name] = ActivityDefinition{
		Name:         name,
		Func:         activityFunc,
		IsClientSafe: isClientSafe,
	}
}

// Get retrieves an activity definition by name
func (r *ActivityRegistry) Get(name string) (ActivityDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.activities[name]
	return def, ok
}

// GetClientSafeActivities returns a list of activities safe for client use
func (r *ActivityRegistry) GetClientSafeActivities() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name, def := range r.activities {
		if def.IsClientSafe {
			names = append(names, name)
		}
	}
	return names
}

// Helper to register globally (Internal/Unsafe by default)
func RegisterActivity(name string, activityFunc interface{}) {
	GlobalRegistry.Register(name, activityFunc, false)
}

// Helper to register globally as Client Safe
func RegisterSafeActivity(name string, activityFunc interface{}) {
	GlobalRegistry.Register(name, activityFunc, true)
}

// Helper to get globally
func GetActivity(name string) interface{} {
	def, ok := GlobalRegistry.Get(name)
	if !ok {
		return nil
	}
	return def.Func
}

// Helper to get safe list globally
func GetClientSafeActivities() []string {
	return GlobalRegistry.GetClientSafeActivities()
}
