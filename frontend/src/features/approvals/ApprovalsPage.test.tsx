import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { ApprovalRow } from './approvalsApi';
import { ApprovalsPage } from './ApprovalsPage';
import { DANIEL, approvalOf, previewOf, roundOf, rowOf } from './testData';

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
    'GET /api/approvals/summary': SUMMARY,
    'GET /api/approvals': page(ROWS),
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
    expect(document.title).toBe('Approvals · NorthStar');
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
    expect(screen.getByRole('tab', { name: 'All open 5 activities' })).toBeInTheDocument();
  });

  it('lists the rows of the tab with a count (FR-029, FR-030)', async () => {
    const { api } = renderPage();

    expect(await screen.findByRole('button', { name: 'Launch film' })).toBeInTheDocument();
    expect(screen.getByText('3 of 3')).toBeInTheDocument();
    expect(api.requests.find((request) => request.path === '/api/approvals')?.search).toBe(
      '?tab=waiting',
    );
  });

  it('switches tabs through the address, keeping the search (FR-029)', async () => {
    const { user, router, api } = renderPage('/approvals', {
      'GET /api/approvals': (request) =>
        new URL(request.url).searchParams.get('tab') === 'open' ? page(ROWS) : page([ROWS[0]]),
    });

    await screen.findByRole('button', { name: 'Launch film' });
    await user.type(screen.getByRole('searchbox', { name: 'Search activity name or ID' }), 'radio');
    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /^All open/ }));

    await waitFor(() => expect(router.state.location.search).toBe('?tab=open'));
    expect(await screen.findByRole('button', { name: 'Spring radio' })).toBeInTheDocument();
    expect(screen.getByText('1 of 3')).toBeInTheDocument();
    expect(api.requests.some((request) => request.search === '?tab=open')).toBe(true);
  });

  it.each([
    ['waiting', "Nothing is waiting for you. You're all caught up."],
    ['open', 'No activities are waiting for approval.'],
    ['decided', "You haven't approved or declined any activities yet."],
  ])('says when the %s tab is empty', async (tab, text) => {
    renderPage(`/approvals?tab=${tab}`, { 'GET /api/approvals': page([]) });

    expect(await screen.findByText(text)).toBeInTheDocument();
  });

  it('searches the name or the ID, ignoring letter case (FR-031)', async () => {
    const { user } = renderPage();
    const search = await screen.findByRole('searchbox', { name: 'Search activity name or ID' });
    await screen.findByRole('button', { name: 'Launch film' });

    await user.type(search, 'a-1044');
    expect(screen.getByText('1 of 3')).toBeInTheDocument();
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

    expect(screen.getByText('2 of 3')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters (1)' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Filters (1)' }));
    await user.selectOptions(
      within(await screen.findByRole('dialog', { name: 'Filters' })).getByRole('combobox', {
        name: 'Plan',
      }),
      'P-1002',
    );
    await user.keyboard('{Escape}');

    expect(screen.getByText('1 of 3')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Remove filter: Owner: Daniel Okoro' }));
    expect(
      screen.getByRole('button', { name: 'Remove filter: Plan: FY2026 Mobility China' }),
    ).toBeInTheDocument();
    expect(screen.getByText('1 of 3')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('3 of 3')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters' })).toBeInTheDocument();
  });

  it('opens the preview from a row, through the address (FR-032)', async () => {
    const { user, router } = renderPage('/approvals', {
      'GET /api/approvals/activities/A-1042': { body: previewOf() },
    });

    await user.click(await screen.findByRole('button', { name: 'Launch film' }));

    expect(await screen.findByRole('dialog', { name: 'Launch film' })).toBeInTheDocument();
    expect(router.state.location.search).toBe('?activity=A-1042');
  });

  it('?activity= opens the preview at once', async () => {
    renderPage('/approvals?tab=waiting&activity=A-1042', {
      'GET /api/approvals/activities/A-1042': { body: previewOf() },
    });

    const preview = await screen.findByRole('dialog', { name: 'Launch film' });
    expect(preview).toHaveTextContent('A-1042 · Waiting at level 1 of 2');
  });

  it('after Approve in the preview, says so and moves on to the next row (FR-032)', async () => {
    const next = previewOf({
      activity: { ...previewOf().activity, id: 'A-1043', name: 'Spring radio' },
    });
    const { user, router } = renderPage('/approvals?activity=A-1042', {
      'GET /api/approvals/activities/A-1042': { body: previewOf() },
      'GET /api/approvals/activities/A-1043': { body: next },
      'POST /api/activities/A-1042/approval/approve': {
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
      'GET /api/approvals': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : page(ROWS);
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('button', { name: 'Launch film' })).toBeInTheDocument();
  });
});
