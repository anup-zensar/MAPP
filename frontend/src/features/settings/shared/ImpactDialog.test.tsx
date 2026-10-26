import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { ImpactDialog } from './ImpactDialog';
import { UsageLine } from './UsageLine';
import { NOT_USED, USED_BY_TWO_PLANS } from '../testRecords';

const GOALS =
  "Used by 2 plans (31 activities). Their calculated MQL and revenue goals will be recalculated; typed goals ('Set later stages myself') are kept.";
const TYPE = "5 activities use 'Event'. They keep it; new activities can't choose it.";

describe('ImpactDialog (FR-011, FR-012)', () => {
  it('lists what the change affects, with Keep editing focused', async () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const { user } = renderWithProviders(
      <ImpactDialog
        kind="model"
        name="Lubricants 1.0"
        warnings={[
          { kind: 'goalsRecalculated', message: GOALS, count: 2 },
          { kind: 'activityTypeRemoved', message: TYPE, count: 5 },
        ]}
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    const dialog = screen.getByRole('alertdialog', {
      name: "Save changes to model 'Lubricants 1.0'?",
    });
    expect(
      within(dialog)
        .getAllByRole('listitem')
        .map((item) => item.textContent),
    ).toEqual([GOALS, TYPE]);
    expect(within(dialog).getByRole('button', { name: 'Keep editing' })).toHaveFocus();

    await user.click(within(dialog).getByRole('button', { name: 'Save changes' }));
    expect(onConfirm).toHaveBeenCalledOnce();

    await user.click(within(dialog).getByRole('button', { name: 'Keep editing' }));
    expect(onCancel).toHaveBeenCalledOnce();
  });
});

describe('UsageLine (FR-003)', () => {
  it('says where a record is used and links to the plans using it', () => {
    renderWithProviders(<UsageLine usage={USED_BY_TWO_PLANS} />);

    expect(screen.getByText('Used by 2 plans')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'FY2026 Mobility' })).toHaveAttribute(
      'href',
      '/plans/P-1002',
    );
  });

  it('says "Not used yet" for an unused record', () => {
    renderWithProviders(<UsageLine usage={NOT_USED} />);

    expect(screen.getByText('Not used yet')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });
});
