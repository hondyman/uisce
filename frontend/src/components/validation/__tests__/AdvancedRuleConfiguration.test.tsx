import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import AdvancedRuleConfiguration from '../AdvancedRuleConfiguration';

describe('AdvancedRuleConfiguration', () => {
  it('renders Material Design table with default rules', () => {
    render(<AdvancedRuleConfiguration />);

    // Check for the table layout - tabs should be present
    expect(screen.getByText(/Rules Overview/i)).toBeInTheDocument();
    // Check for table column headers
    expect(screen.getByRole('columnheader', { name: /Rule Name/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /Severity/i })).toBeInTheDocument();
    // Check for one of the default rules rendered
    expect(screen.getByText(/Age Verification/i)).toBeInTheDocument();
  });
});
