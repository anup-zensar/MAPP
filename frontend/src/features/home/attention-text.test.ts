import { describe, expect, it } from 'vitest';
import { formatDate, formatMoney } from '@/format/intl';
import { reasonOf, usd } from '@/testing/fixtures/home';
import { missingFieldsText, reasonLinkName, reasonLinkOf, reasonText } from './attention-text';
import { viewAllText } from './home-text';

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

  it("calls an activity's items below its line items, as My activities does (functional review X-07)", () => {
    expect(missingFieldsText(3, 1, 'activity')).toBe(
      '3 fields needed · 1 line item needs attention',
    );
    expect(missingFieldsText(0, 2, 'activity')).toBe('2 line items need attention');
    expect(missingFieldsText(0, 2, 'campaign')).toBe('2 items below need attention');
  });

  it('offers "Show fields" only where fields are missing (functional review X-07)', () => {
    const fields = reasonOf('missingFields', {
      missing: [{ field: 'gl', label: 'GL code' }],
      itemsBelow: 1,
    });
    const below = reasonOf('missingFields', { missing: [], itemsBelow: 2 });

    expect(reasonLinkOf(fields, 'campaign')).toEqual({
      label: 'Show fields',
      hash: '#completeness',
    });
    expect(reasonLinkOf(below, 'activity')).toEqual({
      label: 'Open line items',
      hash: '#line-items',
    });
    expect(reasonLinkOf(below, 'campaign')).toEqual({
      label: 'Show items below',
      hash: '#completeness',
    });
  });

  it("names a reason's link with its record", () => {
    expect(reasonLinkName('Open approval', 'activity', 'Launch film')).toBe(
      "Open approval for activity 'Launch film'",
    );
  });

  it('offers the rest of a list', () => {
    expect(viewAllText(7)).toBe('View all (7)');
  });
});
