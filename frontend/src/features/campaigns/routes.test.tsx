import { screen, within } from '@testing-library/react';
import { Outlet } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { renderApp } from '@/test/app';

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

// Feature 005's addresses: side panels over the plan page (build guide §5). Feature 005 keeps
// this file up to date as it replaces the stub panels.
describe('Campaign and initiative routes (feature 005)', () => {
  it.each([
    ['/plans/P-1001/campaigns/new', 'New campaign'],
    ['/plans/P-1001/campaigns/C-1001', 'Campaign'],
    ['/plans/P-1001/campaigns/C-1001/edit', 'Edit campaign'],
    ['/plans/P-1001/campaigns/C-1001/initiatives/new', 'New initiative'],
    ['/plans/P-1001/initiatives/I-1001', 'Initiative'],
    ['/plans/P-1001/initiatives/I-1001/edit', 'Edit initiative'],
  ])('opens %s as the "%s" panel over the plan page', async (path, title) => {
    renderApp(path);

    expect(await screen.findByRole('dialog', { name: title })).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 1, name: 'Plan page', hidden: true }),
    ).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
  });

  it('places feature 006’s activities in the initiative panel', async () => {
    renderApp('/plans/P-1001/initiatives/I-1001');

    const panel = await screen.findByRole('dialog', { name: 'Initiative' });
    expect(within(panel).getByRole('region', { name: 'Activities' })).toHaveTextContent(
      'Activities of I-1001',
    );
  });
});
