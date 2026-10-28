import { describe, expect, it } from 'vitest';
import { standard2026 } from '@/testing/fixtures/all-work';
import { BRANDS, COUNTRY, SPEND_TYPE } from '@/testing/fixtures/all-work-breakdown';
import type { BreakdownChoice, ViewState } from './all-work-types';
import {
  breakdownQuery,
  defaultGrouping,
  gridOfGroup,
  withBreakdownChoice,
} from './breakdown-state';
import { sameState, withoutRefused, yearRange } from './view-state';

const breakdownState = (breakdown: BreakdownChoice | null = null): ViewState => ({
  ...standard2026(),
  view: 'breakdown',
  breakdown,
});

describe('016 FR-040, FR-042: the first grouping', () => {
  it.each([
    ['Spend type when Settings has it', [BRANDS, SPEND_TYPE], 'attr:SA-1030'],
    ['the first Shell attribute otherwise', [COUNTRY, BRANDS], 'attr:SA-1013'],
    ['Plan without Shell attributes', [], 'plan'],
  ])('is %s', (_case, attributes, grouping) => {
    expect(defaultGrouping(attributes)).toBe(grouping);
  });

  it('is written into the state an export or a saved view takes, only for the Breakdown', () => {
    expect(withBreakdownChoice(breakdownState(), [SPEND_TYPE]).breakdown).toEqual({
      by: 'attr:SA-1030',
      thenBy: null,
    });
    expect(withBreakdownChoice(standard2026(), [SPEND_TYPE])).toEqual(standard2026());
  });
});

describe('016 FR-040: the Breakdown query', () => {
  it('takes the view’s filters and search with the groupings, without the Grid’s columns and sort', () => {
    const state: ViewState = {
      ...breakdownState(),
      search: 'helix',
      sort: { column: 'name', direction: 'asc' },
      filters: {
        ...standard2026().filters,
        attributes: [{ attributeId: 'SA-1020', values: ['SO-1201'] }],
        finance: ['invoiced'],
      },
    };

    expect(breakdownQuery(state, { by: 'attr:SA-1013', thenBy: 'activityType' })).toEqual({
      year: 2026,
      search: 'helix',
      attr: ['SA-1020:SO-1201'],
      by: 'attr:SA-1013',
      thenBy: 'activityType',
    });
  });

  it('goes back to the first grouping when the API refuses a grouping', () => {
    const state = breakdownState({ by: 'attr:SA-1999', thenBy: null });
    expect(withoutRefused(state, ['by'], standard2026()).breakdown).toBeNull();
  });

  it('counts the groupings in "Modified" only while the Breakdown is shown', () => {
    const bySpend = { by: 'attr:SA-1030', thenBy: null };
    expect(sameState(breakdownState(bySpend), breakdownState({ ...bySpend, thenBy: 'plan' }))).toBe(
      false,
    );
    expect(sameState({ ...breakdownState(bySpend), view: 'grid' }, standard2026())).toBe(true);
  });
});

describe('016 FR-040: a group opens the Grid filtered to it', () => {
  const group = (key: string, label = key) => ({ key, label });

  it.each([
    ['an option', 'attr:SA-1013', 'SO-1101', [{ attributeId: 'SA-1013', values: ['SO-1101'] }]],
    [
      'Not filled in',
      'attr:SA-1013',
      'notFilledIn',
      [{ attributeId: 'SA-1013', values: ['none'] }],
    ],
  ])('by a Shell attribute: %s', (_case, by, key, attributes) => {
    const next = gridOfGroup(breakdownState(), { by, thenBy: null }, [group(key)]);
    expect(next?.view).toBe('grid');
    expect(next?.filters.attributes).toEqual(attributes);
  });

  it.each([
    ['plan', 'P-1001', { plans: [{ id: 'P-1001', label: 'FY2026 Lubricants Europe' }] }],
    ['owner', 'U-1002', { owners: [{ id: 'U-1002', label: 'FY2026 Lubricants Europe' }] }],
    ['planType', 'strategic', { planTypes: ['strategic'] }],
    ['planCurrency', 'GBP', { planCurrencies: ['GBP'] }],
  ])('by %s', (by, key, filters) => {
    const next = gridOfGroup(breakdownState(), { by, thenBy: null }, [
      group(key, 'FY2026 Lubricants Europe'),
    ]);
    expect(next?.filters).toMatchObject(filters);
  });

  it('by year, as that one year', () => {
    const next = gridOfGroup(breakdownState(), { by: 'year', thenBy: null }, [group('2027')]);
    expect(next && yearRange(next)).toEqual({ from: 2027, to: 2027 });
  });

  it('adds the then-by group’s filter to its group’s', () => {
    const next = gridOfGroup(breakdownState(), { by: 'attr:SA-1013', thenBy: 'activityType' }, [
      group('SO-1101', 'Germany'),
      group('AT-1001', 'Event'),
    ]);
    expect(next?.filters).toMatchObject({
      attributes: [{ attributeId: 'SA-1013', values: ['SO-1101'] }],
      activityTypes: [{ id: 'AT-1001', label: 'Event' }],
    });
  });

  it.each([
    ['Not on a line item', 'attr:SA-1030', 'notOnLineItem'],
    ['No owner', 'owner', 'noOwner'],
    ['a built-in grouping’s Not filled in', 'businessUnit', 'notFilledIn'],
    ['an unknown plan type', 'planType', 'other'],
  ])('cannot open %s', (_case, by, key) => {
    expect(gridOfGroup(breakdownState(), { by, thenBy: null }, [group(key)])).toBeNull();
  });
});
