import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { closestTo, nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { gridResult, standard2026 } from '@/testing/fixtures/all-work';
import { seedRows } from '@/testing/fixtures/all-work-seed';
import { usd } from '@/testing/fixtures/all-work-rows';
import { AllWorkGrid } from './AllWorkGrid';
import type { AllWorkResult, AllWorkRow, ViewState } from './all-work-types';
import { continuedPath, pageOf, toggleRow, visibleRows } from './view-rows';
import { nextSort } from './view-columns';

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
  return closestTo(header, 'tr');
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
    const row = closestTo(nth(unassigned, 0), 'tr');
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
    const row = closestTo(totals, 'tr');
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
