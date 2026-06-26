// Permissive JSONValue for GraphQL raw JSON blobs. Using `unknown` lets
// structured TS types (e.g. SemanticModel) be assigned to fields like
// `resolved_config` without needing index signatures on every interface.
export type JSONValue = unknown;
export default JSONValue;
