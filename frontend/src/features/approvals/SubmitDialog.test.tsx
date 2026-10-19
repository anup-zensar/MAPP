import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { SubmitDialog } from './SubmitDialog';
import { submissionOf } from './testData';

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
