import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import {
  ALL_ROWS,
  FT_1006,
  FT_1010,
  FT_1011,
  transactionPage,
} from '@/testing/fixtures/financials';
import { listQueries, renderFinancialsPage } from '@/testing/fixtures/financials-page';

describe('FinancialsPage search, tabs and filters (015 US3, FR-021)', () => {
  it('searches after a pause and keeps the search in the address (US3 scenario 2)', async () => {
    const keynote = [FT_1011, FT_1006, FT_1010];
    const { api, router, user } = renderFinancialsPage('/financials', (request) => {
      const q = new URL(request.url).searchParams.get('q');
      return {
        body: q
          ? transactionPage({
              items: keynote,
              total: 3,
              counts: {
                all: 4,
                purchaseOrder: 1,
                invoice: 1,
                payment: 0,
                accrual: 2,
                notLinked: 0,
              },
            })
          : transactionPage(),
      };
    });
    await screen.findByRole('table', { name: 'Transactions' });

    await user.type(screen.getByRole('searchbox', { name: 'Search transactions' }), 'keynote');

    await waitFor(() => expect(router.state.location.search).toBe('?q=keynote'));
    expect(await screen.findByText('3 of 4 transactions')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Accruals (2)' })).toBeInTheDocument();
    expect(listQueries(api).at(-1)?.get('q')).toBe('keynote');
    // Typing replaces the address, so Back does not step through every letter.
    expect(router.state.historyAction).toBe('REPLACE');
  });

  it('switches tabs, sorts and pages through the address', async () => {
    const { api, router, user } = renderFinancialsPage('/financials', (request) => {
      const query = new URL(request.url).searchParams;
      return {
        body: transactionPage({
          total: 120,
          page: Number(query.get('page') ?? 1),
        }),
      };
    });
    await screen.findByRole('table', { name: 'Transactions' });

    await user.click(screen.getByRole('tab', { name: 'Not linked (1)' }));
    await waitFor(() => expect(router.state.location.search).toBe('?link=notLinked'));
    expect(listQueries(api).at(-1)?.get('link')).toBe('notLinked');
    expect(listQueries(api).at(-1)?.get('type')).toBe('all');

    await user.click(screen.getByRole('tab', { name: 'Invoices (3)' }));
    await waitFor(() => expect(router.state.location.search).toBe('?type=invoice'));

    await user.click(screen.getByRole('button', { name: 'Amount' }));
    await waitFor(() =>
      expect(router.state.location.search).toBe('?type=invoice&sort=amount%3Adesc'),
    );
    expect(screen.getByRole('columnheader', { name: /Amount/ })).toHaveAttribute(
      'aria-sort',
      'descending',
    );
    await user.click(screen.getByRole('button', { name: 'Amount' }));
    await waitFor(() => expect(router.state.location.search).toContain('sort=amount%3Aasc'));

    expect(await screen.findByText('1–50 of 120')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() => expect(router.state.location.search).toContain('page=2'));
    expect(await screen.findByText('51–100 of 120')).toBeInTheDocument();

    await user.selectOptions(screen.getByRole('combobox', { name: 'Per page' }), '100');
    await waitFor(() => expect(router.state.location.search).toContain('pageSize=100'));
    expect(router.state.location.search).not.toContain('page=2');
  });

  it('filters by year and plan, and shows the record filter of a card link as a chip (US3 scenario 5)', async () => {
    const { api, router, user } = renderFinancialsPage('/financials?recordId=A-1002', {
      body: transactionPage({
        items: ALL_ROWS.filter((row) => row.link.path.some((record) => record.id === 'A-1002')),
        total: 7,
        recordFilter: { type: 'activity', id: 'A-1002', name: 'Social always-on' },
      }),
    });

    const chip = await screen.findByRole('button', {
      name: 'Remove filter: Activity: Social always-on (A-1002)',
    });
    expect(screen.getByText('7 of 11 transactions')).toBeInTheDocument();
    expect(nth(listQueries(api), 0).get('recordId')).toBe('A-1002');

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Plan' }),
      'FY2026 Lubricants Europe',
    );
    await waitFor(() =>
      expect(router.state.location.search).toBe('?planId=P-1001&recordId=A-1002'),
    );
    expect(
      screen.getByRole('button', { name: 'Remove filter: Plan: FY2026 Lubricants Europe' }),
    ).toBeInTheDocument();

    await user.click(chip);
    await waitFor(() => expect(router.state.location.search).toBe('?planId=P-1001'));

    await user.selectOptions(screen.getByRole('combobox', { name: 'Year' }), '2027');
    await waitFor(() => expect(router.state.location.search).toBe('?year=2027'));
    expect(listQueries(api).at(-1)?.get('year')).toBe('2027');
  });

  it('says when nothing matches and clears the filters', async () => {
    const { router, user } = renderFinancialsPage('/financials?q=zzz&type=payment', (request) => {
      const q = new URL(request.url).searchParams.get('q');
      return {
        body: q ? transactionPage({ items: [], total: 0 }) : transactionPage(),
      };
    });

    expect(
      await screen.findByText(
        'No transactions match — clear a filter or search for something else',
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    await waitFor(() => expect(router.state.location.search).toBe('?type=payment'));
    expect(await screen.findByRole('table', { name: 'Transactions' })).toBeInTheDocument();
  });
});
