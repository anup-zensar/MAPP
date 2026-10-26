import { describe, expect, it } from 'vitest';
import {
  DIGITAL,
  DIGITAL_2027,
  EUR,
  GBP,
  lineItem,
  PRIYA_REF,
  USD,
} from '@/testing/fixtures/line-items';
import { formOptions } from '@/testing/fixtures/line-item-forms';
import {
  changeBudgetYear,
  changeCurrency,
  changeCustomMonth,
  changePlannedCost,
  changeSpreadMethod,
  editFormState,
  newFormState,
  withField,
} from './line-item-form-state';
import { toCreateBody } from './line-item-requests';

const CURRENCIES = [USD, EUR, GBP];
const MAR_MAY = { startDate: '2026-03-01', endDate: '2026-05-31', ongoing: false };

describe('newFormState (FR-008)', () => {
  it('fills in the defaults, each with its source', () => {
    const state = newFormState(formOptions());
    expect(state).toMatchObject({
      name: 'Launch film – line item',
      ownerId: PRIYA_REF.id,
      currency: 'USD',
      plannedCost: '20,000.00',
      spreadMethod: 'evenly',
      financeBudgetYear: 2026,
      financeBudgetItemId: '',
      requestSwbs: true,
    });
    expect(state.sources).toEqual({
      name: "From the activity's name",
      ownerId: 'From the activity',
      currency: 'Default',
      plannedCost: "The activity's unassigned planned cost",
      spread: "The plan's default",
      financeBudgetYear: "The plan's year",
      requestSwbs: 'On by default',
    });
  });

  it('says why the planned cost is empty (edge cases, D-144)', () => {
    const nothingLeft = formOptions();
    nothingLeft.defaults = {
      ...nothingLeft.defaults,
      plannedCost: null,
      plannedCostSource: 'nothingLeftToAssign',
    };
    expect(newFormState(nothingLeft)).toMatchObject({
      plannedCost: '',
      sources: {
        plannedCost: "Nothing left to assign — the activity's planned cost is fully assigned",
      },
    });
    nothingLeft.defaults.plannedCostSource = 'noActivityPlannedCost';
    expect(newFormState(nothingLeft).sources.plannedCost).toBe(
      'This activity has no planned cost yet, so there is nothing to pre-fill',
    );
  });

  it('says when the finance budget item is pre-selected (R12)', () => {
    const options = formOptions();
    options.defaults = {
      ...options.defaults,
      financeBudgetItemId: DIGITAL.id,
      financeBudgetItemPreselected: true,
    };
    expect(newFormState(options)).toMatchObject({
      financeBudgetItemId: DIGITAL.id,
      sources: {
        financeBudgetItemId: "Pre-selected — the only active item of '2026 Lubricants Budget'",
      },
    });
  });
});

describe('changeCurrency (FR-012)', () => {
  it('converts the pre-filled balance and says so', () => {
    const state = changeCurrency(newFormState(formOptions()), 'EUR', CURRENCIES);
    expect(state.plannedCost).toBe('18,400.00');
    expect(state.costNotes).toEqual(['Changed to 18,400.00 EUR = 20,000.00 USD']);
    expect(state.sources.currency).toBeUndefined();
  });

  it('adds the rounding hint when the USD value cannot match to the cent', () => {
    const options = formOptions();
    options.defaults = { ...options.defaults, plannedCost: { amount: '100.02', currency: 'USD' } };
    const state = changeCurrency(newFormState(options), 'GBP', CURRENCIES);
    expect(state.plannedCost).toBe('79.02');
    expect(state.costNotes[1]).toBe(
      "If a currency rate leaves a few cents unassigned, adjust the activity's planned cost instead.",
    );
  });

  it('does not convert an amount the person typed', () => {
    const typed = changePlannedCost(newFormState(formOptions()), '10,000');
    const state = changeCurrency(typed, 'EUR', CURRENCIES);
    expect(state.plannedCost).toBe('10,000');
    expect(state.costNotes).toEqual([]);
  });
});

describe('spread and finance budget changes', () => {
  it('starts Custom from the current spread (D-152)', () => {
    const state = changeSpreadMethod(
      changePlannedCost(newFormState(formOptions()), '10,000.00'),
      'custom',
      MAR_MAY,
      2026,
    );
    expect(state.customMonths).toEqual([
      { month: '2026-03', amount: '3,333.33' },
      { month: '2026-04', amount: '3,333.33' },
      { month: '2026-05', amount: '3,333.34' },
    ]);
    expect(state.customTyped).toBe(false);
    expect(changeCustomMonth(state, '2026-03', '5,000').customTyped).toBe(true);
  });

  it('keeps a saved custom spread as typed', () => {
    const custom = lineItem({
      spread: {
        method: 'custom',
        state: 'spread',
        months: [
          {
            month: '2026-02',
            amount: { amount: '2000.00', currency: 'USD' },
            amountUsd: { amount: '2000.00', currency: 'USD' },
            outsideActivityDates: true,
          },
        ],
        customTotal: { amount: '2000.00', currency: 'USD' },
        addsUp: false,
      },
    });
    const state = editFormState(custom);
    expect(state.customMonths).toEqual([{ month: '2026-02', amount: '2,000.00' }]);
    expect(state.customTyped).toBe(true);
  });

  it('marks the item of another year to clear, and brings it back with its year (FR-015)', () => {
    const items = [DIGITAL, DIGITAL_2027];
    const chosen = withField(newFormState(formOptions()), 'financeBudgetItemId', DIGITAL.id);
    const next = changeBudgetYear(chosen, 2027, items);
    expect(next.financeBudgetItemId).toBe('');
    expect(next.clearedItem).toEqual(DIGITAL);
    expect(toCreateBody(next).financeBudgetItemId).toBeNull();
    const back = changeBudgetYear(next, 2026, items);
    expect(back.financeBudgetItemId).toBe(DIGITAL.id);
    expect(back.clearedItem).toBeNull();
  });
});
