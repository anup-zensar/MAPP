import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { planForm } from '@/test/attributeForms';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { PlanPage } from './PlanPage';
import type { Plan } from './plansApi';
import { plan } from './testData';

// Feature 005 fills the plan tree; its own tests cover that.
vi.mock('@/features/campaigns/useCampaignTreeContent', () => ({
  useCampaignTreeContent: () => ({
    nodes: [],
    toolbarActions: null,
    planRowActions: [],
    emptyAction: null,
    footer: undefined,
    dialogs: null,
    isLoading: false,
    error: null,
    refetch: () => {},
  }),
}));

function renderPage(
  answer: MockHandler = { body: plan() },
  more: Record<string, MockHandler> = {},
) {
  const api = mockApi({
    'GET /api/plans/P-1001': answer,
    'GET /api/shell-attribute-forms/plan': { body: planForm() },
    ...more,
  });
  const result = renderWithProviders(<PlanPage />, {
    user: testUser(),
    route: '/plans/P-1001',
    path: '/plans/:planId',
  });
  return { ...result, api };
}

function withFigures(figures: Partial<Plan['figures']>, rest: Partial<Plan> = {}) {
  return { body: plan({ figures: { ...plan().figures, ...figures }, ...rest }) };
}

function card(label: string) {
  const summary = screen.getByRole('region', { name: 'Summary' });
  return within(summary).getByText(label).closest('div') as HTMLElement;
}

