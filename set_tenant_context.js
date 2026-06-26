// Quick tenant setup for testing profiler persistence
// Run this in browser console to set the correct tenant context

localStorage.setItem('selected_tenant', JSON.stringify({
  id: '910638ba-a459-4a3f-bb2d-78391b0595f6',
  display_name: 'Test Tenant'
}));

localStorage.setItem('selected_product', JSON.stringify({
  id: 'test-product-id',
  alpha_product: { product_name: 'Test Product' }
}));

localStorage.setItem('selected_datasource', JSON.stringify({
  id: '982aef38-418f-46dc-acd0-35fe8f3b97b0',
  source_name: 'Test Datasource'
}));

console.log('Tenant context set! Refresh the page to see profile results.');