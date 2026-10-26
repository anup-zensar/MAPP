import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { PAGE_SIZES, TABS, writeFilters } from './financials-links';
import type { FinancialFilters } from './financials-links';
import { readFilters } from './schemas';

const SORTS = ['postingDate', 'documentNumber', 'type', 'vendor', 'amount', 'linkedTo'] as const;

const filters: fc.Arbitrary<FinancialFilters> = fc.record({
  tab: fc.constantFrom(...TABS),
  q: fc.string({ maxLength: 200 }).filter((q) => q === '' || q.trim() !== ''),
  year: fc.option(fc.integer({ min: 2000, max: 2100 }), { nil: null }),
  planId: fc.option(
    fc.nat().map((n) => `P-${n}`),
    { nil: '' },
  ),
  recordId: fc.option(
    fc.tuple(fc.constantFrom('P', 'C', 'I', 'A', 'L'), fc.nat()).map(([l, n]) => `${l}-${n}`),
    { nil: '' },
  ),
  sort: fc.constantFrom(...SORTS),
  direction: fc.constantFrom('asc', 'desc'),
  page: fc.integer({ min: 1 }),
  pageSize: fc.constantFrom(...PAGE_SIZES),
});

describe('the Financials address (015 FR-021)', () => {
  it('never fails on any address, and always gives filters the page can use', () => {
    fc.assert(
      fc.property(fc.dictionary(fc.string(), fc.string()), (query) => {
        const read = readFilters(new URLSearchParams(query));
        expect(TABS).toContain(read.tab);
        expect(SORTS).toContain(read.sort);
        expect(PAGE_SIZES).toContain(read.pageSize);
        expect(Number.isInteger(read.page) && read.page >= 1).toBe(true);
      }),
    );
  });

  it('reads back every filter it wrote', () => {
    fc.assert(
      fc.property(filters, (written) => {
        expect(readFilters(writeFilters(written))).toEqual(written);
      }),
    );
  });
});
