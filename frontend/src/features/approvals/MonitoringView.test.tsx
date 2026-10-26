import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ALEX, monitoringOf } from '@/testing/fixtures/approvals';
import { reply } from '@/testing/replies';
import { MonitoringView } from './MonitoringView';

const RUN = 'POST /api/v1/approvals/escalations/run';

function renderView(answers: HttpHandler[]) {
  const api = mockApi(answers);
  const onOpen = vi.fn();
  const rendered = renderWithProviders(<MonitoringView onOpen={onOpen} />, { user: testUser() });
  return { ...rendered, api, onOpen };
}

describe('MonitoringView (feature 017, US6)', () => {
  it('shows the cards, the ageing with counts in words, waiting by level and the overdue rows (FR-030)', async () => {
    const { user, onOpen } = renderView([
      reply('GET /api/v1/approvals/monitoring', { body: monitoringOf() }),
    ]);

    expect(await screen.findByText('Average time to decision')).toBeInTheDocument();
    expect(screen.getByText('2.4 days')).toBeInTheDocument();
    for (const label of [
      'Overdue',
      'Escalated',
      'Average time to decision',
      'Approved automatically',
    ]) {
      expect(screen.getByRole('button', { name: `About ${label}` })).toBeInTheDocument();
    }

    const ageing = screen.getByRole('region', { name: 'How long approvals have waited' });
    const buckets = within(ageing)
      .getAllByRole('listitem')
      .map((item) => item.textContent);
    expect(buckets).toEqual([
      'Under 1 day1',
      '1–2 days0',
      '3–5 days2',
      '6–10 days1',
      'More than 10 days0',
    ]);

    const byLevel = screen.getByRole('region', { name: 'Waiting by level' });
    const headers = within(byLevel)
      .getAllByRole('columnheader')
      .map((header) => header.textContent);
    expect(headers).toEqual(['Workflow', 'Level', 'Waiting', 'Overdue', 'Oldest waiting']);
    expect(within(byLevel).getByRole('row', { name: /Lubricants EU approval/ })).toHaveTextContent(
      'Level 131',
    );

    const overdue = screen.getByRole('region', { name: 'Overdue' });
    expect(within(overdue).getByRole('row', { name: /Launch film/ })).toHaveTextContent('Overdue');
    await user.click(within(overdue).getByRole('button', { name: 'Launch film' }));
    expect(onOpen).toHaveBeenCalledWith('A-1042');
    // Only someone who may change approval workflows runs reminders and escalations.
    expect(
      screen.queryByRole('button', { name: 'Run reminders and escalations' }),
    ).not.toBeInTheDocument();
  });

  it('says when nothing waits, and when nothing is overdue', async () => {
    renderView([
      reply('GET /api/v1/approvals/monitoring', {
        body: monitoringOf({
          cards: {
            overdue: 0,
            escalated: 0,
            averageDecisionDays: null,
            decidedThisMonth: 0,
            autoApprovedThisMonth: 0,
          },
          ageing: monitoringOf().ageing.map((bucket) => ({ ...bucket, count: 0 })),
          byLevel: [],
          overdue: [],
        }),
      }),
    ]);

    expect(await screen.findByText('Nothing is waiting for approval.')).toBeInTheDocument();
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('says Nothing is overdue while approvals wait in time', async () => {
    renderView([
      reply('GET /api/v1/approvals/monitoring', { body: monitoringOf({ overdue: [] }) }),
    ]);

    expect(await screen.findByText('Nothing is overdue.')).toBeInTheDocument();
  });

  it('says when the figures could not be loaded, with Try again', async () => {
    renderView([
      reply('GET /api/v1/approvals/monitoring', {
        status: 500,
        body: { error: { code: 'internal_error', message: 'Boom', fields: [] } },
      }),
    ]);

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});

describe('RunEscalationsButton (feature 017, US5)', () => {
  it('asks first, then says what was sent and what could not be escalated (FR-028)', async () => {
    const { user, api } = renderView([
      reply('GET /api/v1/approvals/monitoring', {
        body: monitoringOf({ canRunEscalations: true }),
      }),
      reply('POST /api/v1/approvals/escalations/run', {
        body: {
          reminded: 2,
          escalated: 1,
          events: [
            {
              activity: { type: 'activity', id: 'A-1042', name: 'Launch film' },
              roundNumber: 1,
              level: 1,
              action: 'escalated',
              to: [ALEX],
              waitedDays: 6,
            },
          ],
          notEscalated: [
            {
              activity: { type: 'activity', id: 'A-1004', name: 'Trade magazine ads' },
              roundNumber: 2,
              level: 1,
              action: 'notEscalated',
              to: [],
              waitedDays: 7,
            },
          ],
        },
      }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Run reminders and escalations' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Run reminders and escalations now?',
    });
    expect(dialog).toHaveTextContent('Each happens once per level.');
    expect(api.bodiesOf(RUN)).toEqual([]);
    await user.click(within(dialog).getByRole('button', { name: 'Run now' }));

    expect(await screen.findByText('2 reminders sent, 1 escalation')).toBeInTheDocument();
    expect(
      screen.getByText(
        "Not escalated: 'Trade magazine ads' (A-1004), level 1: no backup approver can take it.",
      ),
    ).toBeInTheDocument();
    expect(api.requests.filter((r) => r.path === '/api/v1/approvals/escalations/run')).toHaveLength(
      1,
    );
  });

  it('says when nothing was due', async () => {
    const { user } = renderView([
      reply('GET /api/v1/approvals/monitoring', {
        body: monitoringOf({ canRunEscalations: true }),
      }),
      reply('POST /api/v1/approvals/escalations/run', {
        body: { reminded: 0, escalated: 0, events: [], notEscalated: [] },
      }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Run reminders and escalations' }));
    await user.click(await screen.findByRole('button', { name: 'Run now' }));

    expect(await screen.findByText('No reminders, no escalations')).toBeInTheDocument();
  });
});
