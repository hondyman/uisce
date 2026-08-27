import React from 'react';
import { Box, Card, CardHeader, CardContent, Typography, Grid, Divider } from '@mui/material';
import { FormTemplateSpec, FormFieldItem } from './FormManagerTypes';

interface ReportFormRendererProps {
  formSpec: FormTemplateSpec;
  previewData?: Record<string, any>;
}

export const ReportFormRenderer: React.FC<ReportFormRendererProps> = ({ formSpec, previewData = {} }) => {
  const formatValue = (item: FormFieldItem) => {
    const rawVal = previewData[item.fieldKey || ''] ?? item.valueExpression ?? '—';
    if (typeof rawVal === 'number') {
      if (item.formatMask === 'CURRENCY') return `$${rawVal.toLocaleString(undefined, { minimumFractionDigits: 2 })}`;
      if (item.formatMask === 'PERCENT') return `${(rawVal * 100).toFixed(2)}%`;
    }
    return String(rawVal);
  };

  return (
    <Box sx={{ maxWidth: 1000, mx: 'auto', width: '100%' }}>
      {formSpec.sections.map((section) => (
        <Card
          key={section.id}
          sx={{
            bgcolor: '#071526',
            color: '#F8FAFC',
            border: '1px solid #1E293B',
            borderRadius: 2,
            boxShadow: 'none',
            mb: 3,
          }}
        >
          <CardHeader
            title={
              <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#38BDF8' }}>
                {section.title}
              </Typography>
            }
            subheader={
              section.description && (
                <Typography variant="caption" sx={{ color: '#64748B' }}>
                  {section.description}
                </Typography>
              )
            }
            sx={{ pb: 1, borderBottom: '1px solid #1E293B' }}
          />
          <CardContent sx={{ p: 2 }}>
            <Grid container spacing={2}>
              {section.items.map((item) => (
                <Grid item xs={12} sm={item.colSpan} key={item.id}>
                  <Box sx={{ p: 1.5, bgcolor: '#0B1E36', borderRadius: 1.5, border: '1px solid #1E293B' }}>
                    <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, display: 'block', mb: 0.5 }}>
                      {item.label}
                    </Typography>

                    {item.type === 'SIGNATURE_BLOCK' ? (
                      <Box sx={{ mt: 3, pt: 1, borderTop: '1px solid #475569', display: 'flex', justifyContent: 'space-between' }}>
                        <Typography variant="caption" sx={{ color: '#64748B' }}>Authorized Signature</Typography>
                        <Typography variant="caption" sx={{ color: '#64748B' }}>Date: ____ / ____ / ________</Typography>
                      </Box>
                    ) : item.type === 'DIVIDER' ? (
                      <Divider sx={{ my: 1, borderColor: '#334155' }} />
                    ) : (
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontWeight: 600, color: '#F8FAFC' }}>
                        {formatValue(item)}
                      </Typography>
                    )}
                  </Box>
                </Grid>
              ))}
            </Grid>
          </CardContent>
        </Card>
      ))}
    </Box>
  );
};
