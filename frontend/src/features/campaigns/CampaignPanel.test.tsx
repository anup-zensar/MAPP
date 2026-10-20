import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { Outlet } from 'react-router';
import { describe, expect, it } from 'vitest';
import { SessionProvider } from '@/app/SessionProvider';
import { HISTORY_ANSWERS } from '@/features/activity-log/testData';
import { campaignForm } from '@/test/attributeForms';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderRoutes } from '@/test/render';
import { testUser } from '@/test/users';
import { CampaignPanel } from './CampaignPanel';
import {
  budgetAndCost,
  campaign,
  destinationList,
  PEOPLE,
  planAnswer,
  planDestination,
  usd,
} from './fixtures';

const DETAILS = '/plans/P-1001/campaigns/C-1001';

function renderPanel(route: string, answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    ...HISTORY_ANSWERS,
    'GET /api/campaigns/C-1001': { body: campaign() },
    'GET /api/plans/P-1001': { body: planAnswer() },
    'GET /api/shell-attribute-forms/campaign': { body: campaignForm() },
    'GET /api/people': { body: PEOPLE },
    ...answers,
  });
  const rendered = renderRoutes(
    [
      {
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/plans/:planId',
            element: <Outlet />,
            children: [
              { path: 'campaigns/:campaignId', element: <CampaignPanel mode="details" /> },
              { path: 'campaigns/:campaignId/edit', element: <CampaignPanel mode="edit" /> },
              { path: '*', element: <p>Another panel</p> },
              { index: true, element: <p>Plan page</p> },
            ],
          },
          { path: '*', element: <p>Another page</p> },
        ],
      },
    ],
    { route, user: testUser() },
  );
  return { api, ...rendered };
}

async function findDetails(name = 'Helix launch') {
  return screen.findByRole('dialog', { name });
}

/** The edit form once it is shown (the loading panel is another one). */
async function findForm() {
  await screen.findByLabelText('Campaign name');
  await screen.findByText(/Alex Morgan/);
  return screen.getByRole('dialog', { name: 'Edit campaign' });
}

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
    expect(document.title).toBe('Helix launch · NorthStar');

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
    const { user } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001': {
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
    });
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
    const { user, router } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001': {
        body: campaign({
          completeness: {
            missing: [{ field: 'startDate', label: 'Dates' }],
            itemsBelowNeedingAttention: 0,
          },
        }),
      },
    });
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: /1 field needed/ }));
    await user.click(await screen.findByRole('button', { name: 'Dates' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(`${DETAILS}/edit`));
    expect(router.state.location.search).toBe('?focus=startDate');
    await findForm();
    await waitFor(() => expect(screen.getByLabelText('Start date')).toHaveFocus());
  });

  it('says when the campaign has no initiatives and offers Add initiative (FR-016)', async () => {
    renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001': {
        body: campaign({
          initiatives: [],
          counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
        }),
      },
    });
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
    const asked = api.requests.find((request) => request.path === '/api/activity-log');
    expect(new URLSearchParams(asked?.search).get('recordId')).toBe('C-1001');
  });

  it('deletes from More, closes the panel and offers Undo (US5, FR-028, FR-029)', async () => {
    const { user, router, api } = renderPanel(DETAILS, {
      'DELETE /api/campaigns/C-1001': {
        body: {
          recycleBinEntry: {
            id: 'BIN-1001',
            recordType: 'campaign',
            recordTypeLabel: 'campaign',
            recordId: 'C-1001',
            recordName: 'Helix launch',
            path: [],
            parent: null,
            deletedAt: '2026-03-12T09:00:00Z',
            deletedBy: { id: 'U-1002', name: 'Priya Raman' },
            contents: [],
          },
        },
      },
    });
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete campaign 'Helix launch'?",
    });
    expect(dialog).toHaveTextContent('With it go 1 initiative.');
    await user.click(within(dialog).getByRole('button', { name: 'Delete campaign' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(api.bodiesOf('DELETE /api/campaigns/C-1001')).toHaveLength(1);
    expect(await findMessages()).toHaveTextContent(
      "Campaign 'Helix launch' moved to the recycle bin",
    );
  });

  it('moves from More and opens the plan it moved to, with the campaign highlighted (US6, FR-025)', async () => {
    const { user, router, api } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001/destinations': {
        body: destinationList(
          [planDestination()],
          'Campaigns move only between plans of the same year',
        ),
      },
      'POST /api/campaigns/C-1001/move': { body: campaign({ planId: 'P-1002', version: 4 }) },
    });
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Move to…' }));
    const dialog = await screen.findByRole('dialog', { name: "Move campaign 'Helix launch'" });
    await user.click(await within(dialog).findByRole('radio', { name: 'FY2026 Mobility China' }));
    await user.click(within(dialog).getByRole('button', { name: 'Move campaign' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002'));
    expect(router.state.location.state).toEqual({ highlight: 'C-1001' });
    expect(api.bodiesOf('POST /api/campaigns/C-1001/move')).toEqual([
      { planId: 'P-1002', version: 3 },
    ]);
    expect(await findMessages()).toHaveTextContent(
      "Campaign 'Helix launch' moved to 'FY2026 Mobility China'",
    );
  });

  it('copies from More (US6, FR-026)', async () => {
    const { user } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001/destinations': {
        body: destinationList(
          [
            planDestination({
              id: 'P-1001',
              name: 'FY2026 Lubricants Europe',
              current: true,
              suggestedName: 'Helix launch (copy)',
            }),
          ],
          'A copy can go into this plan or another plan of the same year',
        ),
      },
    });
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Copy' }));

    const dialog = await screen.findByRole('dialog', { name: "Copy campaign 'Helix launch'" });
    expect(await within(dialog).findByRole('textbox', { name: /Name of the copy/ })).toHaveValue(
      'Helix launch (copy)',
    );
    expect(dialog).toHaveTextContent('It copies 1 initiative.');
  });

  it('opens under the right plan when the campaign has moved (FR-017)', async () => {
    const { router } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001': { body: campaign({ planId: 'P-1002' }) },
      // The panel at its new address may ask for its plan before the test ends.
      'GET /api/plans/P-1002': { body: { ...planAnswer(), id: 'P-1002' } },
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1002/campaigns/C-1001'),
    );
  });

  it('says when the campaign no longer exists, with Back to the plan', async () => {
    const { user, router } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001': {
        status: 404,
        body: {
          code: 'not_found',
          message: 'This campaign no longer exists — someone may have deleted it.',
        },
      },
    });

    expect(
      await screen.findByText('This campaign no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Back to the plan' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });

  it('shows a loading state, then a failure with Try again', async () => {
    let calls = 0;
    const { user } = renderPanel(DETAILS, {
      'GET /api/campaigns/C-1001': () =>
        ++calls === 1
          ? { status: 500, body: { code: 'internal_error', message: 'Oops' } }
          : { body: campaign() },
    });

    expect(screen.getByRole('dialog', { name: 'Campaign' })).toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await findDetails()).toBeInTheDocument();
  });
});

