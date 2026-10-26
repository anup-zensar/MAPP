import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { planForm } from '@/testing/fixtures/attribute-forms';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { PlanPage } from './PlanPage';
import type { Plan } from './plansApi';
import { financialsNotServed, plan } from '@/testing/fixtures/plans';

// Feature 005 fills the plan tree; its own tests cover that.
vi.mock('./campaigns/useCampaignTreeContent', () => ({
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
  route = '/plans/P-1001',
) {
  const api = mockApi({
    ...HISTORY_ANSWERS,
    ...financialsNotServed('P-1001', 'P-1004'),
    'GET /api/v1/plans/P-1001': answer,
    'GET /api/v1/shell-attribute-forms/plan': { body: planForm() },
    ...more,
  });
  const result = renderWithProviders(<PlanPage />, {
    user: testUser(),
    route,
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
    expect(document.title).toBe('FY2026 Lubricants Europe · MAPP');
    expect(within(header).getByRole('link', { name: 'Plans' })).toHaveAttribute('href', '/plans');
    expect(within(header).getByText('PLAN')).toBeInTheDocument();
    expect(within(header).getByText('Draft')).toBeInTheDocument();
    expect(within(header).getByText('All filled in')).toBeInTheDocument();
    // The meta row of the one page header (design system §2.3).
    const facts = within(header).getByRole('group', { name: 'Plan facts' });
    expect(facts).toHaveTextContent('P-1001');
    expect(facts).toHaveTextContent('Priya Raman');
    expect(facts).toHaveTextContent('01 Jan – 31 Dec 2026');
    expect(facts).toHaveTextContent('AOP plan');
    expect(facts).toHaveTextContent('Lubricants Europe');
    expect(facts).toHaveTextContent('Business unit: Lubricants');
    expect(within(header).getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/plans/P-1001/edit',
    );
    expect(within(header).getByRole('button', { name: 'More' })).toBeInTheDocument();
    // The tabs of the record template, the tree first (design system §2.4 B).
    expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
      'Plan tree',
      'Details',
      'History',
    ]);
    expect(screen.getByRole('tab', { name: 'Plan tree' })).toHaveAttribute('aria-selected', 'true');
  });

  it('keeps the chosen tab in the address (design system §2.4 B)', async () => {
    const { user, router } = renderPage();

    await user.click(await screen.findByRole('tab', { name: 'Details' }));

    expect(router.state.location.search).toBe('?tab=details');
    expect(
      await screen.findByText(
        'Created by Alex Morgan on 01 Mar 2026, last changed by Daniel Okoro on 10 Mar 2026',
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'Plan tree' }));
    expect(router.state.location.search).toBe('');
  });

  it('has the sections Home links to: #budget and #completeness (feature 009, FR-012)', async () => {
    renderPage();

    await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' });
    expect(screen.getByRole('region', { name: 'Summary' })).toHaveAttribute('id', 'budget');
    const completeness = document.getElementById('completeness');
    expect(completeness).not.toBeNull();
    expect(completeness?.closest('header')).not.toBeNull();
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
    // What remains for campaigns is part of the plan budget card (four cards, §2.4 B).
    expect(
      within(card('Plan budget')).getByText('Over-allocated by $100,000.00'),
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
    expect(card('Budget used')).toHaveTextContent('80%');
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
    expect(card('Budget used')).toHaveTextContent('—No plan budget set');
    expect(card('Planned cost')).toHaveTextContent('Nothing planned yet');
    // "Nothing" is "—" with the reason under it (functional review FR·X-08).
    expect(card('Activities approved')).toHaveTextContent('—No activities yet');
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
    // One completeness chip: its list holds the missing field and the records below (§4.1).
    await user.click(within(header).getByRole('button', { name: '1 field needed' }));
    expect(
      await screen.findByRole('link', { name: "Campaign 'Retail growth' — 2 fields needed" }),
    ).toHaveAttribute('href', '/plans/P-1001/campaigns/C-1001');
    await user.click(screen.getByRole('button', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
  });

  it('shows the goals beside what the activities plan (US3#5, FR-016)', async () => {
    renderPage(
      {
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
      },
      {},
      '/plans/P-1001?tab=details',
    );

    const goals = (await screen.findByRole('heading', { name: 'Goals' })).closest(
      'section',
    ) as HTMLElement;
    const rows = within(goals)
      .getAllByRole('row')
      .slice(1)
      .map((row) => row.textContent);
    expect(rows).toEqual([
      'INQ250,000150,000',
      'MQLCalculated5,0003,000',
      'RevenueCalculated$2,500,000.00$1,500,000.00',
    ]);
    expect(within(goals).getByRole('button', { name: 'About MQL goal' })).toBeInTheDocument();
    expect(within(goals).getByText('60% of the INQ goal planned')).toBeInTheDocument();
  });

  it('says when no goal is set yet', async () => {
    renderPage(
      {
        body: plan({
          goalAmount: null,
          goals: {
            ...plan().goals,
            plan: { inq: null, mql: null, deals: null, revenue: null, reasons: {} },
            plannedPercent: null,
          },
        }),
      },
      {},
      '/plans/P-1001?tab=details',
    );

    expect(
      await screen.findByText('No goal set yet — set a goal amount in Edit.'),
    ).toBeInTheDocument();
  });

  it('shows every detail read-only, with "—" and "(inactive)" (FR-015)', async () => {
    renderPage(
      { body: plan({ description: '', model: { ...plan().model, active: false } }) },
      {},
      '/plans/P-1001?tab=details',
    );

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
        'POST /api/v1/plans/P-1001/publish': () => {
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
        'DELETE /api/v1/plans/P-1001': {
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

    await user.click(await screen.findByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(await screen.findByRole('button', { name: 'Delete plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    expect(
      await screen.findByText("Plan 'FY2026 Lubricants Europe' moved to the recycle bin"),
    ).toBeInTheDocument();
  });

  it('copies the plan from More and opens the copy (US6#1–2, FR-026, FR-027)', async () => {
    const copy = plan({ id: 'P-1004', name: 'FY2026 Lubricants Europe (copy)', version: 1 });
    const { user, router } = renderPage(
      { body: plan() },
      {
        'GET /api/v1/plans/P-1001/copy-options': {
          body: {
            name: 'FY2026 Lubricants Europe (copy)',
            sameYear: 2026,
            nextYear: { year: 2027, available: true, reason: '' },
            contents: {
              campaigns: 0,
              initiatives: 0,
              activities: 0,
              approvedActivities: 0,
              lineItems: 0,
            },
          },
        },
        'POST /api/v1/plans/P-1001/copy': { status: 201, body: { plan: copy, notes: [] } },
        'GET /api/v1/plans/P-1004': { body: copy },
      },
    );

    await user.click(await screen.findByRole('button', { name: 'More' }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'Copy',
      'View history',
      'Delete',
    ]);
    await user.click(screen.getByRole('menuitem', { name: 'Copy' }));
    await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.click(screen.getByRole('button', { name: 'Copy plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1004'));
    expect(
      await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe (copy)' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Plan 'FY2026 Lubricants Europe (copy)' created as a copy of 'FY2026 Lubricants Europe'",
      ),
    ).toBeInTheDocument();
  });

  it('opens the History tab from More and focuses it (FR-031, 011 FR-011)', async () => {
    const { user, router } = renderPage();

    await user.click(await screen.findByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 2, name: 'History' })).toHaveFocus(),
    );
    expect(router.state.location.pathname).toBe('/plans/P-1001');
    expect(router.state.location.search).toBe('?tab=history');
  });

  it('shows the plan’s History in its tab, with Include items inside (011 FR-011 – FR-013)', async () => {
    const { api } = renderPage({ body: plan() }, {}, '/plans/P-1001?tab=history');

    const history = await screen.findByRole('region', { name: 'History' });
    expect(within(history).getByText('Every change to this plan, newest first')).toBeVisible();
    expect(within(history).getByRole('checkbox', { name: 'Include items inside' })).toBeVisible();
    expect(within(history).getByRole('link', { name: 'Open in activity log' })).toHaveAttribute(
      'href',
      '/activity-log?record=P-1001',
    );
    expect(await within(history).findByText('No changes recorded yet.')).toBeVisible();
    const asked = api.requests.find((request) => request.path === '/api/v1/activity-log');
    expect(new URLSearchParams(asked?.search).get('recordId')).toBe('P-1001');
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
        error: {
          code: 'not_found',
          message: 'This plan no longer exists — someone may have deleted it.',
          fields: [],
        },
      },
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Plan not found' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });
});
