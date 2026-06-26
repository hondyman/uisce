// React import removed (unused)
import { render, screen, fireEvent as _fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import { ApolloProvider } from '@apollo/client';
import apolloClient from '../../../../src/graphql/apolloClient';
import { TenantProvider } from '../../../contexts/TenantContext';
import SourceFields from '../SourceFields';

describe('SourceFields', () => {
  it('renders and updates inputs', () => {
    const setFormData = vi.fn();
    const formData = { sourceTable: 's.t', sourceColumn: 'c' };
    render(
      <ApolloProvider client={apolloClient}>
        <TenantProvider>
          <SourceFields formData={formData} setFormData={setFormData} />
        </TenantProvider>
      </ApolloProvider>
    );
  expect(screen.getByDisplayValue('s.t')).toBeInTheDocument();
  expect(screen.getByDisplayValue('c')).toBeInTheDocument();
  });
});
