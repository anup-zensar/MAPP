import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  activityRecord,
  figures,
  lineItemRecord,
  transaction,
  transactionPage,
  usd,
} from '@/testing/fixtures/line-items';
import { ActivityFinancials, LineItemFinancials } from './FinancialsSection';

function renderActivity(answer: MockHandler) {
  mockApi({ 'GET /api/v1/financials/records/A-1001': answer });
  return renderWithProviders(<ActivityFinancials activityId="A-1001" />, { user: testUser() });
}

function renderLineItem(
  answer: MockHandler,
  transactions: MockHandler = { body: transactionPage([transaction()]) },
) {
  mockApi({
    'GET /api/v1/financials/records/L-1001': answer,
    'GET /api/v1/financials/transactions': transactions,
  });
  return renderWithProviders(<LineItemFinancials lineItemId="L-1001" />, { user: testUser() });
}

describe('ActivityFinancials (015 US1, FR-011 – FR-015)', () => {
  it('shows the figures, the warning, the line items and the latest transactions, open', async () => {
    renderActivity({ body: activityRecord() });

    const card = await screen.findByRole('region', { name: 'Financials' });
    expect(card).toHaveAttribute('id', 'financials');
    const tiles = await within(card).findByRole('list', { name: 'Financial figures' });
    expect(tiles).toHaveTextContent('Committed (POs)');
    expect(tiles).toHaveTextContent('82,000.00 USD');
    expect(tiles).toHaveTextContent('68% of planned cost committed');
    expect(tiles).toHaveTextContent('Accrued (Mar 2026)');
    expect(tiles).toHaveTextContent('Not yet committed');
    expect(
      within(card).getByRole('link', { name: 'Committed above planned cost on 1 line item' }),
    ).toHaveAttribute('href', '/line-items/L-1002');

    const byLineItem = within(card).getByRole('table', { name: 'Financial figures by line item' });
    const rows = within(byLineItem).getAllByRole('row');
    expect(within(rows[1]).getByRole('link', { name: 'Agency fee' })).toHaveAttribute(
      'href',
      '/line-items/L-1001',
    );
    expect(within(rows[2]).getAllByRole('cell')[1]).toHaveTextContent('52,000.00 USD');
    expect(within(rows[2]).getAllByRole('cell')[1]).toHaveTextContent('Above planned cost');

    const latest = within(card).getByRole('table', {
      name: 'The latest transactions, newest first',
    });
    expect(within(latest).getAllByRole('row')).toHaveLength(4);
    expect(
      within(card).getByRole('link', { name: 'View all 3 transactions in Financials' }),
    ).toHaveAttribute('href', '/financials?recordId=A-1001');
    expect(within(card).getByText(/^GSAP data as of 12 Mar 2026/)).toHaveTextContent(
      'Demo data (simulated GSAP)',
    );
  });

  it('says when GSAP has sent nothing yet for the activity (FR-014)', async () => {
    renderActivity({
      body: activityRecord({
        figures: figures({ plannedCost: usd('120000.00'), notYetCommitted: usd('120000.00') }),
        breakdown: [],
        latest: [],
      }),
    });

    expect(
      await screen.findByText(
        "No purchase orders or invoices yet — they appear here once GSAP sends them for this activity's line items.",
      ),
    ).toBeVisible();
    expect(screen.getByText('120,000.00 USD')).toBeVisible();
    expect(screen.queryByRole('link', { name: /View all/ })).not.toBeInTheDocument();
  });

  it('says when no GSAP data was ever loaded', async () => {
    renderActivity({
      body: activityRecord({
        figures: figures(),
        breakdown: [],
        latest: [],
        source: { dataAsOf: null, feedName: null, fileName: null, accrualMonth: null },
      }),
    });

    expect(await screen.findByText('No GSAP data loaded yet.')).toBeVisible();
  });

  it('says in place when the person may not see financial data (015 T044)', async () => {
    renderActivity({
      status: 409,
      body: { error: { code: 'not_allowed', message: 'Not allowed', fields: [] } },
    });

    expect(
      await screen.findByText('You don’t have access to financial data — ask an administrator.'),
    ).toBeVisible();
  });
});

describe('LineItemFinancials (015 FR-017)', () => {
  it('shows the figures, every transaction and the SWBS code from GSAP data', async () => {
    renderLineItem({ body: lineItemRecord() });

    const section = await screen.findByRole('region', { name: 'Financials' });
    expect(await within(section).findByText('PO 4500001001 / 10')).toBeVisible();
    expect(within(section).getByText('Purchase order · Brightlight Agency')).toBeVisible();
    expect(within(section).getByText('DE03-M26-0003 (from GSAP data)')).toBeVisible();
  });

  it('says the SWBS code is filled by GSAP later when no transaction carries one', async () => {
    renderLineItem({ body: lineItemRecord({ swbsCodes: [] }) });

    expect(await screen.findByText('Filled by GSAP later')).toBeVisible();
  });
});
