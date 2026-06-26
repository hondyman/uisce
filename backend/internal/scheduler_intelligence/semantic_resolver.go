package scheduler_intelligence

import (
	"context"
)

// SemanticBindingResolver bridges the scheduler and the semantic graph
type SemanticBindingResolver struct {
	semanticClient SemanticClient
}

// NewSemanticBindingResolver creates a new resolver
func NewSemanticBindingResolver(client SemanticClient) *SemanticBindingResolver {
	return &SemanticBindingResolver{
		semanticClient: client,
	}
}

// ResolveForJobSpec resolves semantic references in a Job request
func (r *SemanticBindingResolver) ResolveForJobSpec(
	ctx context.Context,
	spec SemanticBinding,
) (SemanticBinding, error) {
	// Collect all reference IDs
	var refIDs []string
	refIDs = append(refIDs, spec.BOIDs...)
	refIDs = append(refIDs, spec.APIIDs...)
	refIDs = append(refIDs, spec.PageIDs...)
	refIDs = append(refIDs, spec.WorkflowIDs...)
	refIDs = append(refIDs, spec.PreAggIDs...)

	if len(refIDs) == 0 {
		return SemanticBinding{}, nil
	}

	// Resolve via semantic client
	return r.semanticClient.ResolveBindings(ctx, refIDs)
}

// ResolveForDAGSpec resolves semantic references in a DAG request
func (r *SemanticBindingResolver) ResolveForDAGSpec(
	ctx context.Context,
	spec SemanticBinding,
) (SemanticBinding, error) {
	return r.ResolveForJobSpec(ctx, spec)
}
