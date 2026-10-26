import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { OwnerImpactDialog } from './OwnerImpactDialog';
import { removedIds } from './removed-ids';
import type { ImpactWarning } from './api';

const WARNINGS: ImpactWarning[] = [
  {
    kind: 'ownerNeedsAttention',
    count: 1,
    message:
      'Priya Raman owns 1 plan that uses this team: FY2026 Lubricants Europe. Priya will no longer belong to its teams, so its owner will need attention.',
  },
  {
    kind: 'ownerNeedsAttention',
    count: 2,
    message:
      'Alex Morgan owns 2 plans that use this team: FY2026 A, FY2026 B. Alex will no longer belong to their teams, so their owner will need attention.',
  },
];

describe('OwnerImpactDialog (feature 003, FR-013)', () => {
  it('lists every warning, with Keep editing focused', () => {
    renderWithProviders(
      <OwnerImpactDialog
        warnings={WARNINGS}
        busy={false}
        onKeepEditing={vi.fn()}
        onSave={vi.fn()}
      />,
    );

    const dialog = screen.getByRole('alertdialog', { name: 'Some plans will need a new owner' });
    expect(
      within(dialog)
        .getAllByRole('listitem')
        .map((item) => item.textContent),
    ).toEqual(WARNINGS.map((warning) => warning.message));
    expect(within(dialog).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
  });

  it('ignores Save changes while saving', async () => {
    const onSave = vi.fn();
    const { user } = renderWithProviders(
      <OwnerImpactDialog warnings={WARNINGS} busy onKeepEditing={vi.fn()} onSave={onSave} />,
    );

    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    expect(onSave).not.toHaveBeenCalled();
  });

  it('is closed without warnings', () => {
    renderWithProviders(
      <OwnerImpactDialog warnings={[]} busy={false} onKeepEditing={vi.fn()} onSave={vi.fn()} />,
    );
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('tells which ids a form removed', () => {
    expect(removedIds(['T-1', 'T-2', 'T-3'], ['T-3', 'T-4'])).toEqual(['T-1', 'T-2']);
    expect(removedIds(['T-1'], ['T-1', 'T-2'])).toEqual([]);
  });
});
