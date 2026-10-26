import { describe, expect, it } from 'vitest';
import { formatDateRange } from '@/format/intl';
import {
  activitiesApprovedText,
  contentsText,
  goalAmountLabel,
  groupLabel,
  SPREAD_LABELS,
  STATUS,
  yearDates,
} from './planText';

describe('planText', () => {
  it('names statuses with a word and a tone', () => {
    expect(STATUS.draft).toEqual({ word: 'Draft', tone: 'neutral' });
    expect(STATUS.published).toEqual({ word: 'Published', tone: 'positive' });
  });

  it('labels spreads and goal amounts', () => {
    expect(Object.values(SPREAD_LABELS)).toEqual([
      'Evenly',
      'All in first month',
      'All in last month',
      'Custom',
    ]);
    expect(goalAmountLabel('inq')).toBe('Goal amount (INQ)');
    expect(goalAmountLabel('revenue')).toBe('Goal amount (Revenue, USD)');
  });

  it('joins the counts of what a plan holds (FR-029)', () => {
    expect(
      contentsText({
        campaigns: 2,
        initiatives: 4,
        activities: 7,
        approvedActivities: 4,
        lineItems: 15,
      }),
    ).toBe('2 campaigns, 4 initiatives, 7 activities and 15 line items');
    expect(
      contentsText({
        campaigns: 1,
        initiatives: 1,
        activities: 0,
        approvedActivities: 0,
        lineItems: 0,
      }),
    ).toBe('1 campaign and 1 initiative');
    expect(
      contentsText({
        campaigns: 0,
        initiatives: 0,
        activities: 0,
        approvedActivities: 0,
        lineItems: 0,
      }),
    ).toBe('');
  });

  it('writes the year, approvals and record groups', () => {
    expect(yearDates(2026, formatDateRange)).toBe('01 Jan – 31 Dec 2026');
    expect(yearDates(2026, (start, end) => formatDateRange(start, end, 'YYYY-MM-DD'))).toBe(
      '2026-01-01 – 2026-12-31',
    );
    expect(activitiesApprovedText(4, 7)).toBe('4 of 7 activities approved');
    expect(groupLabel({ type: 'plan', id: 'P-1001', name: 'FY2026' })).toBe('Plan');
    expect(groupLabel({ type: 'campaign', id: 'C-1001', name: 'Retail growth' })).toBe(
      "Campaign 'Retail growth'",
    );
    expect(groupLabel({ type: 'lineItem', id: 'L-1001', name: 'Print' })).toBe("Line item 'Print'");
  });
});
