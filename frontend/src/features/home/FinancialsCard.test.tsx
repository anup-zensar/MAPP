import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { FinancialsCard } from './FinancialsCard';
import type { FinancialOverview, HomeScope } from './homeApi';
import { emptyOverview, overviewOf, usd } from './testData';

function renderCard(answer: MockHandler = { body: overviewOf() }, scope: HomeScope = 'myTeams') {
  const api = mockApi({ 'GET /api/v1/financials/overview': answer });
  const rendered = renderWithProviders(<FinancialsCard scope={scope} />, {
    user: testUser(),
    path: '/',
  });
  return { ...rendered, api };
}

function figureOf(label: string) {
  const term = screen.getByText(label, { selector: 'dt' });
  return within(term.parentElement as HTMLElement);
}

function overviewAbove(): FinancialOverview {
  const overview = overviewOf();
  return {
    ...overview,
    figures: {
      ...overview.figures,
      plannedCost: usd('30000.00'),
      notYetCommitted: usd('-5652.17'),
      committedPercent: 119,
      committedAbovePlanned: true,
    },
  };
}

describe('FinancialsCard (015 US6, FR-025)', () => {
  it('shows committed against planned cost, invoiced, paid and accrued for the year (US6 scenario 1)', async () => {
    const { api } = renderCard();

    const card = await screen.findByRole('region', { name: 'Financials · 2026' });
    expect(
      api.requests.find((request) => request.path === '/api/v1/financials/overview')?.search,
    ).toBe('?scope=myTeams');
    expect(
      within(card).getByText(
        /^GSAP data as of 19 Oct 2026, \d\d:\d\d · Demo data \(simulated GSAP\)$/,
      ),
    ).toBeInTheDocument();

    const committed = figureOf('Committed');
    expect(committed.getByText('$35,652.17')).toBeInTheDocument();
    expect(committed.getByText('of $460,000.00 planned')).toBeInTheDocument();
    expect(
      committed.getByRole('progressbar', { name: 'Committed of planned cost' }),
    ).toHaveAttribute('aria-valuenow', '8');
    expect(committed.getByText('8%')).toBeInTheDocument();
    expect(figureOf('Invoiced').getByText('$33,152.17')).toBeInTheDocument();
    expect(figureOf('Paid').getByText('$30,652.17')).toBeInTheDocument();
    expect(figureOf('Accrued (Sep 2026)').getByText('$2,500.00')).toBeInTheDocument();
    // Each figure explains itself.
    expect(within(card).getByRole('button', { name: 'About Committed' })).toBeInTheDocument();
    expect(
      within(card).getByRole('button', { name: 'About Accrued (Sep 2026)' }),
    ).toBeInTheDocument();
  });

  it('links to All work filtered to the activities above planned cost, and to Not linked (US6 scenario 2)', async () => {
    renderCard();

    const above = await screen.findByRole('link', {
      name: '1 activity committed above planned cost',
    });
    expect(above).toHaveAttribute('href', '/all-work?finance=committedAbovePlanned&year=2026');
    expect(
      screen.getByRole('link', { name: '1 transaction not linked to a line item' }),
    ).toHaveAttribute('href', '/financials?link=notLinked');
    // Open Financials is a secondary button: Home has no primary action (design system §2.4 E).
    const open = screen.getByRole('link', { name: 'Open Financials' });
    expect(open).toHaveAttribute('href', '/financials');
    expect(open.className).not.toMatch(/primary/);
  });

  it('writes counts above one in the plural and leaves out the links with nothing to show', async () => {
    renderCard({ body: overviewOf({ activitiesAbovePlanned: 3, notLinked: 0 }) });

    expect(
      await screen.findByRole('link', { name: '3 activities committed above planned cost' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /not linked/ })).not.toBeInTheDocument();
  });

  it('says in words, not colour alone, when committed is above planned cost (015 FR-013)', async () => {
    renderCard({ body: overviewAbove() });

    const committed = await screen.findByText('Committed above planned cost by $5,652.17');
    expect(committed.className).toMatch(/problem/);
    expect(screen.getByText('119%').className).toMatch(/problem/);
  });

  it('says "No planned cost yet" without a planned cost', async () => {
    const overview = overviewOf();
    renderCard({
      body: {
        ...overview,
        figures: {
          ...overview.figures,
          plannedCost: null,
          notYetCommitted: null,
          committedPercent: null,
        },
      },
    });

    expect(await screen.findByText('No planned cost yet')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
  });

  it('says so when the plans of the scope have no transactions yet, with Open Financials (US6 scenario 3)', async () => {
    renderCard({ body: emptyOverview() });

    expect(
      await screen.findByText('No purchase orders or invoices for your plans yet.'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Committed', { selector: 'dt' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open Financials' })).toBeInTheDocument();
  });

  it('says when no GSAP data was ever loaded', async () => {
    const overview = emptyOverview();
    renderCard({
      body: {
        ...overview,
        source: { dataAsOf: null, feedName: null, fileName: null, accrualMonth: null },
      },
    });

    // Said once, by the empty state, not again in the source line.
    expect(await screen.findAllByText('No GSAP data loaded yet.')).toHaveLength(1);
  });

  it('fails on its own with a retry (009 FR-022)', async () => {
    renderCard({ status: 500, body: { error: { code: 'internal', message: 'Boom', fields: [] } } });

    expect(await screen.findByText("This section couldn't load")).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Financials' })).toBeInTheDocument();
  });

  it('asks for the Everything scope when Home shows everything', async () => {
    const { api } = renderCard({ body: overviewOf({ scope: 'everything' }) }, 'everything');

    await screen.findByRole('region', { name: 'Financials · 2026' });
    expect(
      api.requests.find((request) => request.path === '/api/v1/financials/overview')?.search,
    ).toBe('?scope=everything');
  });
});
