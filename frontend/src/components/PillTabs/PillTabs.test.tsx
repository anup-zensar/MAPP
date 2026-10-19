import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { PillTabs } from './PillTabs';

type Tab = 'waiting' | 'open' | 'decided';

function Example() {
  const [tab, setTab] = useState<Tab>('waiting');
  return (
    <PillTabs
      label="Approvals"
      value={tab}
      onChange={setTab}
      tabs={[
        { value: 'waiting', label: 'Waiting for me', badge: <span> 3</span> },
        { value: 'open', label: 'All open' },
        { value: 'decided', label: 'Decided by me' },
      ]}
    >
      <p>Showing {tab}</p>
    </PillTabs>
  );
}

describe('PillTabs', () => {
  it('is a tab list whose chosen tab labels the panel (feature 004)', async () => {
    const user = userEvent.setup();
    render(<Example />);

    const list = screen.getByRole('tablist', { name: 'Approvals' });
    expect(list).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Waiting for me 3' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(screen.getByRole('tabpanel', { name: 'Waiting for me 3' })).toHaveTextContent(
      'Showing waiting',
    );

    await user.click(screen.getByRole('tab', { name: 'Decided by me' }));
    expect(screen.getByRole('tabpanel', { name: 'Decided by me' })).toHaveTextContent(
      'Showing decided',
    );
  });

  it('moves between tabs with the arrow keys, Home and End', async () => {
    const user = userEvent.setup();
    render(<Example />);

    await user.tab();
    expect(screen.getByRole('tab', { name: 'Waiting for me 3' })).toHaveFocus();

    await user.keyboard('{ArrowRight}');
    expect(screen.getByRole('tab', { name: 'All open' })).toHaveFocus();
    expect(screen.getByText('Showing open')).toBeInTheDocument();

    await user.keyboard('{End}');
    expect(screen.getByRole('tab', { name: 'Decided by me' })).toHaveFocus();
    await user.keyboard('{ArrowRight}');
    expect(screen.getByRole('tab', { name: 'Waiting for me 3' })).toHaveFocus();
    await user.keyboard('{ArrowLeft}');
    expect(screen.getByText('Showing decided')).toBeInTheDocument();
  });
});
