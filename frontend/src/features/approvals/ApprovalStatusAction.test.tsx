import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { ApprovalStatusAction } from './ApprovalStatusAction';
import type { ApprovalActivity } from './approvalTypes';
import { approvalOf, submissionOf } from './testData';

const SUBMISSION = 'GET /api/v1/activities/A-1042/approval/submission';
const SUBMIT = 'POST /api/v1/activities/A-1042/approval/submit';

const CREATED: ApprovalActivity = {
  id: 'A-1042',
  name: 'Launch film',
  status: 'created',
  version: 3,
};

function renderAction(activity: ApprovalActivity, answers: Record<string, MockHandler> = {}) {
  const api = mockApi(answers);
  const onShowProblems = vi.fn();
  const rendered = renderWithProviders(
    <ApprovalStatusAction activity={activity} onShowProblems={onShowProblems} />,
    { user: testUser() },
  );
  return { ...rendered, api, onShowProblems };
}

describe('ApprovalStatusAction (feature 008, US1)', () => {
  it('offers Submit for a Created activity and Re-submit for a Declined one (FR-009)', () => {
    renderAction(CREATED);
    expect(screen.getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
  });

  it('offers Re-submit for a Declined activity (FR-020)', () => {
    renderAction({ ...CREATED, status: 'declined' });
    expect(screen.getByRole('button', { name: 'Re-submit for approval' })).toBeInTheDocument();
  });

  it('offers nothing for an Approved activity', () => {
    renderAction({ ...CREATED, status: 'approved' });
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('offers Decline and Approve side by side to an approver and opens the dialog (FR-014)', async () => {
    const approver = approvalOf({
      viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
    });
    const { user } = renderAction(
      { ...CREATED, status: 'submitted' },
      { 'GET /api/v1/activities/A-1042/approval': { body: approver } },
    );

    const approve = await screen.findByRole('button', { name: 'Approve' });
    expect(screen.getAllByRole('button').map((button) => button.textContent)).toEqual([
      'Decline',
      'Approve',
    ]);
    // Secondary while Edit is the page's primary action (design system §11 rule 1).
    expect(approve.className).not.toMatch(/primary/);
    await user.click(approve);

    expect(screen.getByRole('alertdialog', { name: "Approve 'Launch film'?" })).toBeInTheDocument();
  });

  it('makes Approve the primary action when the page asks for it (design system §6.3)', async () => {
    mockApi({
      'GET /api/v1/activities/A-1042/approval': {
        body: approvalOf({ viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] } }),
      },
    });
    renderWithProviders(
      <ApprovalStatusAction
        activity={{ ...CREATED, status: 'submitted' }}
        onShowProblems={vi.fn()}
        primaryDecision
      />,
      { user: testUser() },
    );

    expect((await screen.findByRole('button', { name: 'Approve' })).className).toMatch(/primary/);
  });

  it('never offers the owner Approve as a button (functional review A-01)', async () => {
    const { api } = renderAction(
      { ...CREATED, status: 'submitted' },
      { 'GET /api/v1/activities/A-1042/approval': { body: approvalOf() } },
    );

    await waitFor(() =>
      expect(api.requests.some((request) => request.path.endsWith('/approval'))).toBe(true),
    );
    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('with problems, shows every one of them in 006 list and sends nothing (FR-010)', async () => {
    const problems = [
      { field: 'ownerId', group: 'activity', lineItemId: '', message: 'Owner — needed to submit' },
      {
        field: 'approvalWorkflowId',
        group: 'activity',
        lineItemId: '',
        message: 'Approval workflow — needed to submit',
      },
    ] as const;
    const { user, api, onShowProblems } = renderAction(CREATED, {
      [SUBMISSION]: { body: submissionOf({ ready: false, problems: [...problems] }) },
    });

    await user.click(screen.getByRole('button', { name: 'Submit for approval' }));

    await waitFor(() => expect(onShowProblems).toHaveBeenCalledWith(2));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf(SUBMIT)).toEqual([]);
  });

  it('when ready, confirms, submits with the version and says so (FR-011)', async () => {
    const { user, api } = renderAction(CREATED, {
      [SUBMISSION]: { body: submissionOf() },
      [SUBMIT]: { body: approvalOf() },
    });

    await user.click(screen.getByRole('button', { name: 'Submit for approval' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Submit 'Launch film' for approval?",
    });
    await user.type(screen.getByRole('textbox'), 'Please check the dates');
    await user.click(within(dialog).getByRole('button', { name: 'Submit for approval' }));

    expect(
      await screen.findByText("Activity 'Launch film' submitted for approval"),
    ).toBeInTheDocument();
    expect(dialog).not.toBeInTheDocument();
    expect(api.bodiesOf(SUBMIT)).toEqual([{ version: 3, comment: 'Please check the dates' }]);
  });

  it('says re-submitted after a re-submit', async () => {
    const { user } = renderAction(
      { ...CREATED, status: 'declined' },
      {
        [SUBMISSION]: { body: submissionOf({ kind: 'resubmit', roundNumber: 2 }) },
        [SUBMIT]: { body: approvalOf() },
      },
    );

    await user.click(screen.getByRole('button', { name: 'Re-submit for approval' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Re-submit for approval' }));

    expect(
      await screen.findByText("Activity 'Launch film' re-submitted for approval"),
    ).toBeInTheDocument();
  });

  it('a refused submit with problems opens 006 list with their number (FR-010)', async () => {
    const { user, onShowProblems } = renderAction(CREATED, {
      [SUBMISSION]: { body: submissionOf() },
      [SUBMIT]: {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Fix the problems.',
            fields: [{ field: 'ownerId', message: 'Owner — needed to submit' }],
          },
        },
      },
    });

    await user.click(screen.getByRole('button', { name: 'Submit for approval' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Submit for approval' }));

    await waitFor(() => expect(onShowProblems).toHaveBeenCalledWith(1));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('says who changed the activity when the version is old (D-030)', async () => {
    const changedAt = new Date().toISOString();
    const { user } = renderAction(CREATED, {
      [SUBMISSION]: { body: submissionOf() },
      [SUBMIT]: {
        status: 409,
        body: {
          error: {
            code: 'version_conflict',
            message: 'Someone changed it.',
            fields: [],
            conflict: { changedBy: 'Daniel Okoro', changedAt },
          },
        },
      },
    });

    await user.click(screen.getByRole('button', { name: 'Submit for approval' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Submit for approval' }));

    expect(
      await screen.findByText(/^Not submitted — Daniel Okoro changed this activity at \d\d:\d\d\./),
    ).toBeInTheDocument();
  });
});
