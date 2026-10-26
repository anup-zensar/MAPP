import { configure, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { approvalOf } from '@/testing/fixtures/approvals';
import { activityRecord } from '@/testing/fixtures/line-item-financials';
import type { ShowSubmitProblems } from '@/features/approvals';
import { mockApi } from '@/testing/mock-api';
import type { LooseAnswer } from '@/testing/replies';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  activityAttributeForm,
  editForm,
  incompleteLaunchFilm,
  launchFilm,
} from '@/testing/fixtures/activities';
import { looseReply } from '@/testing/replies';
import { ActivityPage } from './ActivityPage';

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
vi.mock('@/features/approvals/SubmitButton', () => ({
  SubmitButton: ({ onShowProblems }: { onShowProblems: ShowSubmitProblems }) => (
    <button type="button" onClick={() => onShowProblems(5)}>
      Submit for approval
    </button>
  ),
}));

const PATH = '/plans/:planId/activities/:activityId';

/** The approval of a Created activity the viewer owns: nothing to decide (feature 008). */
const NOT_SUBMITTED = approvalOf({
  status: 'created',
  displayStatus: 'created',
  rounds: [],
  actions: {
    submit: { available: true, reason: '' },
    resubmit: { available: false, reason: '' },
    approve: { available: false, reason: '' },
    decline: { available: false, reason: '' },
    recall: { available: false, reason: '' },
  },
});

/** Today's rates, for the ⓘ beside an amount in another currency (016 FR-009). */
const RATES = {
  today: '2026-03-12',
  total: 2,
  items: [
    {
      currency: 'USD',
      rate: '1',
      inverseRate: '1',
      rateDate: null,
      source: null,
      importFileName: null,
    },
    {
      currency: 'EUR',
      rate: '0.92',
      inverseRate: '1.086957',
      rateDate: '2026-03-01',
      source: 'settings',
      importFileName: null,
    },
  ],
};

function openPage(
  answer: LooseAnswer,
  route = '/plans/P-1001/activities/A-1042',
  approval: LooseAnswer = { body: NOT_SUBMITTED },
) {
  const api = mockApi([
    ...HISTORY_ANSWERS,
    looseReply('GET /api/v1/activities/{activityId}', answer, { activityId: 'A-1042' }),
    looseReply('GET /api/v1/activities/{activityId}/approval', approval, { activityId: 'A-1042' }),
    looseReply('GET /api/v1/exchange-rates/current', { body: RATES }),
    looseReply(
      'GET /api/v1/financials/records/{recordId}',
      {
        body: activityRecord({ record: { type: 'activity', id: 'A-1042', name: 'Launch film' } }),
      },
      { recordId: 'A-1042' },
    ),
    looseReply(
      'GET /api/v1/activities/{activityId}/form',
      { body: editForm() },
      { activityId: 'A-1042' },
    ),
    looseReply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: activityAttributeForm() },
      { level: 'activity' },
    ),
  ]);
  return { ...renderWithProviders(<ActivityPage />, { route, path: PATH, user: testUser() }), api };
}

