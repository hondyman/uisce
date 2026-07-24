package observability

import (
	"fmt"
	"strings"
	"sync"
)

// ServiceDependency represents a dependency between two services
type ServiceDependency struct {
	Source          string
	Target          string
	CallCount       int64
	ErrorCount      int64
	AverageDuration int64
	P99Duration     int64
}

// DependencyGraph builds and analyzes service dependency graphs
type DependencyGraph struct {
	tp           *TracerProvider
	dependencies map[string]map[string]*ServiceDependency
	mu           sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph analyzer
func NewDependencyGraph(tp *TracerProvider) *DependencyGraph {
	return &DependencyGraph{
		tp:           tp,
		dependencies: make(map[string]map[string]*ServiceDependency),
	}
}

// BuildGraph builds the service dependency graph from spans
func (dg *DependencyGraph) BuildGraph() error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	spans := dg.tp.GetSpans()

	// Create dependency map
	for _, span := range spans {
		if span.ParentSpanID == "" {
			// Root span, no dependency
			continue
		}

		// Find parent span to determine dependency
		for _, other := range spans {
			if other.SpanID == span.ParentSpanID && other.ServiceName != span.ServiceName {
				// Add dependency
				dg.addDependency(other.ServiceName, span.ServiceName, span)
				break
			}
		}
	}

	return nil
}

// addDependency adds or updates a dependency
func (dg *DependencyGraph) addDependency(source, target string, span *Span) {
	if _, exists := dg.dependencies[source]; !exists {
		dg.dependencies[source] = make(map[string]*ServiceDependency)
	}

	dep, exists := dg.dependencies[source][target]
	if !exists {
		dep = &ServiceDependency{
			Source: source,
			Target: target,
		}
		dg.dependencies[source][target] = dep
	}

	dep.CallCount++
	if span.Status != "ok" {
		dep.ErrorCount++
	}

	// Update average duration
	if span.Duration > 0 {
		dep.AverageDuration = (dep.AverageDuration*(dep.CallCount-1) + span.Duration) / dep.CallCount
	}

	// Update P99
	if span.Duration > dep.P99Duration {
		dep.P99Duration = span.Duration
	}
}

// GetDependencies returns all dependencies
func (dg *DependencyGraph) GetDependencies() []*ServiceDependency {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps := make([]*ServiceDependency, 0)
	for _, targets := range dg.dependencies {
		for _, dep := range targets {
			deps = append(deps, dep)
		}
	}

	return deps
}

// GetDependenciesFor returns dependencies for a specific service
func (dg *DependencyGraph) GetDependenciesFor(serviceName string) []*ServiceDependency {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps := make([]*ServiceDependency, 0)
	if targets, exists := dg.dependencies[serviceName]; exists {
		for _, dep := range targets {
			deps = append(deps, dep)
		}
	}

	return deps
}

// ExportJSONGraph exports the dependency graph as JSON
func (dg *DependencyGraph) ExportJSONGraph() string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	var b strings.Builder

	b.WriteString("{\n  \"services\": [\n")

	services := make(map[string]bool)
	for source := range dg.dependencies {
		services[source] = true
		for _, targets := range dg.dependencies {
			for target := range targets {
				services[target] = true
			}
		}
	}

	i := 0
	for service := range services {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "    {\"name\": \"%s\"}", service)
		i++
	}

	b.WriteString("\n  ],\n  \"dependencies\": [\n")

	j := 0
	for source, targets := range dg.dependencies {
		for target, dep := range targets {
			if j > 0 {
				b.WriteString(",\n")
			}
			fmt.Fprintf(&b, "    {\n")
			fmt.Fprintf(&b, "      \"source\": \"%s\",\n", source)
			fmt.Fprintf(&b, "      \"target\": \"%s\",\n", target)
			fmt.Fprintf(&b, "      \"callCount\": %d,\n", dep.CallCount)
			fmt.Fprintf(&b, "      \"errorCount\": %d,\n", dep.ErrorCount)
			fmt.Fprintf(&b, "      \"averageDuration\": %d,\n", dep.AverageDuration)
			fmt.Fprintf(&b, "      \"p99Duration\": %d\n", dep.P99Duration)
			fmt.Fprintf(&b, "    }")
			j++
		}
	}

	b.WriteString("\n  ]\n}")

	return b.String()
}

