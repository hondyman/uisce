/**
 * Extended Entity Schema Definitions for FieldAutocomplete
 *
 * This file demonstrates how to extend the ENTITY_SCHEMAS with real-world
 * business entities and their field definitions. Import this into FieldAutocomplete
 * or use it as a reference for building your own schema.
 */

export interface Field {
  name: string;
  type: string;
  description?: string;
  nullable?: boolean;
  relatedEntity?: string;
}

/**
 * Complete Entity Schemas for Validation and Business Process Management
 */
export const EXTENDED_ENTITY_SCHEMAS: Record<string, Field[]> = {
  // Business Process Entity
  BusinessProcess: [
    {
      name: 'bp_id',
      type: 'uuid',
      description: 'Unique business process identifier',
      nullable: false,
    },
    {
      name: 'bp_name',
      type: 'varchar',
      description: 'Name of the business process',
      nullable: false,
    },
    {
      name: 'description',
      type: 'text',
      description: 'Detailed process description',
      nullable: true,
    },
    {
      name: 'category',
      type: 'varchar',
      description: 'Business process category (e.g., Finance, Operations)',
      nullable: false,
    },
    {
      name: 'status',
      type: 'varchar',
      description: 'Process status (Active, Deprecated, In Review)',
      nullable: false,
    },
    {
      name: 'owner_id',
      type: 'uuid',
      description: 'References the process owner',
      nullable: false,
      relatedEntity: 'User',
    },
    {
      name: 'created_at',
      type: 'timestamp',
      description: 'Process creation timestamp',
      nullable: false,
    },
    {
      name: 'updated_at',
      type: 'timestamp',
      description: 'Last update timestamp',
      nullable: false,
    },
  ],

  // Validation Result Entity
  ValidationResult: [
    {
      name: 'result_id',
      type: 'uuid',
      description: 'Unique validation result identifier',
      nullable: false,
    },
    {
      name: 'bp_name',
      type: 'varchar',
      description: 'Associated business process name',
      nullable: false,
    },
    {
      name: 'step_name',
      type: 'varchar',
      description: 'Process step that was validated',
      nullable: false,
    },
    {
      name: 'passed',
      type: 'boolean',
      description: 'Whether validation passed',
      nullable: false,
    },
    {
      name: 'error_count',
      type: 'integer',
      description: 'Number of validation errors',
      nullable: false,
    },
    {
      name: 'warning_count',
      type: 'integer',
      description: 'Number of validation warnings',
      nullable: false,
    },
    {
      name: 'execution_time_ms',
      type: 'integer',
      description: 'Execution time in milliseconds',
      nullable: false,
    },
    {
      name: 'executed_at',
      type: 'timestamp',
      description: 'When the validation was executed',
      nullable: false,
    },
    {
      name: 'user_id',
      type: 'uuid',
      description: 'User who triggered the validation',
      nullable: false,
      relatedEntity: 'User',
    },
    {
      name: 'errors',
      type: 'jsonb',
      description: 'Array of error messages',
      nullable: true,
    },
    {
      name: 'warnings',
      type: 'jsonb',
      description: 'Array of warning messages',
      nullable: true,
    },
  ],

  // Employee Entity (Example)
  Employee: [
    {
      name: 'employee_id',
      type: 'uuid',
      description: 'Unique employee identifier',
      nullable: false,
    },
    {
      name: 'first_name',
      type: 'varchar',
      description: 'Employee first name',
      nullable: false,
    },
    {
      name: 'last_name',
      type: 'varchar',
      description: 'Employee last name',
      nullable: false,
    },
    {
      name: 'email',
      type: 'varchar',
      description: 'Employee email address',
      nullable: false,
    },
    {
      name: 'phone',
      type: 'varchar',
      description: 'Employee phone number',
      nullable: true,
    },
    {
      name: 'hire_date',
      type: 'date',
      description: 'Date employee was hired',
      nullable: false,
    },
    {
      name: 'salary',
      type: 'numeric',
      description: 'Employee salary',
      nullable: true,
    },
    {
      name: 'department_id',
      type: 'uuid',
      description: 'References Department entity',
      nullable: false,
      relatedEntity: 'Department',
    },
    {
      name: 'manager_id',
      type: 'uuid',
      description: 'References direct manager (Employee)',
      nullable: true,
      relatedEntity: 'Employee',
    },
    {
      name: 'is_active',
      type: 'boolean',
      description: 'Whether employee is currently active',
      nullable: false,
    },
    {
      name: 'created_at',
      type: 'timestamp',
      description: 'Record creation timestamp',
      nullable: false,
    },
    {
      name: 'updated_at',
      type: 'timestamp',
      description: 'Last update timestamp',
      nullable: false,
    },
  ],

  // Department Entity
  Department: [
    {
      name: 'department_id',
      type: 'uuid',
      description: 'Unique department identifier',
      nullable: false,
    },
    {
      name: 'name',
      type: 'varchar',
      description: 'Department name',
      nullable: false,
    },
    {
      name: 'description',
      type: 'text',
      description: 'Department description and purpose',
      nullable: true,
    },
    {
      name: 'budget',
      type: 'numeric',
      description: 'Annual department budget',
      nullable: true,
    },
    {
      name: 'manager_id',
      type: 'uuid',
      description: 'Department manager reference',
      nullable: false,
      relatedEntity: 'Employee',
    },
    {
      name: 'parent_department_id',
      type: 'uuid',
      description: 'Parent department for hierarchy',
      nullable: true,
      relatedEntity: 'Department',
    },
    {
      name: 'created_at',
      type: 'timestamp',
      description: 'Record creation timestamp',
      nullable: false,
    },
  ],

  // User Entity
  User: [
    {
      name: 'user_id',
      type: 'uuid',
      description: 'Unique user identifier',
      nullable: false,
    },
    {
      name: 'username',
      type: 'varchar',
      description: 'Unique username for login',
      nullable: false,
    },
    {
      name: 'email',
      type: 'varchar',
      description: 'User email address',
      nullable: false,
    },
    {
      name: 'role',
      type: 'varchar',
      description: 'User role (admin, analyst, viewer)',
      nullable: false,
    },
    {
      name: 'is_active',
      type: 'boolean',
      description: 'Whether user account is active',
      nullable: false,
    },
    {
      name: 'last_login',
      type: 'timestamp',
      description: 'Last login timestamp',
      nullable: true,
    },
    {
      name: 'created_at',
      type: 'timestamp',
      description: 'Account creation timestamp',
      nullable: false,
    },
  ],

  // Transaction Entity
  Transaction: [
    {
      name: 'transaction_id',
      type: 'uuid',
      description: 'Unique transaction identifier',
      nullable: false,
    },
    {
      name: 'transaction_date',
      type: 'date',
      description: 'Date of transaction',
      nullable: false,
    },
    {
      name: 'amount',
      type: 'numeric',
      description: 'Transaction amount',
      nullable: false,
    },
    {
      name: 'currency',
      type: 'varchar',
      description: 'Currency code (USD, EUR, etc.)',
      nullable: false,
    },
    {
      name: 'type',
      type: 'varchar',
      description: 'Transaction type (Credit, Debit, Transfer)',
      nullable: false,
    },
    {
      name: 'status',
      type: 'varchar',
      description: 'Transaction status (Pending, Completed, Failed)',
      nullable: false,
    },
    {
      name: 'account_id',
      type: 'uuid',
      description: 'Associated account reference',
      nullable: false,
      relatedEntity: 'Account',
    },
    {
      name: 'reference_number',
      type: 'varchar',
      description: 'External reference number',
      nullable: true,
    },
    {
      name: 'description',
      type: 'text',
      description: 'Transaction description',
      nullable: true,
    },
    {
      name: 'created_at',
      type: 'timestamp',
      description: 'Record creation timestamp',
      nullable: false,
    },
  ],

  // Account Entity
  Account: [
    {
      name: 'account_id',
      type: 'uuid',
      description: 'Unique account identifier',
      nullable: false,
    },
    {
      name: 'account_number',
      type: 'varchar',
      description: 'Account number or code',
      nullable: false,
    },
    {
      name: 'account_type',
      type: 'varchar',
      description: 'Type of account (Checking, Savings, etc.)',
      nullable: false,
    },
    {
      name: 'balance',
      type: 'numeric',
      description: 'Current account balance',
      nullable: false,
    },
    {
      name: 'currency',
      type: 'varchar',
      description: 'Account currency',
      nullable: false,
    },
    {
      name: 'customer_id',
      type: 'uuid',
      description: 'Associated customer reference',
      nullable: false,
      relatedEntity: 'Customer',
    },
    {
      name: 'status',
      type: 'varchar',
      description: 'Account status (Active, Closed, Suspended)',
      nullable: false,
    },
    {
      name: 'opened_date',
      type: 'date',
      description: 'Account opening date',
      nullable: false,
    },
    {
      name: 'closed_date',
      type: 'date',
      description: 'Account closing date (if closed)',
      nullable: true,
    },
  ],

  // Customer Entity
  Customer: [
    {
      name: 'customer_id',
      type: 'uuid',
      description: 'Unique customer identifier',
      nullable: false,
    },
    {
      name: 'name',
      type: 'varchar',
      description: 'Customer full name',
      nullable: false,
    },
    {
      name: 'email',
      type: 'varchar',
      description: 'Customer email address',
      nullable: false,
    },
    {
      name: 'phone',
      type: 'varchar',
      description: 'Customer phone number',
      nullable: true,
    },
    {
      name: 'customer_type',
      type: 'varchar',
      description: 'Customer type (Individual, Business)',
      nullable: false,
    },
    {
      name: 'registration_date',
      type: 'date',
      description: 'Customer registration date',
      nullable: false,
    },
    {
      name: 'credit_limit',
      type: 'numeric',
      description: 'Customer credit limit',
      nullable: true,
    },
    {
      name: 'is_active',
      type: 'boolean',
      description: 'Whether customer is active',
      nullable: false,
    },
  ],

  // Metric Entity (for analytics/PoP)
  Metric: [
    {
      name: 'metric_id',
      type: 'uuid',
      description: 'Unique metric identifier',
      nullable: false,
    },
    {
      name: 'name',
      type: 'varchar',
      description: 'Metric name',
      nullable: false,
    },
    {
      name: 'display_name',
      type: 'varchar',
      description: 'Human-readable metric name',
      nullable: false,
    },
    {
      name: 'description',
      type: 'text',
      description: 'Metric description',
      nullable: true,
    },
    {
      name: 'metric_type',
      type: 'varchar',
      description: 'Type (sum, count, average, ratio)',
      nullable: false,
    },
    {
      name: 'domain',
      type: 'varchar',
      description: 'Business domain (Finance, Sales, Operations)',
      nullable: false,
    },
    {
      name: 'category',
      type: 'varchar',
      description: 'Metric category',
      nullable: false,
    },
    {
      name: 'base_query',
      type: 'text',
      description: 'SQL query for metric calculation',
      nullable: false,
    },
    {
      name: 'owner_user_id',
      type: 'uuid',
      description: 'Metric owner reference',
      nullable: false,
      relatedEntity: 'User',
    },
    {
      name: 'status',
      type: 'varchar',
      description: 'Metric status (draft, published, archived)',
      nullable: false,
    },
    {
      name: 'created_at',
      type: 'timestamp',
      description: 'Creation timestamp',
      nullable: false,
    },
  ],
};

/**
 * Helper function to get schema for an entity
 */
export function getEntitySchema(entityName: string): Field[] {
  return EXTENDED_ENTITY_SCHEMAS[entityName] || [];
}

/**
 * Helper function to get a specific field from an entity schema
 */
export function getFieldInfo(
  entityName: string,
  fieldName: string
): Field | undefined {
  const schema = EXTENDED_ENTITY_SCHEMAS[entityName];
  return schema?.find((f) => f.name === fieldName);
}

/**
 * Helper function to get all related entities for a field
 */
export function getRelatedEntities(entityName: string): string[] {
  const schema = EXTENDED_ENTITY_SCHEMAS[entityName];
  const related = new Set<string>();
  schema?.forEach((field) => {
    if (field.relatedEntity) {
      related.add(field.relatedEntity);
    }
  });
  return Array.from(related);
}

export default EXTENDED_ENTITY_SCHEMAS;