describe('ActivityPage (feature 006, US2)', () => {
  it('shows status, completeness, ID, owner, dates, planned cost and unassigned planned cost (FR-019 – FR-021)', async () => {
    openPage({ body: incompleteLaunchFilm({ owner: launchFilm().owner }) });

    expect(await screen.findByRole('heading', { level: 1, name: 'Launch film' })).toBeVisible();
    expect(document.title).toBe('Launch film · MAPP');
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
    const header = screen.getByRole('banner');
    expect(within(header).getByText('Priya Raman')).toBeVisible();
    expect(within(header).getByText('01 Mar – 31 May 2026')).toBeVisible();
    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getByText('120,000.00 USD')).toBeVisible();
    expect(within(summary).getByText('20,000.00 USD')).toBeVisible();
    expect(screen.getByRole('link', { name: 'Edit' })).toBeVisible();
    expect(screen.getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042/edit',
    );
    expect(screen.getByRole('button', { name: 'Actions for Launch film' })).toBeVisible();
  });

  it('shows the planned and unassigned cost in the reader’s display currency (D-220)', async () => {
    openPage({
      body: launchFilm({
        display: {
          plannedCost: { amount: '110400.00', currency: 'EUR' },
          unassignedPlannedCost: { amount: '18400.00', currency: 'EUR' },
          revenue: null,
        },
      }),
    });

    expect(await screen.findByRole('heading', { level: 1, name: 'Launch film' })).toBeVisible();
    // The plan currency first, the reader's currency beside it (016 FR-013).
    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getByText('120,000.00 USD')).toBeVisible();
    expect(within(summary).getByText(/≈ 110,400.00 EUR/)).toBeVisible();
    expect(within(summary).getByText(/≈ 18,400.00 EUR/)).toBeVisible();
  });

  it('marks an unassigned planned cost below zero in red words', async () => {
    openPage({
      body: launchFilm({ unassignedPlannedCost: { amount: '-500.00', currency: 'USD' } }),
    });

    expect(
      await screen.findByText('Line items exceed the planned cost by 500.00 USD'),
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

  it('shows the summary, then the tabs Overview · Details · Approval · History (design system §2.4 B, FR-020)', async () => {
    const { user, router } = openPage({ body: launchFilm({ externalName: '' }) });

    await screen.findByRole('heading', { level: 1, name: 'Launch film' });
    const tabs = screen.getByRole('tablist', { name: 'Activity' });
    expect(
      within(tabs)
        .getAllByRole('tab')
        .map((tab) => tab.textContent),
    ).toEqual(['Overview', 'Details', 'Approval', 'History']);
    // Overview: Ready to submit, GSAP's Financials (open, 015 FR-015), then the line items.
    expect(screen.getByRole('tab', { name: 'Overview' })).toHaveAttribute('aria-selected', 'true');
    const ready = screen.getByRole('region', { name: 'Ready to submit' });
    const financials = screen.getByRole('region', { name: 'Financials' });
    const lineItems = screen.getByRole('region', { name: 'Line items' });
    expect(
      screen.getByRole('region', { name: 'Summary' }).compareDocumentPosition(ready) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      ready.compareDocumentPosition(financials) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      financials.compareDocumentPosition(lineItems) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(financials.id).toBe('financials');
    expect(await within(financials).findByText('82,000.00 USD')).toBeVisible();
    expect(screen.queryByText('From finance (coming later)')).not.toBeInTheDocument();
    expect(lineItems.closest('#line-items')).not.toBeNull();
    expect(screen.queryByRole('region', { name: 'Details' })).not.toBeInTheDocument();

    // Details: details, Shell attributes and goals; the tab is kept in the address.
    await user.click(screen.getByRole('tab', { name: 'Details' }));
    expect(await screen.findByRole('region', { name: 'Details' })).toBeVisible();
    expect(screen.getByRole('region', { name: 'Shell attributes' })).toBeVisible();
    expect(screen.getByRole('region', { name: 'Goals' })).toBeVisible();
    expect(router.state.location.search).toBe('?tab=details');

    await user.click(screen.getByRole('tab', { name: 'Approval' }));
    expect(
      (await screen.findByRole('region', { name: 'Approval' })).closest('#approval'),
    ).not.toBeNull();

    await user.click(screen.getByRole('tab', { name: 'History' }));
    const history = await screen.findByRole('region', { name: 'History' });
    expect(history.closest('#history')).not.toBeNull();
    expect(screen.getByRole('checkbox', { name: 'Include line items' })).not.toBeChecked();

    await user.click(screen.getByRole('tab', { name: 'Overview' }));
    await waitFor(() => expect(router.state.location.search).toBe(''));
  });

  it('links the activities of a split, "(deleted)" without a link (016 FR-031)', async () => {
    openPage({
      body: launchFilm({
        splitFrom: { id: 'A-1012', name: 'UK launch event', planId: 'P-1001', deleted: false },
        splitInto: [{ id: 'A-1014', name: 'Ireland roadshow', planId: 'P-1001', deleted: true }],
      }),
    });

    expect(
      await screen.findByRole('link', { name: "Split from 'UK launch event' (A-1012)" }),
    ).toHaveAttribute('href', '/plans/P-1001/activities/A-1012');
    expect(screen.getByText("Split into 'Ireland roadshow' (A-1014) (deleted)")).toBeVisible();
    expect(
      screen.queryByRole('link', { name: /Split into 'Ireland roadshow'/ }),
    ).not.toBeInTheDocument();
  });

  it('opens the tab of a #part: #approval from Home and the notifications', async () => {
    openPage({ body: launchFilm() }, '/plans/P-1001/activities/A-1042#approval');

    expect(await screen.findByRole('region', { name: 'Approval' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Approval' })).toHaveAttribute('aria-selected', 'true');
  });

  it('opens the History tab and focuses its heading for View history', async () => {
    const { user, router } = openPage({ body: launchFilm() });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() => expect(router.state.location.search).toBe('?tab=history'));
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 2, name: 'History' })).toHaveFocus(),
    );
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
        error: {
          code: 'not_found',
          message: 'This activity no longer exists — someone may have deleted it.',
          fields: [],
        },
      },
    });

    expect(
      await screen.findByText('This activity no longer exists — someone may have deleted it.'),
    ).toBeVisible();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toBeVisible();
  });

  it('offers Try again when the activity cannot be loaded', async () => {
    openPage({
      status: 500,
      body: { error: { code: 'internal_error', message: 'Broken', fields: [] } },
    });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeVisible();
  });
});
