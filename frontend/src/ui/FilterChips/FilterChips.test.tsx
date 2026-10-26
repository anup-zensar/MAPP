import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { FilterChips } from './FilterChips';

describe('FilterChips', () => {
  it('removes one filter or all of them (design system §6.1)', async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    const onClearAll = vi.fn();
    render(
      <FilterChips
        chips={[
          { key: 'status', label: 'Status: Draft' },
          { key: 'year', label: 'Year: 2026' },
        ]}
        onRemove={onRemove}
        onClearAll={onClearAll}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Remove filter: Year: 2026' }));
    expect(onRemove).toHaveBeenCalledWith('year');

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(onClearAll).toHaveBeenCalledOnce();
  });

  it('shows nothing when no filter is in use', () => {
    const { container } = render(<FilterChips chips={[]} onRemove={vi.fn()} />);

    expect(container).toBeEmptyDOMElement();
  });
});
