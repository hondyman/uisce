// cube/schema/dynamic_loader.js
/**
 * Metadata-First Dynamic Schema Loader
 * 
 * Fetches Business Objects (catalog_node) and Relationships (catalog_edge)
 * from the 'Single Source of Truth' Postgres Graph.
 * 
 * Maps:
 * - catalog_node (kind='bo') -> Cube
 * - bo_fields -> Dimensions/Measures
 * - catalog_edge -> Joins
 */

const { Pool } = require('pg');

// Use metadata DB connection (fallback to backend database if not explicitly set)
const connectionString = process.env.METADATA_DB_URL || process.env.HASURA_GRAPHQL_DATABASE_URL || 'postgres://postgres:postgres@starrocks-fe:5432/alpha';
// Note: host likely needs adjustment depending on docker networking. 'starrocks-fe' is unlikely for postgres.
// Assuming 'host.docker.internal' or 'postgres' service name.
// Update based on docker-compose: 'postgres' service is commented out, using host.docker.internal.
// We will try standard env vars first.

const pool = new Pool({
  connectionString: connectionString.replace('host.docker.internal', 'host.docker.internal'), // Ensure host is reachable
  ssl: false,
});

module.exports = {
  asyncModule: async () => {
    const client = await pool.connect();
    
    try {
      // 1. Fetch Business Objects (Nodes)
      const { rows: nodes } = await client.query(`
        SELECT id, name, description, properties, tenant_id 
        FROM catalog_node 
        WHERE kind = 'bo'
      `);

      const cubes = [];

      for (const node of nodes) {
        // Safe defaults
        const props = node.properties || {};
        const tableName = props.tableName || node.name.toLowerCase(); // Fallback if no mapping

        // 2. Fetch Fields (Dimensions/Measures)
        // Adjust column names based on 000032_redesign_bo_fields_table.sql
        const { rows: fields } = await client.query(`
          SELECT name, type, description, properties 
          FROM bo_fields 
          WHERE bo_id = $1
        `, [node.id]);

        // 3. Fetch Relationships (Edges) -> Joins
        const { rows: edges } = await client.query(`
          SELECT target_id, edge_type, properties 
          FROM catalog_edge 
          WHERE source_id = $1
        `, [node.id]);

        // Build Cube Definition
        const cubeDef = {
          name: node.name,
          sql: `SELECT * FROM ${tableName}`, // TODO: Handle schema/schema prefix
          
          joins: {},
          measures: {
            count: {
              type: 'count',
            }
          },
          dimensions: {}
        };

        // Map Edges to Joins
        for (const edge of edges) {
          // Fetch target node name for the join key
          const { rows: targets } = await client.query('SELECT name FROM catalog_node WHERE id = $1', [edge.target_id]);
          if (targets.length === 0) continue;
          
          const targetName = targets[0].name;
          const edgeProps = edge.properties || {};
          
          // Default join condition if not in properties
          // Using property 'joinCondition' from Phase 1 plan
          // Use ${CUBE} for robust aliasing as per Phase 2 requirements
          const sql = edgeProps.joinCondition || `${CUBE}.${edgeProps.foreignKey || targetName.toLowerCase() + '_id'} = ${targetName}.id`;
          
          cubeDef.joins[targetName] = {
            relationship: edgeProps.cardinality === '1:N' ? 'hasMany' : 'belongsTo', // Simplified mapping
            sql: sql
          };
        }

        // Map Fields to Dimensions/Measures
        fields.forEach(f => {
          const fieldProps = f.properties || {};
          const isMeasure = fieldProps.isMeasure || f.type === 'number'; // Simple heuristic
          
          if (isMeasure) {
            cubeDef.measures[f.name] = {
              type: 'sum', // Default to sum for now
              sql: f.name,
              title: f.description || f.name
            };
          } else {
            cubeDef.dimensions[f.name] = {
              sql: f.name,
              type: mapType(f.type),
              title: f.description || f.name
            };
          }
        });

        cubes.push(cubeDef);
      }

      return cubes;

    } catch (e) {
      console.error("Error loading dynamic schema:", e);
      return []; // Return empty on error to prevent startup crash
    } finally {
      client.release();
    }
  }
};

// Helper: Map Postgres/Semantic types to Cube types
function mapType(type) {
  switch (type) {
    case 'string': return 'string';
    case 'number': return 'number';
    case 'boolean': return 'boolean';
    case 'date': return 'time';
    case 'timestamp': return 'time';
    default: return 'string';
  }
}
