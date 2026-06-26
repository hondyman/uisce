/**
 * GraphQL Query Builder
 * 
 * Schema-driven GraphQL query generator that eliminates hardcoded queries
 * across builders (ValidationRulesBuilderPage, ReportBuilderUI, etc.)
 */

export interface GraphQLField {
    name: string;
    type: string;
    required?: boolean;
    fields?: GraphQLField[]; // For nested objects
}

export interface GraphQLQueryOptions {
    fields: string[] | GraphQLField[];
    filters?: Record<string, any>;
    orderBy?: { field: string; direction: 'asc' | 'desc' };
    limit?: number;
    offset?: number;
}

export interface PaginationOptions {
    limit?: number;
    offset?: number;
}

/**
 * Builds a GraphQL query string from field specifications
 */
function buildFieldSelection(fields: string[] | GraphQLField[], indent = 2): string {
    const indentStr = ' '.repeat(indent);

    return fields.map((field) => {
        if (typeof field === 'string') {
            return `${indentStr}${field}`;
        }

        // Handle nested fields
        if (field.fields && field.fields.length > 0) {
            const nestedFields = buildFieldSelection(field.fields, indent + 2);
            return `${indentStr}${field.name} {\n${nestedFields}\n${indentStr}}`;
        }

        return `${indentStr}${field.name}`;
    }).join('\n');
}

/**
 * Builds filter arguments for GraphQL where clause
 */
function buildFilterArgs(filters: Record<string, any>): string {
    if (!filters || Object.keys(filters).length === 0) {
        return '';
    }

    const filterPairs: string[] = [];

    for (const [key, value] of Object.entries(filters)) {
        if (value === null || value === undefined) {
            continue;
        }

        if (typeof value === 'string') {
            filterPairs.push(`${key}: { _eq: "${value}" }`);
        } else if (typeof value === 'number' || typeof value === 'boolean') {
            filterPairs.push(`${key}: { _eq: ${value} }`);
        } else if (Array.isArray(value)) {
            const arrayStr = value.map((v) =>
                typeof v === 'string' ? `"${v}"` : v
            ).join(', ');
            filterPairs.push(`${key}: { _in: [${arrayStr}] }`);
        } else if (typeof value === 'object') {
            // Handle complex filters like { _eq, _neq, _gt, _lt, etc. }
            const operators = Object.entries(value as Record<string, any>)
                .map(([op, val]) => {
                    const formattedVal = typeof val === 'string' ? `"${val}"` : val;
                    return `${op}: ${formattedVal}`;
                })
                .join(', ');
            filterPairs.push(`${key}: { ${operators} }`);
        }
    }

    return filterPairs.length > 0 ? `where: { ${filterPairs.join(', ')} }` : '';
}

/**
 * Builds pagination arguments (limit, offset)
 */
function buildPaginationArgs(pagination?: PaginationOptions): string {
    const args: string[] = [];

    if (pagination?.limit !== undefined) {
        args.push(`limit: ${pagination.limit}`);
    }

    if (pagination?.offset !== undefined) {
        args.push(`offset: ${pagination.offset}`);
    }

    return args.join(', ');
}

/**
 * Builds order_by argument for sorting
 */
function buildOrderByArg(orderBy?: { field: string; direction: 'asc' | 'desc' }): string {
    if (!orderBy) {
        return '';
    }

    return `order_by: { ${orderBy.field}: ${orderBy.direction} }`;
}

/**
 * Main query builder function
 * 
 * @example
 * ```typescript
 * const query = buildGraphQLQuery('validation_rules', {
 *   fields: ['id', 'rule_name', 'rule_type', 'entity_id'],
 *   filters: { tenant_id: 'tenant-123', is_active: true },
 *   orderBy: { field: 'created_at', direction: 'desc' },
 *   limit: 50
 * });
 * ```
 */
