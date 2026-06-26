# Semantic Mapping Service - Go

This service provides a RESTful API for transforming database column names into standardized, business-friendly semantic terms. It is built with Go, making it performant and easy to integrate into existing Go-based ecosystems and serve a React frontend.

This service replicates the core logic of the Go-based `semlayer` project, including abbreviation expansion and context inference. It also generates suggested Cube.dev properties for each term.

## 🎯 Features

- **Generate Semantic Term**: Takes a schema, table, and column name and applies a series of transformations to produce a standardized semantic term.
- **Generate Cube.dev Properties**: Suggests `name`, `sql`, `type`, `title`, and `description` for use in a Cube.dev data model.
- **Context Inference**: Adds table context to generic column names (e.g., `ID` in a `users` table becomes `USER_ID`).
- **Abbreviation Expansion**: Expands common abbreviations found in column names (e.g., `CUST_ID` becomes `CUSTOMER_IDENTIFIER`).
- **Prefix Removal**: Strips common BI/DW prefixes like `DIM_` and `FCT_`.

## 🛠️ Prerequisites

- Go 1.19 or later
- PostgreSQL database

## 🚀 How to Run

The semantic mapping service is integrated into the main Go backend application. To run:

1. **Build the application:**
   ```bash
   cd backend
   go build -o semlayer-server ./cmd/server
   ```

2. **Run the application:**
   ```bash
   ./semlayer-server
   ```
   The service will start and include the semantic mapping endpoints.

## ⚙️ Configuration

Configure the service through the main application configuration file (`config.yaml`):

```yaml
database:
  host: localhost
  port: 5432
  database: semlayer
  user: postgres
  password: postgres
```

## 🌐 API Endpoints

The following endpoint is available under the `/api/semantic-mapping` path.

### Generate Semantic Term
- **Endpoint**: `POST /api/semantic-mapping/generate-term`
- **Description**: Generates a standardized semantic term and Cube.dev properties from a physical column name and its context.
- **Body**:
  ```json
  {
    "schemaName": "public",
    "tableName": "dim_customers",
    "columnName": "CUST_ID"
  }
  ```
- **Example**:
  ```bash
  curl -X POST -H "Content-Type: application/json" \
    -d '{"schemaName": "public", "tableName": "dim_customers", "columnName": "CUST_ID"}' \
    http://localhost:8080/api/semantic-mapping/generate-term
  ```
- **Example Response**:
  ```json
  {
    "generatedTerm": "CUSTOMER_IDENTIFIER",
    "cubeProperties": {
      "name": "customerIdentifier",
      "sql": "${CUBE.TABLE}.CUST_ID",
      "type": "string",
      "title": "Customer Identifier",
      "description": "The Customer Identifier."
    }
  }
  ```

## 🗑️ Cleanup

The following files from the context appear to be from an older or incorrect location and can be removed, as their functionality is now correctly implemented within `/backend/java/`:

- `/Users/eganpj/GitHub/semlayer/tools/temporal-ops/admin/README.md`
- `/Users/eganpj/GitHub/semlayer/tools/temporal-ops/admin/Abbreviation.java`
- `/Users/eganpj/GitHub/semlayer/tools/temporal-ops/admin/Synonym.java`
- All other Java files under `/Users/eganpj/GitHub/semlayer/tools/temporal-ops/admin/`

The Go files under `/Users/eganpj/GitHub/semlayer/tools/temporal-ops/` are part of a separate CLI tool and should be evaluated independently.