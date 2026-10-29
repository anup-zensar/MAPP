import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { spaView } from '@/testing/fixtures/spa';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { SpaPage } from './SpaPage';

const VIEW = 'GET /api/v1/plans/{planId}/spa';

function renderPage(
  answer: Answer<typeof VIEW> = { body: spaView() },
  route = '/plans/P-1001/spa',
) {
  const api = mockApi([reply(VIEW, answer)]);
  const result = renderWithProviders(<SpaPage />, {
    user: testUser(),
    route,
    path: '/plans/:planId/spa',
  });
  return { ...result, api };
}

const headline = () => screen.getByRole('region', { name: 'Headline figures' });

describe('SpaPage states (013 US1)', () => {
  it('shows a placeholder while the view loads', () => {
    renderPage(() => pending());
    expect(screen.getByRole('heading', { level: 1, name: 'SP&A view' })).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Headline figures' })).not.toBeInTheDocument();
  });

  it('says so when the plan is not found (013 FR-012)', async () => {
    renderPage({ status: 404, body: apiError('not_found', 'This plan no longer exists.') });
    expect(
      await screen.findByText(/no longer exists, or you don’t have access to it/),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });

  it('shows the no-access page on a refusal', async () => {
    mockApi([
      reply(VIEW, { status: 403, body: apiError('forbidden', 'You may not see this plan.') }),
      reply('GET /api/v1/me', { body: testUser() }),
    ]);
    renderWithProviders(<SpaPage />, {
      user: testUser(),
      route: '/plans/P-1001/spa',
      path: '/plans/:planId/spa',
    });
    expect(await screen.findByText('You may not see this plan.')).toBeInTheDocument();
  });

  it('shows the problem the API answers, with Try again', async () => {
    renderPage({ status: 400, body: apiError('validation_failed', 'Choose a dimension.') });
    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says a scope without activities is empty, with the way to the plan tree', async () => {
    renderPage({ body: spaView({ emptyMessage: 'This campaign has no activities yet.' }) });
    expect(await screen.findByText('This campaign has no activities yet.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to the plan tree' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
  });
});

describe('SpaPage figures (013 FR-007, FR-009, FR-014 – FR-016)', () => {
  it('shows the headline figures as the API answers them, with the plan facts', async () => {
    renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    const figures = within(headline());
    expect(figures.getByRole('button', { name: /^\$400,000\.00/ })).toBeInTheDocument();
    expect(figures.getByText('$250,000.00')).toBeInTheDocument();
    expect(figures.getByText('37.5%')).toBeInTheDocument();
    expect(figures.getByText('−62.5%')).toBeInTheDocument();
    expect(screen.getByRole('group', { name: 'Plan facts' })).toHaveTextContent('BCPIR');
    expect(screen.getByText('Finance data as of 21 Oct 2026, 14:05')).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('marks overspend in red and in words', async () => {
    const view = spaView();
    renderPage({
      body: spaView({
        headline: {
          ...view.headline,
          remaining: { amount: '-50000.00', currency: 'USD' },
          utilisationPercent: '112.5',
          variancePercent: '12.5',
          overspend: true,
        },
      }),
    });
    await screen.findByRole('region', { name: 'Headline figures' });
    expect(within(headline()).getAllByText('Overspend')).toHaveLength(2);
    expect(within(headline()).getByText('+12.5%')).toBeInTheDocument();
  });

  it('lists the activities behind a figure, adding up to it', async () => {
    renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    await userEvent.click(
      within(headline()).getByRole('button', {
        name: /^\$200,000\.00, show what makes up Committed/,
      }),
    );
    const panel = await screen.findByRole('dialog', { name: 'What makes up Committed' });
    expect(within(panel).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1001',
    );
    expect(within(panel).getByRole('row', { name: /Total/ })).toHaveTextContent('$200,000.00');
  });

  it('shows "—" with the reason without access to financial data', async () => {
    const view = spaView();
    renderPage({
      body: spaView({
        financeAccess: false,
        financeNote: "You don't have access to financial data in this plan.",
        headline: {
          ...view.headline,
          committed: null,
          actualSpend: null,
          remaining: null,
          utilisationPercent: null,
          variancePercent: null,
        },
      }),
    });
    await screen.findByRole('region', { name: 'Headline figures' });
    expect(
      screen.getAllByText("You don't have access to financial data in this plan.").length,
    ).toBeGreaterThan(0);
    expect(within(headline()).getAllByText('—').length).toBeGreaterThanOrEqual(3);
  });
});

describe('SpaPage choices (013 FR-005, FR-017 – FR-019, FR-022)', () => {
  it('asks for the chosen campaign and keeps it in the address', async () => {
    const { api, router } = renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Scope' }), 'C-1001');
    await waitFor(() => expect(api.requests.at(-1)?.search).toBe('?campaignId=C-1001'));
    expect(router.state.location.search).toBe('?campaign=C-1001');
  });

  it('switches the periods to months, and asks for another dimension', async () => {
    const { api } = renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    await userEvent.click(screen.getByRole('button', { name: 'Months' }));
    expect(screen.getByRole('rowheader', { name: 'Aug 2026' })).toBeInTheDocument();
    await userEvent.selectOptions(
      screen.getByRole('combobox', { name: 'Group by' }),
      'activityType',
    );
    await waitFor(() => expect(api.requests.at(-1)?.search).toBe('?dimension=activityType'));
  });

  it('sorts the activities and opens one to its line items', async () => {
    renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    await userEvent.click(screen.getByRole('button', { name: 'Planned cost' }));
    const table = screen.getByRole('table', { name: 'Activities' });
    const names = () =>
      within(table)
        .getAllByRole('link')
        .map((link) => link.textContent);
    expect(names()).toEqual(['Launch film', 'Social always-on']);
    await userEvent.click(within(table).getByRole('button', { name: 'Planned cost' }));
    expect(names()).toEqual(['Social always-on', 'Launch film']);
    await userEvent.click(
      within(table).getByRole('button', { name: 'Show the line items of Launch film' }),
    );
    expect(within(table).getByRole('link', { name: 'Agency fee' })).toHaveAttribute(
      'href',
      '/line-items/L-1001',
    );
  });

  it('raises a purchase request for one of the activity’s line items, and explains when it cannot', async () => {
    const { router } = renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    await userEvent.click(screen.getByRole('button', { name: 'Actions for Social always-on' }));
    expect(screen.getByRole('menuitem', { name: /Raise purchase request/ })).toHaveTextContent(
      'Add a line item to this activity first.',
    );
    await userEvent.keyboard('{Escape}');
    await userEvent.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await userEvent.click(
      screen.getByRole('menuitem', { name: 'Raise purchase request for Film production' }),
    );
    await waitFor(() => expect(router.state.location.search).toBe('?lineItemId=L-1002'));
  });

  it('shows the budget and the purchase requests, the declined one not counted', async () => {
    renderPage();
    await screen.findByRole('region', { name: 'Headline figures' });
    expect(screen.getByText('−81.8%')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Agency retainer' })).toHaveAttribute(
      'href',
      '/purchase-requests/PR-1001',
    );
    expect(screen.getByText(/Not counted/)).toBeInTheDocument();
  });
});
