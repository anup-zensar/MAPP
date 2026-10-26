import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import type { ActivityApproval } from './approvalsApi';
import { DecisionDialog } from './DecisionDialog';
import type { DecisionDialogProps } from './DecisionDialog';
import { ALEX, approvalOf, CARLOS, DANIEL, PRIYA, roundOf } from '@/testing/fixtures/approvals';

const APPROVE = 'POST /api/v1/activities/A-1042/approval/approve';
const DECLINE = 'POST /api/v1/activities/A-1042/approval/decline';

/** Level 1 approved by Carlos: the round now waits at level 2 for Alex. */
const AT_LEVEL_2 = approvalOf({
  rounds: [
    roundOf({
      currentLevel: 2,
      levels: [
        { ...roundOf().levels[0], state: 'approved' },
        { ...roundOf().levels[1], state: 'waiting' },
      ],
    }),
  ],
});

function renderDialog(
  props: Partial<DecisionDialogProps> & Pick<DecisionDialogProps, 'mode'>,
  answers: Record<string, MockHandler> = {},
) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const onDecided = vi.fn();
  const rendered = renderWithProviders(
    <DecisionDialog approval={approvalOf()} onClose={onClose} onDecided={onDecided} {...props} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose, onDecided };
}

describe('DecisionDialog (feature 008, US2)', () => {
  it('shows the level, the next level, the owner note and the on-behalf sentence (FR-017)', () => {
    renderDialog({ mode: 'approve' });

    const dialog = screen.getByRole('alertdialog', { name: "Approve 'Launch film'?" });
    expect(dialog).toHaveTextContent('Level 1 of 2 · Carlos Mendes');
    expect(dialog).toHaveTextContent('Next: level 2 · Alex Morgan.');
    expect(dialog).toHaveTextContent('You own this activity.');
    expect(dialog).toHaveTextContent(
      'You are not an approver of level 1. Your decision will be recorded as made on behalf of Carlos Mendes.',
    );
    expect(screen.getByRole('textbox', { name: 'Comment (optional)' })).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });

  it('says when it is the last level, and leaves out the notes that do not apply', () => {
    renderDialog({
      mode: 'approve',
      approval: {
        ...AT_LEVEL_2,
        viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
      },
    });

    const dialog = screen.getByRole('alertdialog');
    expect(dialog).toHaveTextContent('Level 2 of 2 · Alex Morgan');
    expect(dialog).toHaveTextContent('This is the last level — the activity becomes Approved.');
    expect(dialog).not.toHaveTextContent('You own this activity.');
    expect(dialog).not.toHaveTextContent('You are not an approver');
  });

  it('approves the level the person saw and names the next approvers (FR-015)', async () => {
    const { user, api, onClose, onDecided } = renderDialog(
      { mode: 'approve' },
      { [APPROVE]: { body: AT_LEVEL_2 } },
    );

    await user.click(screen.getByRole('button', { name: 'Approve' }));

    expect(
      await screen.findByText(
        "Level 1 approved — 'Launch film' now waits for Alex Morgan (level 2).",
      ),
    ).toBeInTheDocument();
    expect(api.bodiesOf(APPROVE)).toEqual([{ roundId: 'AR-1001', level: 1 }]);
    expect(onClose).toHaveBeenCalled();
    expect(onDecided).toHaveBeenCalledWith(AT_LEVEL_2);
  });

  it('says the activity is approved after the last level', async () => {
    const approved: ActivityApproval = {
      ...AT_LEVEL_2,
      status: 'approved',
      rounds: [roundOf({ outcome: 'approved', currentLevel: 2 })],
    };
    const { user } = renderDialog(
      { mode: 'approve', approval: AT_LEVEL_2 },
      { [APPROVE]: { body: approved } },
    );

    await user.type(screen.getByRole('textbox', { name: 'Comment (optional)' }), 'Fine');
    await user.click(screen.getByRole('button', { name: 'Approve' }));

    expect(await screen.findByText("Activity 'Launch film' approved")).toBeInTheDocument();
  });

  it('needs a comment to decline, before anything is sent (FR-016)', async () => {
    const { user, api } = renderDialog({ mode: 'decline' });

    const dialog = screen.getByRole('alertdialog', { name: "Decline 'Launch film'?" });
    expect(dialog).toHaveTextContent(
      'The activity goes back to Priya Raman, who can fix it and re-submit.',
    );
    await user.click(screen.getByRole('button', { name: 'Decline activity' }));

    const comment = screen.getByRole('textbox', { name: /Why are you declining it\?/ });
    expect(comment).toHaveAccessibleDescription(
      expect.stringContaining('Add a comment explaining why the activity is declined.'),
    );
    expect(api.bodiesOf(DECLINE)).toEqual([]);
  });

  it('declines with the comment', async () => {
    const { user, api } = renderDialog(
      { mode: 'decline' },
      { [DECLINE]: { body: approvalOf({ status: 'declined' }) } },
    );

    await user.type(
      screen.getByRole('textbox', { name: /Why are you declining it\?/ }),
      'GL code missing',
    );
    await user.click(screen.getByRole('button', { name: 'Decline activity' }));

    expect(await screen.findByText("Activity 'Launch film' declined")).toBeInTheDocument();
    expect(api.bodiesOf(DECLINE)).toEqual([
      { roundId: 'AR-1001', level: 1, comment: 'GL code missing' },
    ]);
  });

  it('names the owner from the approval itself, for an approver who does not own it (D-214)', () => {
    const { api } = renderDialog({
      mode: 'decline',
      approval: approvalOf({
        ownerId: DANIEL.id,
        owner: DANIEL,
        viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
      }),
    });

    expect(screen.getByRole('alertdialog')).toHaveTextContent(
      'The activity goes back to Daniel Okoro, who can fix it and re-submit.',
    );
    expect(api.requests.some((request) => request.path === '/api/v1/people')).toBe(false);
  });

  it('names the owner it goes back to, or says "its owner" without one', () => {
    renderDialog({
      mode: 'decline',
      approval: approvalOf({
        ownerId: '',
        owner: null,
        viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
      }),
    });

    expect(screen.getByRole('alertdialog')).toHaveTextContent(
      'The activity goes back to its owner, who can fix it and re-submit.',
    );
  });

  it('explains a decision someone else made first, with its time (FR-018)', async () => {
    const changedAt = new Date().toISOString();
    const { user, onClose } = renderDialog(
      { mode: 'approve' },
      {
        [APPROVE]: {
          status: 409,
          body: {
            error: {
              code: 'version_conflict',
              message: 'Level 1 was already approved by Carlos Mendes',
              fields: [],
              conflict: { changedBy: CARLOS.name, changedAt },
            },
          },
        },
      },
    );

    await user.click(screen.getByRole('button', { name: 'Approve' }));

    expect(
      await screen.findByText(/^Level 1 was already approved by Carlos Mendes at \d\d:\d\d\.$/),
    ).toBeInTheDocument();
    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it('lists every approver of the level', () => {
    renderDialog({
      mode: 'approve',
      approval: approvalOf({
        rounds: [
          roundOf({
            levels: [{ ...roundOf().levels[0], approvers: [CARLOS, ALEX] }, roundOf().levels[1]],
          }),
        ],
      }),
    });

    expect(screen.getByRole('alertdialog')).toHaveTextContent(
      'Level 1 of 2 · Carlos Mendes, Alex Morgan',
    );
  });
});

/** Level 1 needs Carlos, Daniel and Alex (all must approve); Daniel approved already. */
function allLevel(viewer: Partial<ActivityApproval['viewer']>): ActivityApproval {
  return approvalOf({
    viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [], ...viewer },
    rounds: [
      roundOf({
        levels: [
          {
            number: 1,
            label: 'Level 1',
            approvers: [CARLOS, DANIEL, ALEX],
            state: 'waiting',
            decision: null,
            mode: 'all',
            approvedCount: 1,
            requiredCount: 3,
            decisions: [
              {
                action: 'approved',
                actor: { id: DANIEL.id, name: DANIEL.name },
                onBehalfOf: [],
                at: '2026-03-11T10:00:00Z',
                comment: '',
                capacity: 'approver',
                asDelegateOf: null,
              },
            ],
          },
          roundOf().levels[1],
        ],
      }),
    ],
  });
}

describe('DecisionDialog rules (feature 017, US2 and US3)', () => {
  it('on a level where all must approve, says who still waits after this approval (FR-015)', () => {
    renderDialog({ mode: 'approve', approval: allLevel({ capacity: 'approver' }) }, {});

    const dialog = screen.getByRole('alertdialog');
    // The signed-in test user approves for themself; Carlos and Alex still wait after it.
    expect(dialog).toHaveTextContent('1 of 3 already approved.');
    expect(dialog).toHaveTextContent('Still waiting for');
    expect(dialog).not.toHaveTextContent('Next: level 2');
  });

  it('says the level is complete when a backup approver approves it (FR-015, FR-019)', () => {
    renderDialog({ mode: 'approve', approval: allLevel({ capacity: 'backup' }) });

    const dialog = screen.getByRole('alertdialog');
    expect(dialog).toHaveTextContent('You approve as backup approver of level 1.');
    expect(dialog).toHaveTextContent('Level 1 is complete after this approval.');
    expect(dialog).toHaveTextContent('Next: level 2 · Alex Morgan.');
  });

  it('names whom a delegate approves for, without the on-behalf sentence (FR-019)', () => {
    renderDialog({
      mode: 'approve',
      approval: approvalOf({
        viewer: {
          isOwner: false,
          isCurrentApprover: false,
          onBehalfOf: [],
          capacity: 'delegate',
          asDelegateOf: [CARLOS],
        },
      }),
    });

    const dialog = screen.getByRole('alertdialog');
    expect(dialog).toHaveTextContent('You approve as delegate of Carlos Mendes.');
    expect(dialog).not.toHaveTextContent('on behalf of');
    expect(dialog).not.toHaveTextContent('You own this activity.');
  });

  it('passes levels that are not needed when naming the next level (FR-010)', () => {
    renderDialog({
      mode: 'approve',
      approval: approvalOf({
        viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
        rounds: [
          roundOf({
            levelCount: 3,
            levels: [
              roundOf().levels[0],
              {
                number: 2,
                label: 'Level 2',
                approvers: [],
                state: 'skipped',
                decision: null,
                ruleText: 'only when planned cost is at least $100,000.00',
              },
              {
                number: 3,
                label: 'Level 3',
                approvers: [PRIYA],
                state: 'notReached',
                decision: null,
              },
            ],
          }),
        ],
      }),
    });

    expect(screen.getByRole('alertdialog')).toHaveTextContent('Next: level 3 · Priya Raman.');
  });
});
