import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { SegmentedFilter } from './SegmentedFilter';

type Status = 'active' | 'inactive' | 'all';

function Example() {
  const [status, setStatus] = useState<Status>('active');
  return (
    <>
      <SegmentedFilter
        label="Status"
        value={status}
        onChange={setStatus}
        options={[
          { value: 'active', label: 'Active', count: 5 },
          { value: 'inactive', label: 'Inactive', count: 1217 },
          { value: 'all', label: 'All' },
        ]}
      />
      <p>Showing {status}</p>
    </>
  );
}

describe('SegmentedFilter', () => {
  it('is a named group of pills; the chosen one is pressed (design system §6.1)', () => {
    renderWithProviders(<Example />);

    const group = screen.getByRole('group', { name: 'Status' });
    expect(within(group).getByRole('button', { name: 'Active 5' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(within(group).getByRole('button', { name: 'Inactive 1,217' })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
    expect(within(group).getByRole('button', { name: 'All' })).toBeInTheDocument();
  });

  it('reports the choice', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(screen.getByRole('button', { name: 'All' }));

    expect(screen.getByText('Showing all')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'true');
  });
});
