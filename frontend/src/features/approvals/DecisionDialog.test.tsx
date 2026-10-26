import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderDialog } from '@/testing/fixtures/approval-screens';
import {
  ALEX,
  approvalOf,
  CARLOS,
  DANIEL,
  LEVEL_1,
  LEVEL_2,
  roundOf,
} from '@/testing/fixtures/approvals';
import { reply } from '@/testing/replies';
import type { ActivityApproval } from './api';

const APPROVE = 'POST /api/v1/activities/A-1042/approval/approve';
const DECLINE = 'POST /api/v1/activities/A-1042/approval/decline';

/** Level 1 approved by Carlos: the round now waits at level 2 for Alex. */
const AT_LEVEL_2 = approvalOf({
  rounds: [
    roundOf({
      currentLevel: 2,
      levels: [
        { ...LEVEL_1, state: 'approved' },
        { ...LEVEL_2, state: 'waiting' },
      ],
    }),
  ],
});

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
    const { user, api, onClose, onDecided } = renderDialog({ mode: 'approve' }, [
      reply(
        'POST /api/v1/activities/{activityId}/approval/approve',
        { body: AT_LEVEL_2 },
        { activityId: 'A-1042' },
      ),
    ]);

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
    const { user } = renderDialog({ mode: 'approve', approval: AT_LEVEL_2 }, [
      reply(
        'POST /api/v1/activities/{activityId}/approval/approve',
        { body: approved },
        { activityId: 'A-1042' },
      ),
    ]);

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
    const { user, api } = renderDialog({ mode: 'decline' }, [
      reply(
        'POST /api/v1/activities/{activityId}/approval/decline',
        { body: approvalOf({ status: 'declined' }) },
        { activityId: 'A-1042' },
      ),
    ]);

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
    const { user, onClose } = renderDialog({ mode: 'approve' }, [
      reply(
        'POST /api/v1/activities/{activityId}/approval/approve',
        {
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
        { activityId: 'A-1042' },
      ),
    ]);

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
            levels: [{ ...LEVEL_1, approvers: [CARLOS, ALEX] }, LEVEL_2],
          }),
        ],
      }),
    });

    expect(screen.getByRole('alertdialog')).toHaveTextContent(
      'Level 1 of 2 · Carlos Mendes, Alex Morgan',
    );
  });
});
