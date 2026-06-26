export const PIPELINE_NODES_DOC = `
# Uisce Pipeline Nodes Reference

Uisce Pipelines allow you to visually define business logic, validations, and integrations. Each pipeline consists of a **Trigger** (input) and a series of **Nodes** (steps).

## Core Concepts

*   **Business Object (BO)**: The target data entity (e.g., *Trade*, *Customer*). Selecting a BO enables context-aware configuration.
*   **Data Flow**: Data flows from left to right. Each node receives the output of the previous node.
*   **Trace Status**: During debugging, nodes show *PASS* or *FAIL* status.

---

## Validation Nodes

These nodes verify data integrity and business rules. If a validation fails, the pipeline may stop or route to an error handler.

### **Limit Check**
Validates numeric fields against a threshold.
*   **Configuration**:
    *   **Field to Check**: Select a numeric field from your Business Object (e.g., \`amount\`, \`credit_limit\`).
    *   **Operator**: \`<\`, \`>\`, \`<=\`, \`>=\`, \`==\`.
    *   **Limit Amount**: The threshold value.
*   **Use Case**: Reject trades over $1M.

### **Sanctions Screening**
Checks names or countries against global watchlists (OFAC, EU, UN).
*   **Configuration**:
    *   **Entity to Screen**: The name or country field.
    *   **Lists**: Select specific watchlists.
*   **Use Case**: Ensure compliance with AML regulations.

### **List Lookup**
Validates that a value exists in a defined list.
*   **Configuration**:
    *   **Field to Lookup**: The field to validate (e.g., \`currency_code\`).
    *   **Source**:
        *   *Manual*: Enter a comma-separated list (e.g., \`USD, EUR, GBP\`).
        *   *Dataset*: Select a managed reference dataset (e.g., *ISO Country Codes*).
*   **Use Case**: Validate supported currencies.

### **Cross Reference**
Checks if an entity exists in another system or Business Object.
*   **Configuration**:
    *   **Target BO**: The object to check against (e.g., *Customer*).
    *   **Match Field**: The key to match (e.g., \`customer_id\`).
*   **Use Case**: Ensure a Trade references a valid Customer.

### **Formula**
Executes custom logic using expression syntax.
*   **Configuration**:
    *   **Expression**: Mathematical or logical expression (e.g., \`amount * risk_factor > 1000\`).
*   **Use Case**: Complex derived validations.

---

## Control Flow

### **Conditional Router**
Splits the pipeline integrity based on logic.
*   **Configuration**:
    *   **Condition**: An expression that evaluates to true/false.
*   **Use Case**: Route "High Value" trades to a different approval process than "Standard" trades.

### **Approval Gate**
Pauses the pipeline for human intervention.
*   **Configuration**:
    *   **Required Roles**: Roles authorized to approve (e.g., \`Supervisor\`).
    *   **Timeout**: How long to wait before default action (e.g., \`24h\`).
*   **Use Case**: Large transactions requiring manual sign-off.

---

## Intelligence (AI)

### **AI Anomaly Detection**
Uses Machine Learning to detect outliers.
*   **Configuration**:
    *   **Model Version**: Select the ML model.
    *   **Sensitivity**: 1-100% threshold.
*   **Use Case**: Detect fraud patterns or unusual trading activity.

### **AI Prediction**
Predicts a score or class for the data.
*   **Configuration**:
    *   **Target**: What to predict (e.g., *Settlement Risk*).
*   **Use Case**: Score the likelihood of a settlement failure.

---

## Integration & Actions

### **External API**
Calls a third-party REST API.
*   **Configuration**:
    *   **URL**: Endpoint address.
    *   **Method**: GET, POST, PUT.
*   **Use Case**: Fetch credit score from an external bureau.

### **Durable Ledger**
Records the transaction to an immutable or audit ledger.
*   **Configuration**:
    *   **Ledger Type**:
        *   *Immutable*: Write-once, crypto-signed.
        *   *Audit*: Standard audit log.
*   **Use Case**: Finalize a trade or log a compliance event.
`;
