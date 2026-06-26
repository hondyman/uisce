// React import removed (unused)
import { render, screen, waitFor } from '@testing-library/react';
import SemanticModelOverview from '../SemanticModelOverview';
import { TenantProvider } from '../../../contexts/TenantContext';

const baseModel = {
  id: 'mdl1',
  name: 'test-model',
  is_custom: true,
  dimensions: [{ id: 'dim1', name: 'created_at', is_custom: true, type: 'time', sql: '${CUBE}.created_at', sourceTable: 'orders', sourceColumn: 'created_at' }],
  measures: [{ id: 'm1', name: 'count', is_custom: true, type: 'count', sql: 'COUNT(*)', sourceTable: 'orders', sourceColumn: 'id' }],
  joins: [{ id: 'j1', name: 'customer', is_custom: true, relationship: 'belongsTo', sql: '${CUBE}.customer_id = ${customer}.id', sourceTable: 'orders', sourceColumn: 'customer_id', leftTable: 'orders', rightTable: 'customer' }],
  pre_aggregations: [{ id: 'p1', name: 'agg1', is_custom: true, type: 'rollup' }],
  filters: [],
  tenant_instance_id: 'ds1'
};

describe('SemanticModelOverview integration', () => {
  it('shows error badge on tile when validationIssues contains element_id', async () => {
  render(<TenantProvider><SemanticModelOverview semanticModel={baseModel} removeSemanticElement={() => {}} toggleElementEdit={() => {}} updateSemanticElement={() => {}} /></TenantProvider>);

    // dispatch a validationIssues event with element_id referencing a join id
    const issues = [{ level: 'error', message: 'join missing', element_id: 'j1' }];
    window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } }));

    // wait for the error badge to appear on the join card
    await waitFor(() => {
      const joinCard = screen.getByText('customer').closest('.element-card');
      expect(joinCard).toBeTruthy();
      const badge = joinCard?.querySelector('.error-badge');
      expect(badge).toBeTruthy();
    });
  });

  it('shows error tooltip with message when validationIssues contains element_id', async () => {
  render(<TenantProvider><SemanticModelOverview semanticModel={baseModel} removeSemanticElement={() => {}} toggleElementEdit={() => {}} updateSemanticElement={() => {}} /></TenantProvider>);
    const issues = [{ level: 'error', message: 'join missing', element_id: 'j1' }];
    window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } }));

    await waitFor(() => {
      const joinCard = screen.getByText('customer').closest('.element-card');
      const tooltip = joinCard?.querySelector('.error-tooltip');
      expect(tooltip).toBeTruthy();
      expect(tooltip?.textContent).toContain('join missing');
    });
  });
});
