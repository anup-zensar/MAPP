import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { calendarResult, standard2026, usd } from '@/testing/fixtures/all-work';
import type { AllWorkResult, AllWorkRow } from './all-work-types';
import { CalendarView } from './CalendarView';
import { visibleRows } from './view-rows';

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
  const table = screen.getByRole('table', { name: /^Calendar 2026/ });
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

    const table = screen.getByRole('table', { name: /^Calendar 2026/ });
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
    // The bar says the status, so the name column shows no chip (UI audit AW-06); a plan has no
    // bar and keeps its chip.
    const header = screen.getByRole('rowheader', { name: /Helix launch event/ });
    expect(within(header).queryByText('Approved')).not.toBeInTheDocument();
    const plan = screen.getByRole('rowheader', { name: /FY2026 Lubricants Europe/ });
    expect(within(plan).getByText('Published')).toBeInTheDocument();
  });

  it('writes the words of a short bar beside it instead of cutting them (UI audit AW-06)', () => {
    renderCalendar(withRow('A-1001', { startDate: '2026-03-01', endDate: '2026-03-10' }));

    const bar = barOf(/Helix launch event/);
    expect(bar).toHaveTextContent('Helix launch eventApproved');
    expect(bar.className).toMatch(/wordsAfter/);
    // A bar long enough for its words keeps them on it.
    expect(barOf(/Helix launch$/).className).not.toMatch(/wordsAfter|wordsBefore/);
  });

  it('puts the words of a short bar near the end of the year before it', () => {
    renderCalendar(withRow('A-1001', { startDate: '2026-12-01', endDate: '2026-12-10' }));

    expect(barOf(/Helix launch event/).className).toMatch(/wordsBefore/);
  });

  it('explains the colours with a legend of words, each word once (FR-025, SC-008, UI audit X-14)', () => {
    renderCalendar();

    expect(screen.getByText('Status')).toBeInTheDocument();
    const legend = screen.getByText('Status').parentElement as HTMLElement;
    // The approval roll-up of campaigns and initiatives reads "Partly approved": "In progress"
    // is only a running activity (functional review A-02).
    expect(
      within(legend)
        .getAllByRole('listitem')
        .map((item) => item.textContent),
    ).toEqual(['Partly approved', 'Approved', 'Declined', 'Created', 'No activities']);
    expect(barOf(/Helix launch$/)).toHaveTextContent('Helix launchPartly approved');
  });

  it('shows a range of years as one timeline with year and quarter bands (016 FR-036)', () => {
    renderCalendar(calendarResult({ yearTo: 2027, today: '2027-01-05' }));

    const table = screen.getByRole('table', { name: 'Calendar 2026–2027' });
    const [, timeline] = within(table).getAllByRole('columnheader');
    expect(timeline).toHaveTextContent('20262027Q1Q2Q3Q4Q1Q2Q3Q4');
    // 1 Mar 2026 is day 59 of the 730 days of 2026 and 2027.
    const bar = barOf(/Helix launch event/);
    expect(percent(bar.style.left)).toBeCloseTo((59 / 730) * 100, 4);
    expect(percent(screen.getByText('Today').style.left)).toBeCloseTo((369.5 / 730) * 100, 4);
  });

  it('says "Continues into" only at the end of the range (016 FR-036)', () => {
    const result = withRow('C-1003', { startDate: '2026-11-01', endDate: '2028-02-28' });
    renderCalendar({ ...result, yearTo: 2027 });

    expect(barOf(/Retail push/)).toHaveTextContent('Continues into 2028');
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
