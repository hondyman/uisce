// React default import removed — using automatic JSX runtime
import { render, fireEvent } from '@testing-library/react';
import { useState } from 'react';
import userEvent from '@testing-library/user-event';
import BuilderTabs from '../BuilderTabs';

test('renders tabs and navigates with arrow keys', async () => {
  const tabs = [
    { id: 'canvas', label: 'Canvas' },
    { id: 'core', label: 'Core' },
    { id: 'custom', label: 'Custom' },
  ];

  const Wrapper: React.FC = () => {
    const [active, setActive] = useState('canvas');
    return <BuilderTabs activeTab={active} setActiveTab={setActive} tabs={tabs} />;
  };

  const { getByRole, getByText } = render(<Wrapper />);

  const list = getByRole('tablist');
  const coreBtn = getByText('Core') as HTMLButtonElement;

  const user = userEvent.setup();
  // click core
  await user.click(coreBtn);
  // after clicking, Core should be selected (aria-selected true)
  expect(coreBtn.getAttribute('aria-selected')).toBe('true');

  // arrow right from core -> custom
  fireEvent.keyDown(list, { key: 'ArrowRight' });
  const customBtn = getByText('Custom') as HTMLButtonElement;
  expect(customBtn.getAttribute('aria-selected')).toBe('true');

  // arrow left from custom -> core
  fireEvent.keyDown(list, { key: 'ArrowLeft' });
  expect(coreBtn.getAttribute('aria-selected')).toBe('true');
});
