import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { focusFieldOf, gridPeriodOf, highlightedLineItemOf } from './schemas';

describe('the line items’ outside data (007 FR-010, FR-020, FR-028)', () => {
  it('reads any stored grid period as Monthly or Quarterly, and keeps a stored choice', () => {
    fc.assert(
      fc.property(fc.anything(), (stored) => {
        expect(['monthly', 'quarterly']).toContain(gridPeriodOf(stored));
      }),
    );
    fc.assert(
      fc.property(fc.constantFrom('monthly', 'quarterly'), (period) => {
        expect(gridPeriodOf(period)).toBe(period);
      }),
    );
  });

  it('reads the highlighted line item from any router state, and the one a create sends', () => {
    fc.assert(
      fc.property(fc.anything(), (state) => {
        expect(['string', 'undefined']).toContain(typeof highlightedLineItemOf(state));
      }),
    );
    fc.assert(
      fc.property(fc.string(), (id) => {
        expect(highlightedLineItemOf({ highlightLineItemId: id })).toBe(id);
      }),
    );
  });

  it('reads ?field= from any address, and the field a completeness link names', () => {
    fc.assert(
      fc.property(fc.string(), (query) => {
        const field = focusFieldOf(new URLSearchParams(query));
        expect(field === null || typeof field === 'string').toBe(true);
      }),
    );
    fc.assert(
      fc.property(fc.string(), (field) => {
        expect(focusFieldOf(new URLSearchParams({ field }))).toBe(field);
      }),
    );
  });
});