describe('PlanPage (US3, FR-011 – FR-017)', () => {
  it('shows the header facts and actions (US3#1, FR-011)', async () => {
    renderPage();

    const header = (
      await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' })
    ).closest('header') as HTMLElement;
    expect(document.title).toBe('FY2026 Lubricants Europe · NorthStar');
    expect(within(header).getByRole('link', { name: 'Plans' })).toHaveAttribute('href', '/plans');
    expect(within(header).getByText('PLAN')).toBeInTheDocument();
    expect(within(header).getByText('Draft')).toBeInTheDocument();
    expect(within(header).getByText('All filled in')).toBeInTheDocument();
    expect(within(header).getByText('AOP')).toBeInTheDocument();
    const facts = within(header).getByRole('list', { name: 'Plan facts' });
    expect(facts).toHaveTextContent('P-1001');
    expect(facts).toHaveTextContent('Owner: Priya Raman');
    expect(facts).toHaveTextContent('Year 2026');
    expect(facts).toHaveTextContent('Lubricants Europe');
    expect(facts).toHaveTextContent('Business unit: Lubricants');
    expect(
      within(header).getByText(
        'Created by Alex Morgan on 01 Mar 2026, last changed by Daniel Okoro on 10 Mar 2026',
      ),
    ).toBeInTheDocument();
    expect(within(header).getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/plans/P-1001/edit',
    );
  });

  it('shows the owner problem in words (FR-021)', async () => {
    renderPage({
      body: plan({
        ownerProblem:
          "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field.",
      }),
    });

    expect(
      await screen.findByText(
        "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field.",
      ),
    ).toBeInTheDocument();
  });

  it('shows a negative remaining budget in red with the amount (US3#2, FR-012)', async () => {
    renderPage(
      withFigures({
        campaignBudgets: { amount: '1300000.00', currency: 'USD' },
        remainingBudget: { amount: '-100000.00', currency: 'USD' },
        overAllocatedBy: { amount: '100000.00', currency: 'USD' },
      }),
    );

    await screen.findByRole('region', { name: 'Summary' });
    expect(
      within(card('Remaining budget')).getByText('Over-allocated by $100,000.00'),
    ).toBeInTheDocument();
    expect(
      within(card('Remaining budget')).getByRole('button', { name: /\(-\$100,000\.00\)$/ }),
    ).toBeInTheDocument();
  });

  it('shows the budget used with its bar, red above 100 % (US3#3)', async () => {
    renderPage(
      withFigures({
        plannedCost: { amount: '960000.00', currency: 'USD' },
        budgetUsedPercent: 80,
      }),
    );

    await screen.findByRole('region', { name: 'Summary' });
    expect(card('Budget used')).toHaveTextContent('80 %');
    expect(within(card('Budget used')).getByRole('progressbar')).toHaveAttribute(
      'aria-valuenow',
      '80',
    );
  });

  it('says why the budget cards are empty without a plan budget (US3#3)', async () => {
    renderPage(
      withFigures(
        { budget: null, remainingBudget: null, budgetUsedPercent: null },
        { budget: null },
      ),
    );

    await screen.findByRole('region', { name: 'Summary' });
    expect(card('Plan budget')).toHaveTextContent('—No plan budget set');
    expect(card('Remaining budget')).toHaveTextContent('—No plan budget set');
    expect(card('Budget used')).toHaveTextContent('—No plan budget set');
    expect(card('Planned cost')).toHaveTextContent('Nothing planned yet');
    expect(card('Activities approved')).toHaveTextContent('0 of 0No activities yet');
  });

  it('links the missing field and the items below that need attention (US3#4, FR-017)', async () => {
    const { user, router } = renderPage({
      body: plan({
        businessUnit: [],
        attributes: {},
        completeness: {
          missing: [{ field: 'attributes.SA-BU', label: 'Business unit' }],
          itemsBelow: [
            {
              record: { type: 'campaign', id: 'C-1001', name: 'Retail growth' },
              missing: [
                { field: 'attributes.SA-CLUSTER', label: 'Cluster' },
                { field: 'attributes.SA-REGION', label: 'Region' },
              ],
            },
          ],
        },
      }),
    });

    const header = (await screen.findByRole('heading', { level: 1 })).closest(
      'header',
    ) as HTMLElement;
    expect(within(header).getByText('No business unit yet')).toBeInTheDocument();
    await user.click(within(header).getByRole('button', { name: '1 item below needs attention' }));
    expect(
      await screen.findByRole('link', { name: "Campaign 'Retail growth' — 2 fields needed" }),
    ).toHaveAttribute('href', '/plans/P-1001/campaigns/C-1001');
    await user.keyboard('{Escape}');
    await user.click(within(header).getByRole('button', { name: '1 field needed' }));
    await user.click(await screen.findByRole('button', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
  });

  it('shows the goals beside what the activities plan (US3#5, FR-016)', async () => {
    renderPage({
      body: plan({
        goals: {
          ...plan().goals,
          activities: {
            inq: 150000,
            mql: 3000,
            revenue: { amount: '1500000.00', currency: 'USD' },
          },
          plannedPercent: 60,
        },
      }),
    });

    const goals = (await screen.findByRole('heading', { name: 'Goals' })).closest(
      'section',
    ) as HTMLElement;
    const rows = within(goals)
      .getAllByRole('row')
      .slice(1)
      .map((row) => row.textContent);
    expect(rows).toEqual([
      'INQ250,000150,000',
      'MQL5,000Calculated3,000',
      'Revenue$2,500,000.00Calculated$1,500,000.00',
    ]);
    expect(within(goals).getByRole('button', { name: 'About MQL goal' })).toBeInTheDocument();
    expect(within(goals).getByText('60 % of the INQ goal planned')).toBeInTheDocument();
  });

  it('says when no goal is set yet', async () => {
    renderPage({
      body: plan({
        goalAmount: null,
        goals: {
          ...plan().goals,
          plan: { inq: null, mql: null, deals: null, revenue: null, reasons: {} },
          plannedPercent: null,
        },
      }),
    });

    expect(
      await screen.findByText('No goal set yet — set a goal amount in Edit.'),
    ).toBeInTheDocument();
  });

  it('shows every detail read-only, with "—" and "(inactive)" (FR-015)', async () => {
    renderPage({
      body: plan({ description: '', model: { ...plan().model, active: false } }),
    });

    const details = (await screen.findByRole('heading', { name: 'Details' })).closest(
      'section',
    ) as HTMLElement;
    expect(within(details).getByText('Amounts in USD')).toBeInTheDocument();
    expect(within(details).getByText('Description').nextElementSibling).toHaveTextContent('—');
    expect(within(details).getByText('Model').nextElementSibling).toHaveTextContent(
      'Lubricants 1.0 (inactive)INQ→MQL 2.00 % · MQL→deal 5.00 % · average deal size 10,000.00 USD',
    );
    expect(
      within(details).getByText('Default line-item spread').nextElementSibling,
    ).toHaveTextContent('Evenly');
    expect(await within(details).findByText('Lubricants')).toBeInTheDocument();
  });

  it('offers Publish on a Draft plan and publishes it (US3#1, US5#1)', async () => {
    let published = false;
    const { user } = renderPage(
      () => ({ body: plan({ status: published ? 'published' : 'draft' }) }),
      {
        'POST /api/plans/P-1001/publish': () => {
          published = true;
          return { body: plan({ status: 'published' }) };
        },
      },
    );

    await user.click(await screen.findByRole('button', { name: 'Publish' }));

    expect(
      await screen.findByText("Plan 'FY2026 Lubricants Europe' published"),
    ).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Unpublish' })).toBeInTheDocument();
    expect(screen.getByRole('banner')).toHaveTextContent('Published');
  });

  it('goes to the Plans list after a delete, with Undo (US6#4, FR-030)', async () => {
    const { user, router } = renderPage(
      { body: plan() },
      {
        'DELETE /api/plans/P-1001': {
          body: {
            recycleBinEntry: {
              id: 'RB-1001',
              recordType: 'plan',
              recordTypeLabel: 'plan',
              recordId: 'P-1001',
              recordName: 'FY2026 Lubricants Europe',
              path: [],
              parent: null,
              deletedAt: '2026-03-12T09:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [],
            },
          },
        },
      },
    );

    await user.click(
      await screen.findByRole('button', { name: 'Actions for FY2026 Lubricants Europe' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(await screen.findByRole('button', { name: 'Delete plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    expect(
      await screen.findByText("Plan 'FY2026 Lubricants Europe' moved to the recycle bin"),
    ).toBeInTheDocument();
  });

  it('opens the plan’s history from More (FR-031)', async () => {
    const { user, router } = renderPage();

    await user.click(
      await screen.findByRole('button', { name: 'Actions for FY2026 Lubricants Europe' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/activity-log'));
    expect(router.state.location.search).toBe('?recordType=plan&recordId=P-1001');
  });

  it('shows grey shapes while loading, then an error with Try again', async () => {
    renderPage({ status: 500 });

    expect(screen.getByLabelText('Loading the plan')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says when the plan does not exist', async () => {
    renderPage({
      status: 404,
      body: {
        code: 'not_found',
        message: 'This plan no longer exists — someone may have deleted it.',
        fields: [],
      },
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Plan not found' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });
});
