import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { CompletenessChip } from './CompletenessChip';

const THREE_MISSING = [
  { id: 'owner', label: 'Owner' },
  { id: 'budget', label: 'Budget' },
  { id: 'teams', label: 'Teams' },
];

describe('CompletenessChip', () => {
  it('says "All filled in" when nothing is missing (FR-019)', () => {
    render(<CompletenessChip missing={[]} />);

    expect(screen.getByText('All filled in')).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('counts one missing field in the singular (FR-019)', () => {
    render(<CompletenessChip missing={[{ id: 'owner', label: 'Owner' }]} />);

    expect(screen.getByRole('button', { name: '1 field needed' })).toBeInTheDocument();
  });

  it('lists the missing fields when clicked (FR-019, constitution III)', async () => {
    const user = userEvent.setup();
    render(<CompletenessChip missing={THREE_MISSING} onSelectField={vi.fn()} />);

    await user.click(screen.getByRole('button', { name: '3 fields needed' }));

    const list = screen.getByRole('dialog', { name: '3 fields needed' });
    expect(list).toContainElement(screen.getByRole('button', { name: 'Owner' }));
    expect(list).toContainElement(screen.getByRole('button', { name: 'Budget' }));
    expect(list).toContainElement(screen.getByRole('button', { name: 'Teams' }));
  });

  it('takes the person to the field they choose (FR-019, constitution III)', async () => {
    const user = userEvent.setup();
    const onSelectField = vi.fn((id: string) => document.getElementById(id)?.focus());
    render(
      <>
        <CompletenessChip missing={THREE_MISSING} onSelectField={onSelectField} />
        <label htmlFor="budget">Budget</label>
        <input id="budget" />
      </>,
    );

    await user.click(screen.getByRole('button', { name: '3 fields needed' }));
    await user.click(screen.getByRole('button', { name: 'Budget' }));

    await waitFor(() => expect(onSelectField).toHaveBeenCalledWith('budget'));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    // The focus stays on the chosen field instead of jumping back to the chip.
    await waitFor(() => expect(screen.getByRole('textbox', { name: 'Budget' })).toHaveFocus());
  });

  it('lists the fields as plain text when there is nowhere to take the person (FR-019)', async () => {
    const user = userEvent.setup();
    render(<CompletenessChip missing={THREE_MISSING} />);

    await user.click(screen.getByRole('button', { name: '3 fields needed' }));

    expect(screen.getByRole('dialog', { name: '3 fields needed' })).toHaveTextContent('Owner');
    expect(screen.queryByRole('button', { name: 'Owner' })).not.toBeInTheDocument();
  });
});
