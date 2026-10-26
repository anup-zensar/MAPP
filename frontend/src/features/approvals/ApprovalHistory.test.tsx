import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ApprovalHistory } from './ApprovalHistory';
import { ALEX, CARLOS, DANIEL, roundOf, SUBMITTED } from '@/testing/fixtures/approvals';

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

const MAPP = { id: 'system', name: 'MAPP' };
const ref = (person: { id: string; name: string }) => ({ id: person.id, name: person.name });

describe('ApprovalHistory rules entries (feature 017, US7)', () => {
  it('writes the new entries, capacities, waited days and the rules facts (FR-034)', () => {
    const round = roundOf({
      facts: {
        plannedCost: { amount: '50000.00', currency: 'USD' },
        activityType: { type: 'activityType', id: 'AT-1001', name: 'Event' },
        businessUnits: [{ id: 'OP-1001', label: 'Lubricants', active: true }],
      },
      history: [
        {
          ...SUBMITTED,
          id: 'AH-6',
          action: 'approved',
          level: 2,
          levelLabel: 'Level 2',
          actor: ref(CARLOS),
          capacity: 'delegate',
          asDelegateOf: ref(DANIEL),
          at: '2026-03-15T10:00:00Z',
          comment: '',
        },
        {
          ...SUBMITTED,
          id: 'AH-5',
          action: 'escalated',
          level: 2,
          levelLabel: 'Level 2',
          actor: MAPP,
          escalatedTo: ref(ALEX),
          waitedDays: 5,
          at: '2026-03-14T06:00:00Z',
          comment: '',
        },
        {
          ...SUBMITTED,
          id: 'AH-4',
          action: 'reminded',
          level: 2,
          levelLabel: 'Level 2',
          actor: MAPP,
          waitedDays: 1,
          at: '2026-03-12T06:00:00Z',
          comment: '',
        },
        {
          ...SUBMITTED,
          id: 'AH-3',
          action: 'approved',
          level: 1,
          actor: ref(ALEX),
          capacity: 'backup',
          at: '2026-03-11T10:00:00Z',
          comment: '',
        },
        {
          ...SUBMITTED,
          id: 'AH-2',
          action: 'skipped',
          level: 3,
          levelLabel: 'Level 3',
          approvers: [],
          actor: MAPP,
          rule: 'planned cost is at least $100,000.00',
          comment: '',
        },
        {
          ...SUBMITTED,
          id: 'AH-1',
          action: 'autoApproved',
          level: 1,
          approvers: [],
          actor: MAPP,
          rule: 'activity type is Event',
          comment: '',
        },
      ],
    });
    renderWithProviders(<ApprovalHistory rounds={[round]} />, { user: testUser() });

    const history = screen.getByRole('region', { name: 'Approval history' });
    expect(history).toHaveTextContent(
      'Rules applied to: planned cost $50,000.00 · Event · Lubricants',
    );
    const items = within(history)
      .getAllByRole('listitem')
      .filter((item) => item.parentElement?.closest('li'));
    expect(items[0]).toHaveTextContent('Approved · Level 2');
    expect(items[0]).toHaveTextContent('Carlos Mendes as delegate of Daniel Okoro');
    expect(items[1]).toHaveTextContent('Escalated to Alex Morgan · Level 2');
    expect(items[1]).toHaveTextContent('MAPP');
    expect(items[1]).toHaveTextContent('waited 5 days');
    expect(items[2]).toHaveTextContent('Reminder sent · Level 2');
    expect(items[2]).toHaveTextContent('waited 1 day');
    expect(items[3]).toHaveTextContent('Alex Morgan as backup approver');
    expect(items[4]).toHaveTextContent('Not needed · Level 3');
    expect(items[4]).toHaveTextContent('Only when planned cost is at least $100,000.00');
    expect(items[5]).toHaveTextContent('Approved automatically · Level 1');
    expect(items[5]).toHaveTextContent('Activity type is Event');
  });

  it('reads rounds and entries from before the rules as before (data-model §10)', () => {
    renderWithProviders(<ApprovalHistory rounds={[roundOf()]} />, { user: testUser() });

    const history = screen.getByRole('region', { name: 'Approval history' });
    expect(history).not.toHaveTextContent('Rules applied to');
    expect(history).toHaveTextContent('Submitted · Level 1');
    expect(history).toHaveTextContent('Priya Raman');
  });
});
