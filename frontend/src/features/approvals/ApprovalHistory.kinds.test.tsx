import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { roundOf, SUBMITTED } from '@/testing/fixtures/approvals';
import { ApprovalHistory } from './ApprovalHistory';

const rounds = () => screen.getAllByRole('heading', { level: 4 }).map((head) => head.textContent);

describe('ApprovalHistory of any record (feature 031)', () => {
  it('names the record type of rounds that are not activity rounds (031 FR-003)', () => {
    renderWithProviders(
      <ApprovalHistory
        rounds={[
          roundOf({ id: 'AR-3', number: 3, kind: 'modifiedPurchaseRequest' }),
          roundOf({ id: 'AR-1', number: 1, kind: 'activity', outcome: 'approved' }),
        ]}
      />,
      { user: testUser() },
    );

    expect(rounds()).toEqual([
      'Round 3 · Modified purchase request · Waiting at level 1 of 2 · Lubricants EU approval',
      'Round 1 · Approved · Lubricants EU approval',
    ]);
  });

  it('shows a replaced round and its entry as Replaced by a new submission (031 FR-017)', () => {
    renderWithProviders(
      <ApprovalHistory
        rounds={[
          roundOf({
            outcome: 'superseded',
            endedAt: '2026-03-11T09:00:00Z',
            history: [
              { ...SUBMITTED, id: 'AH-2', action: 'superseded', at: '2026-03-11T09:00:00Z' },
              SUBMITTED,
            ],
          }),
        ]}
      />,
      { user: testUser() },
    );

    expect(rounds()[0]).toContain('Replaced by a new submission');
    const history = screen.getByRole('region', { name: 'Approval history' });
    expect(within(history).getAllByRole('listitem')[1]).toHaveTextContent(
      'Replaced by a new submission',
    );
  });

  it('marks a decision the owner or submitter made as a self-approval (031 FR-016)', () => {
    renderWithProviders(
      <ApprovalHistory
        rounds={[
          roundOf({
            kind: 'purchaseRequest',
            history: [{ ...SUBMITTED, id: 'AH-2', action: 'approved', selfApproval: true }],
          }),
        ]}
      />,
      { user: testUser() },
    );

    expect(screen.getByText('Self-approval')).toBeInTheDocument();
  });
});
