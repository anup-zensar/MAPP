import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { EVENT, PRINT, TYPES } from '@/testing/fixtures/activity-types';
import { deleted, inUse } from '@/testing/fixtures/settings';
import { chooseAction, renderActivityTypesPage, rowOf } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

function table() {
  return screen.getByRole('table');
}

describe('ActivityTypesPage: the list (FR-002)', () => {
  it('shows the active types with their default cost, models and usage', async () => {
    renderActivityTypesPage();

    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    expect(screen.getByRole('heading', { level: 1, name: 'Activity types' })).toBeInTheDocument();
    expect(document.title).toBe('Activity types · SP&A One');
    expect(screen.getByText('The kinds of work activities can be.')).toBeInTheDocument();

    await screen.findByRole('table');
    const event = rowOf('Event');
    expect(event).toHaveTextContent('$5,000.00');
    expect(event).toHaveTextContent('Lubricants 1.0, Retail 0.9 (inactive)');
    expect(event).toHaveTextContent('Used by 4 activities · Allowed in 2 models');
    expect(within(event).getByText('Active')).toBeInTheDocument();
    expect(rowOf('Podcast')).toHaveTextContent('Not used yet');
    expect(within(table()).queryByText('Print advert')).not.toBeInTheDocument();
    expect(screen.getByText('2 of 3 activity types')).toBeInTheDocument();
  });

  it('filters by status and searches by name, keeping both in the address', async () => {
    const { user, router } = renderActivityTypesPage();
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Inactive 1' }));
    expect(within(rowOf('Print advert')).getByText('Inactive')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'All 3' }));
    await user.type(screen.getByRole('searchbox', { name: 'Search activity types' }), 'pod');

    expect(
      within(table())
        .getAllByRole('rowheader')
        .map((cell) => cell.textContent),
    ).toEqual(['Podcast']);
    expect(router.state.location.search).toBe('?status=all&q=pod');
  });

  it('says when nothing matches the filters, and clears them', async () => {
    const { user } = renderActivityTypesPage([], '/settings/activity-types?q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(within(table()).getAllByRole('rowheader')).toHaveLength(3);
  });

  it('invites to create the first activity type when there are none', async () => {
    renderActivityTypesPage([
      reply('GET /api/v1/activity-types', { body: { items: [], total: 0 } }),
    ]);

    expect(
      await screen.findByText('No activity types yet — create your first activity type'),
    ).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'New activity type' })).toHaveLength(2);
  });

  it('says when the activity types cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderActivityTypesPage([
      reply('GET /api/v1/activity-types', () => {
        calls += 1;
        return calls === 1
          ? {
              status: 500,
              body: {
                error: {
                  code: 'internal_error',
                  message: 'Something went wrong. Please try again.',
                  fields: [],
                },
              },
            }
          : { body: TYPES };
      }),
    ]);

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong.');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('table')).toBeInTheDocument();
  });
});

describe('ActivityTypesPage: details (US5 scenario 2)', () => {
  it('shows its default cost, its models, where it is used and who changed it', async () => {
    const { user, router } = renderActivityTypesPage();

    await user.click(await screen.findByRole('button', { name: 'Event' }));

    const panel = await screen.findByRole('dialog', { name: 'Event' });
    expect(router.state.location.search).toBe('?id=AT-1002');
    expect(panel).toHaveTextContent('Default planned cost$5,000.00');
    expect(within(panel).getByText('Lubricants 1.0')).toBeInTheDocument();
    expect(within(panel).getByText('Retail 0.9 (inactive)')).toBeInTheDocument();
    expect(panel).toHaveTextContent('Used by 4 activities · Allowed in 2 models');
    expect(panel).toHaveTextContent(
      'Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026',
    );
    expect(within(panel).getByRole('button', { name: 'Edit' })).toBeInTheDocument();
  });

  it('says when the activity type of the address no longer exists', async () => {
    renderActivityTypesPage([], '/settings/activity-types?id=AT-9999');

    const panel = await screen.findByRole('dialog', { name: 'Activity type' });
    expect(panel).toHaveTextContent(
      'This activity type no longer exists — someone may have deleted it.',
    );
    expect(within(panel).getByRole('button', { name: 'Back to Activity types' })).toBeVisible();
  });
});

describe('ActivityTypesPage: status and delete (US5 scenario 5, FR-004)', () => {
  it('moves an unused type to the recycle bin and offers Undo', async () => {
    const { user } = renderActivityTypesPage([
      reply(
        'DELETE /api/v1/activity-types/{activityTypeId}',
        {
          body: deleted('activity type', 'AT-1006', 'Podcast'),
        },
        { activityTypeId: 'AT-1006' },
      ),
    ]);

    await screen.findByRole('table');
    await chooseAction('Podcast', 'Delete', user);
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete activity type 'Podcast'?",
    });
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    await user.click(within(dialog).getByRole('button', { name: 'Delete activity type' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Activity type 'Podcast' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('offers Set inactive instead of deleting a type in use', async () => {
    const { user, api } = renderActivityTypesPage([
      reply(
        'POST /api/v1/activity-types/{activityTypeId}/status',
        { body: { ...EVENT, active: false } },
        { activityTypeId: 'AT-1002' },
      ),
    ]);

    await screen.findByRole('table');
    await chooseAction('Event', 'Delete', user);

    const dialog = await screen.findByRole('alertdialog', { name: "'Event' is in use" });
    expect(dialog).toHaveTextContent(
      "Used by 4 activities · Allowed in 2 models — set it inactive instead. New activities can't choose an inactive type; activities using it keep it.",
    );
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(await findMessages()).toHaveTextContent("Activity type 'Event' set inactive");
    expect(api.bodiesOf('POST /api/v1/activity-types/AT-1002/status')).toEqual([
      { active: false, version: 3 },
    ]);
    expect(api.requests.some((r) => r.method === 'DELETE')).toBe(false);
  });

  it('switches to the in-use dialog when the API finds the type in use', async () => {
    const message = "'Podcast' is in use (used by 1 activity) — set it inactive instead.";
    const { user } = renderActivityTypesPage([
      reply(
        'DELETE /api/v1/activity-types/{activityTypeId}',
        { status: 409, body: inUse(message) },
        { activityTypeId: 'AT-1006' },
      ),
    ]);

    await screen.findByRole('table');
    await chooseAction('Podcast', 'Delete', user);
    await user.click(await screen.findByRole('button', { name: 'Delete activity type' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'Podcast' is in use" });
    expect(dialog).toHaveTextContent(message);
  });

  it('reactivates an inactive type', async () => {
    const { user, api } = renderActivityTypesPage(
      [
        reply(
          'POST /api/v1/activity-types/{activityTypeId}/status',
          { body: { ...PRINT, active: true } },
          { activityTypeId: 'AT-1007' },
        ),
      ],
      '/settings/activity-types?status=inactive',
    );

    await screen.findByRole('table');
    await chooseAction('Print advert', 'Reactivate', user);

    expect(await findMessages()).toHaveTextContent("Activity type 'Print advert' reactivated");
    expect(api.bodiesOf('POST /api/v1/activity-types/AT-1007/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });
});
