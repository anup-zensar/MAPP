import { screen, waitFor, within } from '@testing-library/react';
import { Outlet } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { HISTORY_ANSWERS } from '@/features/activity-log/testData';
import { financialsNotServed } from '@/features/plans/testData';
import { campaignForm } from '@/test/attributeForms';
import { renderApp } from '@/test/app';
import type { MockHandler } from '@/test/mockApi';
import { campaign, initiative, PEOPLE, planAnswer, planCampaigns } from './fixtures';

// The plan page (004) and an initiative's activities (006) are tested by their features; here
// they stand in, so those features never break these tests.
vi.mock('@/features/plans/PlanPage', () => ({
  PlanPage: () => (
    <>
      <h1>Plan page</h1>
      <Outlet />
    </>
  ),
}));
vi.mock('@/features/activities/InitiativeActivities', () => ({
  InitiativeActivities: ({ initiativeId }: { initiativeId: string }) => (
    <section aria-label="Activities">{`Activities of ${initiativeId}`}</section>
  ),
}));

const ANSWERS: Record<string, MockHandler> = {
  ...HISTORY_ANSWERS,
  ...financialsNotServed('P-1001', 'C-1001', 'I-1001'),
  'GET /api/v1/plans/P-1001/campaigns': { body: planCampaigns() },
  'GET /api/v1/plans/P-1001': { body: planAnswer() },
  'GET /api/v1/campaigns/C-1001': { body: campaign() },
  'GET /api/v1/initiatives/I-1001': { body: initiative() },
  'GET /api/v1/shell-attribute-forms/campaign': { body: campaignForm() },
  'GET /api/v1/shell-attribute-forms/initiative': {
    body: { ...campaignForm(), level: 'initiative' },
  },
  'GET /api/v1/people': { body: PEOPLE },
};

// Feature 005's addresses: side panels over the plan page (build guide §5).
describe('Campaign and initiative routes (feature 005)', () => {
  it.each([
    ['/plans/P-1001/campaigns/new', 'New campaign', 'New campaign'],
    ['/plans/P-1001/campaigns/C-1001', 'Helix launch', 'Helix launch'],
    ['/plans/P-1001/campaigns/C-1001/edit', 'Edit campaign', 'Helix launch'],
    ['/plans/P-1001/campaigns/C-1001/initiatives/new', 'New initiative', 'New initiative'],
    ['/plans/P-1001/initiatives/I-1001', 'Digital & social', 'Digital & social'],
    ['/plans/P-1001/initiatives/I-1001/edit', 'Edit initiative', 'Digital & social'],
  ])('opens %s as the "%s" panel over the plan page', async (path, panel, tab) => {
    renderApp(path, ANSWERS);

    expect(await screen.findByRole('dialog', { name: panel })).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 1, name: 'Plan page', hidden: true }),
    ).toBeInTheDocument();
    await waitFor(() => expect(document.title).toBe(`${tab} · MAPP`));
  });

  it('places feature 006’s activities in the initiative panel', async () => {
    renderApp('/plans/P-1001/initiatives/I-1001', ANSWERS);

    const panel = await screen.findByRole('dialog', { name: 'Digital & social' });
    expect(within(panel).getByRole('region', { name: 'Activities' })).toHaveTextContent(
      'Activities of I-1001',
    );
  });
});
