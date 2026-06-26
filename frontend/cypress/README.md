Cypress visual smoke test for Model Generator

How to run locally:

1. Install Cypress (dev dependency) from the frontend folder:

   npm install --save-dev cypress

2. Run Cypress in interactive mode:

   npx cypress open

   or run headless:

   npx cypress run --spec "cypress/e2e/model_generator_spec.ts"

Notes:
- The test stubs GraphQL `/graphql` POST requests and the model generation API `/api/fabric/models/generate`.
- If your app requires authentication or tenant setup, add the necessary stubs in `cypress/e2e/model_generator_spec.ts`.
- The test returns a non-gzipped JSON string for `tenant_chart[0].chart` so the frontend `transformChartData` will parse it.
