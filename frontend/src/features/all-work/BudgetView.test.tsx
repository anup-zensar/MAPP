import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { AllWorkResult } from './allWorkTypes';
import { BudgetView } from './BudgetView';
import { budgetResult, standard2026, usd } from './testData';
import { visibleRows } from './viewState';

const OPEN_ALL = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };

function renderBudget(result: AllWorkResult = budgetResult()) {
  const onOpen = vi.fn();
  const rendered = renderWithProviders(
    <BudgetView
      result={result}
      rows={visibleRows(result.rows, OPEN_ALL)}
      continued=""
      search=""
      onToggle={() => {}}
      onOpen={onOpen}
    />,
    { user: testUser() },
  );
  return { ...rendered, onOpen };
}

/** The cells of a row after its name, by column heading (the first row of that name). */
function cellsOf(name: string | RegExp): Record<string, string> {
  const table = screen.getByRole('table', { name: 'Budget & cost' });
  const headings = within(table)
    .getAllByRole('columnheader')
    .map((heading) => heading.textContent ?? '');
  const [header] = within(table).getAllByRole('rowheader', { name });
  const cells = within(header.closest('tr') as HTMLElement).getAllByRole('cell');
  return Object.fromEntries(cells.map((cell, index) => [headings[index + 1], cell.textContent]));
}

describe('BudgetView (US4)', () => {
  it('shows Plan budget, the months of the year and Total, read-only (FR-022, FR-024)', () => {
    renderBudget();

    expect(
      within(screen.getByRole('table', { name: 'Budget & cost' }))
        .getAllByRole('columnheader')
        .map((heading) => heading.textContent),
    ).toEqual([
      'Name',
      'Plan budget',
      'Jan',
      'Feb',
      'Mar',
      'Apr',
      'May',
      'Jun',
      'Jul',
      'Aug',
      'Sep',
      'Oct',
      'Nov',
      'Dec',
      'Total',
    ]);
    expect(screen.getByText('To change how a cost is spread, open its line item')).toBeVisible();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  });

  it('shows a line item’s spread over the activity’s months and its total (US4 scenario 1, FR-023)', () => {
    renderBudget();

    expect(cellsOf(/Venue hire/)).toMatchObject({
      'Plan budget': '',
      Feb: '—',
      Mar: '$3,000.00',
      Apr: '$3,000.00',
      May: '$3,000.00',
      Jun: '—',
      Total: '$9,000.00',
    });
  });

  it('spreads the unassigned planned cost evenly and adds it to the activity (US4 scenario 2, FR-023)', () => {
    renderBudget();

    expect(cellsOf(/^Unassigned planned cost/)).toMatchObject({
      Mar: '$1,000.00',
      Apr: '$1,000.00',
      May: '$1,000.00',
      Total: '$3,000.00',
    });
    expect(cellsOf(/^ACTIVITYHelix launch event/)).toMatchObject({
      Mar: '$4,000.00',
      May: '$4,000.00',
      Total: '$12,000.00',
    });
  });

  it('says that line items exceed the planned cost instead of spreading a negative balance (FR-023)', () => {
    renderBudget();

    const activity = screen.getByRole('rowheader', { name: /Helix social ads/ });
    const warning = within(activity).getByText('Line items exceed the planned cost by $1,000.00');
    expect(warning.className).toMatch(/exceeded/);
    const unassigned = screen.getAllByRole('rowheader', { name: /^Unassigned planned cost/ })[1];
    expect(within(unassigned.closest('tr') as HTMLElement).queryByText(/\$/)).toBeNull();
  });

  it('adds up containers per period, with the plan budget on container rows only (FR-024)', () => {
    renderBudget();

    expect(cellsOf(/^PLANFY2026 Lubricants Europe/)).toMatchObject({
      'Plan budget': '$1,200,000.00',
      Mar: '$4,000.00',
      Apr: '$6,000.00',
      Jun: '$2,000.00',
      Total: '$18,000.00',
    });
    expect(cellsOf(/^CAMPAIGNRetail push/)).toMatchObject({ 'Plan budget': '—', Total: '—' });
  });

  it('closes with the period totals, the grand total and the plans’ budget (FR-024)', () => {
    renderBudget();

    const footer = screen
      .getByRole('rowheader', { name: 'Total · 3 activities' })
      .closest('tr') as HTMLElement;
    expect(
      within(footer)
        .getAllByRole('cell')
        .map((cell) => cell.textContent),
    ).toEqual([
      '$2,000,000.00',
      '—',
      '—',
      '$4,000.00',
      '$6,000.00',
      '$6,000.00',
      '$2,000.00',
      '—',
      '—',
      '—',
      '—',
      '—',
      '—',
      '$18,000.00',
    ]);
  });

  it('shows quarters, each the sum of its months (US4 scenario 3, FR-022)', () => {
    renderBudget(budgetResult('quarter'));

    expect(cellsOf(/^PLANFY2026 Lubricants Europe/)).toMatchObject({
      Q1: '$4,000.00',
      Q2: '$14,000.00',
      Q3: '—',
      Q4: '—',
      Total: '$18,000.00',
    });
  });

  it('shows Other years and No dates when the API answers them (FR-022)', () => {
    const result = budgetResult();
    result.periods = [
      ...result.periods,
      { key: 'other', label: 'Other years' },
      { key: 'noDates', label: 'No dates' },
    ];
    result.rows = result.rows.map((row) =>
      row.key === 'A-1003'
        ? { ...row, periods: [...(row.periods ?? []), null, usd('500.00')], total: usd('500.00') }
        : { ...row, periods: [...(row.periods ?? []), null, null] },
    );
    renderBudget(result);

    expect(cellsOf(/Fleet roadshow/)).toMatchObject({
      'Other years': '—',
      'No dates': '$500.00',
      Total: '$500.00',
    });
  });

  it('opens a row’s record (FR-024)', async () => {
    const { user, onOpen } = renderBudget();

    await user.click(cellsOfRow(/Venue hire/)[3]);

    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ key: 'L-1001' }));
  });
});

function cellsOfRow(name: RegExp): HTMLElement[] {
  const header = screen.getByRole('rowheader', { name });
  return within(header.closest('tr') as HTMLElement).getAllByRole('cell');
}
