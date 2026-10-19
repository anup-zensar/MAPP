import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
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
