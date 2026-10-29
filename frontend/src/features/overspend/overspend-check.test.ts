import { describe, expect, it } from 'vitest';
import { budgetCheck, overLimitMessages } from './overspend-check';

const usd = (amount: string) => ({ amount, currency: 'USD' });
const PLANNED = usd('120000.00');

describe('037 FR-015 budgetCheck', () => {
  it.each([
    ['Warn never refuses', 'warn', '100,000', '150000.00', null, null],
    ['at the planned cost is within it', 'refuse', '120,000', '150000.00', null, null],
    [
      'lowered below the planned cost is refused',
      'refuse',
      '100,000',
      '150000.00',
      "The planned cost is 120,000.00 USD; this budget refuses overspending, so it can't be lower than that.",
      null,
    ],
    [
      'typed where there was none is refused',
      'refuse',
      '1',
      null,
      "The planned cost is 120,000.00 USD; this budget refuses overspending, so it can't be lower than that.",
      null,
    ],
    [
      'already over and kept is only noted',
      'refuse',
      '100,000',
      '100000.00',
      null,
      'The planned cost is already over this budget by 20,000.00 USD. Anything that adds cost will be refused until it is back within the budget.',
    ],
    ['no budget typed has nothing to go over', 'refuse', '', '150000.00', null, null],
  ] as const)('%s', (_name, mode, typed, stored, error, note) => {
    expect(budgetCheck({ mode, typed, stored, planned: PLANNED })).toEqual({ error, note });
  });

  it('has nothing to say without a planned cost (a new record)', () => {
    expect(budgetCheck({ mode: 'refuse', typed: '1', stored: null, planned: null })).toEqual({
      error: null,
      note: null,
    });
  });
});

describe('037 FR-014 overLimitMessages', () => {
  const limits = [
    {
      budget: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
      leftToPlan: usd('5000.00'),
    },
    { budget: { type: 'plan', id: 'P-1001', name: 'Lubricants' }, leftToPlan: usd('-100.00') },
  ];

  it('names every budget a raise would take over, with the overage', () => {
    expect(overLimitMessages(limits, '10000.00', '16,000')).toEqual([
      "This is 1,000.00 USD over the budget of campaign 'Helix launch', which refuses overspending.",
      "This is 6,100.00 USD over the budget of plan 'Lubricants', which refuses overspending.",
    ]);
  });

  it('allows keeping or lowering the cost, even under a budget already over', () => {
    expect(overLimitMessages(limits, '10000.00', '10,000')).toEqual([]);
    expect(overLimitMessages(limits, '10000.00', '9,000')).toEqual([]);
  });

  it('counts a new activity from nothing, and ignores text that is not an amount', () => {
    expect(overLimitMessages(limits.slice(0, 1), null, '5,000.01')).toHaveLength(1);
    expect(overLimitMessages(limits, null, 'abc')).toEqual([]);
  });
});
