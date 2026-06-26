import express from 'express';
const app = express();

app.use(express.json());

// Mock Hasura GraphQL endpoint
app.post('/v1/graphql', (req, res) => {
  // Mock response for tenant_chart query
  if (req.body.query && req.body.query.includes('tenant_chart')) {
    res.json({
      data: {
        tenant_chart: [{
          chart: "eyJub2RlcyI6W10sImVkZ2VzIjpbXSwidmlld3BvcnQiOnsiY2VudGVyIjpbMCwwXSwiem9vbSI6MX0sIm1ldGEiOnt9fQ==" // base64 empty chart
        }]
      }
    });
  } else if (req.body.query && req.body.query.includes('business_terms')) {
    res.json({
      data: {
        business_terms: [{
          id: "customer_id",
          node_name: "Customer ID",
          description: "Unique identifier for customers",
          properties: {}
        }]
      }
    });
  } else {
    res.json({
      data: {}
    });
  }
});

// Health check
app.get('/healthz', (req, res) => {
  res.json({ status: 'ok' });
});

app.listen(8082, () => {
  console.error('Mock Hasura server running on port 8082');
});
