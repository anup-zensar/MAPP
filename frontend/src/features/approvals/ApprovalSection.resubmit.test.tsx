import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { card, NONE, renderSection } from '@/testing/fixtures/approval-screens';
import { approvalOf, submissionOf } from '@/testing/fixtures/approvals';
import { reply } from '@/testing/replies';

const SUBMIT = 'POST /api/v1/activities/A-1042/approval/submit';
const AVAILABLE = { available: true, reason: '' };

const WAITING = approvalOf({
  actions: {
    submit: NONE,
    resubmit: AVAILABLE,
    approve: NONE,
    decline: NONE,
    recall: AVAILABLE,
  },
});

const REPLACING = submissionOf({
  kind: 'resubmit',
  roundNumber: 2,
  replaces: { roundNumber: 1, level: 1, levelLabel: 'Level 1' },
});

describe('Re-submit a Submitted activity (feature 031, US1)', () => {
  it('offers Re-submit for approval next to Recall while the activity waits (031 FR-017)', async () => {
    renderSection({ body: WAITING });

    const resubmit = await within(card()).findByRole('button', { name: 'Re-submit for approval' });
    const buttons = within(card())
      .getAllByRole('button')
      .map((button) => button.textContent);
    expect(buttons.indexOf('Recall submission')).toBe(buttons.indexOf(resubmit.textContent) + 1);
    await expectNoAxeViolations(card());
  });

  it('states which round ends and at which level before anything changes (031 FR-018)', async () => {
    const { user, api } = renderSection({ body: WAITING }, [
      reply(
        'GET /api/v1/activities/{activityId}/approval/submission',
        { body: REPLACING },
        { activityId: 'A-1042' },
      ),
    ]);

    await user.click(await within(card()).findByRole('button', { name: 'Re-submit for approval' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "Re-submit 'Launch film' for approval?",
    });
    expect(dialog).toHaveTextContent(
      'Round 1, waiting at Level 1, ends and round 2 starts at the first level that needs a decision.',
    );
    expect(dialog).not.toHaveTextContent('Round 2 starts at level 1.');
    expect(api.bodiesOf(SUBMIT)).toEqual([]);
  });

  it('re-submits with the version shown and says so (031 FR-020)', async () => {
    const { user, api } = renderSection({ body: WAITING }, [
      reply(
        'GET /api/v1/activities/{activityId}/approval/submission',
        { body: REPLACING },
        { activityId: 'A-1042' },
      ),
      reply(
        'POST /api/v1/activities/{activityId}/approval/submit',
        { body: approvalOf() },
        { activityId: 'A-1042' },
      ),
    ]);

    await user.click(await within(card()).findByRole('button', { name: 'Re-submit for approval' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Re-submit for approval' }));

    expect(
      await screen.findByText("Activity 'Launch film' re-submitted for approval"),
    ).toBeInTheDocument();
    expect(api.bodiesOf(SUBMIT)).toEqual([{ version: 3 }]);
  });
});
