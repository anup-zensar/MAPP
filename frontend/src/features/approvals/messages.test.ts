import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '@/api/errors';
import {
  activitiesCount,
  alreadyDecidedMessage,
  approvedToast,
  approversText,
  chainStateText,
  declineBoxText,
  roundOutcomeText,
  submitConflictMessage,
} from './messages';
import { ALEX, CARLOS, DANIEL, roundOf } from '@/testing/fixtures/approvals';

function conflict(message: string, changedAt: string) {
  return new ApiError('conflict', {
    message,
    status: 409,
    conflict: { changedBy: 'Carlos Mendes', changedAt },
  });
}

describe('approval messages (feature 008, plan "User-visible messages")', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date('2026-03-12T16:00:00Z'));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('adds the time of an earlier decision, and the date when it was not today (FR-018)', () => {
    const message = 'Level 1 was already approved by Carlos Mendes';

    expect(alreadyDecidedMessage(conflict(message, '2026-03-12T14:32:00Z'))).toBe(
      'Level 1 was already approved by Carlos Mendes at 14:32.',
    );
    expect(alreadyDecidedMessage(conflict(message, '2026-03-11T09:05:00Z'))).toBe(
      'Level 1 was already approved by Carlos Mendes on 11 Mar 2026 at 09:05.',
    );
  });

  it('says who changed the activity before a submit (D-030)', () => {
    expect(submitConflictMessage(conflict('Changed', '2026-03-12T14:32:00Z'))).toBe(
      'Not submitted — Carlos Mendes changed this activity at 14:32. Check the changes and submit again.',
    );
  });

  it('keeps the API message when there is no conflict detail', () => {
    const error = new ApiError('badRequest', { message: 'Reload the page and try again.' });
    expect(alreadyDecidedMessage(error)).toBe('Reload the page and try again.');
  });

  it('writes counts, approvers and outcomes', () => {
    expect(activitiesCount(1)).toBe('1 activity');
    expect(activitiesCount(3)).toBe('3 activities');
    expect(approversText([CARLOS])).toBe('Approver: Carlos Mendes');
    expect(approversText([CARLOS, DANIEL])).toBe('Approvers: Carlos Mendes, Daniel Okoro');
    expect(roundOutcomeText(roundOf())).toBe('Waiting at level 1 of 2');
    expect(roundOutcomeText(roundOf({ outcome: 'cancelled' }))).toBe(
      'Cancelled — activity deleted',
    );
  });

  it('describes each chain state (FR-024)', () => {
    const decided = {
      number: 1,
      label: 'Level 1',
      approvers: [CARLOS],
      state: 'approved' as const,
      decision: {
        action: 'approved' as const,
        actor: { id: ALEX.id, name: ALEX.name },
        onBehalfOf: [{ id: CARLOS.id, name: CARLOS.name }],
        at: '2026-03-12T09:00:00Z',
        comment: '',
      },
    };
    expect(chainStateText(decided, 'open')).toBe(
      'Approved by Alex Morgan on behalf of Carlos Mendes on 12 Mar 2026',
    );
    const waiting = { ...decided, state: 'notDecided' as const, decision: null };
    expect(chainStateText(waiting, 'recalled')).toBe('Not decided — submission recalled');
    expect(chainStateText(waiting, 'restarted')).toBe('Not decided — approval restarted');
    expect(chainStateText(waiting, 'cancelled')).toBe('Not decided — activity deleted');
    expect(chainStateText({ ...waiting, state: 'notReached' }, 'open')).toBe('Not reached yet');
  });

  it('writes the latest decline box line (FR-016)', () => {
    expect(
      declineBoxText({
        actor: { id: DANIEL.id, name: DANIEL.name },
        onBehalfOf: [],
        at: '2026-03-11T14:32:00Z',
      }),
    ).toBe('Declined by Daniel Okoro on 11 Mar 2026, 14:32');
  });

  it('names the next level after an approval, or the final approval (FR-015)', () => {
    const next = roundOf({
      currentLevel: 2,
      levels: roundOf().levels.map((level) => ({ ...level, state: 'waiting' as const })),
    });
    expect(approvedToast('Launch film', 1, next)).toBe(
      "Level 1 approved — 'Launch film' now waits for Alex Morgan (level 2).",
    );
    expect(approvedToast('Launch film', 2, roundOf({ outcome: 'approved' }))).toBe(
      "Activity 'Launch film' approved",
    );
  });
});
