import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { highlightedRecordOf } from './schemas';

describe('the highlighted record of ?record= (011 FR-008)', () => {
  it('never fails on any address, and reads the record a link names', () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (query, record) => {
        expect(typeof highlightedRecordOf(new URLSearchParams(query))).toBe('string');
        expect(highlightedRecordOf(new URLSearchParams({ record }))).toBe(record);
      }),
    );
  });

  it('is empty without ?record=', () => {
    expect(highlightedRecordOf(new URLSearchParams('type=plan'))).toBe('');
  });
});
