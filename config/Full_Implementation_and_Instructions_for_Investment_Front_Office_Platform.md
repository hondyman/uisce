# Full Implementation and Instructions for Investment Front Office Platform

This document provides a comprehensive implementation for your investment front office platform, incorporating the Single Table Inheritance (STI) pattern as recommended. This approach consolidates sub-entities into single tables (e.g., one for clients/investors and one for portfolios) using a discriminator column (`type`) to distinguish subtypes, while preserving multitenancy via `tenant_id`, extensibility through `custom_fields` (JSONB), and seamless upgrades. Core attributes remain as typed columns for performance and integrity, with subtype-specific fields as nullable columns.

This design ensures data isolation per tenant, allows tenant-specific customizations without schema changes, and facilitates upgrades by modifying only core columns.

---

## 1. PostgreSQL Database Schema (DDL)

The schema uses single tables for entity groups (e.g., `client_investors` encompasses all subtypes).

```sql
-- Enable UUID extension for primary keys
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table (for multitenancy management)
CREATE TABLE tenants (
    tenant_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Consolidated Client/Investor table (STI pattern)
CREATE TABLE client_investors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('ClientInvestor', 'IndividualInvestor', 'InstitutionalInvestor', 'FamilyOffice')),  -- Discriminator
    name VARCHAR(255) NOT NULL,
    contact_details JSONB,  -- e.g., {"email": "user@example.com", "phone": "+123456789"}
    risk_tolerance VARCHAR(50) CHECK (risk_tolerance IN ('Low', 'Medium', 'High')),
    investment_objectives TEXT[],
    regulatory_status VARCHAR(100),
    -- Subtype-specific fields (nullable)
    age INTEGER,  -- IndividualInvestor
    tax_id VARCHAR(50),  -- IndividualInvestor
    beneficiary VARCHAR(255),  -- IndividualInvestor
    org_type VARCHAR(100),  -- InstitutionalInvestor
    signatories TEXT[],  -- InstitutionalInvestor
    custody VARCHAR(255),  -- InstitutionalInvestor
    generations INTEGER,  -- FamilyOffice
    aggregated_reporting BOOLEAN DEFAULT FALSE,  -- FamilyOffice
    custom_fields JSONB DEFAULT '{}',  -- Tenant custom extensions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Consolidated Portfolio table (STI pattern)
CREATE TABLE portfolios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES client_investors(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('Portfolio', 'DiscretionaryPortfolio', 'NonDiscretionaryPortfolio', 'ModelPortfolio')),  -- Discriminator
    benchmark VARCHAR(100),
    asset_allocation_targets JSONB,  -- e.g., {"equity": 60, "fixedIncome": 40}
    performance_metrics JSONB,  -- e.g., {"return": 5.2, "volatility": 10.1}
    -- Subtype-specific fields (nullable)
    advisor_discretion BOOLEAN DEFAULT TRUE,  -- DiscretionaryPortfolio
    client_approval_required BOOLEAN DEFAULT TRUE,  -- NonDiscretionaryPortfolio
    template_id UUID,  -- ModelPortfolio
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_client_investors_tenant ON client_investors(tenant_id);
CREATE INDEX idx_client_investors_type ON client_investors(type);
CREATE INDEX idx_portfolios_tenant ON portfolios(tenant_id);
CREATE INDEX idx_portfolios_client ON portfolios(client_id);
CREATE INDEX idx_portfolios_type ON portfolios(type);

-- Trigger for updated_at (apply to each table)
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_client_investors_ts
BEFORE UPDATE ON client_investors
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_portfolios_ts
BEFORE UPDATE ON portfolios
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- View for unified querying across entities (for Hasura)
CREATE VIEW all_entities AS
SELECT 
    'client_investors' AS entity_type, 
    id, 
    tenant_id, 
    name, 
    custom_fields, 
    created_at, 
    row_to_json(t.*) AS full_data
FROM client_investors t
UNION ALL
SELECT 
    'portfolios' AS entity_type, 
    id, 
    tenant_id, 
    benchmark AS name,  -- Alias for consistency in views
    custom_fields, 
    created_at, 
    row_to_json(t.*) AS full_data
FROM portfolios t;
-- Extend UNION ALL for additional entities as needed.
```

