import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import {
  ALL_ROWS,
  FT_1006,
  FT_1010,
  FT_1011,
  transactionPage,
} from '@/testing/fixtures/financials';
import { Outlet } from 'react-router';
import { SessionProvider } from '@/session/SessionProvider';
import { renderRoutes } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FinancialsPage } from './FinancialsPage';
import { TransactionPanel } from './TransactionPanel';

const LIST = 'GET /api/v1/financials/transactions';

function renderPage(route = '/financials', list: MockHandler = { body: transactionPage() }) {
  // A 403 makes the session provider reload the person's access (014 FR-020).
  const api = mockApi({ [LIST]: list, 'GET /api/v1/me': { body: testUser() } });
  return { ...renderFinancials(route), api };
}

/** The query strings the page asked the list with, in order. */
function listQueries(api: ReturnType<typeof mockApi>): URLSearchParams[] {
  return api.requests
    .filter((request) => request.path === '/api/v1/financials/transactions')
    .map((request) => new URLSearchParams(request.search));
}

const rowOf = (name: string) => screen.getByRole('row', { name: new RegExp(name) });

/**
 * Renders the Financials page as the router does: the list at /financials and the details panel
 * as its child route. Other addresses show "Another page". For tests only.
 */
function renderFinancials(route = '/financials') {
  return renderRoutes(
    [
      {
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/financials',
            element: <FinancialsPage />,
            children: [{ path: 'transactions/:transactionId', element: <TransactionPanel /> }],
          },
          { path: '*', element: <p>Another page</p> },
        ],
      },
    ],
    { route, user: testUser() },
  );
}

describe('FinancialsPage (015 US3, FR-020 – FR-021)', () => {
  it('shows the header, the source, the tabs with counts, the totals and the rows newest first', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Financials' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Financials · MAPP');
    expect(
      screen.getByText('Purchase orders, invoices, payments and accruals from GSAP'),
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/^GSAP data as of .+ · Demo data \(simulated GSAP\)$/),
    ).toBeInTheDocument();

    const tabs = screen.getByRole('tablist', { name: 'Transaction type' });
    expect(
      within(tabs)
        .getAllByRole('tab')
        .map((tab) => tab.textContent),
    ).toEqual([
      'All (11)',
      'Purchase orders (4)',
      'Invoices (3)',
      'Payments (2)',
      'Accruals (2)',
      'Not linked (1)',
    ]);
    expect(within(tabs).getByRole('tab', { name: 'All (11)' })).toHaveAttribute(
      'aria-selected',
      'true',
    );

    // The four totals for the filters (US3 scenario 1).
    expect(screen.getByRole('button', { name: '$47.65K ($47,652.17)' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '$33.15K ($33,152.17)' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '$30.65K ($30,652.17)' })).toBeInTheDocument();
    expect(screen.getByText('Accrued (Sep 2026)')).toBeInTheDocument();
    expect(screen.getByText('11 of 11 transactions')).toBeInTheDocument();

    const rows = within(screen.getByRole('table', { name: 'Transactions' })).getAllByRole('row');
    expect(rows).toHaveLength(12);
    expect(rows[1]).toHaveTextContent('Accrual 9100004002');
    expect(rows[11]).toHaveTextContent('PO 4500001003 / 10');
    expect(screen.getByRole('columnheader', { name: /Posting date/ })).toHaveAttribute(
      'aria-sort',
      'descending',
    );
  });

  it('shows each transaction with its amount, vendor and links (FR-021)', async () => {
    renderPage();
    await screen.findByRole('table', { name: 'Transactions' });

    const stand = rowOf('PO 4500001003 / 10');
    expect(stand).toHaveTextContent('Purchase order');
    expect(stand).toHaveTextContent('08 Jan 2026');
    expect(stand).toHaveTextContent('Outdoor Frames Ltd');
    expect(stand).toHaveTextContent('9,800.00 EUR');
    expect(stand).toHaveTextContent('$10,652.17');
    expect(stand).toHaveTextContent('GB01-M26-0006');
    expect(within(stand).getByRole('link', { name: 'PO 4500001003 / 10' })).toHaveAttribute(
      'href',
      '/financials/transactions/FT-1003',
    );
    expect(within(stand).getByRole('link', { name: 'Stand build (L-1006)' })).toHaveAttribute(
      'href',
      '/line-items/L-1006',
    );
    expect(within(stand).getByRole('link', { name: 'Billboard winter wave' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1003',
    );

    // US3 scenario 4: not linked, with a chip.
    const brightside = rowOf('PO 4500001004 / 10');
    expect(brightside).toHaveTextContent('Not linked');
    expect(within(brightside).queryByRole('link', { name: /L-/ })).not.toBeInTheDocument();
    // A USD amount shows once.
    expect(within(rowOf('Invoice 5100002002')).getAllByText('$2,500.00')).toHaveLength(1);
  });

  it('searches after a pause and keeps the search in the address (US3 scenario 2)', async () => {
    const keynote = [FT_1011, FT_1006, FT_1010];
    const { api, router, user } = renderPage('/financials', (request) => {
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
    const { api, router, user } = renderPage('/financials', (request) => {
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
    const { api, router, user } = renderPage('/financials?recordId=A-1002', {
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
    expect(listQueries(api)[0].get('recordId')).toBe('A-1002');

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
    const { router, user } = renderPage('/financials?q=zzz&type=payment', (request) => {
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

  it('says what an empty tab means, and when no GSAP data was ever loaded', async () => {
    const { user } = renderPage('/financials?link=notLinked', (request) =>
      new URL(request.url).searchParams.get('link') === 'notLinked'
        ? { body: transactionPage({ items: [], total: 0 }) }
        : {
            body: transactionPage({
              items: [],
              total: 0,
              counts: {
                all: 0,
                purchaseOrder: 0,
                invoice: 0,
                payment: 0,
                accrual: 0,
                notLinked: 0,
              },
              source: { dataAsOf: null, feedName: null, fileName: null, accrualMonth: null },
            }),
          },
    );

    expect(
      await screen.findByText('Every transaction of 2026 is linked to a line item.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Clear filters' })).toBeNull();

    await user.click(screen.getByRole('tab', { name: 'All (11)' }));
    expect((await screen.findAllByText('No GSAP data loaded yet.')).length).toBe(2);
  });

  it('shows one card per transaction on a phone (US3 scenario 6)', async () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(max-width: 639px)',
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
    renderPage();

    const cards = await screen.findByRole('list', { name: 'Transactions' });
    expect(within(cards).getAllByRole('listitem')).toHaveLength(11);
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
    const stand = within(cards)
      .getAllByRole('listitem')
      .find((card) => card.textContent?.includes('PO 4500001003 / 10'));
    expect(stand).toHaveTextContent('9,800.00 EUR');
    expect(stand).toHaveTextContent('Stand build (L-1006)');
  });

  it('says so in place of the page when the person may not see financial data (FR-028)', async () => {
    renderPage('/financials', {
      status: 403,
      body: {
        error: {
          code: 'forbidden',
          message: "Seeing financial data isn't part of your role (Viewer).",
        },
      },
    });

    expect(
      await screen.findByText("You don't have access to financial data — ask an administrator."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
  });

  it('shows a failure with Try again', async () => {
    let calls = 0;
    const { user } = renderPage('/financials', () => {
      calls += 1;
      return calls === 1
        ? { status: 500, body: { error: { code: 'internal_error', message: 'Boom' } } }
        : { body: transactionPage() };
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('table', { name: 'Transactions' })).toBeInTheDocument();
  });
});
