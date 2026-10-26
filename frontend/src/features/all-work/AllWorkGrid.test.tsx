import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { gridResult, seedRows, standard2026, usd } from '@/testing/fixtures/all-work';
import { AllWorkGrid } from './AllWorkGrid';
import type { AllWorkResult, AllWorkRow, ViewState } from './all-work-types';
import { continuedPath, nextSort, pageOf, toggleRow, visibleRows } from './view-state';

/** The grid with its expansion kept the way the page keeps it. */
function Harness({
  result,
  initial,
  onOpen,
  pageSize = 50,
}: {
  result: AllWorkResult;
  initial: ViewState;
  onOpen: (row: AllWorkRow) => void;
  pageSize?: number;
}) {
  const [state, setState] = useState(initial);
  const [page, setPage] = useState(1);
  const shown = pageOf(visibleRows(result.rows, state), page, pageSize);
  return (
    <>
      <AllWorkGrid
        result={result}
        rows={shown.rows}
        continued={continuedPath(shown.rows, result.rows)}
        search={state.search}
        sort={state.sort}
        onSort={(column) => setState({ ...state, sort: nextSort(state.sort, column) })}
        onToggle={(key) => setState(toggleRow(state, key))}
        onOpen={onOpen}
      />
      <button type="button" onClick={() => setPage(page + 1)}>
        Next page
      </button>
    </>
  );
}

function renderGrid(
  result: AllWorkResult = gridResult(),
  initial: ViewState = standard2026(),
  pageSize?: number,
) {
  const onOpen = vi.fn();
  const rendered = renderWithProviders(
    <Harness result={result} initial={initial} onOpen={onOpen} pageSize={pageSize} />,
    { user: testUser() },
  );
  return { ...rendered, onOpen };
}

function bodyRow(name: string): HTMLElement {
  const header = screen.getByRole('rowheader', { name: new RegExp(name) });
  return header.closest('tr') as HTMLElement;
}