Notes:
- For production deployments consider enabling Row-Level Security (RLS) and defining policies that enforce tenant isolation (e.g., allow access only where tenant_id = X-Hasura-Tenant-Id session variable).
- Consider adding partial indexes and more selective indexes based on query patterns (e.g., GIN indexes on JSONB columns if you search custom_fields frequently).

---

## 2. Hasura GraphQL Configuration

Hasura provides an auto-generated GraphQL API. Configure it to connect to your PostgreSQL database, track the tables (`tenants`, `client_investors`, `portfolios`) and the `all_entities` view, and enable RLS with a policy like `{ "tenant_id": {"_eq": "X-Hasura-Tenant-Id"} }` (passed via session variables).

Example GraphQL Operations

Query Entities:
```graphql
query GetEntities($entityType: String, $tenantId: uuid!, $search: String) {
  all_entities(where: {
    tenant_id: {_eq: $tenantId},
    entity_type: {_eq: $entityType},
    _or: [{name: {_ilike: $search}}, {custom_fields: {_contains: {search: $search}}}]
  }) {
    entity_type
    id
    name
    custom_fields
    created_at
    full_data
  }
}
```

Insert (e.g., ClientInvestor):
```graphql
mutation InsertClientInvestor($object: client_investors_insert_input!) {
  insert_client_investors_one(object: $object) {
    id
  }
}
```

Update:
```graphql
mutation UpdateEntity($id: uuid!, $changes: client_investors_set_input!) {
  update_client_investors_by_pk(pk_columns: {id: $id}, _set: $changes) {
    id
  }
}
```

Delete:
```graphql
mutation DeleteEntity($id: uuid!) {
  delete_client_investors_by_pk(id: $id) {
    id
  }
}
```

For dynamic operations across entities, use Hasura actions pointing to custom SQL or the Go backend.

---

## 3. Go Backend Implementation

The backend serves as an API gateway, handling JWT authentication for tenant resolution and proxying GraphQL requests to Hasura. It supports dynamic CRUD via a single endpoint.

Project Structure

- `main.go`: Entry point.

Use dependencies: `gin`, `github.com/golang-jwt/jwt/v4`, `github.com/machinebox/graphql`.

Code (main.go)

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/machinebox/graphql"
)

// Hasura configuration
const hasuraURL = "http://localhost:8080/v1/graphql" // Adjust to your Hasura endpoint
const hasuraAdminSecret = "your-hasura-secret" // Set via environment variable

// JWT Claims
type Claims struct {
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// Dynamic Request Struct
type DynamicRequest struct {
	EntityType string                 `json:"entity_type"`
	Data       map[string]interface{} `json:"data"`
	IDs        []string               `json:"ids"`
}

// Authentication Middleware
func tenantAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-jwt-secret"), nil // Replace with secure key
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}

