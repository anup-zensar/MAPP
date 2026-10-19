import { describe, expect, it } from 'vitest';
import { formatDate, formatMoney } from '@/lib/format';
import {
  activityCountsText,
  approvedPercent,
  approvedText,
  cardLinkName,
  changedFieldsText,
  datesText,
  initialsOf,
  lineItemsNeedAttentionText,
  recentSentence,
  levelText,
  missingFieldsText,
  reasonLinkName,
  reasonText,
  showAllText,
  viewAllText,
  waitingText,
  ofPlanBudgetText,
  oldestWaitingText,
  percentUsedText,
  planCountsText,
  plansWithoutBudgetText,
  viewAllPlansText,
} from './homeText';
import { logEntryOf, reasonOf, usd } from './testData';

describe('homeText — summary cards (FR-005, FR-006, FR-007)', () => {
  it('names a card link "<label>: <value> — <link label>"', () => {
    expect(cardLinkName("My teams' plans", '3', 'open the Plans list')).toBe(
      "My teams' plans: 3 — open the Plans list",
    );
  });

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
    expect(percentUsedText(84)).toBe('84 % used');
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

describe('homeText — needs your attention (FR-011, FR-012)', () => {
  const format = {
    money: (value: { amount: string; currency: string } | null | undefined) =>
      formatMoney(value?.amount, value?.currency ?? 'USD'),
    date: (date: string) => formatDate(date),
  };

  it('writes each reason as a sentence', () => {
    expect(
      reasonText(
        reasonOf('declined', {
          declinedBy: { id: 'U-1003', name: 'Daniel Okoro' },
          declinedAt: '2026-03-12T10:30:00Z',
          comment: 'Please cut the agency fee',
        }),
        format,
      ),
    ).toBe("Declined by Daniel Okoro on 12 Mar 2026: 'Please cut the agency fee'");
    expect(reasonText(reasonOf('negativeBalance', { amount: usd('1200.00') }), format)).toBe(
      'Line items exceed the planned cost by $1,200.00',
    );
    expect(reasonText(reasonOf('overAllocated', { amount: usd('25000.00') }), format)).toBe(
      'Over-allocated by $25,000.00',
    );
    expect(reasonText(reasonOf('overBudget', { amount: usd('5000.00') }), format)).toBe(
      'Planned cost is $5,000.00 over budget',
    );
  });

  it('leaves the comment out of a decline without one', () => {
    expect(
      reasonText(
        reasonOf('declined', {
          declinedBy: { id: 'U-1003', name: 'Daniel Okoro' },
          declinedAt: '2026-03-12T10:30:00Z',
          comment: '',
        }),
        format,
      ),
    ).toBe('Declined by Daniel Okoro on 12 Mar 2026');
  });

  it('counts missing fields and items below, singular for one', () => {
    expect(missingFieldsText(3, 2)).toBe('3 fields needed · 2 items below need attention');
    expect(missingFieldsText(1, 0)).toBe('1 field needed');
    expect(missingFieldsText(0, 1)).toBe('1 item below needs attention');
  });

  it("names a reason's link with its record", () => {
    expect(reasonLinkName('Open approval', 'activity', 'Launch film')).toBe(
      "Open approval for activity 'Launch film'",
    );
  });

  it('offers the rest of a list', () => {
    expect(showAllText(12)).toBe('Show all 12');
    expect(viewAllText(7)).toBe('View all 7');
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

  it("writes an activity's dates", () => {
    expect(
      datesText({ startDate: '2026-03-01', endDate: '2026-05-31', ongoing: false }, date),
    ).toBe('01 Mar 2026 – 31 May 2026');
    expect(datesText({ startDate: '2026-03-01', endDate: null, ongoing: true }, date)).toBe(
      'Ongoing from 01 Mar 2026',
    );
    expect(datesText({ startDate: null, endDate: null, ongoing: false }, date)).toBe(
      'No dates yet',
    );
    expect(datesText({ startDate: '2026-03-01', endDate: null, ongoing: false }, date)).toBe(
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
