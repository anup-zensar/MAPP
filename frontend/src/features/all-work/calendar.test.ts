import { describe, expect, it } from 'vitest';
import { barOf, hasNoDates, monthDays, monthWeeks, runsOn, todayOffset } from './calendar';

describe('calendar geometry (FR-025 – FR-028)', () => {
  it('places a bar from its start to its end date (US5 scenario 1)', () => {
    const bar = barOf({ level: 'activity', startDate: '2026-03-01', endDate: '2026-05-31' }, 2026);

    // 1 Mar is day 59 of 365; March to May are 92 days.
    expect(bar?.left).toBeCloseTo((59 / 365) * 100, 6);
    expect(bar?.width).toBeCloseTo((92 / 365) * 100, 6);
    expect(bar).toMatchObject({ continuesFrom: false, continuesInto: false, ongoing: false });
  });

  it('runs an ongoing activity to the end of the year with an open end (US5 scenario 3)', () => {
    const bar = barOf({ level: 'activity', startDate: '2026-10-01', ongoing: true }, 2026);

    expect((bar?.left ?? 0) + (bar?.width ?? 0)).toBeCloseTo(100, 6);
    expect(bar).toMatchObject({ ongoing: true, continuesInto: false });
  });

  it('cuts bars at the year’s edges and says so (US5 scenario 3)', () => {
    const into = barOf({ level: 'campaign', startDate: '2026-11-01', endDate: '2027-02-28' }, 2026);
    expect(into).toMatchObject({ continuesInto: true, continuesFrom: false });
    expect((into?.left ?? 0) + (into?.width ?? 0)).toBeCloseTo(100, 6);

    const from = barOf(
      { level: 'initiative', startDate: '2025-12-01', endDate: '2026-01-31' },
      2026,
    );
    expect(from).toMatchObject({ continuesFrom: true, left: 0 });
    expect(from?.width).toBeCloseTo((31 / 365) * 100, 6);
  });

  it('gives plans, records without dates and records outside the year no bar (D-124, FR-026)', () => {
    expect(
      barOf({ level: 'plan', startDate: '2026-01-01', endDate: '2026-12-31' }, 2026),
    ).toBeNull();
    expect(barOf({ level: 'activity' }, 2026)).toBeNull();
    expect(barOf({ level: 'activity', startDate: '2026-03-01' }, 2026)).toBeNull();
    expect(
      barOf({ level: 'activity', startDate: '2027-03-01', endDate: '2027-04-01' }, 2026),
    ).toBeNull();
    expect(hasNoDates({ level: 'activity' })).toBe(true);
    expect(hasNoDates({ level: 'plan' })).toBe(false);
    expect(hasNoDates({ level: 'activity', startDate: '2026-10-01', ongoing: true })).toBe(false);
  });

  it('puts the Today line on the date when it falls in the year (US5 scenario 2)', () => {
    // 18 Oct 2026 is day 290 of the year (0-based), the line in the middle of the day.
    expect(todayOffset('2026-10-18', 2026)).toBeCloseTo((290.5 / 365) * 100, 6);
    expect(todayOffset('2026-10-18', 2027)).toBeNull();
  });

  it('knows the days of each month, leap years included', () => {
    expect(monthDays(2026)).toEqual([31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]);
    expect(monthDays(2028)[1]).toBe(29);
  });

  it('lays out a month Monday to Sunday (FR-028)', () => {
    const weeks = monthWeeks(2026, 10);

    // 1 Oct 2026 is a Thursday.
    expect(weeks[0]).toEqual([
      null,
      null,
      null,
      '2026-10-01',
      '2026-10-02',
      '2026-10-03',
      '2026-10-04',
    ]);
    expect(weeks.at(-1)).toEqual([
      '2026-10-26',
      '2026-10-27',
      '2026-10-28',
      '2026-10-29',
      '2026-10-30',
      '2026-10-31',
      null,
    ]);
    expect(weeks.flat().filter(Boolean)).toHaveLength(31);
  });

  it('knows on which days an activity runs (FR-028)', () => {
    const activity = { startDate: '2026-10-10', endDate: '2026-10-20' };
    expect(runsOn(activity, '2026-10-10', 2026)).toBe(true);
    expect(runsOn(activity, '2026-10-21', 2026)).toBe(false);
    expect(runsOn({ startDate: '2026-10-10', ongoing: true }, '2026-12-31', 2026)).toBe(true);
    expect(runsOn({}, '2026-10-10', 2026)).toBe(false);
  });
});
