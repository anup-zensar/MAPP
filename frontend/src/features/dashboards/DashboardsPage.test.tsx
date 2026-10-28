import { screen, within } from '@testing-library/react';
import { Outlet } from 'react-router';
import { describe, expect, it } from 'vitest';
import { SessionProvider } from '@/session/SessionProvider';
import { expectNoAxeViolations } from '@/testing/axe';
import { dashboard } from '@/testing/fixtures/dashboards';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import type { MockedApi } from '@/testing/mock-api';
import { renderRoutes } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { DashboardsPage } from './DashboardsPage';

type ExecutiveAnswer = Answer<'GET /api/v1/dashboards/executive'>;

function renderPage(
  answer: ExecutiveAnswer = { body: dashboard() },
  route = '/dashboards/executive',
) {
  const api = mockApi([
    reply('GET /api/v1/me', { body: testUser() }),
    reply('GET /api/v1/dashboards/executive', answer),
    reply('GET /api/v1/dashboards/regional', {
      body: dashboard({
        kind: 'regional',
        team: { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true },
      }),
    }),
  ]);
  const rendered = renderRoutes(
    [
      {
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          { path: '/dashboards/executive', element: <DashboardsPage kind="executive" /> },
          { path: '/dashboards/regional', element: <DashboardsPage kind="regional" /> },
        ],
      },
    ],
    { route, user: testUser() },
  );
  return { ...rendered, api };
}

function queries(api: MockedApi, path: string): URLSearchParams[] {
  return api.requests.filter((r) => r.path === path).map((r) => new URLSearchParams(r.search));
}

describe('DashboardsPage (036 US1)', () => {
  it('shows the figures, the tables, the chart and the panels (US1 scenarios 1, 6–11, FR-008 – FR-019)', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Executive dashboard' }),
    ).toBeInTheDocument();
    const summary = await screen.findByLabelText('Summary');
    expect(within(summary).getByText('$2,000,000.00')).toBeInTheDocument();
    expect(within(summary).getByText('Budget used 38%')).toBeInTheDocument();
    const markets = screen.getByRole('table', { name: /Plans and figures by market/ });
    expect(within(markets).getByRole('link', { name: 'Mobility China' })).toHaveAttribute(
      'href',
      '/dashboards/regional?team=T-1002',
    );
    expect(screen.getByRole('img', { name: /Planned cost by month/ })).toBeInTheDocument();
    expect(
      screen.getByText('Not yet spread (activities without dates): $5,000.00'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Fleet push' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/C-1009',
    );
    expect(screen.getByText('1.7 days')).toBeInTheDocument();
    expect(
      screen.getByRole('table', { name: 'Global budgets of the year and their allocations' }),
    ).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('switches the chart to its table of every month (FR-029)', async () => {
    const { user } = renderPage();

    await user.click(await screen.findByRole('button', { name: 'Table' }));

    const table = screen.getByRole('table', { name: /Planned cost spread by month/ });
    expect(within(table).getAllByRole('row')).toHaveLength(13);
    expect(within(table).getByRole('rowheader', { name: 'Mar 2026' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Table' })).toHaveAttribute('aria-pressed', 'true');
  });

  it('keeps the filters in the address and asks for them (FR-006, FR-007)', async () => {
    const { user, api, router } = renderPage();

    await user.selectOptions(await screen.findByLabelText('Currency'), 'EUR');
    await user.selectOptions(screen.getByLabelText('Plan type'), 'strategic');

    expect(router.state.location.search).toBe('?planType=strategic&currency=EUR');
    const last = queries(api, '/api/v1/dashboards/executive').at(-1);
    expect(last?.get('currency')).toBe('EUR');
    expect(last?.get('planType')).toBe('strategic');
  });

  it('falls back to the defaults for an address it cannot use (FR-007)', async () => {
    const { api } = renderPage(
      { body: dashboard() },
      '/dashboards/executive?year=nineteen&currency=eu&planType=x',
    );

    await screen.findByRole('heading', { level: 1, name: 'Executive dashboard' });
    expect(queries(api, '/api/v1/dashboards/executive')[0]?.toString()).toBe('');
  });

  it('says why there is nothing to show (US1 scenario 13)', async () => {
    renderPage({
      body: dashboard({ planCount: 0, filters: { ...dashboard().filters, year: 2030 } }),
    });

    expect(await screen.findByText('No plans to show for 2030')).toBeInTheDocument();
    expect(screen.getByLabelText('Year')).toBeInTheDocument();
  });

  it('says what financial figures it cannot show (US1 scenario 4, FR-003)', async () => {
    renderPage({ body: dashboard({ finance: { visible: false, planCount: 0 } }) });

    expect(
      await screen.findByText("You can't see financial data in these plans."),
    ).toBeInTheDocument();
  });

  it('shows its loading, error and no-access states (FR-028)', async () => {
    renderPage(() => pending());
    expect(await screen.findByLabelText('Loading the dashboard')).toBeInTheDocument();
  });

  it('offers a retry when the dashboard cannot be loaded (FR-028)', async () => {
    renderPage({ status: 500, body: apiError('internal_error', 'Something went wrong.') });
    expect(await screen.findByRole('button', { name: /Try again|Retry/ })).toBeInTheDocument();
  });

  it('shows no access to a person who may not view plans (FR-001)', async () => {
    renderPage({ status: 403, body: apiError('forbidden', "You don't have access to this page.") });
    expect(await screen.findByRole('heading', { name: 'No access' })).toBeInTheDocument();
  });
});

describe('DashboardsPage, Regional (036 US3)', () => {
  it("names its market and links a plan's campaigns (FR-020, FR-021)", async () => {
    renderPage(undefined, '/dashboards/regional?team=T-1001');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Regional dashboard' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('table', { name: /by market/ })).not.toBeInTheDocument();
    expect(
      await screen.findByRole('link', { name: 'Show campaigns of FY2026 Lubricants Europe' }),
    ).toHaveAttribute('href', '/dashboards/regional?team=T-1001&plan=P-1001');
    await expectNoAxeViolations();
  });
});
