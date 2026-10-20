import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { SubmitDialog } from './SubmitDialog';
import { ALEX, CARLOS, DANIEL, PRIYA, submissionOf } from './testData';

function renderDialog(submission = submissionOf(), commentError?: string) {
  const onConfirm = vi.fn();
  const onCancel = vi.fn();
  const rendered = renderWithProviders(
    <SubmitDialog
      submission={submission}
      busy={false}
      commentError={commentError}
      onConfirm={onConfirm}
      onCancel={onCancel}
    />,
    { user: testUser() },
  );
  return { ...rendered, onConfirm, onCancel };
}

describe('SubmitDialog (feature 008, US1)', () => {
  it('names the workflow and lists each level with its approvers (FR-011)', () => {
    renderDialog();

    const dialog = screen.getByRole('alertdialog', { name: "Submit 'Launch film' for approval?" });
    expect(dialog).toHaveTextContent('Approval workflow: Lubricants EU approval');
    const levels = screen.getAllByRole('listitem').map((item) => item.textContent);
    expect(levels).toEqual(['Level 1 · Carlos Mendes', 'Level 2 · Alex Morgan']);
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });

  it('sends the optional comment with Submit for approval', async () => {
    const { user, onConfirm } = renderDialog();

    await user.type(
      screen.getByRole('textbox', { name: 'Comment for the approvers (optional)' }),
      '  Ready for review ',
    );
    await user.click(screen.getByRole('button', { name: 'Submit for approval' }));

    expect(onConfirm).toHaveBeenCalledWith('Ready for review');
  });

  it('says which round a re-submit starts, and notes an unassigned planned cost (FR-020)', () => {
    renderDialog(
      submissionOf({
        kind: 'resubmit',
        roundNumber: 2,
        unassignedPlannedCost: { amount: '6800.00', currency: 'USD' },
      }),
    );

    const dialog = screen.getByRole('alertdialog', {
      name: "Re-submit 'Launch film' for approval?",
    });
    expect(dialog).toHaveTextContent('Round 2 starts at level 1.');
    expect(dialog).toHaveTextContent(
      '$6,800.00 of the planned cost is not yet assigned to a line item.',
    );
    expect(screen.getByRole('button', { name: 'Re-submit for approval' })).toBeInTheDocument();
  });

  it('shows the API problem with the comment under the field', () => {
    renderDialog(submissionOf(), 'Use 2,000 characters or fewer.');

    expect(
      screen.getByRole('textbox', { name: 'Comment for the approvers (optional)' }),
    ).toHaveAccessibleDescription(expect.stringContaining('Use 2,000 characters or fewer.'));
  });

  it('Cancel closes without sending', async () => {
    const { user, onConfirm, onCancel } = renderDialog();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onCancel).toHaveBeenCalled();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});

const outcome = (overrides: Record<string, unknown>) => ({
  number: 1,
  label: 'Level 1',
  outcome: 'decide' as const,
  approvers: [CARLOS],
  source: '',
  mode: 'any' as const,
  ruleText: '',
  excluded: [],
  backupUsed: false,
  noApprover: false,
  advice: '',
  ...overrides,
});

describe('SubmitDialog rules (feature 017, US1)', () => {
  it('says what each level will do, who is left out and the backup (FR-011, FR-013)', () => {
    renderDialog(
      submissionOf({
        outcomes: [
          outcome({ approvers: [DANIEL], source: 'line manager of Priya Raman' }),
          outcome({
            number: 2,
            label: 'Level 2',
            outcome: 'skipped',
            approvers: [],
            ruleText: 'only when planned cost is at least $100,000.00',
          }),
          outcome({
            number: 3,
            label: 'Level 3',
            approvers: [ALEX],
            backupUsed: true,
            excluded: [
              { person: PRIYA, reason: 'owner' },
              { person: CARLOS, reason: 'cannotApprove' },
            ],
          }),
          outcome({
            number: 4,
            label: 'Level 4',
            approvers: [],
            noApprover: true,
            advice:
              'Nobody else can approve level 4. Someone allowed to decide on behalf, such as an administrator, will have to approve it.',
          }),
        ],
      }),
    );

    const levels = screen.getAllByRole('listitem');
    expect(levels[0]).toHaveTextContent('Level 1 · Daniel Okoro (line manager of Priya Raman)');
    expect(levels[1]).toHaveTextContent(
      'Level 2 · Not needed — only when planned cost is at least $100,000.00',
    );
    expect(levels[2]).toHaveTextContent('Level 3 · Alex Morgan');
    expect(levels[2]).toHaveTextContent('Priya Raman is left out: owns this activity.');
    expect(levels[2]).toHaveTextContent(
      "Carlos Mendes is left out: can't approve — no approving role.",
    );
    expect(levels[2]).toHaveTextContent('The backup approver Alex Morgan approves this level.');
    expect(levels[3]).toHaveTextContent('Level 4 · No approver');
    expect(levels[3]).toHaveTextContent('Nobody else can approve level 4.');
    expect(screen.getByRole('alertdialog')).not.toHaveTextContent(
      'This activity will be approved automatically.',
    );
  });

  it('says when the rules approve the activity at once (FR-014)', () => {
    renderDialog(
      submissionOf({
        outcomes: [
          outcome({
            outcome: 'autoApproved',
            approvers: [],
            ruleText: 'planned cost is below $10,000.00',
          }),
        ],
        approvedAutomatically: true,
      }),
    );

    const dialog = screen.getByRole('alertdialog');
    expect(screen.getByRole('listitem')).toHaveTextContent(
      'Level 1 · Approved automatically — planned cost is below $10,000.00',
    );
    expect(dialog).toHaveTextContent('This activity will be approved automatically.');
    expect(screen.getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
  });
});