export function buildGraphQLQuery(
    tableName: string,
    options: GraphQLQueryOptions
): string {
    const { fields, filters, orderBy, limit, offset } = options;

    // Build arguments
    const args: string[] = [];

    const filterArg = filters ? buildFilterArgs(filters) : '';
    if (filterArg) args.push(filterArg);

    const orderByArg = orderBy ? buildOrderByArg(orderBy) : '';
    if (orderByArg) args.push(orderByArg);

    const paginationArg = buildPaginationArgs({ limit, offset });
    if (paginationArg) args.push(paginationArg);

    const argsStr = args.length > 0 ? `(${args.join(', ')})` : '';

    // Build field selection
    const fieldSelection = buildFieldSelection(fields);

    // Construct query
    return `query {
  ${tableName}${argsStr} {
${fieldSelection}
  }
}`;
}

/**
 * Builds a GraphQL mutation for insert operations
 */
export function buildGraphQLInsertMutation(
    tableName: string,
    objects: Record<string, any>[],
    returningFields: string[]
): string {
    const objectsJson = JSON.stringify(objects, null, 2)
        .split('\n')
        .map((line, i) => (i === 0 ? line : `    ${line}`))
        .join('\n');

    const fieldSelection = buildFieldSelection(returningFields);

    return `mutation {
  insert_${tableName}(objects: ${objectsJson}) {
    returning {
${fieldSelection}
    }
  }
}`;
}

/**
 * Builds a GraphQL mutation for update operations
 */
export function buildGraphQLUpdateMutation(
    tableName: string,
    filters: Record<string, any>,
    updates: Record<string, any>,
    returningFields: string[]
): string {
    const whereClause = buildFilterArgs(filters);
    const setClause = Object.entries(updates)
        .map(([key, value]) => {
            const formattedValue = typeof value === 'string' ? `"${value}"` : value;
            return `${key}: ${formattedValue}`;
        })
        .join(', ');

    const fieldSelection = buildFieldSelection(returningFields);

    return `mutation {
  update_${tableName}(${whereClause}, _set: { ${setClause} }) {
    returning {
${fieldSelection}
    }
  }
}`;
}

/**
 * Builds a GraphQL mutation for delete operations
 */
export function buildGraphQLDeleteMutation(
    tableName: string,
    filters: Record<string, any>,
    returningFields: string[] = ['id']
): string {
    const whereClause = buildFilterArgs(filters);
    const fieldSelection = buildFieldSelection(returningFields);

    return `mutation {
  delete_${tableName}(${whereClause}) {
    returning {
${fieldSelection}
    }
  }
}`;
}

/**
 * Helper to build aggregate queries (count, sum, avg, etc.)
 */
export function buildGraphQLAggregateQuery(
    tableName: string,
    filters: Record<string, any>,
    aggregates: {
        count?: boolean;
        sum?: string[];
        avg?: string[];
        max?: string[];
        min?: string[];
    }
): string {
    const whereClause = filters ? buildFilterArgs(filters) : '';
    const argsStr = whereClause ? `(${whereClause})` : '';

    const aggregateFields: string[] = [];

    if (aggregates.count) {
        aggregateFields.push('    aggregate {\n      count\n    }');
    }

    if (aggregates.sum && aggregates.sum.length > 0) {
        const sumFields = aggregates.sum.map((f) => `      ${f}`).join('\n');
        aggregateFields.push(`    aggregate {\n      sum {\n${sumFields}\n      }\n    }`);
    }

    if (aggregates.avg && aggregates.avg.length > 0) {
        const avgFields = aggregates.avg.map((f) => `      ${f}`).join('\n');
        aggregateFields.push(`    aggregate {\n      avg {\n${avgFields}\n      }\n    }`);
    }

    if (aggregates.max && aggregates.max.length > 0) {
        const maxFields = aggregates.max.map((f) => `      ${f}`).join('\n');
        aggregateFields.push(`    aggregate {\n      max {\n${maxFields}\n      }\n    }`);
    }

    if (aggregates.min && aggregates.min.length > 0) {
        const minFields = aggregates.min.map((f) => `      ${f}`).join('\n');
        aggregateFields.push(`    aggregate {\n      min {\n${minFields}\n      }\n    }`);
    }

    return `query {
  ${tableName}_aggregate${argsStr} {
${aggregateFields.join('\n')}
  }
}`;
}

/**
 * Convert a semantic view schema to GraphQL field definitions
 */
export function schemaToFields(schema: { fields: Array<{ field_name: string; field_type: string }> }): string[] {
    return schema.fields.map((f) => f.field_name);
}
