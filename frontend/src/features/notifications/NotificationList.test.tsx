import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { DECLINED, NOW, notificationOf, pageOf } from '@/testing/fixtures/notifications';
import { reply } from '@/testing/replies';
import { NotificationList } from './NotificationList';

const READ_ALL = 'POST /api/v1/notifications/read-all';

function renderList(answers: HttpHandler[]) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const rendered = renderWithProviders(<NotificationList titleId="title" onClose={onClose} />, {
    user: testUser(),
  });
  // Fake timers keep ticking; this user waits with them (README "Testing tips").
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
  return { ...rendered, api, onClose, user };
}

const items = () => screen.getAllByRole('listitem');

describe('NotificationList (feature 008, US6)', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(NOW);
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('lists the notifications newest first with title, text, time and unread marker (FR-033)', async () => {
    renderList([
      reply('GET /api/v1/notifications', { body: pageOf([notificationOf(), DECLINED]) }),
    ]);

    const unread = await screen.findByRole('button', { name: /^Approval needed/ });
    expect(unread).toHaveAccessibleName(
      "Approval needed Priya Raman submitted 'Launch film' (A-1042) 5 minutes ago Unread",
    );
    expect(within(unread).getByText('5 minutes ago')).toHaveAttribute(
      'title',
      '12 Mar 2026, 08:55',
    );
    expect(within(unread).getByText('5 minutes ago')).toHaveAttribute(
      'dateTime',
      '2026-03-12T08:55:00Z',
    );
    expect(items()[1]).toHaveTextContent('Activity declined');
    expect(items()[1]).toHaveTextContent('"The agency fee is missing its GL code."');
    expect(items()[1]).toHaveTextContent('yesterday');
    expect(items()[1]).not.toHaveTextContent('Unread');
    expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeInTheDocument();
  });

  it('shows the kinds of approval rules: automatic, reminder, escalation (017 FR-032)', async () => {
    const MAPP = { id: 'system', name: 'MAPP' };
    renderList([
      reply('GET /api/v1/notifications', {
        body: pageOf([
          notificationOf({
            id: 'NT-1010',
            kind: 'autoApproved',
            title: 'Activity approved automatically',
            text: "'Launch film' (A-1042) was approved automatically: planned cost is below $10,000.00",
            actor: MAPP,
          }),
          notificationOf({
            id: 'NT-1011',
            kind: 'reminder',
            title: 'Reminder — approval waiting 3 days',
            text: "'Launch film' (A-1042) has waited 3 days for you at level 1",
            actor: MAPP,
          }),
          notificationOf({
            id: 'NT-1012',
            kind: 'escalated',
            title: 'Approval escalated to you',
            text: "'Launch film' (A-1042) waited 5 days at level 1. You can now approve it as backup approver.",
            actor: MAPP,
          }),
        ]),
      }),
    ]);

    expect(
      await screen.findByRole('button', { name: /^Activity approved automatically/ }),
    ).toHaveTextContent('was approved automatically: planned cost is below $10,000.00');
    expect(
      screen.getByRole('button', { name: /^Reminder — approval waiting 3 days/ }),
    ).toHaveTextContent('has waited 3 days for you at level 1');
    expect(screen.getByRole('button', { name: /^Approval escalated to you/ })).toHaveTextContent(
      'You can now approve it as backup approver.',
    );
  });

  it('marks a notification read and opens its activity at the Approval section (FR-034)', async () => {
    let read = false;
    const { user, api, router, onClose } = renderList([
      reply('GET /api/v1/notifications', () => ({ body: pageOf([notificationOf({ read })]) })),
      reply(
        'POST /api/v1/notifications/{notificationId}/read',
        () => {
          read = true;
          return { body: notificationOf({ read: true }) };
        },
        { notificationId: 'NT-1003' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: /^Approval needed/ }));

    expect(onClose).toHaveBeenCalled();
    await waitFor(() => expect(router.state.location.pathname).toBe('/activities/A-1042'));
    expect(router.state.location.hash).toBe('#approval');
    await waitFor(() => expect(api.requests.some((r) => r.path.endsWith('/read'))).toBe(true));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /^Approval needed/ })).not.toHaveTextContent(
        'Unread',
      ),
    );
    expect(screen.queryByRole('button', { name: 'Mark all as read' })).not.toBeInTheDocument();
  });

  it('opens an activity again without marking a read notification twice', async () => {
    const { user, api, router } = renderList([
      reply('GET /api/v1/notifications', { body: pageOf([DECLINED]) }),
    ]);

    await user.click(await screen.findByRole('button', { name: /^Activity declined/ }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/activities/A-1031'));
    expect(api.requests.some((r) => r.method === 'POST')).toBe(false);
  });

  it('says when the activity was deleted and opens the recycle bin (edge case)', async () => {
    const gone = notificationOf({
      read: true,
      activity: { id: 'A-1042', name: 'Launch film', exists: false },
    });
    const { user, router } = renderList([
      reply('GET /api/v1/notifications', { body: pageOf([gone]) }),
    ]);

    const item = await screen.findByRole('button', { name: /^Approval needed/ });
    expect(item).toHaveTextContent('This activity was deleted — find it in the recycle bin.');
    await user.click(item);

    await waitFor(() => expect(router.state.location.pathname).toBe('/recycle-bin'));
  });

  it('marks all as read: the count goes, the items stay (FR-034)', async () => {
    let allRead = false;
    const second = notificationOf({
      id: 'NT-1004',
      title: 'Level 1 approved',
      kind: 'levelApproved',
    });
    const { user, api } = renderList([
      reply('GET /api/v1/notifications', () => ({
        body: pageOf([notificationOf({ read: allRead }), { ...second, read: allRead }]),
      })),
      reply('POST /api/v1/notifications/read-all', () => {
        allRead = true;
        return { body: { marked: 2, unread: 0 } };
      }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Mark all as read' }));

    await waitFor(() =>
      expect(screen.queryByRole('button', { name: 'Mark all as read' })).not.toBeInTheDocument(),
    );
    expect(items()).toHaveLength(2);
    expect(screen.queryByText('Unread')).not.toBeInTheDocument();
    expect(api.bodiesOf(READ_ALL)).toHaveLength(1);
  });

  it('says so when Mark all as read fails, and keeps the items unread', async () => {
    const { user } = renderList([
      reply('GET /api/v1/notifications', { body: pageOf([notificationOf()]) }),
      reply('POST /api/v1/notifications/read-all', { status: 500 }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Mark all as read' }));

    expect(
      await screen.findByText(
        "MAPP can't reach its server right now. Check that it is running and try again.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('Unread')).toBeInTheDocument();
  });

  it('says "You\'re all caught up." when there are none (FR-034)', async () => {
    renderList([reply('GET /api/v1/notifications', { body: pageOf([]) })]);

    expect(await screen.findByText("You're all caught up.")).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Mark all as read' })).not.toBeInTheDocument();
  });

  it('shows loading shapes while the notifications are on their way', () => {
    renderList([reply('GET /api/v1/notifications', () => new Promise(() => {}))]);

    expect(screen.getByRole('heading', { name: 'Notifications' })).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
  });

  it('says when the notifications could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderList([
      reply('GET /api/v1/notifications', () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: pageOf([notificationOf()]) };
      }),
    ]);

    expect(await screen.findByRole('alert')).toHaveTextContent("Notifications couldn't be loaded.");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('button', { name: /^Approval needed/ })).toBeInTheDocument();
  });
});
