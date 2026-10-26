import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import type { AllWorkResult, AllWorkRow } from './all-work-types';
import { MonthCalendar } from './MonthCalendar';
import { calendarResult, row } from '@/testing/fixtures/all-work';

/** An activity of Helix digital running between two days of 2026. */
function activity(key: string, name: string, startDate: string, endDate: string): AllWorkRow {
  return row({
    key,
    level: 'activity',
    name,
    depth: 3,
    parentKey: 'I-1001',
    planId: 'P-1001',
    status: 'approved',
    owner: 'Priya Raman',
    startDate,
    endDate,
  });
}

function renderMonth(result: AllWorkResult = calendarResult()) {
  const onOpen = vi.fn();
  const rendered = renderWithProviders(<MonthCalendar result={result} onOpen={onOpen} />, {
    user: testUser(),
  });
  return { ...rendered, onOpen };
}

/** The cell of a day of the month on screen. */
function dayCell(day: number): HTMLElement {
  const table = screen.getByRole('table');
  const cell = within(table)
    .getAllByRole('cell')
    .find(
      (candidate) => candidate.firstElementChild?.firstElementChild?.textContent === String(day),
    );
  if (!cell) throw new Error(`No day ${day} on screen`);
  return cell;
}

describe('MonthCalendar (FR-028)', () => {
  it('opens on today’s month as a Monday-to-Sunday grid', () => {
    renderMonth();

    expect(screen.getByRole('heading', { name: 'October 2026' })).toBeInTheDocument();
    expect(screen.getAllByRole('columnheader').map((heading) => heading.textContent)).toEqual([
      'Mon',
      'Tue',
      'Wed',
      'Thu',
      'Fri',
      'Sat',
      'Sun',
    ]);
    // 1 Oct 2026 is a Thursday: three empty days come first.
    const [firstWeek] = within(screen.getByRole('table')).getAllByRole('row').slice(1);
    expect(
      within(firstWeek)
        .getAllByRole('cell')
        .map((cell) => cell.textContent),
    ).toEqual(['', '', '', '1', '2', '3', '4']);
    expect(dayCell(18).firstElementChild).toHaveAttribute('aria-current', 'date');
  });

  it('moves between the months of the year and back to today', async () => {
    const { user } = renderMonth();

    await user.click(screen.getByRole('button', { name: 'Next month' }));
    expect(screen.getByRole('heading', { name: 'November 2026' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Next month' }));
    expect(screen.getByRole('button', { name: 'Next month' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Previous month' }));
    await user.click(screen.getByRole('button', { name: 'Previous month' }));
    await user.click(screen.getByRole('button', { name: 'Previous month' }));
    expect(screen.getByRole('heading', { name: 'September 2026' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Today' }));
    expect(screen.getByRole('heading', { name: 'October 2026' })).toBeInTheDocument();
  });

  it('shows each activity on the days it runs, with its status word', () => {
    const result = calendarResult();
    result.rows = [
      ...result.rows,
      activity('A-2001', 'Fleet day Rotterdam', '2026-10-10', '2026-10-12'),
    ];
    renderMonth(result);

    for (const day of [10, 11, 12]) {
      expect(
        within(dayCell(day)).getByRole('link', { name: /Fleet day Rotterdam/ }),
      ).toHaveTextContent('Fleet day RotterdamApproved');
    }
    expect(within(dayCell(13)).queryByRole('link')).not.toBeInTheDocument();
  });

  it('shows three activities a day and the rest behind "+ n more"', async () => {
    const result = calendarResult();
    result.rows = [
      ...result.rows,
      ...[1, 2, 3, 4, 5].map((n) =>
        activity(`A-30${n}`, `Webinar ${n}`, '2026-10-20', '2026-10-20'),
      ),
    ];
    const { user, onOpen } = renderMonth(result);

    const day = dayCell(20);
    expect(within(day).getAllByRole('link')).toHaveLength(3);
    await user.click(within(day).getByRole('button', { name: '+ 2 more on 20 Oct 2026' }));

    const list = await screen.findByRole('dialog', { name: '20 Oct 2026' });
    expect(within(list).getAllByRole('link')).toHaveLength(5);
    await user.click(within(list).getByRole('link', { name: /Webinar 5/ }));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ key: 'A-305' }));
  });

  it('starts in January of a year without today, without a Today button', () => {
    renderMonth(calendarResult({ year: 2027, today: '2026-10-18' }));

    expect(screen.getByRole('heading', { name: 'January 2027' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Today' })).not.toBeInTheDocument();
  });
});
