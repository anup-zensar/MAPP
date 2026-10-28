import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DETAILS, findDetails, findForm, renderPanel } from '@/testing/fixtures/campaign-panel';
import { budgetAndCost, campaign, planAnswer, usd } from '@/testing/fixtures/campaigns';
import { reply } from '@/testing/replies';

describe('CampaignPanel — details (feature 005, US3)', () => {
  it('opens by its own address with the header, facts and sections (US3#1, FR-016, FR-017)', async () => {
    renderPanel(DETAILS);
    const panel = await findDetails();

    // The page under the panel has the breadcrumb; the panel names its plan as a value (C-02).
    expect(within(panel).queryByRole('navigation', { name: /breadcrumb/i })).toBeNull();
    expect(within(panel).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(within(panel).getByText('CAMPAIGN')).toBeInTheDocument();
    // The campaign's chips, and its initiative's approval progress in the list.
    expect(within(panel).getAllByText('No activities')).toHaveLength(2);
    expect(within(panel).getByText('All filled in')).toBeInTheDocument();
    expect(within(panel).getByText('C-1001')).toBeInTheDocument();
    expect(within(panel).getByText('Alex Morgan')).toBeInTheDocument();
    // The timing is a word after the dates, not a third chip.
    expect(within(panel).getByText('01 Jan – 31 Dec 2026 · Running')).toBeInTheDocument();
    expect(
      within(panel).getByText(
        'Created by Alex Morgan on 10 Feb 2026, last changed by Priya Raman on 05 Mar 2026',
      ),
    ).toBeInTheDocument();
    expect(within(panel).getByText('Launch of the new Helix range.')).toBeInTheDocument();
    expect(document.title).toBe('Helix launch · SP&A One');

    const money = within(panel).getByRole('region', { name: 'Budget and cost' });
    // Home's "Open budget" and "Show fields" lead here (feature 009).
    expect(money).toHaveAttribute('id', 'budget');
    expect(within(panel).getByText('All filled in').closest('#completeness')).not.toBeNull();
    // The budget, and what is left to plan with nothing planned yet.
    expect(within(money).getAllByText('$400,000.00')).toHaveLength(2);
    expect(within(money).getByText('Allocated to initiatives')).toBeInTheDocument();
    expect(within(money).getByText('$300,000.00')).toBeInTheDocument();
    expect(within(money).getByText('Remaining for initiatives')).toBeInTheDocument();
    expect(within(money).getByText('No planned cost below yet')).toBeInTheDocument();
    expect(within(money).getAllByText('Calculated')).toHaveLength(3);

    const goals = within(panel).getByRole('region', { name: /Goals/ });
    expect(within(goals).getAllByText('—')).toHaveLength(3);

    const initiatives = within(panel).getByRole('region', { name: 'Initiatives (1)' });
    expect(within(initiatives).getByRole('link', { name: 'Digital & social' })).toHaveAttribute(
      'href',
      '/plans/P-1001/initiatives/I-1001',
    );
    expect(within(panel).getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      `${DETAILS}/edit`,
    );
  });

  it('shows planned cost, budget used in red and over-allocation (FR-010, FR-011)', async () => {
    const { user } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          body: campaign({
            budget: usd('120000.00'),
            outsidePlanYear: true,
            budgetAndCost: budgetAndCost({
              allocatedToChildren: usd('150000.00'),
              remainingForChildren: usd('-30000.00'),
              overAllocatedBy: usd('30000.00'),
              plannedCost: usd('150000.00'),
              budgetUsedPercent: 125,
              overBudget: true,
              leftToPlan: usd('-30000.00'),
            }),
            approvalProgress: { state: 'inProgress', approved: 3, total: 5 },
            completeness: {
              missing: [
                { field: 'attributes.SA-CLUSTER', label: 'Cluster' },
                { field: 'attributes.SA-REGION', label: 'Region' },
              ],
              itemsBelowNeedingAttention: 1,
            },
          }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findDetails();

    expect(within(panel).getByText('Over-allocated by $30,000.00')).toBeInTheDocument();
    // Allocated to initiatives and the planned cost.
    expect(within(panel).getAllByText('$150,000.00')).toHaveLength(2);
    expect(within(panel).getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuetext',
      '125 %',
    );
    expect(within(panel).getByText('125%')).toBeInTheDocument();
    expect(within(panel).getByText('3 of 5 activities approved')).toBeInTheDocument();
    expect(
      within(panel).getByText("This campaign runs outside 2026, the plan's year"),
    ).toBeInTheDocument();
    // One completeness chip (design system §4.1); its list also names the records below.
    await user.click(within(panel).getByRole('button', { name: /2 fields needed/ }));
    expect(await screen.findByText('1 item below needs attention')).toBeInTheDocument();
  });

  it('links a needed field to the edit form with that field focused (FR-014)', async () => {
    const { user, router } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          body: campaign({
            completeness: {
              missing: [{ field: 'startDate', label: 'Dates' }],
              itemsBelowNeedingAttention: 0,
            },
          }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: /1 field needed/ }));
    await user.click(await screen.findByRole('button', { name: 'Dates' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(`${DETAILS}/edit`));
    expect(router.state.location.search).toBe('?focus=startDate');
    await findForm();
    await waitFor(() => expect(screen.getByLabelText('Start date')).toHaveFocus());
  });

  it('says when the campaign has no initiatives and offers Add initiative (FR-016)', async () => {
    renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          body: campaign({
            initiatives: [],
            counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
          }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findDetails();

    const initiatives = within(panel).getByRole('region', { name: 'Initiatives (0)' });
    expect(
      within(initiatives).getByText(
        'No initiatives yet — an initiative groups similar work inside a campaign.',
      ),
    ).toBeInTheDocument();
    expect(within(initiatives).getByRole('link', { name: 'Add initiative' })).toHaveAttribute(
      'href',
      `${DETAILS}/initiatives/new`,
    );
  });

  it('offers Add initiative and View history under More (US3#1, FR-016, FR-030)', async () => {
    const { user, router } = renderPanel(DETAILS);
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    const items = await screen.findAllByRole('menuitem');
    expect(items.map((item) => item.textContent)).toEqual([
      'Add initiative',
      'Copy',
      'Move to…',
      'View history',
      'Delete',
    ]);
    await user.click(screen.getByRole('menuitem', { name: 'View history' }));

    // The panel's own History part gets the focus (feature 011, FR-011); the panel stays open.
    const history = within(panel).getByRole('heading', { level: 3, name: 'History' });
    await waitFor(() => expect(history).toHaveFocus());
    expect(history.closest('#history-C-1001')).not.toBeNull();
    expect(router.state.location.pathname).toBe(DETAILS);
  });

  it('shows the campaign’s History with Include items inside (011 FR-011 – FR-013)', async () => {
    const { api } = renderPanel(DETAILS);
    const panel = await findDetails();

    const history = within(panel).getByRole('region', { name: 'History' });
    expect(within(history).getByText('Every change to this campaign, newest first')).toBeVisible();
    expect(
      within(history).getByRole('checkbox', { name: 'Include items inside' }),
    ).not.toBeChecked();
    expect(within(history).getByRole('link', { name: 'Open in activity log' })).toHaveAttribute(
      'href',
      '/activity-log?record=C-1001',
    );
    expect(await within(history).findByText('No changes recorded yet.')).toBeVisible();
    const asked = api.requests.find((request) => request.path === '/api/v1/activity-log');
    expect(new URLSearchParams(asked?.search).get('recordId')).toBe('C-1001');
  });

  it('opens under the right plan when the campaign has moved (FR-017)', async () => {
    const { router } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        { body: campaign({ planId: 'P-1002' }) },
        { campaignId: 'C-1001' },
      ),
      reply(
        'GET /api/v1/plans/{planId}',
        { body: { ...planAnswer(), id: 'P-1002' } },
        { planId: 'P-1002' },
      ),
    ]);

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1002/campaigns/C-1001'),
    );
  });

  it('says when the campaign no longer exists, with Back to the plan', async () => {
    const { user, router } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This campaign no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);

    expect(
      await screen.findByText('This campaign no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Back to the plan' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });

  it('shows a loading state, then a failure with Try again', async () => {
    let calls = 0;
    const { user } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        () =>
          ++calls === 1
            ? {
                status: 500,
                body: { error: { code: 'internal_error', message: 'Oops', fields: [] } },
              }
            : { body: campaign() },
        { campaignId: 'C-1001' },
      ),
    ]);

    expect(screen.getByRole('dialog', { name: 'Campaign' })).toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await findDetails()).toBeInTheDocument();
  });
});
