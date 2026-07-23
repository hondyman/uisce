# Semantic Term Generation Flow

This flowchart visualizes the sequence of transformations applied inside the `SemanticMappingService`.

```mermaid
graph TD
    A[Input: (schema, table, column)] --> B{Start Normalization};
    B --> C[1. Uppercase & Trim];
    C --> D[2. Split camelCase <br> e.g., 'firstName' -> 'FIRST_NAME'];
    D --> E[3. Remove BI Prefixes <br> e.g., 'FCT_ORDERS' -> 'ORDERS'];
    E --> F[4. Singularize <br> e.g., 'ORDERS' -> 'ORDER'];
    F --> G[5. Expand Abbreviations <br> e.g., 'CUST_ID' -> 'CUSTOMER_IDENTIFIER'];
    G --> H[6. Add Context to Generics <br> e.g., 'ORDER' + 'DATE' -> 'ORDER_DATE'];
    H --> I[7. Remove Redundancy <br> e.g., 'ORDER_ORDER_ID' -> 'ORDER_ID'];
    I --> J[Output: Generated Term <br> e.g., 'CUSTOMER_IDENTIFIER'];

    J --> K{Generate Cube.dev Properties};
    K --> L[Infer Type <br> e.g., '..._IDENTIFIER' -> 'string'];
    K --> M[Format Title <br> e.g., 'CUSTOMER_IDENTIFIER' -> 'Customer Identifier'];
    K --> N[Format Name (camelCase) <br> e.g., 'CUSTOMER_IDENTIFIER' -> 'customerIdentifier'];
    
    subgraph Final API Response
        direction LR
        O["generatedTerm: 'CUSTOMER_IDENTIFIER'"]
        P["cubeProperties: {...}"]
    end

    J --> O;
    L --> P;
    M --> P;
    N --> P;

    style A fill:#f9f,stroke:#333,stroke-width:2px
    style J fill:#bbf,stroke:#333,stroke-width:2px
    style O fill:#ccf,stroke:#333,stroke-width:1px
    style P fill:#ccf,stroke:#333,stroke-width:1px
```

### How to Read the Flowchart

1.  **Normalization Pipeline (Top Half):** The process starts with the raw table and column names. It then moves through a series of cleaning and standardization steps, such as removing prefixes, expanding abbreviations (`CUST_ID` -> `CUSTOMER_IDENTIFIER`), and adding context (`ID` in an `orders` table becomes `ORDER_ID`).
2.  **Generated Term:** The result of this pipeline is a clean, standardized `generatedTerm`.
3.  **Cube.dev Properties (Bottom Half):** This `generatedTerm` is then used as input to create a set of suggested properties for a Cube.dev schema. It infers the data type, creates a human-readable title, and formats the name into `camelCase` for use in the data model.
4.  **Final API Response:** The service combines the `generatedTerm` and the `cubeProperties` into a single JSON object that is sent back to the client.

This visual shows how a potentially messy column name like `dim_customers.CUST_ID` is methodically transformed into a valuable and structured semantic asset.

## 🎯 Features

- **Generate Semantic Term**: Takes a schema, table, and column name and applies a series of transformations to produce a standardized semantic term.
- **Generate Cube.dev Properties**: Suggests `name`, `sql`, `type`, `title`, and `description` for use in a Cube.dev data model.
- **Context Inference**: Adds table context to generic column names (e.g., `ID` in a `users` table becomes `USER_ID`).
- **Abbreviation Expansion**: Expands common abbreviations found in column names (e.g., `CUST_ID` becomes `CUSTOMER_IDENTIFIER`).
- **Prefix Removal**: Strips common BI/DW prefixes like `DIM_` and `FCT_`.

## 🛠️ Prerequisites

- Java 17 or later
- Apache Maven 3.6+

## 🚀 How to Run

1.  **Build the application:**
    Open a terminal in the `backend/java` directory and run:
    ```bash
    mvn clean package
    ```
    This will compile the code and create an executable JAR file in the `target/` directory.

2.  **Run the application:**
    ```bash
    java -jar target/semantic-mapper-0.0.1-SNAPSHOT.jar
    ```
    The service will start on port `8080` by default.

## ⚙️ Configuration

The temporal-ops CLI tool uses command-line flags and environment variables for configuration. See the main README for available options.