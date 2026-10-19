import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
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

  it('adds a chip that counts the records below needing attention (feature 004, D-199)', () => {
    render(<CompletenessChip missing={[]} itemsBelow={1} />);

    expect(screen.getByText('All filled in')).toBeInTheDocument();
    expect(screen.getByText('1 item below needs attention')).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('makes the counted chip a button when it can open the list', async () => {
    const user = userEvent.setup();
    const onItemsBelowSelect = vi.fn();
    render(
      <CompletenessChip missing={[]} itemsBelow={3} onItemsBelowSelect={onItemsBelowSelect} />,
    );

    await user.click(screen.getByRole('button', { name: '3 items below need attention' }));

    expect(onItemsBelowSelect).toHaveBeenCalledOnce();
  });

  it('names the records below when they have a name of their own (feature 006)', () => {
    render(
      <CompletenessChip
        missing={[]}
        itemsBelow={2}
        itemsBelowText={(count) => `${count} line items need attention`}
      />,
    );

    expect(screen.getByText('2 line items need attention')).toBeInTheDocument();
    expect(screen.queryByText(/below/)).not.toBeInTheDocument();
  });

  it('lists the records below with links to them', async () => {
    const { user } = renderWithProviders(
      <CompletenessChip
        missing={THREE_MISSING}
        itemsBelow={[
          {
            id: 'C-1002',
            label: "Campaign 'Retail growth'",
            href: '/plans/P-1001/campaigns/C-1002',
          },
          {
            id: 'A-1003',
            label: "Activity 'Launch film'",
            href: '/plans/P-1001/activities/A-1003',
          },
        ]}
      />,
    );

    await user.click(screen.getByRole('button', { name: '2 items below need attention' }));

    expect(screen.getByRole('link', { name: "Activity 'Launch film'" })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1003',
    );
  });

  it('shows no second chip when nothing below needs attention', () => {
    render(<CompletenessChip missing={[]} itemsBelow={[]} />);

    expect(screen.queryByText(/below/)).not.toBeInTheDocument();
  });
});
