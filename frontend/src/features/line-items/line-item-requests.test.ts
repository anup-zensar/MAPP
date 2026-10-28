import { describe, expect, it } from 'vitest';
import { ApiError } from '@/api/errors';
import { formOptions } from '@/testing/fixtures/line-item-forms';
import {
  changeCustomMonth,
  changePlannedCost,
  changeSpreadMethod,
  newFormState,
  withField,
} from './line-item-form-state';
import { formProblems, toCreateBody, toUpdateBody } from './line-item-requests';

const MAR_MAY = { startDate: '2026-03-01', endDate: '2026-05-31', ongoing: false };

describe('request bodies', () => {
  it('sends empty amounts as null and months only for Custom, and no restart flag (034 FR-001)', () => {
    const state = withField(changePlannedCost(newFormState(formOptions()), ''), 'ownerId', '');
    expect(toCreateBody(state)).toEqual({
      name: 'Launch film – line item',
      ownerId: null,
      description: '',
      currency: 'USD',
      plannedCost: null,
      spread: { method: 'evenly', months: [] },
      attributes: {},
      contractedAmount: null,
      requestSwbs: true,
      financeBudgetYear: 2026,
      financeBudgetItemId: null,
    });
  });

  it('sends typed amounts without grouping, and other text as typed for the API to refuse', () => {
    let state = changePlannedCost(newFormState(formOptions()), '-50');
    state = changeSpreadMethod(state, 'custom', MAR_MAY, 2026);
    state = changeCustomMonth(state, '2026-03', '1,500.5');
    state = changeCustomMonth(state, '2026-04', '');
    state = changeCustomMonth(state, '2026-05', 'abc');
    const body = toUpdateBody(state, 4);
    expect(body.plannedCost).toBe('-50.00');
    expect(body.spread.months).toEqual([
      { month: '2026-03', amount: '1500.50' },
      { month: '2026-05', amount: 'abc' },
    ]);
    expect(body.version).toBe(4);
  });

  it('maps custom month problems to their inputs', () => {
    let state = changeSpreadMethod(newFormState(formOptions()), 'custom', MAR_MAY, 2026);
    state = changeCustomMonth(state, '2026-04', '-1');
    const body = toCreateBody(state);
    const error = new ApiError('validation', {
      fields: [
        { field: 'spread.months[1].amount', message: "Month amounts can't be negative." },
        { field: 'name', message: 'Enter a name for the line item.' },
      ],
    });
    expect(formProblems(error, body)).toEqual([
      { field: 'spread.month.2026-04', message: "Month amounts can't be negative." },
      { field: 'name', message: 'Enter a name for the line item.' },
    ]);
  });
});