// ExportDotGraph exports the dependency graph as Graphviz DOT format
func (dg *DependencyGraph) ExportDotGraph() string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	output := "digraph ServiceDependencies {\n"
	output += "  rankdir=LR;\n"
	output += "  node [shape=box, style=rounded];\n\n"

	// Add nodes
	services := make(map[string]bool)
	for source := range dg.dependencies {
		services[source] = true
		for _, targets := range dg.dependencies {
			for target := range targets {
				services[target] = true
			}
		}
	}

	for service := range services {
		output += fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", service, service)
	}

	output += "\n"

	// Add edges
	for source, targets := range dg.dependencies {
		for target, dep := range targets {
			errorColor := "black"
			if dep.ErrorCount > 0 {
				errorColor = "red"
			}

			output += fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%d calls\", color=\"%s\", fontcolor=\"%s\"];\n",
				source, target, dep.CallCount, errorColor, errorColor)
		}
	}

	output += "}\n"

	return output
}

// GetHotPaths returns the most frequently traversed paths (hot paths)
func (dg *DependencyGraph) GetHotPaths() []*ServiceDependency {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps := make([]*ServiceDependency, 0)
	for _, targets := range dg.dependencies {
		for _, dep := range targets {
			deps = append(deps, dep)
		}
	}

	// Simple bubble sort by call count (in production, use quicksort)
	for i := 0; i < len(deps); i++ {
		for j := i + 1; j < len(deps); j++ {
			if deps[j].CallCount > deps[i].CallCount {
				deps[i], deps[j] = deps[j], deps[i]
			}
		}
	}

	// Return top 10
	if len(deps) > 10 {
		return deps[:10]
	}
	return deps
}

// GetSlowPaths returns the slowest service dependencies
func (dg *DependencyGraph) GetSlowPaths() []*ServiceDependency {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps := make([]*ServiceDependency, 0)
	for _, targets := range dg.dependencies {
		for _, dep := range targets {
			deps = append(deps, dep)
		}
	}

	// Sort by P99 duration
	for i := 0; i < len(deps); i++ {
		for j := i + 1; j < len(deps); j++ {
			if deps[j].P99Duration > deps[i].P99Duration {
				deps[i], deps[j] = deps[j], deps[i]
			}
		}
	}

	// Return top 10
	if len(deps) > 10 {
		return deps[:10]
	}
	return deps
}

// GetErrorPaths returns service dependencies with high error rates
func (dg *DependencyGraph) GetErrorPaths() []*ServiceDependency {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps := make([]*ServiceDependency, 0)
	for _, targets := range dg.dependencies {
		for _, dep := range targets {
			if dep.ErrorCount > 0 {
				deps = append(deps, dep)
			}
		}
	}

	return deps
}

// AnalyzeCriticalPath identifies the critical path through the service dependency graph
func (dg *DependencyGraph) AnalyzeCriticalPath(startService string) []*ServiceDependency {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	path := make([]*ServiceDependency, 0)
	visited := make(map[string]bool)

	dg.dfsPath(startService, &path, visited, dg.dependencies)

	return path
}

// dfsPath performs depth-first search to find critical path
func (dg *DependencyGraph) dfsPath(service string, path *[]*ServiceDependency, visited map[string]bool, deps map[string]map[string]*ServiceDependency) {
	if visited[service] {
		return
	}

	visited[service] = true

	if targets, exists := deps[service]; exists {
		for _, dep := range targets {
			*path = append(*path, dep)
			dg.dfsPath(dep.Target, path, visited, deps)
		}
	}
}
