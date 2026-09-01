import { useMemo } from 'react';
import { DynamicFieldUIConfig } from './DynamicPropertyTypes';
import { evaluatePageExpression } from './evaluatePageExpression';
import { usePageEventBus } from './PageEventBusContext';

export function useEvaluatedField(
  fieldConfig: DynamicFieldUIConfig = {},
  rowData: Record<string, any> = {}
) {
  const { parameters } = usePageEventBus();

  return useMemo(() => {
    const context = {
      rowData,
      parameters,
      globalContext: {
        userName: 'Current User',
        executionTime: new Date(),
        tenantId: 'active-tenant',
      },
    };

    return {
      textColor: evaluatePageExpression(fieldConfig.textColor, context) || '#F8FAFC',
      backgroundColor: evaluatePageExpression(fieldConfig.backgroundColor, context) || 'transparent',
      fontWeight: evaluatePageExpression(fieldConfig.fontWeight, context) || 'normal',
      formatMask: evaluatePageExpression(fieldConfig.formatMask, context) || 'AUTO',
      isVisible: fieldConfig.isVisible ? evaluatePageExpression(fieldConfig.isVisible, context) : true,
      isReadOnly: fieldConfig.isReadOnly ? evaluatePageExpression(fieldConfig.isReadOnly, context) : false,
      isRequired: fieldConfig.isRequired ? evaluatePageExpression(fieldConfig.isRequired, context) : false,
    };
  }, [fieldConfig, rowData, parameters]);
}
