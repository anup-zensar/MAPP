import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { standard2026 } from '@/testing/fixtures/all-work';
import type { ViewState } from './all-work-types';
import { fromAddress, hasAddressFilters } from './schemas';
import { sameState, toQueryParams } from './view-state';

const NAMES = [
  ...['year', 'yearFrom', 'yearTo', 'planType', 'planCurrency', 'finance', 'plan', 'planStatus'],
  ...['containerStatus', 'activityStatus', 'owner', 'team', 'businessUnit', 'activityType'],
  ...['strategic', 'showLevels', 'search', 'other'],
];
const VALUES = [
  ...['2026', '2027', '2031', '1999', '2026.0', '', 'operational', 'EUR', 'eur', 'published'],
  ...['committedAbovePlanned', 'declined', 'running', 'only', 'completeBranches', 'U-1003'],
];

const anyAddress = fc
  .array(fc.tuple(fc.constantFrom(...NAMES), fc.oneof(fc.constantFrom(...VALUES), fc.string())))
  .map((pairs) => new URLSearchParams(pairs));

/** The address of a state, written with the API's names as other pages link (D-122). */
function addressOf(state: ViewState): URLSearchParams {
  const params = new URLSearchParams();
  for (const [name, value] of Object.entries(toQueryParams(state))) {
    [value].flat().forEach((one) => params.append(name, String(one)));
  }
  return new URLSearchParams(params.toString());
}

describe('fromAddress (D-122)', () => {
  it('applies the API filter names of the address over the standard view', () => {
    const params = new URLSearchParams(
      'year=2027&owner=U-1003&activityStatus=declined&activityStatus=nonsense&strategic=only&search=helix',
    );

    expect(hasAddressFilters(params)).toBe(true);
    const state = fromAddress(params, standard2026());
    expect(state.year).toBe(2027);
    expect(state.filters.owners).toEqual([{ id: 'U-1003', label: 'U-1003' }]);
    expect(state.filters.activityStatuses).toEqual(['declined']);
    expect(state.filters.strategic).toBe('only');
    expect(state.search).toBe('helix');
    expect(state.columns).toEqual(standard2026().columns);
  });

  it('ignores a year outside 2000–2100 and addresses without filters', () => {
    expect(fromAddress(new URLSearchParams('year=1999'), standard2026()).year).toBe(2026);
    expect(hasAddressFilters(new URLSearchParams('tab=x'))).toBe(false);
  });

  it('reads any address without failing, into a state that survives write-then-read (FR-007)', () => {
    fc.assert(
      fc.property(anyAddress, (address) => {
        const state = fromAddress(address, standard2026());

        expect(sameState(fromAddress(addressOf(state), standard2026()), state)).toBe(true);
      }),
    );
  });
});
