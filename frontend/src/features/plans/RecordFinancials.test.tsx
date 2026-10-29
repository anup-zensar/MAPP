import { screen, waitForElementToBeRemoved, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { mockApi } from '@/testing/mock-api';
import type { Answer } from '@/testing/replies';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { apiError, reply } from '@/testing/replies';
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

function renderSection(
  answer: Answer<'GET /api/v1/financials/records/{recordId}'>,
  variant: 'page' | 'panel' = 'page',
) {
  mockApi([
    reply('GET /api/v1/financials/records/{recordId}', answer, { recordId: 'P-1001' }),
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
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
    ).toHaveAttribute('href', '/financials?recordId=P-1001');
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

  it('marks a child with a line item committed above its planned cost (US2 scenario 3)', async () => {
    renderSection({
      body: view({
        record: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
        breakdownLevel: 'activity',
        breakdown: [
          {
            record: { type: 'activity', id: 'A-1003', name: 'Billboard winter wave' },
            figures: figures({ lineItemsAbovePlanned: 1 }),
          },
          {
            record: { type: 'activity', id: 'A-1004', name: 'Trade magazine ads' },
            figures: figures(),
          },
        ],
      }),
    });

    const table = await screen.findByRole('table', { name: 'By activity' });
    expect(within(table).getByRole('row', { name: /^Billboard winter wave/ })).toHaveTextContent(
      'Billboard winter wave · Committed above planned cost on 1 line item',
    );
    expect(within(table).getByRole('row', { name: /^Trade magazine ads/ })).not.toHaveTextContent(
      'Committed above',
    );
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

  it('says in place of the figures when the person may not see financial data (FR-028)', async () => {
    renderSection(
      {
        status: 403,
        body: apiError('forbidden', "Seeing financial data isn't part of your role (Viewer)."),
      },
      'panel',
    );

    expect(
      await screen.findByText('You don’t have access to financial data — ask an administrator.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3, name: 'Financials' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument();
  });

  it('shows nothing while the API does not serve 015 yet', async () => {
    renderSection({
      status: 404,
      body: { error: { code: 'not_found', message: 'Not found', fields: [] } },
    });

    await waitForElementToBeRemoved(() => screen.queryByRole('region', { name: 'Financials' }));
  });
});
