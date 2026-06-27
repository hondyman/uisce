import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import ImpactAnalysis from './ImpactAnalysis';

import { vi } from 'vitest';

// Mock the rulesApi
vi.mock('../../services/rulesApi', () => ({
  rulesApi: {
    fetchRuleImpact: vi.fn(),
  }
}));

import { rulesApi } from '../../services/rulesApi';

test('renders impact modal lists after API returns', async () => {
  const mockImpact = {
    rule_id: 'r1',
    fields: [ { semantic_term_id: '', bo_field_id: 'f1', business_object_id: 'b1', field_path: ['customer', 'name'] } ],
    semantic_terms: [ { id: 't1', node_name: 'COMPANY_CUSTOMER_NAME', display_name: 'Company Customer Name' } ],
    business_objects: [ { id: 'b1', node_name: 'customer', display_name: 'Customer' } ],
    dependent_rules: [ { id: 'r2', rule_name: 'OtherRule', link_type: 'uses_field' } ],
    overrides: [ { id: 'o1', tenant_id: 'tenant1', rule_name: 'TenantOverride' } ],
  };

  (rulesApi.fetchRuleImpact as unknown as vi.Mock).mockResolvedValue(mockImpact);

  render(<ImpactAnalysis ruleID={'r1'} rule={{ target_entity: 'business_object', field_name: 'name', rule_condition: '', severity: 'error' }} tenantId={'t1'} />);

  const button = screen.getByRole('button', { name: /Analyze Impact/i });
  fireEvent.click(button);

  await waitFor(() => expect(rulesApi.fetchRuleImpact).toHaveBeenCalled());

  // Expect business object chip
  expect(await screen.findByText('Customer')).toBeInTheDocument();
  // Expect field chip
  expect(await screen.findByText('customer.name')).toBeInTheDocument();
  // Expect dependent rule
  expect(await screen.findByText('OtherRule')).toBeInTheDocument();
});
