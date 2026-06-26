/**
 * Browser console script to seed tenant scope for testing
 * 
 * Usage:
 * 1. Open browser DevTools console on the Fabric Builder app
 * 2. Paste this entire script and press Enter
 * 3. The script will automatically seed with Northwind tenant
 * 4. Reload the page after running
 */

function seedTenantScope(tenantId, tenantName, datasourceId, datasourceName) {
  const tenantObj = {
    id: tenantId,
    display_name: tenantName,
    name: tenantName
  };
  
  const datasourceObj = {
    id: datasourceId,
    alpha_datasource_id: datasourceId,
    source_name: datasourceName,
    alpha_datasource: {
      datasource_name: datasourceName
    }
  };
  
  localStorage.setItem('selected_tenant', JSON.stringify(tenantObj));
  localStorage.setItem('selected_datasource', JSON.stringify(datasourceObj));
  
  console.log('✅ Tenant scope seeded successfully!');
  console.log('Tenant:', tenantObj);
  console.log('Datasource:', datasourceObj);
  console.log('\n🔄 Now reload the page to activate the scope.');
}

// Real values from the alpha database:
console.log('🎯 Seeding Northwind tenant scope...');
seedTenantScope(
  '910638ba-a459-4a3f-bb2d-78391b0595f6', // Northwind tenant_id
  'Northwind',                             // tenant_name
  '982aef38-418f-46dc-acd0-35fe8f3b97b0', // datasource_id
  'Northwind Database'                     // datasource_name
);

// Alternative: GOLD_COPY tenant (uncomment to use instead)
// seedTenantScope(
//   'c52a4906-6177-44a6-80c6-0c1b7c5f30b3', // GOLD_COPY tenant_id
//   'GOLD_COPY',                             // tenant_name
//   'f938c8e6-6e11-405c-a700-ce5eacc5f45b', // datasource_id
//   'GOLD_COPY Database'                     // datasource_name
// );

