import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ReapprovalDialog } from './ReapprovalDialog';

const DATES = [
  { label: 'Start date', old: '01 Mar 2026', new: '01 Apr 2026' },
  { label: 'End date', old: '31 May 2026', new: '30 Jun 2026' },
];

function open(
  changes: { label: string; old: string; new: string }[],
  lineItems = { count: 4, autoSpread: 3, customSpread: 1 },
) {
  render(
    <ReapprovalDialog
      open
      onOpenChange={() => {}}
      name="Launch film"
      changes={changes}
      lineItems={lineItems}
      onConfirm={() => {}}
    />,
  );
  return screen.getByRole('alertdialog', { name: 'This activity will need approval again' });
}

describe('ReapprovalDialog (feature 006, FR-027)', () => {
  it('lists each change old → new, and what happens to the line-item spreads when dates change', () => {
    const dialog = open(DATES);

    expect(dialog).toHaveTextContent('Start date: 01 Mar 2026 → 01 Apr 2026');
    expect(dialog).toHaveTextContent('End date: 31 May 2026 → 30 Jun 2026');
    expect(dialog).toHaveTextContent(
      '3 line items will be re-spread over the new dates; 1 custom spread is kept as typed.',
    );
    expect(screen.getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Save and restart approval' })).toBeVisible();
  });

  it('says nothing about spreads for a cost change, or without line items', () => {
    const dialog = open([{ label: 'Planned cost', old: '120,000.00 USD', new: '150,000.00 USD' }]);
    expect(dialog).not.toHaveTextContent('re-spread');
  });

  it('says nothing about spreads when the activity has no line items', () => {
    const dialog = open(DATES, { count: 0, autoSpread: 0, customSpread: 0 });
    expect(dialog).not.toHaveTextContent('re-spread');
  });
});
