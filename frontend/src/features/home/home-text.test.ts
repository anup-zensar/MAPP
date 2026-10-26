import { describe, expect, it } from 'vitest';
import { formatDate, formatDateRange } from '@/format/intl';
import { logEntryOf, usd } from '@/testing/fixtures/home';
import {
  activityCountsText,
  approvedPercent,
  approvedText,
  changedFieldsText,
  datesText,
  greetingText,
  isZero,
  percentText,
  initialsOf,
  lineItemsNeedAttentionText,
  recentSentence,
  levelText,
  waitingText,
  ofPlanBudgetText,
  oldestWaitingText,
  percentUsedText,
  planCountsText,
  plansWithoutBudgetText,
  viewAllPlansText,
} from './home-text';

describe('homeText — the greeting (design system §2.3)', () => {
  it('greets the person by first name (FR-001)', () => {
    expect(greetingText('Priya')).toBe('Welcome back, Priya.');
  });
});

describe('homeText — summary cards (FR-005, FR-006, FR-007)', () => {
  it('counts plans and activities with their statuses', () => {
    expect(planCountsText(1, 0)).toBe('1 published · 0 draft');
    expect(planCountsText(1250, 3)).toBe('1,250 published · 3 draft');
    expect(activityCountsText(2, 1)).toBe('2 declined · 1 submitted');
  });

  it('says how long the oldest activity has waited, or that nothing waits', () => {
    const now = new Date('2026-03-12T09:00:00Z');

    expect(oldestWaitingText('2026-03-09T08:00:00Z', now)).toBe('oldest waiting 3 days');
    expect(oldestWaitingText('2026-03-12T08:30:00Z', now)).toBe('oldest waiting less than an hour');
    expect(oldestWaitingText(null, now)).toBe('Nothing waiting');
  });

  it('writes the budget lines of the planned-cost card', () => {
    expect(ofPlanBudgetText('$1.2M')).toBe('of $1.2M plan budget');
    expect(percentUsedText(84)).toBe('84% used');
    expect(percentText(125)).toBe('125%');
    expect(isZero(usd('0.00'))).toBe(true);
    expect(isZero(null)).toBe(true);
    expect(isZero(usd('0.01'))).toBe(false);
    expect(plansWithoutBudgetText(1)).toBe('Includes 1 plan with no budget');
    expect(plansWithoutBudgetText(2)).toBe('Includes 2 plans with no budget');
  });
});

describe('homeText — Plans at a glance (FR-008 – FR-010)', () => {
  it('writes the approval progress of a plan as words and as a bar value', () => {
    expect(approvedText(3, 7)).toBe('3 of 7 approved');
    expect(approvedPercent(3, 7)).toBe(43);
    expect(approvedPercent(0, 0)).toBe(0);
  });

  it('offers every plan in scope with its count', () => {
    expect(viewAllPlansText(12)).toBe('View all 12 plans');
  });
});

describe('homeText — waiting for my approval (FR-013)', () => {
  it('says the level and how long the activity has waited', () => {
    const now = new Date('2026-03-12T09:00:00Z');

    expect(levelText(1, 2)).toBe('Level 1 of 2');
    expect(waitingText('2026-03-11T08:00:00Z', now)).toBe('waiting 1 day');
    expect(waitingText('2026-03-12T04:00:00Z', now)).toBe('waiting 5 hours');
  });
});

describe('homeText — my activities (FR-014)', () => {
  const date = (value: string) => formatDate(value);

  it("writes an activity's dates as every date range reads (D-216)", () => {
    const format = { date, dateRange: (start: string, end: string) => formatDateRange(start, end) };
    expect(
      datesText({ startDate: '2026-03-01', endDate: '2026-05-31', ongoing: false }, format),
    ).toBe('01 Mar – 31 May 2026');
    expect(datesText({ startDate: '2026-03-01', endDate: null, ongoing: true }, format)).toBe(
      'Ongoing from 01 Mar 2026',
    );
    expect(datesText({ startDate: null, endDate: null, ongoing: false }, format)).toBe(
      'No dates yet',
    );
    expect(datesText({ startDate: '2026-03-01', endDate: null, ongoing: false }, format)).toBe(
      '01 Mar 2026 – —',
    );
  });

  it('counts the line items that need attention, singular for one', () => {
    expect(lineItemsNeedAttentionText(1)).toBe('1 line item needs attention');
    expect(lineItemsNeedAttentionText(3)).toBe('3 line items need attention');
  });
});

describe('homeText — recent changes (FR-015)', () => {
  it('lower-cases field labels in the middle of a sentence, except abbreviations', () => {
    expect(changedFieldsText(['Planned cost'])).toBe('planned cost');
    expect(changedFieldsText(['Name', 'GL code'])).toBe('name and GL code');
    expect(changedFieldsText(['Name', 'Region', 'Owner'])).toBe('name, region and owner');
  });

  it('names three fields and counts the rest', () => {
    expect(changedFieldsText(['Name', 'Region', 'Owner', 'Start date'])).toBe(
      'name, region, owner and 1 more field',
    );
    expect(changedFieldsText(['Name', 'Region', 'Owner', 'Start date', 'End date'])).toBe(
      'name, region, owner and 2 more fields',
    );
  });

  it('builds the sentence of a change, and of any other action with its verb', () => {
    expect(recentSentence(logEntryOf())).toEqual({
      lead: 'Priya Raman changed planned cost of activity',
      name: 'Launch film',
      id: 'A-1042',
    });
    expect(
      recentSentence(logEntryOf({ action: 'statusChanged', verb: 'submitted', changes: [] })).lead,
    ).toBe('Priya Raman submitted activity');
  });

  it("writes a person's initials", () => {
    expect(initialsOf('Priya Raman')).toBe('PR');
    expect(initialsOf('Mei Lin Chen')).toBe('MC');
    expect(initialsOf('Alex')).toBe('A');
  });
});
