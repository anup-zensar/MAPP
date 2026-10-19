import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';

// Feature 005 fills the plan tree; its own tests cover that, so it never breaks these.
vi.mock('@/features/campaigns/useCampaignTreeContent', () => ({
  useCampaignTreeContent: () => ({
    nodes: [],
    toolbarActions: null,
    planRowActions: [],
    emptyAction: null,
    footer: undefined,
    dialogs: <p>Campaign dialogs</p>,
    isLoading: false,
    error: null,
    refetch: () => {},
  }),
}));

// Feature 004's addresses (build guide §5). Feature 004 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('Plans routes (feature 004)', () => {
  it.each([
    ['/plans', 'Plans', 'Plans'],
    ['/plans/new', 'New plan', 'New plan'],
    ['/plans/P-1001', 'Plan', 'Plan'],
    ['/plans/P-1001/edit', 'Edit plan', 'Edit plan'],
  ])('opens %s: "%s", with Plans current in the sidebar', async (path, heading, tab) => {
    renderApp(path);

    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument();
    expect(document.title).toBe(`${tab} · NorthStar`);
    expectCurrentSection('Plans');
  });

  it('places feature 005’s campaign content on the plan page', async () => {
    renderApp('/plans/P-1001');

    expect(await screen.findByText('Campaign dialogs')).toBeInTheDocument();
  });
});
