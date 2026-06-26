package graphql

// Stub types to satisfy generated resolver signatures when gqlgen output is partial.

// mutationResolver/queryResolver are provided by specific resolver files.
// Define the gqlgen interfaces so resolver constructors compile.
type MutationResolver interface{}
type QueryResolver interface{}
type SubscriptionResolver interface{}

// executionContext is referenced by custom scalar helpers.
type executionContext struct{}
