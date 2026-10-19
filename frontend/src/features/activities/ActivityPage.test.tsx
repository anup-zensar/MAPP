import { configure, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { ShowSubmitProblems } from '@/features/approvals/approvalTypes';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { ActivityPage } from './ActivityPage';
import { activityAttributeForm, editForm, incompleteLaunchFilm, launchFilm } from './testData';

// The whole form renders in each test: on a busy computer the first render can take a while.
configure({ asyncUtilTimeout: 10_000 });

// Features 007 and 008 test their own parts; here they stand in. The status action stands in for
// 008's Submit for approval, which finds five problems and hands them to this page.
vi.mock('@/features/line-items/LineItemsSection', () => ({
  LineItemsSection: () => <section aria-label="Line items">Line items</section>,
}));
vi.mock('@/features/approvals/ApprovalSection', () => ({
  ApprovalSection: () => <section aria-label="Approval">Approval</section>,
}));
vi.mock('@/features/approvals/ApprovalStatusAction', () => ({
  ApprovalStatusAction: ({ onShowProblems }: { onShowProblems: ShowSubmitProblems }) => (
    <button type="button" onClick={() => onShowProblems(5)}>
      Submit for approval
    </button>
  ),
}));

const PATH = '/plans/:planId/activities/:activityId';

function openPage(answer: MockHandler, route = '/plans/P-1001/activities/A-1042') {
  const api = mockApi({
    'GET /api/activities/A-1042': answer,
    'GET /api/activities/A-1042/form': { body: editForm() },
    'GET /api/shell-attribute-forms/activity': { body: activityAttributeForm() },
  });
  return { ...renderWithProviders(<ActivityPage />, { route, path: PATH, user: testUser() }), api };
}

describe('ActivityPage (feature 006, US2)', () => {
  it('shows status, completeness, ID, owner, dates, planned cost and unassigned planned cost (FR-019 – FR-021)', async () => {
    openPage({ body: incompleteLaunchFilm({ owner: launchFilm().owner }) });

    expect(await screen.findByRole('heading', { level: 1, name: 'Launch film' })).toBeVisible();
    expect(document.title).toBe('Launch film · NorthStar');
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(
      within(breadcrumb)
        .getAllByRole('link')
        .map((link) => link.textContent),
    ).toEqual(['Plans', 'FY2026 Lubricants Europe', 'Helix launch', 'Digital & social']);
    expect(screen.getByText('ACTIVITY')).toBeVisible();
    expect(screen.getByText('Created')).toBeVisible();
    expect(screen.getByRole('button', { name: '3 fields needed' })).toBeVisible();
    expect(screen.getByRole('button', { name: '1 line item needs attention' })).toBeVisible();
    expect(screen.getByText('A-1042')).toBeVisible();
    const facts = screen.getByText('Unassigned planned cost').closest('dl') as HTMLElement;
    expect(within(facts).getByText('Priya Raman')).toBeVisible();
    expect(within(facts).getByText('01 Mar 2026 – 31 May 2026')).toBeVisible();
    expect(within(facts).getByText('$120,000.00')).toBeVisible();
    expect(within(facts).getByText('$20,000.00')).toBeVisible();
    expect(
      screen.getByText(
        'Created by Alex Morgan on 01 Mar 2026, last changed by Priya Raman on 12 Mar 2026',
      ),
    ).toBeVisible();
    expect(screen.getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042/edit',
    );
    expect(screen.getByRole('button', { name: 'Actions for Launch film' })).toBeVisible();
  });

  it('marks an unassigned planned cost below zero in red words', async () => {
    openPage({
      body: launchFilm({ unassignedPlannedCost: { amount: '-500.00', currency: 'USD' } }),
    });

    expect(
      await screen.findByText("Line items exceed the activity's planned cost by $500.00"),
    ).toBeVisible();
  });

  it('says that a cost or date change restarts the approval of an approved activity (FR-027)', async () => {
    openPage({ body: launchFilm({ status: 'approved', displayStatus: 'inProgress' }) });

    expect(
      await screen.findByText(
        'In progress — changing planned cost or dates will restart approval.',
      ),
    ).toBeVisible();
    expect(screen.getByText('In progress')).toBeVisible();
  });

  it('opens Ready to submit and says how many things are missing when submitting finds problems (FR-025)', async () => {
    const { user } = openPage({ body: incompleteLaunchFilm() });

    const summary = (await screen.findByText('Submitting needs 5 things')).closest('summary');
    const panel = screen.getByRole('region', { name: 'Ready to submit' });
    const owner = within(panel).getByRole('link', { name: 'Owner — needed to submit' });
    expect(owner).toBeVisible();
    await user.click(summary as HTMLElement);
    expect(owner).not.toBeVisible();

    await user.click(screen.getByRole('button', { name: 'Submit for approval' }));

    expect(screen.getByRole('alert')).toHaveTextContent('Submitting needs 5 things — view list');
    expect(owner).toBeVisible();
    await waitFor(() => expect(panel).toHaveFocus());
  });

  it('shows the sections in order: Ready to submit, details, line items, approval, from finance (FR-020)', async () => {
    openPage({ body: launchFilm({ externalName: '' }) });

    await screen.findByRole('heading', { level: 1, name: 'Launch film' });
    const names = [
      screen.getByRole('region', { name: 'Ready to submit' }),
      screen.getByRole('region', { name: 'Details' }),
      screen.getByRole('region', { name: 'Line items' }),
      screen.getByRole('region', { name: 'Approval' }),
      screen.getByRole('region', { name: 'From finance (coming later)' }),
    ];
    for (let index = 1; index < names.length; index += 1) {
      expect(
        names[index - 1].compareDocumentPosition(names[index]) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    }
    expect(
      screen.getByRole('region', { name: 'Line items' }).closest('#line-items'),
    ).not.toBeNull();
    expect(screen.getByRole('region', { name: 'Approval' }).closest('#approval')).not.toBeNull();
    expect(screen.getByText('Total actuals')).not.toBeVisible();
  });

  it('replaces an address with another plan by the right one', async () => {
    const { router } = openPage({ body: launchFilm() }, '/plans/P-9999/activities/A-1042#ready');

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042'),
    );
    expect(router.state.location.hash).toBe('#ready');
  });

  it('opens Ready to submit for the Show fields link of Home (#completeness, feature 009)', async () => {
    openPage({ body: launchFilm() }, '/plans/P-1001/activities/A-1042#completeness');

    const panel = await screen.findByRole('region', { name: 'Ready to submit' });
    await waitFor(() => expect(panel).toHaveFocus());
  });

  it('shows grey shapes while loading', async () => {
    openPage(() => new Promise(() => {}));

    expect(await screen.findAllByText('Loading…')).not.toHaveLength(0);
  });

  it('says when the activity no longer exists', async () => {
    openPage({
      status: 404,
      body: {
        code: 'not_found',
        message: 'This activity no longer exists — someone may have deleted it.',
        fields: [],
      },
    });

    expect(
      await screen.findByText('This activity no longer exists — someone may have deleted it.'),
    ).toBeVisible();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toBeVisible();
  });

  it('offers Try again when the activity cannot be loaded', async () => {
    openPage({ status: 500, body: { code: 'internal_error', message: 'Broken', fields: [] } });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeVisible();
  });
});
