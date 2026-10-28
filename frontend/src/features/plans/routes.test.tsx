import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { planCampaigns } from '@/testing/fixtures/campaigns';
import { NOTHING_INSIDE } from '@/testing/fixtures/plan-page';
import {
  financialsNotServed,
  FORM_ANSWERS,
  plan,
  planDefaults,
  seedList,
} from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';

/** What the four pages ask the API. */
const ANSWERS = [
  ...FORM_ANSWERS,
  ...HISTORY_ANSWERS,
  ...NOTHING_INSIDE,
  ...financialsNotServed('P-1001'),
  reply('GET /api/v1/plans', { body: seedList() }),
  reply('GET /api/v1/plan-defaults', { body: planDefaults() }),
  reply('GET /api/v1/plans/{planId}', { body: plan() }, { planId: 'P-1001' }),
];

// Feature 004's addresses (build guide §5).
describe('Plans routes (feature 004)', () => {
  it.each([
    ['/plans', 'Plans', 'Plans'],
    ['/plans/new', 'New plan', 'New plan'],
    ['/plans/P-1001', 'FY2026 Lubricants Europe', 'FY2026 Lubricants Europe'],
    ['/plans/P-1001/edit', 'Edit plan', 'Edit FY2026 Lubricants Europe'],
  ])('opens %s: "%s", with Plans current in the sidebar', async (path, heading, tab) => {
    renderApp(path, ANSWERS);

    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument();
    await vi.waitFor(() => expect(document.title).toBe(`${tab} · SP&A One`));
    expectCurrentSection('Plans');
  });

  it('places feature 005’s campaign content on the plan page', async () => {
    renderApp('/plans/P-1001', [
      ...ANSWERS,
      reply('GET /api/v1/plans/{planId}/campaigns', { body: planCampaigns() }),
    ]);

    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
  });
});
