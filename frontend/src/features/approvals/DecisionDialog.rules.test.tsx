import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderDialog } from '@/testing/fixtures/approval-screens';
import {
  ALEX,
  approvalOf,
  CARLOS,
  DANIEL,
  LEVEL_1,
  LEVEL_2,
  PRIYA,
  roundOf,
} from '@/testing/fixtures/approvals';
import type { ActivityApproval } from './api';
/** Level 1 needs Carlos, Daniel and Alex (all must approve); Daniel approved already. */
function allLevel(
  viewer: Partial<ActivityApproval['viewer']>,
  approvers = [CARLOS, DANIEL, ALEX],
): ActivityApproval {
  return approvalOf({
    viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [], ...viewer },
    rounds: [
      roundOf({
        levels: [
          {
            number: 1,
            label: 'Level 1',
            approvers,
            state: 'waiting',
            decision: null,
            mode: 'all',
            approvedCount: 1,
            requiredCount: approvers.length,
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
          LEVEL_2,
        ],
      }),
    ],
  });
}

describe('DecisionDialog rules (feature 017, US2 and US3)', () => {
  it('on a level where all must approve, says who still waits after this approval (FR-015)', () => {
    renderDialog({ mode: 'approve', approval: allLevel({ capacity: 'approver' }) }, []);

    const dialog = screen.getByRole('alertdialog');
    // The signed-in test user approves for themself; Carlos and Alex still wait after it.
    expect(dialog).toHaveTextContent('1 of 3 already approved.');
    expect(dialog).toHaveTextContent('Still waiting for');
    expect(dialog).not.toHaveTextContent('Next: level 2');
  });

  it('says the level is complete when the last approver who waits approves it (FR-015)', () => {
    renderDialog({
      mode: 'approve',
      approval: allLevel({ capacity: 'approver' }, [DANIEL, PRIYA]),
    });

    const dialog = screen.getByRole('alertdialog');
    // The signed-in test user is Priya Raman, the only approver still waiting.
    expect(dialog).toHaveTextContent('1 of 2 already approved.');
    expect(dialog).toHaveTextContent('Level 1 is complete after this approval.');
    expect(dialog).not.toHaveTextContent('Still waiting for');
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
              LEVEL_1,
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
