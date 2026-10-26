import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ApprovalSection } from './ApprovalSection';
import type { ActivityApproval } from './api';
import type { ApprovalActivity } from './approval-types';
import { APPROVED_NO_RECALL } from './messages';
import {
  ALEX,
  approvalOf,
  CARLOS,
  DANIEL,
  DECLINE,
  PRIYA,
  roundOf,
  SUBMITTED,
} from '@/testing/fixtures/approvals';

const APPROVAL = 'GET /api/v1/activities/A-1042/approval';

const ACTIVITY: ApprovalActivity = {
  id: 'A-1042',
  name: 'Launch film',
  status: 'submitted',
  version: 3,
};

const NONE = { available: false, reason: '' };

function renderSection(answer: MockHandler, extra: Record<string, MockHandler> = {}) {
  const api = mockApi({ [APPROVAL]: answer, ...extra });
  const rendered = renderWithProviders(
    <ApprovalSection activity={ACTIVITY} onShowProblems={vi.fn()} />,
    { user: testUser() },
  );
  return { ...rendered, api };
}

const card = () => screen.getByRole('region', { name: 'Approval' });

describe('ApprovalSection (feature 008, US2)', () => {
  it('shows the round, the status, the workflow and the chain with each state (FR-024)', async () => {
    renderSection({
      body: approvalOf({
        viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
        rounds: [
          roundOf({
            levels: [
              { ...roundOf().levels[0], approvers: [CARLOS, PRIYA] },
              { ...roundOf().levels[1], approvers: [{ ...ALEX, active: false }] },
            ],
          }),
        ],
      }),
    });

    expect(await screen.findByText('Round 1 · Lubricants EU approval')).toBeInTheDocument();
    expect(card()).toHaveTextContent('Submitted');
    expect(card()).toHaveTextContent('Approval workflow: Lubricants EU approval');
    const levels = within(card())
      .getAllByRole('listitem')
      .filter((item) => item.textContent?.includes('Level'));
    expect(levels[0]).toHaveTextContent('Level 1');
    expect(levels[0]).toHaveTextContent('Priya Raman (you)');
    expect(levels[0]).toHaveTextContent('Waiting');
    expect(card()).toHaveTextContent('Alex Morgan (inactive)');
    expect(card()).toHaveTextContent('Not reached yet');
    // Decline then Approve, side by side (UI audit AC-03).
    const buttons = within(card())
      .getAllByRole('button')
      .map((button) => button.textContent);
    expect(buttons.indexOf('Approve')).toBe(buttons.indexOf('Decline') + 1);
  });

  it('never offers the owner Approve as a button; it waits in ⋯ with a warning (functional review A-01)', async () => {
    const { user } = renderSection({ body: approvalOf() });

    await screen.findByText('Round 1 · Lubricants EU approval');
    expect(within(card()).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(within(card()).queryByRole('button', { name: 'Decline' })).not.toBeInTheDocument();
    await user.click(
      within(card()).getByRole('button', { name: 'Actions for approval of Launch film' }),
    );
    const approve = await screen.findByRole('menuitem', { name: /^Approve/ });
    expect(approve).toHaveTextContent('You own this activity.');
    await user.click(approve);
    const dialog = await screen.findByRole('alertdialog', { name: "Approve 'Launch film'?" });
    expect(dialog).toHaveTextContent('You own this activity.');
  });

  it('says why the person cannot decide when the API gives a reason (014 FR-025)', async () => {
    const reason = 'You own this activity, so someone else must approve it.';
    renderSection({
      body: approvalOf({
        actions: {
          ...approvalOf().actions,
          approve: { available: false, reason },
          decline: { available: false, reason },
        },
      }),
    });

    expect(await within(card()).findByText(reason)).toBeInTheDocument();
    expect(
      within(card()).queryByRole('button', { name: 'Actions for approval of Launch film' }),
    ).not.toBeInTheDocument();
  });

  it('offers Re-submit after a decline and leaves the reason to the page notice (FR-016, UI audit AC-02)', async () => {
    const declined: ActivityApproval = approvalOf({
      status: 'declined',
      displayStatus: 'declined',
      latestDecline: DECLINE,
      rounds: [roundOf({ outcome: 'declined', endedAt: DECLINE.at })],
      actions: {
        submit: NONE,
        resubmit: { available: true, reason: '' },
        approve: NONE,
        decline: NONE,
        recall: NONE,
      },
    });
    renderSection({ body: declined });

    expect(
      await within(card()).findByRole('button', { name: 'Re-submit for approval' }),
    ).toBeInTheDocument();
    // The activity page's notice under the header shows who declined and why (ActivityStatusNotice).
    expect(
      screen.queryByText(
        'Declined by Daniel Okoro on behalf of Carlos Mendes on 11 Mar 2026, 14:32',
      ),
    ).not.toBeInTheDocument();
    expect(within(card()).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
  });

  it('shows the history with on-behalf decisions, comments and change sentences (FR-025)', async () => {
    const round2 = roundOf({
      id: 'AR-1002',
      number: 2,
      history: [
        {
          ...SUBMITTED,
          id: 'AH-1004',
          roundNumber: 2,
          action: 'restarted',
          level: 0,
          levelLabel: '',
          approvers: [],
          actor: { id: PRIYA.id, name: PRIYA.name },
          comment: '',
          at: '2026-03-12T10:15:00Z',
          changes: [
            {
              label: 'Planned cost',
              oldText: '$50,000.00',
              newText: '$65,000.00',
              text: 'Planned cost changed from $50,000.00 to $65,000.00',
            },
          ],
        },
      ],
    });
    const round1 = roundOf({
      outcome: 'approved',
      history: [
        {
          ...SUBMITTED,
          id: 'AH-1002',
          action: 'approved',
          actor: { id: DANIEL.id, name: DANIEL.name },
          onBehalfOf: [{ id: CARLOS.id, name: CARLOS.name }],
          comment: 'Looks good',
          at: '2026-03-11T08:30:00Z',
        },
        SUBMITTED,
      ],
    });
    renderSection({ body: approvalOf({ rounds: [round2, round1] }) });

    const history = await screen.findByRole('region', { name: 'Approval history' });
    const rounds = within(history).getAllByRole('heading', { level: 4 });
    expect(rounds.map((heading) => heading.textContent)).toEqual([
      'Round 2 · Waiting at level 1 of 2 · Lubricants EU approval',
      'Round 1 · Approved · Lubricants EU approval',
    ]);
    expect(history).toHaveTextContent('Planned cost changed from $50,000.00 to $65,000.00');
    expect(history).toHaveTextContent('Daniel Okoro on behalf of Carlos Mendes');
    expect(history).toHaveTextContent('Approver: Carlos Mendes');
    expect(history).toHaveTextContent('"Looks good"');
    expect(history).toHaveTextContent('Approved · Level 1 · 11 Mar 2026, 08:30');
  });

  it('explains why an Approved activity cannot be recalled (FR-021)', async () => {
    renderSection({
      body: approvalOf({
        status: 'approved',
        displayStatus: 'inProgress',
        rounds: [roundOf({ outcome: 'approved' })],
        actions: {
          submit: NONE,
          resubmit: NONE,
          approve: NONE,
          decline: NONE,
          recall: { available: false, reason: APPROVED_NO_RECALL },
        },
      }),
    });

    expect(await screen.findByText(APPROVED_NO_RECALL)).toBeInTheDocument();
    expect(card()).toHaveTextContent('In progress');
    expect(within(card()).queryByRole('button')).not.toBeInTheDocument();
  });

  it('shows "Not submitted yet" with the levels to come and no history', async () => {
    renderSection({
      body: approvalOf({
        status: 'created',
        displayStatus: 'created',
        rounds: [],
        upcomingLevels: [
          { number: 1, name: '', label: 'Level 1', approvers: [CARLOS] },
          { number: 2, name: 'Global brand', label: 'Global brand', approvers: [ALEX] },
        ],
        actions: {
          submit: { available: true, reason: '' },
          resubmit: NONE,
          approve: NONE,
          decline: NONE,
          recall: NONE,
        },
      }),
    });

    expect(await screen.findByRole('heading', { name: 'Not submitted yet' })).toBeInTheDocument();
    expect(card()).toHaveTextContent('Global brand');
    expect(card()).toHaveTextContent('No approval history yet.');
    expect(within(card()).getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
  });

  it('says when no approval workflow is chosen', async () => {
    renderSection({
      body: approvalOf({ status: 'created', displayStatus: 'created', workflow: null, rounds: [] }),
    });

    expect(await screen.findByText('No approval workflow chosen yet.')).toBeInTheDocument();
  });

  it('marks an inactive workflow', async () => {
    renderSection({
      body: approvalOf({
        workflow: { id: 'WF-1001', name: 'Lubricants EU approval', active: false },
      }),
    });

    expect(
      await screen.findByText('Approval workflow: Lubricants EU approval (inactive)'),
    ).toBeInTheDocument();
  });

  it('says when the approval could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderSection(() => {
      calls += 1;
      return calls === 1 ? { status: 500 } : { body: approvalOf() };
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('Round 1 · Lubricants EU approval')).toBeInTheDocument();
  });
});

const RECALL = 'POST /api/v1/activities/A-1042/approval/recall';

/** The actions of an activity back in Created: only Submit. */
const SUBMIT_ONLY = {
  submit: { available: true, reason: '' },
  resubmit: NONE,
  approve: NONE,
  decline: NONE,
  recall: NONE,
};

/** Level 1 of round 1, which was waiting when the round ended; level 2 was never reached. */
const ENDED_LEVELS = [
  { ...roundOf().levels[0], state: 'notDecided' as const },
  roundOf().levels[1],
];

/** Priya changed the planned cost of "Launch film" on 12 Mar 2026 at 10:15. */
const PLANNED_COST_CHANGE = {
  label: 'Planned cost',
  oldText: '$50,000.00',
  newText: '$65,000.00',
  text: 'Planned cost changed from $50,000.00 to $65,000.00',
};

describe('ApprovalSection when approval ends early (feature 008, US5)', () => {
  it('recalls a Submitted activity with a reason, which then shows as Recalled (FR-021)', async () => {
    let recalled = false;
    const recalledApproval = approvalOf({
      status: 'created',
      displayStatus: 'created',
      rounds: [
        roundOf({
          outcome: 'recalled',
          endedAt: '2026-03-12T10:00:00Z',
          levels: ENDED_LEVELS,
          history: [
            {
              ...SUBMITTED,
              id: 'AH-1002',
              action: 'recalled',
              comment: 'Wrong workflow',
              at: '2026-03-12T10:00:00Z',
            },
            SUBMITTED,
          ],
        }),
      ],
      actions: SUBMIT_ONLY,
    });
    const { user, api } = renderSection(
      () => ({ body: recalled ? recalledApproval : approvalOf() }),
      {
        [RECALL]: () => {
          recalled = true;
          return { body: recalledApproval };
        },
      },
    );

    await user.click(await within(card()).findByRole('button', { name: 'Recall submission' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Recall the submission of 'Launch film'?",
    });
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Reason (optional)' }),
      'Wrong workflow',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Recall submission' }));

    expect(await screen.findByText("Submission of 'Launch film' recalled")).toBeInTheDocument();
    expect(api.bodiesOf(RECALL)).toEqual([{ roundId: 'AR-1001', reason: 'Wrong workflow' }]);
    expect(await within(card()).findByText('— submission recalled')).toBeInTheDocument();
    expect(card()).toHaveTextContent('Not decided — submission recalled');
    const history = screen.getByRole('region', { name: 'Approval history' });
    expect(history).toHaveTextContent('Round 1 · Recalled · Lubricants EU approval');
    expect(history).toHaveTextContent('Recalled · Level 1 · 12 Mar 2026, 10:00');
    expect(history).toHaveTextContent('"Wrong workflow"');
    expect(within(card()).getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
    expect(
      within(card()).queryByRole('button', { name: 'Recall submission' }),
    ).not.toBeInTheDocument();
  });

  it('shows a round ended by a change that needs approval again, with the change (FR-022)', async () => {
    renderSection({
      body: approvalOf({
        status: 'created',
        displayStatus: 'created',
        rounds: [
          roundOf({
            outcome: 'restarted',
            endedAt: '2026-03-12T10:15:00Z',
            levels: ENDED_LEVELS,
            history: [
              {
                ...SUBMITTED,
                id: 'AH-1002',
                action: 'restarted',
                comment: '',
                at: '2026-03-12T10:15:00Z',
                changes: [PLANNED_COST_CHANGE],
              },
              SUBMITTED,
            ],
          }),
        ],
        actions: SUBMIT_ONLY,
      }),
    });

    expect(await screen.findByText('Round 1 · Lubricants EU approval')).toBeInTheDocument();
    expect(card()).toHaveTextContent('Created');
    const [first, second] = within(card())
      .getAllByRole('listitem')
      .filter((item) => /^L\d/.test(item.textContent ?? ''));
    expect(first).toHaveTextContent('Not decided — approval restarted');
    expect(second).toHaveTextContent('Not reached yet');
    const history = screen.getByRole('region', { name: 'Approval history' });
    expect(history).toHaveTextContent('Round 1 · Approval restarted · Lubricants EU approval');
    expect(history).toHaveTextContent('Approval restarted · Level 1 · 12 Mar 2026, 10:15');
    expect(history).toHaveTextContent('Planned cost changed from $50,000.00 to $65,000.00');
    expect(within(card()).getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
  });

  it('shows the restart of an Approved activity on its approved round (FR-022)', async () => {
    const approvedLevels = roundOf().levels.map((level) => ({
      ...level,
      state: 'approved' as const,
    }));
    renderSection({
      body: approvalOf({
        status: 'created',
        displayStatus: 'created',
        rounds: [
          roundOf({
            outcome: 'approved',
            currentLevel: 2,
            endedAt: '2026-03-11T08:30:00Z',
            levels: approvedLevels,
            history: [
              {
                ...SUBMITTED,
                id: 'AH-1004',
                action: 'restarted',
                level: 0,
                levelLabel: '',
                approvers: [],
                comment: '',
                at: '2026-03-12T10:15:00Z',
                changes: [
                  {
                    label: "Line item 'Agency fee' (L-1003) — Currency",
                    oldText: 'EUR',
                    newText: 'GBP',
                    text: "Line item 'Agency fee' (L-1003) — Currency changed from EUR to GBP",
                  },
                ],
              },
              SUBMITTED,
            ],
          }),
        ],
        actions: SUBMIT_ONLY,
      }),
    });

    const history = await screen.findByRole('region', { name: 'Approval history' });
    expect(history).toHaveTextContent('Round 1 · Approved · Lubricants EU approval');
    const restart = within(history)
      .getAllByRole('listitem')
      .find((item) => item.textContent?.startsWith('Approval restarted'));
    expect(restart).toHaveTextContent('Approval restarted · 12 Mar 2026, 10:15');
    expect(restart).toHaveTextContent(
      "Line item 'Agency fee' (L-1003) — Currency changed from EUR to GBP",
    );
    expect(restart).not.toHaveTextContent('Approver');
    expect(card()).toHaveTextContent('Created');
  });

  it('shows a round cancelled because the activity was deleted (FR-023)', async () => {
    renderSection({
      body: approvalOf({
        status: 'created',
        displayStatus: 'created',
        rounds: [
          roundOf({
            outcome: 'cancelled',
            endedAt: '2026-03-11T16:00:00Z',
            levels: ENDED_LEVELS,
            history: [
              {
                ...SUBMITTED,
                id: 'AH-1002',
                action: 'cancelled',
                actor: { id: DANIEL.id, name: DANIEL.name },
                comment: '',
                at: '2026-03-11T16:00:00Z',
              },
              SUBMITTED,
            ],
          }),
        ],
        actions: SUBMIT_ONLY,
      }),
    });

    expect(await within(card()).findByText('— activity deleted')).toBeInTheDocument();
    expect(card()).toHaveTextContent('Not decided — activity deleted');
    const history = screen.getByRole('region', { name: 'Approval history' });
    expect(history).toHaveTextContent(
      'Round 1 · Cancelled — activity deleted · Lubricants EU approval',
    );
    expect(history).toHaveTextContent(
      'Cancelled — activity deleted · Level 1 · 11 Mar 2026, 16:00',
    );
    expect(history).toHaveTextContent('Daniel Okoro');
  });
});
