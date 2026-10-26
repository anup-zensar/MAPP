import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  CARLOS,
  DANIEL,
  NO_DELEGATIONS,
  PRIYA,
  approvalOf,
  delegationOf,
  monitoringOf,
  previewOf,
  roundOf,
  rowOf,
} from '@/testing/fixtures/approvals';
import type { ApprovalRow } from './api';
import { ApprovalsPage } from './ApprovalsPage';

const SUMMARY = {
  body: { waitingForMe: 3, allOpen: 5, approvedThisMonth: 2, declinedThisMonth: 1 },
};

const CHINA = [
  { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
  { type: 'campaign', id: 'C-1005', name: 'EV launch' },
  { type: 'initiative', id: 'I-1009', name: 'Dealer events' },
];

const ROWS: ApprovalRow[] = [
  rowOf(),
  rowOf({
    activity: {
      id: 'A-1043',
      name: 'Spring radio',
      status: 'submitted',
      displayStatus: 'submitted',
    },
    path: CHINA,
    owner: DANIEL,
    workflow: { id: 'WF-1002', name: 'China BC approval workflow', active: true },
    submittedAt: '2026-03-01T09:00:00Z',
  }),
  rowOf({
    activity: { id: 'A-1044', name: 'Trade show', status: 'submitted', displayStatus: 'submitted' },
    owner: DANIEL,
  }),
];

const page = (items: ApprovalRow[]) => ({ body: { items, total: items.length } });

function renderPage(route = '/approvals', answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    'GET /api/v1/approvals/summary': SUMMARY,
    'GET /api/v1/approvals': page(ROWS),
    'GET /api/v1/approval-delegations': NO_DELEGATIONS,
    ...answers,
  });
  const rendered = renderWithProviders(<ApprovalsPage />, {
    route,
    path: '/approvals',
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('ApprovalsPage (feature 008, US3)', () => {
  it('shows the four summary cards with their tips (FR-028)', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { level: 1, name: 'Approvals' })).toBeInTheDocument();
    expect(document.title).toBe('Approvals · MAPP');
    await screen.findByText('Approved this month');
    for (const label of [
      'Waiting for me',
      'All open',
      'Approved this month',
      'Declined this month',
    ]) {
      expect(screen.getByRole('button', { name: `About ${label}` })).toBeInTheDocument();
    }
    expect(screen.getByRole('tab', { name: 'Waiting for me 3 activities' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(screen.getByRole('tab', { name: 'All open (5)' })).toBeInTheDocument();
  });

  it('lists the rows of the tab with a count (FR-029, FR-030)', async () => {
    const { api } = renderPage();

    expect(await screen.findByRole('button', { name: 'Launch film' })).toBeInTheDocument();
    expect(screen.getByText('3 of 3 activities')).toBeInTheDocument();
    expect(api.requests.find((request) => request.path === '/api/v1/approvals')?.search).toBe(
      '?tab=waiting',
    );
  });

  it('switches tabs through the address, keeping the search (FR-029)', async () => {
    const { user, router, api } = renderPage('/approvals', {
      'GET /api/v1/approvals': (request) =>
        new URL(request.url).searchParams.get('tab') === 'open' ? page(ROWS) : page([ROWS[0]]),
    });

    await screen.findByRole('button', { name: 'Launch film' });
    await user.type(screen.getByRole('searchbox', { name: 'Search activity name or ID' }), 'radio');
    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /^All open/ }));

    await waitFor(() => expect(router.state.location.search).toBe('?tab=open'));
    expect(await screen.findByRole('button', { name: 'Spring radio' })).toBeInTheDocument();
    expect(screen.getByText('1 of 3 activities')).toBeInTheDocument();
    expect(api.requests.some((request) => request.search === '?tab=open')).toBe(true);
  });

  it.each([
    ['waiting', "Nothing is waiting for you. You're all caught up."],
    ['open', 'No activities are waiting for approval.'],
    ['decided', "You haven't approved or declined any activities yet."],
  ])('says when the %s tab is empty', async (tab, text) => {
    renderPage(`/approvals?tab=${tab}`, { 'GET /api/v1/approvals': page([]) });

    expect(await screen.findByText(text)).toBeInTheDocument();
  });

  it('searches the name or the ID, ignoring letter case (FR-031)', async () => {
    const { user } = renderPage();
    const search = await screen.findByRole('searchbox', { name: 'Search activity name or ID' });
    await screen.findByRole('button', { name: 'Launch film' });

    await user.type(search, 'a-1044');
    expect(screen.getByText('1 of 3 activities')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Trade show' })).toBeInTheDocument();

    await user.clear(search);
    await user.type(search, 'LAUNCH');
    expect(screen.getByRole('button', { name: 'Launch film' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Trade show' })).not.toBeInTheDocument();
  });

  it('filters by plan and owner with chips, and clears them (FR-031)', async () => {
    const { user } = renderPage();
    await screen.findByRole('button', { name: 'Launch film' });

    await user.click(screen.getByRole('button', { name: 'Filters' }));
    const filters = await screen.findByRole('dialog', { name: 'Filters' });
    await user.selectOptions(within(filters).getByRole('combobox', { name: 'Owner' }), 'U-1003');
    await user.keyboard('{Escape}');

    expect(screen.getByText('2 of 3 activities')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters (1)' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Filters (1)' }));
    await user.selectOptions(
      within(await screen.findByRole('dialog', { name: 'Filters' })).getByRole('combobox', {
        name: 'Plan',
      }),
      'P-1002',
    );
    await user.keyboard('{Escape}');

    expect(screen.getByText('1 of 3 activities')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Remove filter: Owner: Daniel Okoro' }));
    expect(
      screen.getByRole('button', { name: 'Remove filter: Plan: FY2026 Mobility China' }),
    ).toBeInTheDocument();
    expect(screen.getByText('1 of 3 activities')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('3 of 3 activities')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters' })).toBeInTheDocument();
  });

  it('opens the preview from a row, through the address (FR-032)', async () => {
    const { user, router } = renderPage('/approvals', {
      'GET /api/v1/approvals/activities/A-1042': { body: previewOf() },
    });

    await user.click(await screen.findByRole('button', { name: 'Launch film' }));

    expect(await screen.findByRole('dialog', { name: 'Launch film' })).toBeInTheDocument();
    expect(router.state.location.search).toBe('?activity=A-1042');
  });

  it('?activity= opens the preview at once', async () => {
    renderPage('/approvals?tab=waiting&activity=A-1042', {
      'GET /api/v1/approvals/activities/A-1042': { body: previewOf() },
    });

    const preview = await screen.findByRole('dialog', { name: 'Launch film' });
    expect(preview).toHaveTextContent('A-1042 · Waiting at level 1 of 2');
  });

  it('after Approve in the preview, says so and moves on to the next row (FR-032)', async () => {
    const next = previewOf({
      activity: { ...previewOf().activity, id: 'A-1043', name: 'Spring radio' },
    });
    const { user, router } = renderPage('/approvals?activity=A-1042', {
      'GET /api/v1/approvals/activities/A-1042': { body: previewOf() },
      'GET /api/v1/approvals/activities/A-1043': { body: next },
      'POST /api/v1/activities/A-1042/approval/approve': {
        body: approvalOf({ status: 'approved', rounds: [roundOf({ outcome: 'approved' })] }),
      },
    });
    await screen.findByRole('button', { name: 'Launch film', hidden: true });
    const preview = await screen.findByRole('dialog', { name: 'Launch film' });

    await user.click(within(preview).getByRole('button', { name: 'Approve' }));
    const dialog = await screen.findByRole('alertdialog', { name: "Approve 'Launch film'?" });
    await user.click(within(dialog).getByRole('button', { name: 'Approve' }));

    expect(await screen.findByText("Activity 'Launch film' approved")).toBeInTheDocument();
    await waitFor(() => expect(router.state.location.search).toBe('?activity=A-1043'));
    expect(await screen.findByRole('dialog', { name: 'Spring radio' })).toBeInTheDocument();
  });

  it('says when the rows could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderPage('/approvals', {
      'GET /api/v1/approvals': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : page(ROWS);
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('button', { name: 'Launch film' })).toBeInTheDocument();
  });

  it('opens All open when nothing waits for the person (UI audit AP-03)', async () => {
    const { api } = renderPage('/approvals', {
      'GET /api/v1/approvals/summary': {
        body: { waitingForMe: 0, allOpen: 3, approvedThisMonth: 0, declinedThisMonth: 0 },
      },
    });

    expect(await screen.findByRole('tab', { name: 'All open (3)' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    await screen.findByRole('button', { name: 'Launch film' });
    expect(api.requests.filter((request) => request.path === '/api/v1/approvals')).toEqual([
      expect.objectContaining({ search: '?tab=open' }),
    ]);
  });

  it('keeps the tab the address names, even when nothing waits (UI audit AP-03)', async () => {
    renderPage('/approvals?tab=waiting', {
      'GET /api/v1/approvals/summary': {
        body: { waitingForMe: 0, allOpen: 3, approvedThisMonth: 0, declinedThisMonth: 0 },
      },
      'GET /api/v1/approvals': page([]),
    });

    expect(
      await screen.findByText("Nothing is waiting for you. You're all caught up."),
    ).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /^Waiting for me/ })).toHaveAttribute(
      'aria-selected',
      'true',
    );
  });
});

describe('ApprovalsPage rules (feature 017, US4 and US6)', () => {
  it('opens Monitoring as the fourth tab, without search or filters (FR-030)', async () => {
    const { user, api } = renderPage('/approvals?tab=waiting', {
      'GET /api/v1/approvals/monitoring': { body: monitoringOf() },
    });
    await screen.findByRole('row', { name: /Launch film/ });

    await user.click(screen.getByRole('tab', { name: 'Monitoring' }));

    expect(await screen.findByText('Average time to decision')).toBeInTheDocument();
    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Filters/ })).not.toBeInTheDocument();
    expect(api.requests.some((r) => r.path === '/api/v1/approvals/monitoring')).toBe(true);
  });

  it('opens the delegation panel from Delegate my approvals, through the address (FR-024)', async () => {
    const { user } = renderPage();

    await user.click(await screen.findByRole('button', { name: 'Delegate my approvals' }));

    const panel = await screen.findByRole('dialog', { name: 'Approval delegation' });
    const closes = within(panel).getAllByRole('button', { name: 'Close' });
    await user.click(closes[closes.length - 1]);
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Approval delegation' })).not.toBeInTheDocument(),
    );
  });

  it('says above the cards who approves for whom today (FR-024)', async () => {
    renderPage('/approvals', {
      'GET /api/v1/approval-delegations': {
        body: {
          items: [
            delegationOf({ person: PRIYA, delegate: CARLOS, state: 'current' }),
            delegationOf({ id: 'DLG-1002', person: DANIEL, delegate: PRIYA, state: 'current' }),
          ],
          total: 2,
          canManageOthers: false,
        },
      },
    });

    expect(
      await screen.findByText('Carlos Mendes approves for you until 14 Nov 2026.'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('You are approving for Daniel Okoro until 14 Nov 2026.'),
    ).toBeInTheDocument();
  });
});
