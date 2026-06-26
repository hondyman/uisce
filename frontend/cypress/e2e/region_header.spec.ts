describe('Region header propagation', () => {
  it('includes X-Tenant-Region header on REST requests', () => {
    // Set the selected region in localStorage
    cy.visit('/');
    cy.window().then(win => {
      win.localStorage.setItem('selected_region', 'eu-west');
    });

    // Intercept the datasources request and assert header presence
    cy.intercept('GET', '/api/semantic/datasources', (req) => {
      expect(req.headers['x-tenant-region']).to.equal('eu-west');
      req.reply({ statusCode: 200, body: [] });
    }).as('getDatasources');

    // Trigger client code to call the endpoint (direct fetch to avoid UI flakiness)
    cy.window().then(win => {
      return win.fetch('/api/semantic/datasources', { headers: { 'X-Tenant-ID': 'default' } });
    });

    cy.wait('@getDatasources');
  });
});