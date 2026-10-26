import { describe, expect, it } from 'vitest';
import {
  DEFAULT_FILTERS,
  financialsHref,
  toQuery,
  transactionHref,
  writeFilters,
} from './financials-links';
import { readFilters } from './schemas';

const read = (query: string) => readFilters(new URLSearchParams(query));

describe('financialsLinks (015 FR-021: the address is the state)', () => {
  it('reads every parameter of the address', () => {
    expect(
      read(
        'type=invoice&q=keynote&year=2027&planId=P-1001&recordId=A-1002&sort=amount:asc&page=2&pageSize=100',
      ),
    ).toEqual({
      tab: 'invoice',
      q: 'keynote',
      year: 2027,
      planId: 'P-1001',
      recordId: 'A-1002',
      sort: 'amount',
      direction: 'asc',
      page: 2,
      pageSize: 100,
    });
  });

  it('reads the Not linked tab from link=notLinked', () => {
    expect(read('link=notLinked').tab).toBe('notLinked');
    expect(read('type=payment&link=notLinked').tab).toBe('notLinked');
  });

  it('falls back to the defaults for anything it cannot use', () => {
    expect(
      read('type=bogus&year=99&planId=X-1&recordId=Z-9&sort=colour:up&page=0&pageSize=7'),
    ).toEqual(DEFAULT_FILTERS);
  });

  it('writes only what differs from the defaults', () => {
    expect(writeFilters(DEFAULT_FILTERS).toString()).toBe('');
    expect(
      writeFilters({ tab: 'notLinked', q: 'HX1099', sort: 'vendor', direction: 'asc' }).toString(),
    ).toBe('link=notLinked&q=HX1099&sort=vendor%3Aasc');
  });

  it('builds the links of the cards, Home and the panel', () => {
    expect(financialsHref()).toBe('/financials');
    expect(financialsHref({ recordId: 'A-1002' })).toBe('/financials?recordId=A-1002');
    expect(financialsHref({ tab: 'notLinked' })).toBe('/financials?link=notLinked');
    expect(transactionHref('FT-1003', '?q=keynote')).toBe(
      '/financials/transactions/FT-1003?q=keynote',
    );
  });

  it('asks the API with the tab as type and link', () => {
    expect(toQuery(DEFAULT_FILTERS)).toEqual({
      type: 'all',
      link: 'all',
      sort: 'postingDate:desc',
      page: 1,
      pageSize: 50,
    });
    expect(
      toQuery({ ...DEFAULT_FILTERS, tab: 'notLinked', q: ' 4500001003 ', year: 2026 }),
    ).toEqual({
      type: 'all',
      link: 'notLinked',
      q: '4500001003',
      year: 2026,
      sort: 'postingDate:desc',
      page: 1,
      pageSize: 50,
    });
    expect(toQuery({ ...DEFAULT_FILTERS, tab: 'purchaseOrder', planId: 'P-1001' })).toMatchObject({
      type: 'purchaseOrder',
      link: 'all',
      planId: 'P-1001',
    });
  });
});
