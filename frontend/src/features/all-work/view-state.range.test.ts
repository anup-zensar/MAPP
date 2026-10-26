import { describe, expect, it } from 'vitest';
import { standard2026 } from '@/testing/fixtures/all-work';
import type { ViewState } from './all-work-types';
import { chipsOf, clearAll, removeChip } from './view-chips';
import { fromAddress, hasAddressFilters } from './schemas';
import {
  sameState,
  savedStateOf,
  toQueryParams,
  withoutRefused,
  withYears,
  yearRange,
} from './view-state';

const withPlan = (state: ViewState): ViewState => ({
  ...state,
  filters: { ...state.filters, plans: [{ id: 'P-1001', label: 'FY2026 Lubricants Europe' }] },
});

describe('Years (016 FR-034, FR-035, FR-043)', () => {
  it('reads a state without a range as one year', () => {
    expect(yearRange(standard2026())).toEqual({ from: 2026, to: 2026 });
  });

  it('chooses a range: Budget & cost opens on Yearly and chosen plans go', () => {
    const state = withYears(withPlan(standard2026()), 2025, 2027);

    expect(yearRange(state)).toEqual({ from: 2025, to: 2027 });
    expect(state).toMatchObject({ year: 2025, yearFrom: 2025, yearTo: 2027, period: 'year' });
    expect(state.filters.plans).toEqual([]);
  });

  it('goes back to one plain year, and from Yearly to Monthly', () => {
    const state = withYears(withYears(standard2026(), 2025, 2027), 2026, 2026);

    expect(state.year).toBe(2026);
    expect(state).not.toHaveProperty('yearTo');
    expect(state.period).toBe('month');
  });

  it('never reads more than five years', () => {
    expect(yearRange({ year: 2024, yearTo: 2031 })).toEqual({ from: 2024, to: 2028 });
  });

  it('asks one year as `year` and a range as yearFrom and yearTo', () => {
    expect(toQueryParams(standard2026())).toMatchObject({ year: 2026 });
    const query = toQueryParams(withYears(standard2026(), 2026, 2028));
    expect(query).toMatchObject({ yearFrom: 2026, yearTo: 2028 });
    expect(query).not.toHaveProperty('year');
  });

  it('opens a range named in the address, and ignores one longer than five years', () => {
    const params = new URLSearchParams('yearFrom=2025&yearTo=2026');
    expect(hasAddressFilters(params)).toBe(true);
    expect(yearRange(fromAddress(params, standard2026()))).toEqual({ from: 2025, to: 2026 });

    const tooLong = fromAddress(new URLSearchParams('yearFrom=2020&yearTo=2026'), standard2026());
    expect(yearRange(tooLong)).toEqual({ from: 2020, to: 2020 });
  });

  it('sets a refused range back to the standard year', () => {
    const state = withoutRefused(withYears(standard2026(), 2025, 2029), ['yearTo'], standard2026());

    expect(yearRange(state)).toEqual({ from: 2026, to: 2026 });
  });

  it('treats one year saved before ranges and the same year as a range alike (FR-043)', () => {
    expect(sameState(standard2026(), { ...standard2026(), yearFrom: 2026, yearTo: 2026 })).toBe(
      true,
    );
  });
});

describe('Plan type, Plan currency and Financials (016 FR-037, 015 FR-024)', () => {
  const filtered: ViewState = {
    ...standard2026(),
    filters: {
      ...standard2026().filters,
      planTypes: ['strategic', 'crossMarket'],
      planCurrencies: ['EUR'],
      finance: ['committedAbovePlanned'],
    },
  };

  it('sends each filter in use with the API name', () => {
    expect(toQueryParams(filtered)).toMatchObject({
      planType: ['strategic', 'crossMarket'],
      planCurrency: ['EUR'],
      finance: ['committedAbovePlanned'],
    });
    expect(toQueryParams(standard2026())).not.toHaveProperty('planType');
    expect(toQueryParams(standard2026())).not.toHaveProperty('finance');
  });

  it('shows a removable chip per value', () => {
    expect(chipsOf(filtered).map((chip) => chip.label)).toEqual([
      'Plan type: Strategic',
      'Plan type: Cross-market',
      'Plan currency: EUR',
      'Financials: Committed above planned cost',
    ]);
    const fewer = removeChip(filtered, 'finance:committedAbovePlanned');
    expect(fewer.filters.finance).toEqual([]);
    expect(chipsOf(clearAll(filtered))).toEqual([]);
  });

  it('reads them from the address, as Home links to committed above planned cost', () => {
    const state = fromAddress(
      new URLSearchParams(
        'finance=committedAbovePlanned&planType=strategic&planCurrency=EUR&year=2026',
      ),
      standard2026(),
    );

    expect(state.filters).toMatchObject({
      finance: ['committedAbovePlanned'],
      planTypes: ['strategic'],
      planCurrencies: ['EUR'],
    });
    const wrong = fromAddress(
      new URLSearchParams('finance=lots&planType=other&planCurrency=euro'),
      standard2026(),
    );
    expect(chipsOf(wrong)).toEqual([]);
  });

  it('keeps Financials out of a saved view: the contract stores the other filters only', () => {
    const saved = savedStateOf(filtered);

    expect(saved.filters).not.toHaveProperty('finance');
    expect(saved.filters.planTypes).toEqual(['strategic', 'crossMarket']);
  });

  it('asks Compare with phased budget only in Budget & cost (016 FR-038)', () => {
    const compare: ViewState = { ...standard2026(), comparePhased: true };

    expect(toQueryParams({ ...compare, view: 'budget' })).toMatchObject({ comparePhased: true });
    expect(toQueryParams(compare)).not.toHaveProperty('comparePhased');
  });
});
