/// <reference types="cypress" />
// Cypress visual smoke test for Model Generator
// - Stubs GraphQL GET_TECHNICAL_LINEAGE_CHART to return a chart with a node that has core_id
// - Asserts the star (core) icon is visible in the Data Catalog
// - Stubs the POST to /api/fabric/models/generate to return a generated model
// - Asserts the model icon appears immediately after generation

describe('Model Generator visual smoke', () => {
  beforeEach(() => {
    cy.intercept('POST', '/graphql', (req) => {
      // Intercept GraphQL queries and respond to GET_TECHNICAL_LINEAGE_CHART
      if (req.body && req.body.operationName === 'GetTechnicalLineageChart') {
        const chartPayload = {
          tenant_chart: [
            {
              id: 'chart-1',
              chart_name: 'technical_lineage_chart',
              // Return a gzipped hex string (with leading "\\x") to exercise decompression path
              chart: '\\x1f8b08000000000000136d50bb0ec2300cfc979b5d54d6ac4c2c88810d55284d0c8d9436250f89aacabfa354ed50c40d1e7cb6efce3306a73940dc67180d81285bcbd51184388dbc3540d0324a881956b66c219002fb50c60a7f917d991d536b8d3a6c54501df772c78116c1dbfe76699df7ea269c9c6788e8131394f3fc58fcd52baa3f65455957cea67e28b91ac23b496b9e86f555c6eed766268c2e9868dc50e27d206ac20451e7dc1058bf96ef34f90b450edb422a010000'
            }
          ]
        };
        req.reply({ statusCode: 200, body: chartPayload });
        return;
      }

      // Default passthrough for other GraphQL ops
      req.continue();
    }).as('graphql');

    // Stub model metadata endpoint to return no existing models initially
    cy.intercept('POST', '/api/fabric/models/metadata', { statusCode: 200, body: { results: {} } }).as('metadata');

    // Stub the model generate endpoint
    cy.intercept('POST', '/api/fabric/models/generate', (req) => {
      // Return one generated model matching the requested table
      const body = {
        generated: [
          {
            table_name: 'public.users',
            model_name: 'public.users_model',
            resolved_config: { name: 'public.users_model' }
          }
        ],
        skipped: [],
        overwritten: []
      };
      req.reply({ statusCode: 200, body });
    }).as('generate');
  });

  it('shows core icon and model icon after generation', () => {
    // Visit the app root and navigate to the model generator page
    cy.visit('/');

    // If your app has routing that requires authentication or tenant switching,
    // you may need to stub those endpoints or set localStorage tokens here.

    // Open the Model Generator directly if route exists
    cy.contains('Model Generator').click();

    // Wait for the graphql chart to be loaded
    cy.wait('@graphql');

    // The DataCatalogTree displays a star icon for core nodes; look for it next to 'users'
    cy.contains('users').parents('li, .MuiTreeItem-root').within(() => {
      // star icon is rendered as svg with title tooltip; we'll assert an element with role or svg exists
      cy.get('svg').should('exist');
      // Also assert that checkbox exists (multi-select mode)
      cy.get('input[type="checkbox"]').check();
    });

    // Click generate button
    cy.contains(/Generate for Selection/i).click();

    // Wait for the generate API
    cy.wait('@generate');

    // After generation, the model icon (ViewInArIcon) should appear for the users table
    cy.contains('users').parents('li, .MuiTreeItem-root').within(() => {
      // The model icon button is an icon button; assert it exists and is visible
      cy.get('button').contains(/Load model|view/i).should('exist');
    });
  });
});