describe('CampaignPanel — edit (feature 005, US3)', () => {
  it('saves the form with the version it started from and goes back to the details (US3#2, FR-018)', async () => {
    const { user, api, router } = renderPanel(`${DETAILS}/edit`, {
      'PATCH /api/campaigns/C-1001': { body: campaign({ budget: usd('450000.00'), version: 4 }) },
    });
    const form = await findForm();
    expect(within(form).getByLabelText('Campaign name')).toHaveValue('Helix launch');
    expect(within(form).getByLabelText('Campaign budget')).toHaveValue('400,000.00');
    // The plan's remaining budget counts this campaign's saved budget once.
    expect(within(form).getByText('$200,000.00')).toBeInTheDocument();

    const budget = within(form).getByLabelText('Campaign budget');
    await user.clear(budget);
    await user.type(budget, '450000');
    expect(within(form).getByText('$150,000.00')).toBeInTheDocument();
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(DETAILS));
    expect(api.bodiesOf('PATCH /api/campaigns/C-1001')).toEqual([
      {
        name: 'Helix launch',
        description: 'Launch of the new Helix range.',
        ownerId: 'U-1001',
        startDate: '2026-01-01',
        endDate: '2026-12-31',
        budget: { amount: '450000.00', currency: 'USD' },
        attributes: {},
        version: 3,
      },
    ]);
    await findDetails();
    // The details panel is open over the messages area, which stays readable for screen readers.
    await waitFor(() =>
      expect(screen.getByRole('region', { name: 'Messages (F8)', hidden: true })).toHaveTextContent(
        "Campaign 'Helix launch' saved",
      ),
    );
  });

  it('shows the FR-007 problem under the dates and saves nothing (US3#3, FR-007)', async () => {
    const problem =
      'Digital & social (01 Feb – 30 Nov 2026) would be outside these dates. Change its dates first, or choose dates that include it.';
    const { user, router } = renderPanel(`${DETAILS}/edit`, {
      'PATCH /api/campaigns/C-1001': {
        status: 400,
        body: {
          code: 'validation_failed',
          message: 'Some fields need your attention.',
          fields: [{ field: 'startDate', message: problem }],
        },
      },
    });
    const form = await findForm();

    fireEvent.change(within(form).getByLabelText('Start date'), {
      target: { value: '2026-03-01' },
    });
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('1 problem needs your attention')).toBeInTheDocument();
    expect(within(form).getByLabelText('Start date')).toHaveAccessibleDescription(problem);
    expect(router.state.location.pathname).toBe(`${DETAILS}/edit`);
  });

  it('keeps the typed values after a conflict and saves again with the new version (FR-018)', async () => {
    let saves = 0;
    let reads = 0;
    const { user, api } = renderPanel(`${DETAILS}/edit`, {
      'GET /api/campaigns/C-1001': () => ({ body: campaign({ version: ++reads > 1 ? 4 : 3 }) }),
      'PATCH /api/campaigns/C-1001': () =>
        ++saves === 1
          ? {
              status: 409,
              body: {
                code: 'version_conflict',
                message: 'Not saved — Daniel Okoro changed this campaign',
                conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-12T09:00:00Z' },
              },
            }
          : { body: campaign({ name: 'Helix relaunch', version: 5 }) },
    });
    const form = await findForm();

    const name = within(form).getByLabelText('Campaign name');
    await user.clear(name);
    await user.type(name, 'Helix relaunch');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(
      await within(form).findByText(/^Not saved — Daniel Okoro changed this campaign/),
    ).toBeInTheDocument();
    expect(name).toHaveValue('Helix relaunch');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(api.bodiesOf('PATCH /api/campaigns/C-1001')).toHaveLength(2));
    expect(
      api
        .bodiesOf('PATCH /api/campaigns/C-1001')
        .map((body) => (body as { version: number }).version),
    ).toEqual([3, 4]);
  });

  it('lists the values a removed cluster clears, before saving, and guards unsaved changes (US3#4, FR-009, FR-018)', async () => {
    const { user, api } = renderPanel(`${DETAILS}/edit`, {
      'GET /api/campaigns/C-1001': {
        body: campaign({
          attributes: { 'SA-CLUSTER': ['SO-RETAIL', 'SO-B2B'], 'SA-REGION': ['SO-CHINA', 'SO-EU'] },
        }),
      },
      'PATCH /api/campaigns/C-1001': { body: campaign() },
    });
    const form = await findForm();

    await user.click(within(form).getByRole('checkbox', { name: 'B2B' }));
    expect(within(form).getByText(/'B2B – EUROPE' will be cleared/)).toBeInTheDocument();

    // Closing asks first.
    await user.click(within(form).getByRole('button', { name: 'Close' }));
    const question = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(question).getByRole('button', { name: 'Keep editing' }));

    await user.click(within(form).getByRole('button', { name: 'Save' }));
    const clear = await screen.findByRole('alertdialog', {
      name: 'Clear values that no longer fit?',
    });
    expect(within(clear).getByText('Region: B2B – EUROPE')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/campaigns/C-1001')).toHaveLength(0);
    await user.click(within(clear).getByRole('button', { name: 'Clear and save' }));

    await waitFor(() => expect(api.bodiesOf('PATCH /api/campaigns/C-1001')).toHaveLength(1));
  });

  it('goes back to the details on Cancel when nothing changed', async () => {
    const { user, router } = renderPanel(`${DETAILS}/edit`);
    const form = await findForm();

    await user.click(within(form).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(DETAILS));
  });

  it('shows a failed save that is not about a field as a message and keeps the values', async () => {
    const { user, router } = renderPanel(`${DETAILS}/edit`, {
      'PATCH /api/campaigns/C-1001': {
        status: 500,
        body: { code: 'internal_error', message: 'Something went wrong. Please try again.' },
      },
    });
    const form = await findForm();

    await user.type(within(form).getByLabelText('Campaign name'), ' 2');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(screen.getByRole('region', { name: 'Messages (F8)', hidden: true })).toHaveTextContent(
        'Something went wrong. Please try again.',
      ),
    );
    expect(within(form).getByLabelText('Campaign name')).toHaveValue('Helix launch 2');
    expect(router.state.location.pathname).toBe(`${DETAILS}/edit`);
  });
});
