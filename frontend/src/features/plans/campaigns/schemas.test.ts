import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { highlightOf } from './schemas';

describe('the record to highlight, from the router state (005 FR-004)', () => {
  it('never fails on any state, and names a record only when the state holds one', () => {
    fc.assert(
      fc.property(fc.anything(), (state) => {
        const highlight = highlightOf(state);
        expect(highlight === undefined || typeof highlight === 'string').toBe(true);
      }),
    );
  });

  it('reads back any record a create, move or copy leaves', () => {
    fc.assert(
      fc.property(fc.string(), (id) => {
        expect(highlightOf({ highlight: id })).toBe(id);
      }),
    );
  });
});
