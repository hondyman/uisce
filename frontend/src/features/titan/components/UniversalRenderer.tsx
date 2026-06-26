import React, { useEffect, useState, useMemo } from 'react';
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';
import { AgGridReact } from 'ag-grid-react';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';

// Types for Titan Schema
interface FieldDefinition {
  name: string;
  label: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'enum';
  required?: boolean;
  options?: string[]; // For enum
}

interface ObjectDefinition {
  name: string;
  fields: FieldDefinition[];
}

interface UniversalRendererProps {
  objectType: string;
  onSubmit: (data: any) => void;
}

// Helper to convert Titan Schema to JSON Schema for RJSF
const convertToJSONSchema = (def: ObjectDefinition) => {
  const schema: any = {
    title: def.name,
    type: 'object',
    required: [],
    properties: {},
  };

  def.fields.forEach(field => {
    if (field.required) {
      schema.required.push(field.name);
    }

    let prop: any = { title: field.label };

    switch (field.type) {
      case 'string':
        prop.type = 'string';
        break;
      case 'number':
        prop.type = 'number';
        break;
      case 'boolean':
        prop.type = 'boolean';
        break;
      case 'date':
        prop.type = 'string';
        prop.format = 'date';
        break;
      case 'enum':
        prop.type = 'string';
        prop.enum = field.options;
        break;
      default:
        prop.type = 'string';
    }
    schema.properties[field.name] = prop;
  });

  return schema;
};

export const UniversalRenderer: React.FC<UniversalRendererProps> = ({ objectType, onSubmit }) => {
  const [schema, setSchema] = useState<ObjectDefinition | null>(null);

  useEffect(() => {
    // Mock Fetch Schema
    if (objectType === 'Trade') {
      setSchema({
        name: 'Trade',
        fields: [
          { name: 'symbol', label: 'Symbol', type: 'string', required: true },
          { name: 'quantity', label: 'Quantity', type: 'number', required: true },
          { name: 'side', label: 'Side', type: 'enum', options: ['Buy', 'Sell'], required: true },
          { name: 'price', label: 'Price', type: 'number', required: true },
          { name: 'counterparty', label: 'Counterparty', type: 'string', required: false },
        ]
      });
    }
  }, [objectType]);

  const jsonSchema = useMemo(() => schema ? convertToJSONSchema(schema) : null, [schema]);

  if (!jsonSchema) return <div>Loading Schema...</div>;

  return (
    <div className="p-6 border rounded-xl shadow-sm bg-white max-w-lg">
      <Form
        schema={jsonSchema}
        validator={validator}
        onSubmit={({ formData }: { formData: any }) => onSubmit(formData)}
        uiSchema={{
            "ui:submitButtonOptions": {
                "props": {
                    "className": "w-full bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors font-medium shadow-sm mt-4"
                },
                "submitText": `Create ${objectType}`
            }
        }}
      />
    </div>
  );
};

// Metadata-Driven Data Grid
export const MetadataGrid: React.FC<{ objectType: string }> = ({ objectType }) => {
    const [columnDefs, setColumnDefs] = useState<any[]>([]);
    const [rowData, setRowData] = useState<any[]>([]);

    useEffect(() => {
        // Mock Fetch Schema & Data
        if (objectType === 'Trade') {
            setColumnDefs([
                { field: 'id', headerName: 'ID', sortable: true, filter: true },
                { field: 'symbol', headerName: 'Symbol', sortable: true, filter: true },
                { field: 'quantity', headerName: 'Quantity', sortable: true, filter: 'agNumberColumnFilter' },
                { field: 'side', headerName: 'Side', sortable: true, filter: true },
                { field: 'price', headerName: 'Price', sortable: true, filter: 'agNumberColumnFilter' },
                { field: 'status', headerName: 'Status', sortable: true, filter: true },
            ]);
            setRowData([
                { id: '1', symbol: 'AAPL', quantity: 100, side: 'Buy', price: 150.00, status: 'Settled' },
                { id: '2', symbol: 'GOOGL', quantity: 50, side: 'Sell', price: 2800.00, status: 'Pending' },
            ]);
        }
    }, [objectType]);

    return (
        <div className="ag-theme-alpine" style={{ height: 400, width: '100%' }}>
            <AgGridReact
                rowData={rowData}
                columnDefs={columnDefs}
                pagination={true}
                paginationPageSize={10}
            />
        </div>
    );
};
