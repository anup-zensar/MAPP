import { screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { ApprovalTab } from './approvalsApi';
import { ApprovalsTable } from './ApprovalsTable';
import { rowOf } from './testData';

function renderTable(rows = [rowOf()], tab: ApprovalTab = 'waiting') {
  const onOpen = vi.fn();
  const rendered = renderWithProviders(<ApprovalsTable rows={rows} tab={tab} onOpen={onOpen} />, {
    user: testUser(),
  });
  return { ...rendered, onOpen };
}

const headers = () => screen.getAllByRole('columnheader').map((header) => header.textContent);

describe('ApprovalsTable (feature 008, US3)', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date('2026-03-13T09:00:00Z'));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the columns of the waiting and open tabs (FR-030)', () => {
    renderTable();

    expect(headers()).toEqual([
      'Activity',
      'Plan › campaign',
      'Owner',
      'Planned cost',
      'Workflow',
      'Level',
      'Submitted',
    ]);
    const row = screen.getByRole('row', { name: /Launch film/ });
    expect(row).toHaveTextContent('A-1042');
    expect(row).toHaveTextContent('FY2026 Lubricants Europe › Spring launch');
    expect(row).toHaveTextContent('Priya Raman');
    expect(row).toHaveTextContent('$50,000.00');
    expect(row).toHaveTextContent('Lubricants EU approval');
    expect(row).toHaveTextContent('Level 1 of 2 · Carlos Mendes');
    expect(row).toHaveTextContent('10 Mar 2026waiting 3 days');
    expect(row).not.toHaveTextContent('Start date passed');
  });

  it('marks a passed start date and an inactive workflow (edge case)', () => {
    renderTable([
      rowOf({
        startDatePassed: true,
        workflow: { id: 'WF-1001', name: 'Lubricants EU approval', active: false },
      }),
    ]);

    const row = screen.getByRole('row', { name: /Launch film/ });
    expect(within(row).getByText('Start date passed')).toBeInTheDocument();
    expect(row).toHaveTextContent('Lubricants EU approval (inactive)');
  });

  it('adds my decision and the status in Decided by me, with the level only while open', () => {
    renderTable(
      [
        rowOf({
          activity: {
            id: 'A-1042',
            name: 'Launch film',
            status: 'approved',
            displayStatus: 'inProgress',
          },
          round: { ...rowOf().round, outcome: 'approved' },
          waitingSince: null,
          myDecision: { action: 'approved', at: '2026-03-12T09:00:00Z', roundNumber: 1, level: 2 },
        }),
      ],
      'decided',
    );

    expect(headers()).toContain('My decision');
    expect(headers()).toContain('Status');
    const row = screen.getByRole('row', { name: /Launch film/ });
    expect(row).toHaveTextContent('Approved · 12 Mar 2026');
    expect(row).toHaveTextContent('In progress');
    expect(row).not.toHaveTextContent('Level 1 of 2');
    expect(row).not.toHaveTextContent('waiting');
  });

  it('shows my decision on a round that a change restarted, back in Created (FR-022)', () => {
    renderTable(
      [
        rowOf({
          activity: {
            id: 'A-1042',
            name: 'Launch film',
            status: 'created',
            displayStatus: 'created',
          },
          round: { ...rowOf().round, outcome: 'restarted', currentLevel: 2 },
          waitingSince: null,
          myDecision: { action: 'approved', at: '2026-03-11T09:00:00Z', roundNumber: 1, level: 1 },
        }),
      ],
      'decided',
    );

    const row = screen.getByRole('row', { name: /Launch film/ });
    expect(row).toHaveTextContent('Approved · 11 Mar 2026');
    expect(row).toHaveTextContent('Created');
    expect(row).not.toHaveTextContent('Level 2 of 2');
    expect(row).not.toHaveTextContent('waiting');
  });

  it('opens the preview from the activity name', async () => {
    const { user, onOpen } = renderTable();

    await user.click(screen.getByRole('button', { name: 'Launch film' }));

    expect(onOpen).toHaveBeenCalledWith('A-1042');
  });
});
