import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { statusFilterOf } from './useListFilters';

describe('the status filter of a Settings list in the address (002 FR-002)', () => {
  it.each(['active', 'inactive', 'all'] as const)('reads %s', (status) => {
    expect(statusFilterOf(status)).toBe(status);
  });

  it('reads anything else, or nothing, as Active', () => {
    fc.assert(
      fc.property(fc.option(fc.string(), { nil: null }), (raw) => {
        const expected = raw === 'inactive' || raw === 'all' ? raw : 'active';
        expect(statusFilterOf(raw)).toBe(expected);
      }),
    );
  });
});
