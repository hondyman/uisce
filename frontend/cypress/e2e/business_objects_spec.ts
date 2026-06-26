/// <reference types="cypress" />

describe('Business Objects UI', () => {
  beforeEach(() => {
    // Set tenant and datasource in localStorage (app reads these to set headers)
    window.localStorage.setItem('selected_tenant', JSON.stringify({ id: 'tenant-1' }));
    window.localStorage.setItem('selected_datasource', JSON.stringify({ id: 'datasource-1' }));
  });

  it('lists business objects when datasource header is set', () => {
    // Stub backend response for /api/business-objects
    cy.intercept('GET', '/api/business-objects', {
      statusCode: 200,
      body: {
        'bo-1': { id: 'bo-1', name: 'Customers', displayName: 'Customers' },
        'bo-2': { id: 'bo-2', name: 'Orders', displayName: 'Orders' }
      }
    }).as('getBOs');

    cy.visit('/business-objects');

    // Wait for API call and assert it included the proper header (Cypress can inspect the request)
    cy.wait('@getBOs').its('request.headers').then((headers) => {
      expect(headers['x-tenant-id']).to.exist;
      expect(headers['x-tenant-datasource-id']).to.exist; // new header name
    });

    // Check UI shows expected objects
    cy.contains('Customers').should('exist');
    cy.contains('Orders').should('exist');
  });
});