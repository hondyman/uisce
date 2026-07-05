import re

with open("backend/internal/api/api.go", "r") as f:
    content = f.read()

# Fix debug query
q1 = """					SELECT id, node_name, COALESCE(description, ''), tenant_id, tenant_datasource_id, created_at, updated_at, COALESCE(properties, '{}'::jsonb) as properties
					FROM catalog_node"""
q1_new = """					SELECT id, node_name, COALESCE(description, ''), tenant_id, tenant_datasource_id, created_at, updated_at, COALESCE(properties, '{}'::jsonb) as properties, node_type_id
					FROM catalog_node"""
content = content.replace(q1, q1_new)

scan1 = "err := rows.Scan(&id, &nodeName, &description, &tenantID, &datasourceID, &createdAt, &updatedAt, &propsJSON)"
scan1_new = """var nodeTypeID string\n\t\t\terr := rows.Scan(&id, &nodeName, &description, &tenantID, &datasourceID, &createdAt, &updatedAt, &propsJSON, &nodeTypeID)"""
content = content.replace(scan1, scan1_new)

node1 = """"properties":           props,
			}"""
node1_new = """"properties":           props,
				"node_type_id":         nodeTypeID,
			}"""
content = content.replace(node1, node1_new)

# Fix main query
q2 = """SELECT DISTINCT ON (cn.node_name) cn.id, cn.node_name, COALESCE(cn.description, ''), cn.tenant_id, cn.tenant_datasource_id, cn.created_at, cn.updated_at, COALESCE(cn.properties, '{}'::jsonb) as properties, COALESCE(cnt.catalog_type_name, 'table') as catalog_type
				FROM catalog_node cn"""
q2_new = """SELECT DISTINCT ON (cn.node_name) cn.id, cn.node_name, COALESCE(cn.description, ''), cn.tenant_id, cn.tenant_datasource_id, cn.created_at, cn.updated_at, COALESCE(cn.properties, '{}'::jsonb) as properties, COALESCE(cnt.catalog_type_name, 'table') as catalog_type, cn.node_type_id
				FROM catalog_node cn"""
content = content.replace(q2, q2_new)

scan2 = "err := rows.Scan(&id, &nodeName, &description, &tID, &dsID, &createdAt, &updatedAt, &propsJSON, &catalogTypeName)"
scan2_new = """var dbNodeTypeID string\n\t\terr := rows.Scan(&id, &nodeName, &description, &tID, &dsID, &createdAt, &updatedAt, &propsJSON, &catalogTypeName, &dbNodeTypeID)"""
content = content.replace(scan2, scan2_new)

node2 = """"properties":           props,
		}"""
node2_new = """"properties":           props,
			"node_type_id":         dbNodeTypeID,
		}"""
content = content.replace(node2, node2_new)

with open("backend/internal/api/api.go", "w") as f:
    f.write(content)
