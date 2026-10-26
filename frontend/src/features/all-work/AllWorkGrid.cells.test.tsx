import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { closestTo } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { gridResult, standard2026 } from '@/testing/fixtures/all-work';
import { seedRows } from '@/testing/fixtures/all-work-seed';
import { AllWorkGrid } from './AllWorkGrid';
import type { AllWorkResult, ViewState } from './all-work-types';
import { visibleRows } from './view-rows';

/** The grid's first page as the page shows it; nothing is opened, closed or sorted here. */
function renderGrid(result: AllWorkResult, state: ViewState = standard2026()) {
  return renderWithProviders(
    <AllWorkGrid
      result={result}
      rows={visibleRows(result.rows, state)}
      continued=""
      search={state.search}
      sort={state.sort}
      onSort={vi.fn()}
      onToggle={vi.fn()}
      onOpen={vi.fn()}
    />,
    { user: testUser() },
  );
}

function bodyRow(name: string): HTMLElement {
  return closestTo(screen.getByRole('rowheader', { name: new RegExp(name) }), 'tr');
}

describe('AllWorkGrid cells (US1)', () => {
  it('shows money as the API answers it in the display currency, and local amounts with theirs (FR-017)', () => {
    const columns = [
      ...gridResult().columns,
      {
        key: 'lineItem.localPlannedCost',
        label: 'Planned cost in local currency',
        group: 'lineItem' as const,
        kind: 'money' as const,
        levels: ['lineItem' as const],
        sortable: true,
        inactive: false,
      },
    ];
    const eur = (amount: string) => ({ amount, currency: 'EUR' });
    const rows = seedRows().map((row) => ({
      ...row,
      cells: [
        ...row.cells.map((cell) =>
          cell?.kind === 'money' && cell.money ? { ...cell, money: eur(cell.money.amount) } : cell,
        ),
        row.level === 'lineItem'
          ? { kind: 'money' as const, money: { amount: '7110.00', currency: 'GBP' } }
          : null,
      ],
    }));
    const state = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
    renderGrid(gridResult({ columns, rows, displayCurrency: 'EUR' }), state);

    expect(within(bodyRow('Helix launch event')).getByText('12,000.00 EUR')).toBeInTheDocument();
    expect(within(bodyRow('Venue hire')).getByText('7,110.00 GBP')).toBeInTheDocument();
  });

  it('cuts long text with the full text on focus, and labels inactive values (edge cases)', async () => {
    const description = 'A very long description of the campaign that goes on and on';
    const rows = seedRows().map((row) =>
      row.key === 'C-1001'
        ? {
            ...row,
            cells: row.cells.map((cell, index) =>
              index === 0
                ? { kind: 'text' as const, text: description }
                : index === 2
                  ? { kind: 'text' as const, text: 'Tom Weber', inactive: true }
                  : cell,
            ),
          }
        : row,
    );
    const { user } = renderGrid(gridResult({ rows }));

    const campaign = bodyRow('Helix launch$');
    expect(within(campaign).getByText('Tom Weber (inactive)')).toBeInTheDocument();
    await user.tab();
    within(campaign).getByText(description).focus();
    expect(await screen.findByRole('tooltip', { name: description })).toBeInTheDocument();
  });

  it('marks the searched text and shows the excerpt of a hit in the description (FR-006)', () => {
    const rows = seedRows().map((row) =>
      row.key === 'A-1002'
        ? {
            ...row,
            searchHit: {
              fields: ['name' as const, 'description' as const],
              excerpt: '…paid social for the Helix launch…',
            },
          }
        : row,
    );
    renderGrid(gridResult({ rows }), { ...standard2026(), search: 'social' });

    const activity = bodyRow('Helix social ads');
    const marked = [...activity.querySelectorAll('mark')].map((mark) => mark.textContent);
    expect(marked).toEqual(['social', 'social']);
    expect(within(activity).getByText(/paid/)).toHaveTextContent(
      '…paid social for the Helix launch…',
    );
  });
});
