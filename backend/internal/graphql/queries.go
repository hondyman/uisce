package graphql

// GraphQL queries for interacting with the catalog.
const (
	// GetCatalogNodesForHash retrieves all qualified paths for a datasource to compute a schema hash.
	GetCatalogNodesForHash = `
		query GetCatalogNodesForHash($datasourceId: uuid!) {
			catalog_node(
				where: { tenant_datasource_id: { _eq: $datasourceId } },
				order_by: { qualified_path: asc }
			) {
				qualified_path
			}
		}
	`

	// GetGoldCopyNodes retrieves all nodes for a gold copy datasource for mapping.
	GetGoldCopyNodes = `
		query GetGoldCopyNodes($datasourceId: uuid!) {
			catalog_node(
				where: {
					tenant_datasource_id: { _eq: $datasourceId },
					core_id: { _is_null: true }
				}
			) {
				id
				node_type_id
				qualified_path
			}
		}
	`
)
