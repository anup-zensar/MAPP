import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DeleteActivityDialog } from './DeleteActivityDialog';
import type { DeletedActivity } from './DeleteActivityDialog';
import { launchFilm } from './testData';

function ask(overrides: Partial<DeletedActivity> = {}) {
  render(
    <DeleteActivityDialog
      open
      onOpenChange={() => {}}
      activity={{ ...launchFilm(), ...overrides }}
      onConfirm={() => {}}
    />,
  );
  return screen.getByRole('alertdialog', { name: "Delete activity 'Launch film'?" });
}

const LINE_ITEMS = (count: number) => ({
  count,
  plannedCost: { amount: '0.00', currency: 'USD' },
  autoSpread: count,
  customSpread: 0,
});

describe('DeleteActivityDialog (feature 006, FR-031)', () => {
  it('says what goes with it and what it does to the planned costs, with Cancel focused', () => {
    const dialog = ask({ lineItems: LINE_ITEMS(3) });

    expect(dialog).toHaveTextContent('3 line items go with it.');
    expect(dialog).toHaveTextContent(
      'The initiative, campaign and plan planned costs go down by 120,000.00 USD.',
    );
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    expect(dialog).not.toHaveTextContent("It leaves the approvers' inbox.");
    expect(dialog).not.toHaveTextContent('comes back as Created');
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Delete activity' })).toBeVisible();
  });

  it("names the amount in the reader's display currency (D-220)", () => {
    const dialog = ask({
      display: {
        plannedCost: { amount: '110400.00', currency: 'EUR' },
        unassignedPlannedCost: { amount: '18400.00', currency: 'EUR' },
        revenue: null,
      },
    });

    expect(dialog).toHaveTextContent(
      'The initiative, campaign and plan planned costs go down by 110,400.00 EUR.',
    );
  });

  it('says one line item goes, and nothing about line items or costs when there are none', () => {
    expect(ask({ lineItems: LINE_ITEMS(1) })).toHaveTextContent('1 line item goes with it.');
  });

  it('leaves out the line items and planned cost sentences when there are none', () => {
    const dialog = ask({
      lineItems: LINE_ITEMS(0),
      display: { plannedCost: null, unassignedPlannedCost: null, revenue: null },
    });

    expect(dialog).not.toHaveTextContent('go with it');
    expect(dialog).not.toHaveTextContent('planned costs go down');
  });

  it("says a submitted activity leaves the approvers' inbox and comes back as Created", () => {
    const dialog = ask({ status: 'submitted' });

    expect(dialog).toHaveTextContent("It leaves the approvers' inbox.");
    expect(dialog).toHaveTextContent(
      "If it's restored later from the recycle bin, it comes back as Created.",
    );
  });

  it('says an approved activity comes back as Created', () => {
    const dialog = ask({ status: 'approved' });

    expect(dialog).not.toHaveTextContent("It leaves the approvers' inbox.");
    expect(dialog).toHaveTextContent(
      "If it's restored later from the recycle bin, it comes back as Created.",
    );
  });
});
