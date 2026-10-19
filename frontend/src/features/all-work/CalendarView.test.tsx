import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { AllWorkResult, AllWorkRow } from './allWorkTypes';
import { CalendarView } from './CalendarView';
import { calendarResult, standard2026, usd } from './testData';
import { visibleRows } from './viewState';

function renderCalendar(result: AllWorkResult = calendarResult()) {
  const onOpen = vi.fn();
  const rendered = renderWithProviders(
    <CalendarView
      result={result}
      rows={visibleRows(result.rows, standard2026())}
      continued=""
      search=""
      onToggle={() => {}}
      onOpen={onOpen}
    />,
    { user: testUser() },
  );
  return { ...rendered, onOpen };
}

/** The bar of a record: the link in the timeline cell of its row. */
function barOf(name: RegExp): HTMLElement {
  const table = screen.getByRole('table', { name: 'Calendar 2026' });
  const [header] = within(table).getAllByRole('rowheader', { name });
  const cell = within(header.closest('tr') as HTMLElement).getByRole('cell');
  return within(cell).getByRole('link');
}

function percent(value: string): number {
  return Number(value.replace('%', ''));
}

function withRow(key: string, change: Partial<AllWorkRow>): AllWorkResult {
  const result = calendarResult();
  return {
    ...result,
    rows: result.rows.map((row) => (row.key === key ? { ...row, ...change } : row)),
  };
}

describe('CalendarView (US5)', () => {
  it('shows the months of the year across the top (FR-025)', () => {
    renderCalendar();

    const table = screen.getByRole('table', { name: 'Calendar 2026' });
    const [, timeline] = within(table).getAllByRole('columnheader');
    expect(timeline).toHaveTextContent('JanFebMarAprMayJunJulAugSepOctNovDec');
  });

  it('draws a bar from the start to the end date with its status word (US5 scenario 1, FR-025)', () => {
    renderCalendar();

    const bar = barOf(/Helix launch event/);
    expect(bar).toHaveTextContent('Helix launch eventApproved');
    expect(percent(bar.style.left)).toBeCloseTo((59 / 365) * 100, 4);
    expect(percent(bar.style.width)).toBeCloseTo((92 / 365) * 100, 4);
    expect(bar.className).toMatch(/positive/);
    expect(barOf(/Helix social ads/).className).toMatch(/problem/);
    // The name column shows the status chip too.
    const header = screen.getByRole('rowheader', { name: /Helix launch event/ });
    expect(within(header).getByText('Approved')).toBeInTheDocument();
  });

  it('explains the colours with a legend of words (FR-025, SC-008)', () => {
    renderCalendar();

    expect(screen.getByText('Status')).toBeInTheDocument();
    const legend = screen.getByText('Status').parentElement as HTMLElement;
    expect(
      within(legend)
        .getAllByRole('listitem')
        .map((item) => item.textContent),
    ).toEqual(['In progress', 'Approved', 'Declined', 'Created', 'No activities']);
  });

  it('gives plans no bar (D-124)', () => {
    renderCalendar();

    const header = screen.getByRole('rowheader', { name: /FY2026 Lubricants Europe/ });
    const cell = within(header.closest('tr') as HTMLElement).getByRole('cell');
    expect(within(cell).queryByRole('link')).not.toBeInTheDocument();
  });

  it('draws the Today line at 18 Oct 2026 (US5 scenario 2, FR-026)', () => {
    renderCalendar();

    const today = screen.getByText('Today');
    expect(percent(today.style.left)).toBeCloseTo((290.5 / 365) * 100, 4);
  });

  it('has no Today line in a year without today', () => {
    renderCalendar(calendarResult({ today: '2027-01-05' }));

    expect(screen.queryByText('Today')).not.toBeInTheDocument();
  });

  it('runs an ongoing activity to the end of the year with an open end (US5 scenario 3)', async () => {
    const { user } = renderCalendar(
      withRow('A-1002', { startDate: '2026-10-01', endDate: undefined, ongoing: true }),
    );

    const bar = barOf(/Helix social ads/);
    expect(bar).toHaveTextContent('Ongoing');
    expect(percent(bar.style.left) + percent(bar.style.width)).toBeCloseTo(100, 4);
    expect(bar.className).toMatch(/openEnd/);
    await user.tab();
    bar.focus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent('From 01 Oct 2026, ongoing');
  });

  it('says where a bar is cut by the year’s edges (US5 scenario 3, FR-026)', () => {
    renderCalendar(withRow('C-1003', { startDate: '2025-11-01', endDate: '2027-02-28' }));

    const bar = barOf(/Retail push/);
    expect(bar).toHaveTextContent('Continues from 2025');
    expect(bar).toHaveTextContent('Continues into 2027');
    expect(percent(bar.style.left)).toBe(0);
  });

  it('lists the records without dates with links (US5 scenario 4, FR-026)', async () => {
    const { user, onOpen } = renderCalendar();

    const undated = screen.getByRole('region', { name: 'No dates yet (3)' });
    expect(
      within(undated)
        .getAllByRole('link')
        .map((link) => link.textContent),
    ).toEqual(['Rimula fleet', 'Fleet days', 'Fleet roadshow']);
    await user.click(within(undated).getByRole('link', { name: 'Fleet roadshow' }));

    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ key: 'A-1003' }));
  });

  it('tells a bar’s name, ID, level, status, dates, owner and planned cost on focus (FR-026)', async () => {
    const { user } = renderCalendar(withRow('A-1001', { plannedCost: usd('12000.00') }));

    await user.tab();
    barOf(/Helix launch event/).focus();

    const tip = await screen.findByRole('tooltip');
    expect(tip).toHaveTextContent('Helix launch event · A-1001');
    expect(tip).toHaveTextContent('Activity · Approved');
    expect(tip).toHaveTextContent('01 Mar – 31 May 2026');
    expect(tip).toHaveTextContent('Owner: Priya Raman');
    expect(tip).toHaveTextContent('Planned cost: $12,000.00');
    expect(tip).toHaveTextContent("Change dates on the record's page");
  });

  it('opens the record from its bar and cannot be dragged (FR-026, FR-027)', async () => {
    const { user, onOpen } = renderCalendar();

    const bar = barOf(/Helix launch event/);
    expect(bar).toHaveAttribute('draggable', 'false');
    await user.click(bar);

    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ key: 'A-1001' }));
  });
});
