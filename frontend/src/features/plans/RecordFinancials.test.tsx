import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/schema';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { RecordFinancials } from './RecordFinancials';

type FinancialRecordView = components['schemas']['FinancialRecordView'];
type FinancialFigures = components['schemas']['FinancialFigures'];

const usd = (amount: string) => ({ amount, currency: 'USD' });

function figures(overrides: Partial<FinancialFigures> = {}): FinancialFigures {
  return {
    plannedCost: usd('460000.00'),
    committed: usd('230000.00'),
    invoiced: usd('120000.00'),
    paid: usd('80000.00'),
    accrued: usd('15000.00'),
    notYetCommitted: usd('230000.00'),
    committedPercent: 50,
    committedAbovePlanned: false,
    lineItemsAbovePlanned: 0,
    firstLineItemAbovePlanned: null,
    transactionCount: 12,
    ...overrides,
  };
}

function view(overrides: Partial<FinancialRecordView> = {}): FinancialRecordView {
  return {
    record: { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
    planId: 'P-1001',
    path: [],
    figures: figures(),
    breakdownLevel: 'campaign',
    breakdown: [
      {
        record: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
        figures: figures({ committed: usd('150000.00') }),
      },
    ],
    latest: [],
    swbsCodes: [],
    source: {
      dataAsOf: '2026-10-19T06:00:00Z',
      feedName: 'GSAP demo feed',
      fileName: null,
      accrualMonth: '2026-09',
    },
    ...overrides,
  };
}

function renderSection(answer: MockHandler, variant: 'page' | 'panel' = 'page') {
  mockApi({ 'GET /api/financials/records/P-1001': answer });
  return renderWithProviders(
    <RecordFinancials recordId="P-1001" planId="P-1001" variant={variant} />,
    { user: testUser(), route: '/plans/P-1001', path: '/plans/:planId' },
  );
}

describe('RecordFinancials (015 FR-018, FR-019)', () => {
  it('shows the figures, the campaigns and View all, open', async () => {
    renderSection({ body: view() });

    const section = await screen.findByRole('region', { name: 'Financials' });
    // Committed and Not yet committed (half of the planned cost each).
    expect(await within(section).findAllByText('$230,000.00')).toHaveLength(2);
    expect(within(section).getByText('50% of planned cost committed')).toBeInTheDocument();
    expect(within(section).getByText('Accrued (Sep 2026)')).toBeInTheDocument();
    const table = within(section).getByRole('table', { name: 'By campaign' });
    expect(within(table).getByRole('link', { name: 'Helix launch' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/C-1001',
    );
    expect(
      within(section).getByRole('link', { name: 'View all 12 transactions in Financials' }),
    ).toHaveAttribute('href', '/financials?record=P-1001');
    expect(within(section).getByText(/GSAP data as of .* · GSAP demo feed/)).toBeInTheDocument();
  });

  it('says in red words when committed is above the planned cost (FR-013)', async () => {
    renderSection({
      body: view({
        figures: figures({
          committed: usd('500000.00'),
          notYetCommitted: usd('-40000.00'),
          committedPercent: 109,
          committedAbovePlanned: true,
          lineItemsAbovePlanned: 2,
          firstLineItemAbovePlanned: { type: 'lineItem', id: 'L-1003', name: 'Stand build' },
        }),
      }),
    });

    expect(
      await screen.findByText('Committed above planned cost by $40,000.00'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'Committed above planned cost on 2 line items' }),
    ).toHaveAttribute('href', '/line-items/L-1003');
  });

  it('says when there are no transactions yet (FR-014)', async () => {
    renderSection(
      {
        body: view({
          record: { type: 'campaign', id: 'P-1001', name: 'Helix launch' },
          figures: figures({ transactionCount: 0, committed: usd('0.00') }),
          breakdown: [],
        }),
      },
      'panel',
    );

    expect(
      await screen.findByText(
        'No purchase orders or invoices yet — they appear here once GSAP sends them for this campaign’s line items.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3, name: 'Financials' })).toBeInTheDocument();
  });

  it('shows nothing while the API does not serve 015 yet', async () => {
    renderSection({ status: 404, body: { code: 'not_found', message: 'Not found', fields: [] } });

    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(screen.queryByRole('region', { name: 'Financials' })).not.toBeInTheDocument();
  });
});
