import React from 'react';
import { Calculate, VpnKey, Storage } from '@mui/icons-material';
import { useTheme } from '@mui/material/styles';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

export interface FieldMeta {
  field_name: string;
  semantic_term_key: string;
  display_label: string;
  data_type: 'string' | 'number' | 'date' | 'boolean';
  field_role: 'DIMENSION' | 'MEASURE' | 'KEY';
  binding_status: 'RESOLVED' | 'UNRESOLVED' | 'CALCULATED';
  is_editable: boolean;
  column_width: number;
  component_hint: string;
}

interface Props {
  fields: FieldMeta[];
  data: any[];
}

export const DynamicBODataGrid: React.FC<Props> = ({ fields, data }) => {
  const theme = useTheme();

  const renderCellContent = (field: FieldMeta, value: any) => {
    if (value === null || value === undefined) {
      return (
        <Typography
          component="span"
          sx={{
            fontFamily: 'monospace',
            fontSize: '0.75rem',
            color: 'text.secondary',
          }}
        >
          —
        </Typography>
      );
    }

    switch (field.data_type) {
      case 'number':
        return (
          <Typography
            component="span"
            sx={{
              fontFamily: 'monospace',
              color: '#34d399',
              fontWeight: 500,
            }}
          >
            {typeof value === 'number' ? value.toLocaleString(undefined, { minimumFractionDigits: 2 }) : value}
          </Typography>
        );
      case 'date':
        return (
          <Typography
            component="span"
            sx={{
              fontFamily: 'monospace',
              fontSize: '0.75rem',
              color: 'grey.300',
            }}
          >
            {new Date(value).toLocaleDateString()}
          </Typography>
        );
      default:
        return (
          <Typography
            component="span"
            sx={{
              fontSize: '0.75rem',
              color: 'grey.200',
            }}
          >
            {String(value)}
          </Typography>
        );
    }
  };

  return (
    <Box
      sx={{
        overflowX: 'auto',
        backgroundColor: 'rgba(30, 41, 59, 0.8)',
        border: '1px solid rgba(71, 85, 105, 0.6)',
        borderRadius: '12px',
        boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
      }}
    >
      <TableContainer component={Paper} sx={{ backgroundColor: 'transparent', boxShadow: 'none' }}>
        <Table>
          <TableHead>
            <TableRow
              sx={{
                backgroundColor: 'rgba(3, 7, 18, 0.8)',
                borderBottom: '1px solid rgb(51, 65, 85)',
              }}
            >
              {fields.map((field) => (
                <TableCell
                  key={field.field_name}
                  sx={{
                    width: field.column_width,
                    py: 1.5,
                    px: 2,
                    fontSize: '0.6875rem',
                    fontWeight: 700,
                    color: 'grey.400',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    borderBottom: '1px solid rgb(51, 65, 85)',
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                    {field.field_role === 'KEY' && (
                      <VpnKey sx={{ fontSize: 12, color: '#fbbf24' }} />
                    )}
                    {field.binding_status === 'CALCULATED' && (
                      <Calculate sx={{ fontSize: 12, color: '#38bdf8' }} />
                    )}
                    {field.binding_status === 'RESOLVED' && (
                      <Storage sx={{ fontSize: 12, color: 'grey.500' }} />
                    )}
                    <span>{field.display_label}</span>
                  </Box>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody sx={{ borderTop: '1px solid rgb(30, 41, 59)' }}>
            {data.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={fields.length}
                  align="center"
                  sx={{ py: 8, color: 'grey.500', fontFamily: 'monospace' }}
                >
                  No records available for active Business Object context.
                </TableCell>
              </TableRow>
            ) : (
              data.map((row, idx) => (
                <TableRow
                  key={idx}
                  sx={{
                    '&:hover': {
                      backgroundColor: 'rgba(51, 65, 85, 0.3)',
                    },
                    transition: 'background-color 0.2s',
                  }}
                >
                  {fields.map((field) => (
                    <TableCell
                      key={field.field_name}
                      sx={{ py: 1, px: 2, fontSize: '0.75rem' }}
                    >
                      {renderCellContent(field, row[field.field_name])}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};
