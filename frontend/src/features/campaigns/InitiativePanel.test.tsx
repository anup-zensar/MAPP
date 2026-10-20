import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { Outlet } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { SessionProvider } from '@/app/SessionProvider';
import { HISTORY_ANSWERS } from '@/features/activity-log/testData';
import { campaignForm } from '@/test/attributeForms';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderRoutes } from '@/test/render';
import { testUser } from '@/test/users';
import { initiative, PEOPLE, planAnswer, usd } from './fixtures';
import { InitiativePanel } from './InitiativePanel';

// Feature 006 fills the initiative's extension points; here they stand in.
vi.mock('@/features/activities/activityTreeRows', () => ({
  useActivityRows: () => new Map(),
  activityActionsForInitiative: (initiative: { id: string }, navigate: (to: string) => void) => [
    {
      label: 'Add activity',
      onSelect: () => navigate(`/plans/P-1001/initiatives/${initiative.id}/activities/new`),
    },
  ],
}));
vi.mock('@/features/activities/InitiativeActivities', () => ({
  InitiativeActivities: ({ initiativeId }: { initiativeId: string }) => (
    <section aria-label="Activities">{`Activities of ${initiativeId}`}</section>
  ),
}));

const DETAILS = '/plans/P-1001/initiatives/I-1001';

function renderPanel(route: string, answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    ...HISTORY_ANSWERS,
    'GET /api/initiatives/I-1001': { body: initiative() },
    'GET /api/plans/P-1001': { body: planAnswer() },
    'GET /api/shell-attribute-forms/initiative': {
      body: { ...campaignForm(), level: 'initiative' },
    },
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
              { path: 'initiatives/:initiativeId', element: <InitiativePanel mode="details" /> },
              { path: 'initiatives/:initiativeId/edit', element: <InitiativePanel mode="edit" /> },
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

async function findForm() {
  await screen.findByLabelText('Initiative name');
  await screen.findByText(/Priya Raman/);
  return screen.getByRole('dialog', { name: 'Edit initiative' });
}

describe('InitiativePanel (feature 005, US3)', () => {
  it('shows its campaign, the remaining campaign budget and what is below it (FR-016)', async () => {
    renderPanel(DETAILS, {
      'GET /api/initiatives/I-1001': {
        body: initiative({
          counts: { activities: 3, approvedActivities: 1, lineItems: 5 },
          parentRemainingBudget: usd('-50000.00'),
        }),
      },
    });
    const panel = await screen.findByRole('dialog', { name: 'Digital & social' });

    expect(within(panel).getByText('INITIATIVE')).toBeInTheDocument();
    // Its plan and campaign are values with links, not a second breadcrumb (UI audit C-02).
    expect(within(panel).queryByRole('navigation', { name: /breadcrumb/i })).toBeNull();
    expect(within(panel).getByRole('link', { name: 'Helix launch' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/C-1001',
    );
    expect(within(panel).getByText('01 Feb – 30 Nov 2026 · Running')).toBeInTheDocument();
    const money = within(panel).getByRole('region', { name: 'Budget and cost' });
    expect(within(money).getByText('Remaining campaign budget')).toBeInTheDocument();
    expect(within(money).getByText('Over-allocated by $50,000.00')).toBeInTheDocument();
    expect(
      within(panel).getByText('Activities: 3 (1 approved) · Line items: 5'),
    ).toBeInTheDocument();
    expect(within(panel).getByRole('region', { name: 'Activities' })).toHaveTextContent(
      'Activities of I-1001',
    );
  });

  it('says when nothing is below it yet', async () => {
    renderPanel(DETAILS);
    const panel = await screen.findByRole('dialog', { name: 'Digital & social' });

    expect(within(panel).getByText('No activities yet.')).toBeInTheDocument();
  });

  it('offers View history and feature 006’s actions under More', async () => {
    const { user, router } = renderPanel(DETAILS);
    const panel = await screen.findByRole('dialog', { name: 'Digital & social' });

    await user.click(within(panel).getByRole('button', { name: 'Actions for Digital & social' }));
    const items = await screen.findAllByRole('menuitem');
    expect(items.map((item) => item.textContent)).toEqual([
      'Copy',
      'Move to…',
      'View history',
      'Add activity',
      'Delete',
    ]);
    await user.click(screen.getByRole('menuitem', { name: 'Add activity' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe(
        '/plans/P-1001/initiatives/I-1001/activities/new',
      ),
    );
  });

  it('shows the initiative’s History, which View history brings up (011 FR-011, FR-012)', async () => {
    const { user } = renderPanel(DETAILS);
    const panel = await screen.findByRole('dialog', { name: 'Digital & social' });
    const history = within(panel).getByRole('region', { name: 'History' });
    expect(
      within(history).getByText('Every change to this initiative, newest first'),
    ).toBeVisible();
    expect(within(history).getByRole('checkbox', { name: 'Include items inside' })).toBeVisible();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Digital & social' }));
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() =>
      expect(within(history).getByRole('heading', { level: 3, name: 'History' })).toHaveFocus(),
    );
  });

  it('edits with the campaign read-only and its dates as the allowed range (FR-006, FR-018)', async () => {
    const { user, api, router } = renderPanel(`${DETAILS}/edit`, {
      'PATCH /api/initiatives/I-1001': { body: initiative({ version: 4 }) },
    });
    const form = await findForm();

    expect(within(form).getByLabelText('Campaign')).toHaveValue('Helix launch');
    expect(
      within(form).getByText("Must be inside the campaign's dates (01 Jan 2026 – 31 Dec 2026)"),
    ).toBeInTheDocument();
    expect(within(form).getByText('$100,000.00')).toBeInTheDocument();

    await user.type(within(form).getByLabelText('Description'), 'Paid social.');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(DETAILS));
    expect(api.bodiesOf('PATCH /api/initiatives/I-1001')).toEqual([
      expect.objectContaining({ description: 'Paid social.', version: 3 }),
    ]);
  });

  it('shows the API’s R3 and FR-007 messages under the dates (FR-006, FR-007)', async () => {
    const r3 = "Dates must be inside the campaign's dates (01 Jan 2026 – 31 Dec 2026).";
    const fr007 =
      'Launch webinar (01 Mar – 31 Mar 2026) would be outside these dates. Change its dates first, or choose dates that include it.';
    const { user } = renderPanel(`${DETAILS}/edit`, {
      'PATCH /api/initiatives/I-1001': {
        status: 400,
        body: {
          code: 'validation_failed',
          message: 'Some fields need your attention.',
          fields: [
            { field: 'startDate', message: fr007 },
            { field: 'endDate', message: r3 },
          ],
        },
      },
    });
    const form = await findForm();

    fireEvent.change(within(form).getByLabelText('Start date'), {
      target: { value: '2026-04-01' },
    });
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('2 problems need your attention')).toBeInTheDocument();
    expect(within(form).getByLabelText('Start date')).toHaveAccessibleDescription(fr007);
    expect(within(form).getByLabelText('End date')).toHaveAccessibleDescription(r3);
  });

  it('says when the initiative no longer exists', async () => {
    renderPanel(DETAILS, {
      'GET /api/initiatives/I-1001': {
        status: 404,
        body: {
          code: 'not_found',
          message: 'This initiative no longer exists — someone may have deleted it.',
        },
      },
    });

    expect(
      await screen.findByText('This initiative no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back to the plan' })).toBeInTheDocument();
  });
});
