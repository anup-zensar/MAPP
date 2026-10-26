import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { ApprovalChain } from './ApprovalChain';
import { ALEX, CARLOS, DANIEL, PRIYA, roundOf } from './testData';

describe('ApprovalChain (feature 008, US2)', () => {
  it('shows each level in order with its approvers and state (FR-024)', () => {
    const round = roundOf({
      currentLevel: 2,
      levels: [
        {
          number: 1,
          label: 'Level 1',
          approvers: [CARLOS],
          state: 'approved',
          decision: {
            action: 'approved',
            actor: { id: DANIEL.id, name: DANIEL.name },
            onBehalfOf: [{ id: CARLOS.id, name: CARLOS.name }],
            at: '2026-03-12T09:00:00Z',
            comment: '',
          },
        },
        {
          number: 2,
          label: 'Global brand',
          approvers: [PRIYA, { ...ALEX, active: false }],
          state: 'waiting',
          decision: null,
        },
      ],
    });
    renderWithProviders(<ApprovalChain round={round} />, { user: testUser() });

    const [first, second] = screen
      .getAllByRole('listitem')
      .filter((item) => item.parentElement?.tagName === 'OL');
    expect(first).toHaveTextContent('L1');
    expect(first).toHaveTextContent(
      'Approved by Daniel Okoro on behalf of Carlos Mendes on 12 Mar 2026',
    );
    expect(second).toHaveTextContent('Global brand (level 2)');
    expect(second).toHaveTextContent('Priya Raman (you)');
    expect(second).toHaveTextContent('Alex Morgan (inactive)');
    expect(second).toHaveTextContent('Waiting');
  });

  it('lists the levels to come without states', () => {
    renderWithProviders(
      <ApprovalChain
        upcomingLevels={[{ number: 1, name: '', label: 'Level 1', approvers: [CARLOS] }]}
      />,
      { user: testUser() },
    );

    const level = screen.getAllByRole('listitem')[0];
    expect(level).toHaveTextContent('Level 1');
    expect(level).toHaveTextContent('Carlos Mendes');
    expect(level).not.toHaveTextContent('Waiting');
  });
});

describe('ApprovalChain rules (feature 017, US2 and US7)', () => {
  it('shows the new states, the progress of an all level and what happened (FR-033)', () => {
    const round = roundOf({
      levelCount: 4,
      levels: [
        {
          number: 1,
          label: 'Level 1',
          approvers: [],
          state: 'autoApproved',
          decision: null,
          ruleText: 'planned cost is below $10,000.00',
        },
        {
          number: 2,
          label: 'Level 2',
          approvers: [CARLOS, DANIEL],
          state: 'waiting',
          decision: null,
          mode: 'all',
          source: 'role Approver',
          approvedCount: 1,
          requiredCount: 2,
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
          remindedAt: '2026-03-13T06:00:00Z',
          escalatedAt: '2026-03-15T06:00:00Z',
          escalatedTo: ALEX,
          excluded: [{ person: PRIYA, reason: 'owner' }],
        },
        {
          number: 3,
          label: 'Level 3',
          approvers: [],
          state: 'skipped',
          decision: null,
          ruleText: 'only when planned cost is at least $100,000.00',
        },
        {
          number: 4,
          label: 'Level 4',
          approvers: [],
          state: 'notReached',
          decision: null,
          source: 'line manager of Priya Raman',
          noApprover: true,
          advice: 'Nobody else can approve level 4.',
        },
      ],
    });
    renderWithProviders(<ApprovalChain round={round} />, { user: testUser() });

    const levels = screen.getAllByRole('listitem').filter((item) => item.querySelector('p'));
    expect(levels[0]).toHaveTextContent(
      'Approved automatically — planned cost is below $10,000.00',
    );
    expect(levels[1]).toHaveTextContent('Role: Approver');
    expect(levels[1]).toHaveTextContent('1 of 2 approved · waiting for Carlos Mendes');
    expect(levels[1]).toHaveTextContent('Reminder sent on 13 Mar 2026');
    expect(levels[1]).toHaveTextContent('Escalated to Alex Morgan on 15 Mar 2026');
    expect(levels[1]).toHaveTextContent('Priya Raman was left out: owns this activity.');
    expect(levels[1]).toHaveTextContent('Daniel OkoroApproved');
    expect(levels[2]).toHaveTextContent(
      'Not needed — only when planned cost is at least $100,000.00',
    );
    expect(levels[3]).toHaveTextContent('Line manager of Priya Raman');
    expect(levels[3]).toHaveTextContent('No approver. Nobody else can approve level 4.');
  });
});

describe('ApprovalChain access (feature 014, FR-028)', () => {
  it('flags an approver who holds no role that approves', () => {
    renderWithProviders(
      <ApprovalChain
        round={roundOf({
          levels: [
            {
              number: 1,
              label: 'Level 1',
              approvers: [{ ...CARLOS, canApprove: false }],
              state: 'waiting',
              decision: null,
            },
          ],
        })}
      />,
      { user: testUser() },
    );

    expect(document.body).toHaveTextContent("Carlos Mendes(can't approve — no approving role)");
  });
});
