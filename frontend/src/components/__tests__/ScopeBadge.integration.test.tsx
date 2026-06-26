import React from 'react'
import { render, screen } from '@testing-library/react'
import ScopeBadge from '../ScopeBadge'
import { ScopeProvider, useScope } from '../../contexts/ScopeContext'

// Helper component to set scope imperatively for the test
function ScopeSetter({ schemaName, tableName, columnNames }: { schemaName?: string, tableName?: string, columnNames?: string[] }) {
  const { setSchemaName, setTableName, setColumnNames, setSchemaId, setTableId, setColumnIds } = useScope()
  React.useEffect(() => {
    if (schemaName) { setSchemaName(schemaName); setSchemaId('schema-1') }
    if (tableName) { setTableName(tableName); setTableId('table-1') }
    if (columnNames) { setColumnNames(columnNames); setColumnIds(columnNames.map((_, i) => `col-${i}`)) }
  }, [schemaName, tableName, columnNames, setSchemaName, setTableName, setColumnNames, setSchemaId, setTableId, setColumnIds])
  return null
}

test('ScopeBadge shows schema, table and column names when set via ScopeProvider', async () => {
  render(
    <ScopeProvider>
      <ScopeSetter schemaName="public_long_schema_name_that_should_truncate" tableName="orders_table_long_name" columnNames={["id", "amount"]} />
      <ScopeBadge />
    </ScopeProvider>
  )

  // schema should appear
  expect(await screen.findByText(/public_long_schema_name_that_should_truncate/)).toBeInTheDocument()
  // table should appear
  expect(screen.getByText(/orders_table_long_name/)).toBeInTheDocument()
  // columns should appear (joined)
  expect(screen.getByText(/id, amount/)).toBeInTheDocument()
})
