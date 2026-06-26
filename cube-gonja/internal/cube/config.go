package cube

import (
	"sync"
)

// Config represents the configuration object for cube settings
type Config struct {
	mu sync.RWMutex

	// Base path for the cube API
	BasePath string

	// Function to extract app ID from context
	ContextToAppID func(map[string]interface{}) string

	// Function to rewrite queries
	QueryRewrite func(map[string]interface{}, map[string]interface{}) map[string]interface{}

	// Additional configuration options can be added here
	Extra map[string]interface{}
}

// Global config instance
var globalConfig = &Config{
	Extra: make(map[string]interface{}),
}

// GetConfig returns the global config instance
func GetConfig() *Config {
	return globalConfig
}

// Set sets a configuration option by name
func (c *Config) Set(name string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch name {
	case "base_path":
		if v, ok := value.(string); ok {
			c.BasePath = v
		}
	case "context_to_app_id":
		if v, ok := value.(func(map[string]interface{}) string); ok {
			c.ContextToAppID = v
		}
	case "query_rewrite":
		if v, ok := value.(func(map[string]interface{}, map[string]interface{}) map[string]interface{}); ok {
			c.QueryRewrite = v
		}
	default:
		c.Extra[name] = value
	}
}

// Get gets a configuration option by name
func (c *Config) Get(name string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch name {
	case "base_path":
		return c.BasePath
	case "context_to_app_id":
		return c.ContextToAppID
	case "query_rewrite":
		return c.QueryRewrite
	default:
		return c.Extra[name]
	}
}

// ConfigDecorator is a decorator function for setting config options
func ConfigDecorator(name string) func(interface{}) {
	return func(value interface{}) {
		globalConfig.Set(name, value)
	}
}
