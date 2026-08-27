import React from 'react';
import { Box, Typography } from '@mui/material';
import type { FormTemplateSpec } from './form/FormManagerTypes';
import type { FormReferenceElement } from './formElementModel';
import { ReportFormRenderer } from './form/ReportFormRenderer';
import { evaluateDynamicProperty } from './evaluateDynamicProperty';
import { evaluateCondition } from '../ExpressionBuilder/AdvancedConditionBuilder';

interface FormEmbedShellProps {
  element: FormReferenceElement;
  formRegistry: Record<string, FormTemplateSpec>;
  rowData: Record<string, any>;
  globalContext?: {
    pageNumber: number;
    totalPages: number;
    userName: string;
    executionTime: Date;
  };
}

export const FormEmbedShell: React.FC<FormEmbedShellProps> = ({
  element,
  formRegistry,
  rowData,
  globalContext = { pageNumber: 1, totalPages: 1, userName: '', executionTime: new Date() },
}) => {
  if (element.visibilityCondition && !evaluateCondition(element.visibilityCondition, rowData)) {
    return null;
  }

  const resolvedTemplateId = evaluateDynamicProperty(
    element.templateId ?? { isExpression: false, value: '' },
    rowData,
    globalContext as any
  );

  const activeFormSpec = formRegistry[resolvedTemplateId];

  if (!activeFormSpec) {
    return (
      <Box
        sx={{
          p: 2,
          border: '1px dashed #EF4444',
          borderRadius: 1,
          bgcolor: 'rgba(239, 68, 68, 0.05)',
        }}
      >
        <Typography
          variant="caption"
          sx={{ color: '#EF4444', fontFamily: 'monospace', fontSize: 11 }}
        >
          Form Template Resolution Error: Template &apos;{resolvedTemplateId}&apos; not found in registry.
        </Typography>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        p: element.containerStyle?.paddingTop ? `${element.containerStyle.paddingTop}px` : 1,
        bgcolor: element.containerStyle?.backgroundColor || 'transparent',
        borderRadius: 1,
        width: '100%',
      }}
    >
      <ReportFormRenderer formSpec={activeFormSpec} previewData={rowData} />
    </Box>
  );
};
