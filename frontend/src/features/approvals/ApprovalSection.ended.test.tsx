import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { card, NONE, renderSection } from '@/testing/fixtures/approval-screens';
import {
  approvalOf,
  DANIEL,
  LEVEL_1,
  LEVEL_2,
  roundOf,
  SUBMITTED,
} from '@/testing/fixtures/approvals';
import { reply } from '@/testing/replies';

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
const ENDED_LEVELS = [{ ...LEVEL_1, state: 'notDecided' as const }, LEVEL_2];

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
      [
        reply(
          'POST /api/v1/activities/{activityId}/approval/recall',
          () => {
            recalled = true;
            return { body: recalledApproval };
          },
          { activityId: 'A-1042' },
        ),
      ],
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