// Proxy GraphQL to Hasura
func proxyGraphQL(c *gin.Context) {
	client := graphql.NewClient(hasuraURL)
	reqBody := make(map[string]interface{})
	if err := c.BindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := graphql.NewRequest(reqBody["query"].(string))
	req.Header.Set("X-Hasura-Admin-Secret", hasuraAdminSecret)
	req.Header.Set("X-Hasura-Tenant-Id", c.GetString("tenant_id"))

	var resp interface{}
	if err := client.Run(context.Background(), req, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Dynamic CRUD Endpoint
func dynamicCRUD(c *gin.Context) {
	var req DynamicRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	req.Data["tenant_id"] = tenantID

	client := graphql.NewClient(hasuraURL)
	var mutation string
	switch c.Request.Method {
	case "POST": // Insert
		mutation = fmt.Sprintf(`mutation { insert_%s_one(object: %s) { id } }`, req.EntityType, jsonMarshal(req.Data))
	case "PUT": // Update
		mutation = fmt.Sprintf(`mutation { update_%s_by_pk(pk_columns: {id: "%s"}, _set: %s) { id } }`, req.EntityType, req.Data["id"], jsonMarshal(req.Data))
	case "DELETE": // Delete
		mutation = fmt.Sprintf(`mutation { delete_%s(where: {id: {_in: %s}}) { affected_rows } }`, req.EntityType, jsonMarshal(req.IDs))
	}

	reqGQL := graphql.NewRequest(mutation)
	reqGQL.Header.Set("X-Hasura-Admin-Secret", hasuraAdminSecret)

	var resp interface{}
	if err := client.Run(context.Background(), reqGQL, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func jsonMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func main() {
	r := gin.Default()
	r.Use(tenantAuthMiddleware())

	r.POST("/graphql", proxyGraphQL)
	r.POST("/crud", dynamicCRUD)   // For insert
	r.PUT("/crud", dynamicCRUD)    // For update
	r.DELETE("/crud", dynamicCRUD) // For delete

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(r.Run(":" + port))
}
```

Notes:
- Install dependencies: `go get github.com/gin-gonic/gin github.com/golang-jwt/jwt/v4 github.com/machinebox/graphql`.
- Run: `go run main.go`.
- Secure the JWT key and Hasura secret in environment variables.

---

## 4. Vite + React + TypeScript Frontend Implementation

The frontend features a single unified CRUD page. Use Vite for development, React for UI, Ant Design for components, and Apollo Client for GraphQL.

Project Setup

1. Create the project:

```bash
npm create vite@latest -- --template react-ts
```

2. Install dependencies:

```bash
npm install @apollo/client graphql antd @ant-design/icons dayjs
```

Key Files

`src/App.tsx`:

```tsx
import { ApolloClient, InMemoryCache, ApolloProvider } from '@apollo/client';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import UnifiedCRUDPage from './pages/UnifiedCRUDPage';

const client = new ApolloClient({
  uri: 'http://localhost:8080/graphql', // Backend endpoint
  cache: new InMemoryCache(),
  headers: { Authorization: 'your-jwt-token' } // Replace with auth logic
});

function App() {
  return (
    <ApolloProvider client={client}>
      <Router>
        <Routes>
          <Route path="/" element={<UnifiedCRUDPage />} />
        </Routes>
      </Router>
    </ApolloProvider>
  );
}

export default App;
```

`src/pages/UnifiedCRUDPage.tsx` (Full Code):

```tsx
import React, { useState } from 'react';
import { useQuery, useMutation, gql } from '@apollo/client';
import {
  Table, Button, Input, Modal, Form, Select, Tag, Space, Popconfirm
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';

const { Option } = Select;
const { Search } = Input;

// GraphQL Definitions
const GET_ENTITIES = gql`
  query GetEntities($entityType: String, $tenantId: uuid!, $search: String) {
    all_entities(where: {
      tenant_id: {_eq: $tenantId},
      entity_type: {_eq: $entityType},
      _or: [{name: {_ilike: $search}}, {custom_fields: {_contains: {search: $search}}}]
    }) {
      entity_type
      id
      name
      custom_fields
      created_at
      full_data
    }
  }
`;

const INSERT_ENTITY = gql`
  mutation InsertEntity($entityType: String!, $object: jsonb!) {
    dynamic_insert(entity_type: $entityType, object: $object) {
      id
    }
  }
`;

const UPDATE_ENTITY = gql`
  mutation UpdateEntity($entityType: String!, $id: uuid!, $changes: jsonb!) {
    dynamic_update(entity_type: $entityType, id: $id, changes: $changes) {
      id
    }
  }
`;

const DELETE_ENTITIES = gql`
  mutation DeleteEntities($entityType: String!, $ids: [uuid!]!) {
    dynamic_delete(entity_type: $entityType, ids: $ids) {
      affected_rows
    }
  }
`;

interface EntityRecord {
  id: string;
  entity_type: string;
  name: string;
  custom_fields: any;
  created_at: string;
  full_data: any;
}

const ENTITY_SCHEMAS: { [key: string]: { fields: { key: string; label: string; type: string; options?: string[] }[] } } = {
  client_investors: {
    fields: [
      { key: 'type', label: 'Type', type: 'select', options: ['ClientInvestor', 'IndividualInvestor', 'InstitutionalInvestor', 'FamilyOffice'] },
      { key: 'name', label: 'Name', type: 'text' },
      { key: 'risk_tolerance', label: 'Risk Tolerance', type: 'select', options: ['Low', 'Medium', 'High'] },
      { key: 'custom_fields', label: 'Custom Fields', type: 'json' },
      // Add subtype fields as needed, e.g., { key: 'age', label: 'Age', type: 'number' }
    ]
  },
  portfolios: {
    fields: [
      { key: 'type', label: 'Type', type: 'select', options: ['Portfolio', 'DiscretionaryPortfolio', 'NonDiscretionaryPortfolio', 'ModelPortfolio'] },
      { key: 'benchmark', label: 'Benchmark', type: 'text' },
      { key: 'custom_fields', label: 'Custom Fields', type: 'json' },
      // Add subtype fields
    ]
  }
  // Extend for other entities
};

const UnifiedCRUDPage: React.FC = () => {
  const tenantId = 'your-tenant-uuid'; // Integrate with auth
  const [entityType, setEntityType] = useState('client_investors');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedRecord, setSelectedRecord] = useState<EntityRecord | null>(null);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const { loading, data, refetch } = useQuery(GET_ENTITIES, {
    variables: { entityType, tenantId, search: `%${searchTerm}%` },
  });

  const [insertEntity] = useMutation(INSERT_ENTITY);
  const [updateEntity] = useMutation(UPDATE_ENTITY);
  const [deleteEntities] = useMutation(DELETE_ENTITIES);

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Custom Fields', key: 'custom_fields', render: (record: EntityRecord) => <Tag>{JSON.stringify(record.custom_fields)}</Tag> },
    { title: 'Created At', dataIndex: 'created_at', key: 'created_at' },
    {
      title: 'Actions',
      key: 'actions',
      render: (record: EntityRecord) => (
        <Space>
          <Button icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Popconfirm title="Confirm delete?" onConfirm={() => handleDelete([record.id])}>
            <Button icon={<DeleteOutlined />} danger />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const handleEdit = (record: EntityRecord) => {
    setSelectedRecord(record);
    form.setFieldsValue(record.full_data);
    setIsModalVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    try {
      if (selectedRecord) {
        await updateEntity({ variables: { entityType, id: selectedRecord.id, changes: values } });
      } else {
        await insertEntity({ variables: { entityType, object: values } });
      }
      setIsModalVisible(false);
      refetch();
    } catch (error) {
      console.error(error);
    }
  };

  const handleDelete = async (ids: string[]) => {
    await deleteEntities({ variables: { entityType, ids } });
    refetch();
  };

  const renderFormFields = () => {
    const schema = ENTITY_SCHEMAS[entityType];
    return schema.fields.map(field => (
      <Form.Item key={field.key} name={field.key} label={field.label}>
        {field.type === 'select' ? (
          <Select>{field.options?.map(opt => <Option key={opt} value={opt}>{opt}</Option>)}</Select>
        ) : field.type === 'json' ? (
          <Input.TextArea />
        ) : (
          <Input />
        )}
      </Form.Item>
    ));
  };

  return (
    <div>
      <Select value={entityType} onChange={setEntityType}>
        <Option value="client_investors">Client Investors</Option>
        <Option value="portfolios">Portfolios</Option>
      </Select>
      <Search placeholder="Search" onSearch={setSearchTerm} />
      <Button icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>Add</Button>
      <Table
        rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
        dataSource={data?.all_entities || []}
        columns={columns}
        rowKey="id"
        loading={loading}
      />
      <Modal visible={isModalVisible} onOk={handleSubmit} onCancel={() => setIsModalVisible(false)}>
        <Form form={form}>{renderFormFields()}</Form>
      </Modal>
    </div>
  );
};

export default UnifiedCRUDPage;
```

Instructions

- Run development server: `npm run dev`.
- Build for production: `npm run build`.
- Extend `ENTITY_SCHEMAS` for additional fields/entities.

---

## 5. Entity Manager (New Section)

The Entity Manager provides a centralized admin UI and API for registering entity types, mapping tenant-specific UI fields, and managing migrations/transformations. It sits alongside Hasura and the Go backend and offers the following responsibilities:

- Entity Registry: stores metadata about entity groups (e.g., `client_investors`, `portfolios`), available subtypes, display labels, and default field schemas.
- Tenant Customizations: stores per-tenant overrides for field visibility, labels, validation rules, and default values in a `tenant_entity_customizations` table.
- Migration Jobs: tracks schema migrations that must be applied across tenants (e.g., adding a core column), and supports zero-downtime migration patterns.
- Validation / Enrichment Hooks: allows registering webhook endpoints or Hasura actions to validate or enrich data during create/update operations.

Suggested Schema for Entity Manager tables:

```sql
CREATE TABLE entity_registry (
  entity_name TEXT PRIMARY KEY, -- e.g., 'client_investors'
  display_name TEXT NOT NULL,
  default_schema JSONB NOT NULL, -- canonical field list and types
  subtypes JSONB DEFAULT '[]', -- allowed discriminator values
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tenant_entity_customizations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  entity_name TEXT NOT NULL REFERENCES entity_registry(entity_name),
  schema_overrides JSONB DEFAULT '{}', -- overrides for field visibility, labels, validation
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE migration_jobs (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  job_name TEXT NOT NULL,
  job_spec JSONB NOT NULL, -- description of the migration e.g., ALTER TABLE ...
  status TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
  started_at TIMESTAMP WITH TIME ZONE,
  completed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

Entity Manager responsibilities and flow:

1. Admin registers entity in `entity_registry` with `default_schema` describing fields and types.
2. Tenant admin creates customization rows in `tenant_entity_customizations` to hide fields or change labels.
3. UI (frontend) fetches the registry and tenant customizations to build forms dynamically.
4. On submit, frontend calls the Go backend which enforces tenant_id and validation hooks, then proxies to Hasura.
5. Migration jobs are queued in `migration_jobs`; a worker service applies DDL safely and updates job status.

Hasura Integration

- Track `entity_registry` and `tenant_entity_customizations` in Hasura to allow UI and admin APIs to use GraphQL.
- Use Hasura event triggers or actions to call the backend's validation/enrichment endpoints before inserts/updates.

Security

- Enforce RLS on `tenant_entity_customizations` and entity tables so tenants can only see and modify their own customizations.
- Admin roles (platform operators) can read/write `entity_registry` and `migration_jobs` with higher privileges.

---

## 6. Setup and Deployment Instructions

Database:

1. Install PostgreSQL (v14+ recommended).
2. Create database: `createdb investment_db`.
3. Execute DDL: `psql -d investment_db -f schema.sql`.

Hasura:

1. Install Hasura (docker or CLI).
2. Start: `docker run -p 8080:8080 hasura/graphql-engine:latest`.
3. Console: Access at `http://localhost:8080/console`, connect to PostgreSQL, track tables/views.

Backend:

1. Set environment variables: `HASURA_ADMIN_SECRET`, `JWT_SECRET`.
2. Run: `go run main.go`.

Frontend:

1. Install: `npm install`.
2. Run: `npm run dev` (access at `http://localhost:5173`).

Integration:

- Auth: Implement JWT generation for tenants (e.g., login endpoint) and issue tokens containing `tenant_id` in claims.
- Testing: Use tools like Postman for backend, browser for frontend.
- Deployment: Use Docker for each component; host on AWS/GCP with secrets management.

Upgrades:

- Add core columns via `ALTER TABLE` (e.g., `ALTER TABLE client_investors ADD COLUMN new_field TEXT;`).
- Tenant customizations in `custom_fields` remain unaffected.

---

## 7. Next Steps and Extensions

- Add full RLS policies examples for Hasura and test them.
- Implement a small worker for `migration_jobs` to apply DDL with safety checks.
- Add API tests and frontend integration tests.
- Expand `ENTITY_SCHEMAS` and UI components for subtype-specific fields and conditional field rendering.

---

This implementation is ready for extension. If you want, I can:

- Add a sample `schema.sql` file to the repo with the DDL above.
- Implement the Go backend boilerplate under `backend/` in this repo.
- Add the Vite frontend scaffold under `frontend/` with the `UnifiedCRUDPage` component.

Tell me which of the above you'd like next and I'll proceed.