describe('AllWorkGrid (US1)', () => {
  it('shows each row with its level tag, its name as a link and its ID (FR-011)', () => {
    renderGrid();

    const table = screen.getByRole('table', { name: 'All work grid' });
    expect(
      within(table)
        .getAllByRole('columnheader')
        .map((header) => header.textContent),
    ).toEqual([
      'Name',
      'ID',
      'Status',
      'Owner',
      'Start date',
      'End date',
      'Plan budget',
      'Planned cost',
      'Line items',
      'Unassigned planned cost',
    ]);
    const plan = bodyRow('FY2026 Lubricants Europe');
    expect(within(plan).getByText('PLAN')).toBeInTheDocument();
    expect(within(plan).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(within(plan).getByText('P-1001')).toBeInTheDocument();
    expect(within(plan).getByText('Published')).toBeInTheDocument();
    expect(within(plan).getByText('01 Jan 2026')).toBeInTheDocument();
    expect(within(plan).getByText('$1,200,000.00')).toBeInTheDocument();
    expect(within(bodyRow('Helix launch$')).getByText('CAMPAIGN')).toBeInTheDocument();
    expect(within(bodyRow('Helix digital')).getByRole('link')).toHaveAttribute(
      'href',
      '/plans/P-1001/initiatives/I-1001',
    );
    expect(within(bodyRow('Helix social ads')).getByText('Declined')).toBeInTheDocument();
  });

  it('marks strategic plans with AOP (FR-011)', () => {
    renderGrid();

    expect(within(bodyRow('FY2026 Mobility China')).getByText('AOP')).toBeInTheDocument();
    expect(within(bodyRow('FY2026 Lubricants Europe')).queryByText('AOP')).not.toBeInTheDocument();
  });

  it('shows "—" for an empty value and nothing on levels a column does not belong to (FR-020)', () => {
    renderGrid();

    const activity = bodyRow('Fleet roadshow');
    // Fleet roadshow has no dates, no planned cost and no line items.
    expect(within(activity).getAllByText('—').length).toBeGreaterThanOrEqual(4);
    // Plan budget belongs to plans, campaigns and initiatives only.
    const cells = within(activity).getAllByRole('cell');
    expect(cells[5]).toBeEmptyDOMElement();
  });

  it('moves the status chip into the name when the Status column is hidden (D-118)', () => {
    const columns = gridResult().columns.filter((column) => column.key !== 'status');
    const rows = seedRows(columns.map((column) => column.key));
    renderGrid(gridResult({ columns, rows }));

    expect(screen.queryByRole('columnheader', { name: 'Status' })).not.toBeInTheDocument();
    expect(within(bodyRow('Helix social ads')).getByText('Declined')).toBeInTheDocument();
  });

  it('opens and closes rows with the mouse and the keyboard (FR-011)', async () => {
    const { user } = renderGrid();

    expect(screen.queryByRole('rowheader', { name: /Venue hire/ })).not.toBeInTheDocument();
    const expand = screen.getByRole('button', { name: 'Expand Helix launch event' });
    expect(expand).toHaveAttribute('aria-expanded', 'false');
    await user.click(expand);

    expect(screen.getByRole('rowheader', { name: /Venue hire/ })).toBeInTheDocument();
    const collapse = screen.getByRole('button', { name: 'Collapse Helix launch event' });
    expect(collapse).toHaveAttribute('aria-expanded', 'true');

    collapse.focus();
    await user.keyboard('{Enter}');
    expect(screen.queryByRole('rowheader', { name: /Venue hire/ })).not.toBeInTheDocument();

    screen.getByRole('button', { name: 'Collapse FY2026 Lubricants Europe' }).focus();
    await user.keyboard(' ');
    expect(screen.queryByRole('rowheader', { name: /Helix launch$/ })).not.toBeInTheDocument();
  });

  it('shows the unassigned planned cost under an open activity, in red with a minus sign when negative (FR-012)', async () => {
    const { user } = renderGrid();

    await user.click(screen.getByRole('button', { name: 'Expand Helix social ads' }));

    const unassigned = screen.getAllByRole('rowheader', {
      name: /Unassigned planned cost \(Sys_Gen_Balance\)/,
    });
    expect(unassigned).toHaveLength(1);
    const row = unassigned[0].closest('tr') as HTMLElement;
    const amounts = within(row).getAllByText('-$1,000.00');
    expect(amounts).toHaveLength(2);
    amounts.forEach((amount) => expect(amount.className).toMatch(/problem/));
    expect(within(row).getByRole('link')).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1002',
    );
  });

  it('labels context rows and does not count them (FR-005)', async () => {
    const rows = seedRows().map((row) =>
      row.key === 'P-1001' || row.key === 'C-1001' ? { ...row, role: 'context' as const } : row,
    );
    const { user } = renderGrid(gridResult({ rows }));

    const plan = bodyRow('FY2026 Lubricants Europe');
    expect(within(plan).getByText('Shown for context')).toBeInTheDocument();
    expect(within(bodyRow('Helix digital')).queryByText('Shown for context')).toBeNull();

    await user.click(within(plan).getByRole('button', { name: 'About Shown for context' }));
    expect(
      await screen.findByText(
        "This row doesn't match the filters; it's shown so you can see where the matching rows are.",
        { selector: '[role="dialog"], [role="dialog"] *' },
      ),
    ).toBeInTheDocument();
  });

  it('marks a roll-up that filters make smaller and gives its unfiltered value (FR-013)', async () => {
    const rows = seedRows().map((row) =>
      row.key === 'P-1001'
        ? {
            ...row,
            filtered: true,
            cells: row.cells.map((cell, index) =>
              index === 6
                ? {
                    kind: 'money' as const,
                    money: usd('5000.00'),
                    unfiltered: { kind: 'money' as const, money: usd('17000.00') },
                  }
                : cell,
            ),
          }
        : row,
    );
    const { user } = renderGrid(gridResult({ rows }));

    const plan = bodyRow('FY2026 Lubricants Europe');
    const mark = within(plan).getByRole('button', { name: 'Filtered' });
    await user.tab();
    mark.focus();
    expect(
      await screen.findByRole('tooltip', {
        name: 'Filters hide some records below. Without filters: $17,000.00.',
      }),
    ).toBeInTheDocument();
  });

  it('closes the grid with the totals row (FR-014)', () => {
    renderGrid();

    const totals = screen.getByRole('rowheader', { name: 'Total · 3 activities' });
    const row = totals.closest('tr') as HTMLElement;
    expect(within(row).getByText('$2,000,000.00')).toBeInTheDocument();
    expect(within(row).getByText('$17,000.00')).toBeInTheDocument();
    expect(within(row).getByText('$15,000.00')).toBeInTheDocument();
    expect(within(row).getByText('$3,000.00')).toBeInTheDocument();
  });

  it('explains that Unassigned planned cost adds positive balances only (functional review X-02)', async () => {
    const { user } = renderGrid();

    const heading = screen.getByRole('columnheader', { name: /Unassigned planned cost/ });
    await user.click(
      within(heading).getByRole('button', { name: 'About Unassigned planned cost' }),
    );
    expect(await screen.findByText(/Only positive balances are added/)).toBeInTheDocument();
  });

  it('writes the approval roll-up as "Partly approved", "In progress" only for running activities (functional review A-02)', () => {
    renderGrid();

    const campaign = bodyRow('Helix launch$');
    expect(within(campaign).getByText('Partly approved')).toBeInTheDocument();
    expect(within(campaign).queryByText('In progress')).not.toBeInTheDocument();
  });

  it('opens a row’s record from a click on the row or its name, not from its controls (FR-016)', async () => {
    const { user, onOpen, router } = renderGrid();

    await user.click(within(bodyRow('Helix digital')).getByText('I-1001'));
    expect(onOpen).toHaveBeenLastCalledWith(expect.objectContaining({ key: 'I-1001' }));

    await user.click(screen.getByRole('link', { name: 'Helix social ads' }));
    expect(onOpen).toHaveBeenLastCalledWith(expect.objectContaining({ key: 'A-1002' }));
    // The page decides when to go: the link alone does not navigate.
    expect(router.state.location.pathname).toBe('/');

    onOpen.mockClear();
    await user.click(screen.getByRole('button', { name: 'Expand Helix social ads' }));
    expect(onOpen).not.toHaveBeenCalled();
  });

  it('opens the record with Enter on its name (FR-016)', async () => {
    const { user, onOpen } = renderGrid();

    screen.getByRole('link', { name: 'Fleet roadshow' }).focus();
    await user.keyboard('{Enter}');

    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ key: 'A-1003' }));
  });

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

  it('sorts by a column heading: ascending, descending, then the plan-tree order (FR-015)', async () => {
    const { user } = renderGrid();

    const heading = screen.getByRole('columnheader', { name: 'Planned cost' });
    expect(heading).not.toHaveAttribute('aria-sort');
    await user.click(within(heading).getByRole('button', { name: 'Planned cost' }));
    expect(heading).toHaveAttribute('aria-sort', 'ascending');
    await user.click(within(heading).getByRole('button', { name: 'Planned cost' }));
    expect(heading).toHaveAttribute('aria-sort', 'descending');
    await user.click(within(heading).getByRole('button', { name: 'Planned cost' }));
    expect(heading).not.toHaveAttribute('aria-sort');

    // The rows come sorted from the API (D-114); Name is not a column and is not sortable.
    expect(
      within(screen.getByRole('columnheader', { name: 'Name' })).queryByRole('button'),
    ).not.toBeInTheDocument();
  });

  it('names the rows above a page that starts inside a branch (D-115)', async () => {
    const state = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
    const { user } = renderGrid(gridResult(), state, 5);

    expect(screen.queryByText(/^Continued:/)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Next page' }));

    expect(
      screen.getByText(
        'Continued: FY2026 Lubricants Europe › Helix launch › Helix digital › Helix launch event',
      ),
    ).toBeInTheDocument();
    // The totals row closes every page.
    expect(screen.getByRole('rowheader', { name: 'Total · 3 activities' })).toBeInTheDocument();
  });
});
