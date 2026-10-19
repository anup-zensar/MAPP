import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { ApprovalHistory } from './ApprovalHistory';
import { roundOf, SUBMITTED } from './testData';

const entry = (id: string, at: string, roundNumber = 2) => ({
  ...SUBMITTED,
  id,
  at,
  roundNumber,
  action: 'approved' as const,
});

describe('ApprovalHistory (feature 008, US2)', () => {
  it('compact: only the latest 3 entries, each with its round (FR-025)', () => {
    const rounds = [
      roundOf({
        number: 2,
        history: [entry('AH-4', '2026-03-12T10:00:00Z'), entry('AH-3', '2026-03-12T09:00:00Z')],
      }),
      roundOf({
        id: 'AR-1000',
        history: [entry('AH-2', '2026-03-11T10:00:00Z', 1), SUBMITTED],
      }),
    ];
    renderWithProviders(<ApprovalHistory rounds={rounds} compact />, { user: testUser() });

    const history = screen.getByRole('region', { name: 'Approval history' });
    const items = within(history).getAllByRole('listitem');
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveTextContent('Approved · Round 2 · Level 1 · 12 Mar 2026, 10:00');
    expect(items[2]).toHaveTextContent('Round 1');
    expect(within(history).queryByRole('heading', { level: 4 })).not.toBeInTheDocument();
  });

  it('says when there is no history yet', () => {
    renderWithProviders(<ApprovalHistory rounds={[]} />, { user: testUser() });

    expect(screen.getByText('No approval history yet.')).toBeInTheDocument();
  });
});
