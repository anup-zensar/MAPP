import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { transactionPage } from '@/testing/fixtures/financials';
import { renderFinancialsPage, transactionRow } from '@/testing/fixtures/financials-page';

describe('FinancialsPage (015 US3, FR-020 – FR-021)', () => {
  it('shows the header, the source, the tabs with counts, the totals and the rows newest first', async () => {
    renderFinancialsPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Financials' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Financials · SP&A One');
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
    renderFinancialsPage();
    await screen.findByRole('table', { name: 'Transactions' });

    const stand = transactionRow('PO 4500001003 / 10');
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

    // US3 scenario 4: not linked, with a chip and the reason.
    const brightside = transactionRow('PO 4500001004 / 10');
    expect(brightside).toHaveTextContent('Not linked');
    expect(brightside).toHaveTextContent(
      'No line item for HX key HX1099 — check the tracking key in GSAP.',
    );
    expect(within(brightside).queryByRole('link', { name: /L-/ })).not.toBeInTheDocument();
    // A USD amount shows once.
    expect(within(transactionRow('Invoice 5100002002')).getAllByText('$2,500.00')).toHaveLength(1);
  });

  it('says what an empty tab means, and when no GSAP data was ever loaded', async () => {
    const { user } = renderFinancialsPage('/financials?link=notLinked', (request) =>
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

  it('shows no Not linked tab or count when the answer has no notLinked count (038 FR-004, H-4)', async () => {
    const counts = { ...transactionPage().counts };
    delete counts.notLinked;
    renderFinancialsPage('/financials', { body: transactionPage({ counts }) });

    const tabs = await screen.findByRole('tablist', { name: 'Transaction type' });
    await within(tabs).findByRole('tab', { name: 'All (11)' });
    expect(
      within(tabs)
        .getAllByRole('tab')
        .map((tab) => tab.textContent),
    ).toEqual(['All (11)', 'Purchase orders (4)', 'Invoices (3)', 'Payments (2)', 'Accruals (2)']);
  });

  it('shows one card per transaction on a phone (US3 scenario 6)', async () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(max-width: 639px)',
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
    renderFinancialsPage();

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
    renderFinancialsPage('/financials', {
      status: 403,
      body: {
        error: {
          code: 'forbidden',
          message: "Seeing financial data isn't part of your role (Viewer).",
          fields: [],
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
    const { user } = renderFinancialsPage('/financials', () => {
      calls += 1;
      return calls === 1
        ? { status: 500, body: { error: { code: 'internal_error', message: 'Boom', fields: [] } } }
        : { body: transactionPage() };
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('table', { name: 'Transactions' })).toBeInTheDocument();
  });
});
