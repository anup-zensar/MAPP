import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  deleted,
  inUse,
  LUBRICANTS,
  MADE,
  MODELS,
  NOT_USED,
  RETAIL,
  validationFailed,
  versionConflict,
} from '@/testing/fixtures/settings';
import { ActivityTypesPage } from './ActivityTypesPage';

type ActivityType = components['schemas']['ActivityType'];

/** Event: 5,000 USD by default, allowed in Lubricants 1.0 and the inactive Retail 0.9, in use. */
const EVENT: ActivityType = {
  id: 'AT-1002',
  name: 'Event',
  defaultPlannedCost: { amount: '5000.00', currency: 'USD' },
  models: [
    { id: LUBRICANTS.id, name: LUBRICANTS.name, active: true },
    { id: RETAIL.id, name: RETAIL.name, active: false },
  ],
  active: true,
  usage: {
    inUse: true,
    summary: 'Used by 4 activities · Allowed in 2 models',
    parts: [
      { kind: 'activity', text: 'Used by 4 activities', count: 4, inRecycleBin: 0 },
      { kind: 'model', text: 'Allowed in 2 models', count: 2, inRecycleBin: 0 },
    ],
    records: [],
  },
  version: 3,
  ...MADE,
};

/** Podcast: no default cost, no models, unused. */
const PODCAST: ActivityType = {
  ...EVENT,
  id: 'AT-1006',
  name: 'Podcast',
  defaultPlannedCost: null,
  models: [],
  usage: NOT_USED,
  version: 1,
};

/** Print advert: inactive and unused. */
const PRINT: ActivityType = { ...PODCAST, id: 'AT-1007', name: 'Print advert', active: false };

const TYPES = { items: [EVENT, PODCAST, PRINT], total: 3 };

function renderTypes(
  answers: Record<string, MockHandler> = {},
  route = '/settings/activity-types',
) {
  const api = mockApi({
    'GET /api/v1/activity-types': { body: TYPES },
    'GET /api/v1/models': { body: MODELS },
    ...answers,
  });
  return {
    ...renderWithProviders(<ActivityTypesPage />, {
      route,
      user: testUser(),
      path: '/settings/activity-types',
    }),
    api,
  };
}

function table() {
  return screen.getByRole('table');
}

function row(name: string): HTMLElement {
  return within(table()).getByRole('rowheader', { name }).closest('tr') as HTMLElement;
}

async function chooseAction(
  name: string,
  action: string,
  user: ReturnType<typeof renderTypes>['user'],
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

describe('ActivityTypesPage: the list (FR-002)', () => {
  it('shows the active types with their default cost, models and usage', async () => {
    renderTypes();

    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    expect(screen.getByRole('heading', { level: 1, name: 'Activity types' })).toBeInTheDocument();
    expect(document.title).toBe('Activity types · MAPP');
    expect(screen.getByText('The kinds of work activities can be.')).toBeInTheDocument();

    await screen.findByRole('table');
    const event = row('Event');
    expect(event).toHaveTextContent('$5,000.00');
    expect(event).toHaveTextContent('Lubricants 1.0, Retail 0.9 (inactive)');
    expect(event).toHaveTextContent('Used by 4 activities · Allowed in 2 models');
    expect(within(event).getByText('Active')).toBeInTheDocument();
    expect(row('Podcast')).toHaveTextContent('Not used yet');
    expect(within(table()).queryByText('Print advert')).not.toBeInTheDocument();
    expect(screen.getByText('2 of 3 activity types')).toBeInTheDocument();
  });

  it('filters by status and searches by name, keeping both in the address', async () => {
    const { user, router } = renderTypes();
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Inactive 1' }));
    expect(within(row('Print advert')).getByText('Inactive')).toBeInTheDocument();
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
    const { user } = renderTypes({}, '/settings/activity-types?q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(within(table()).getAllByRole('rowheader')).toHaveLength(3);
  });

  it('invites to create the first activity type when there are none', async () => {
    renderTypes({ 'GET /api/v1/activity-types': { body: { items: [], total: 0 } } });

    expect(
      await screen.findByText('No activity types yet — create your first activity type'),
    ).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'New activity type' })).toHaveLength(2);
  });

  it('says when the activity types cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderTypes({
      'GET /api/v1/activity-types': () => {
        calls += 1;
        return calls === 1
          ? {
              status: 500,
              body: {
                error: {
                  code: 'internal_error',
                  message: 'Something went wrong. Please try again.',
                },
              },
            }
          : { body: TYPES };
      },
    });

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong.');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('table')).toBeInTheDocument();
  });
});

describe('ActivityTypesPage: details (US5 scenario 2)', () => {
  it('shows its default cost, its models, where it is used and who changed it', async () => {
    const { user, router } = renderTypes();

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
    renderTypes({}, '/settings/activity-types?id=AT-9999');

    const panel = await screen.findByRole('dialog', { name: 'Activity type' });
    expect(panel).toHaveTextContent(
      'This activity type no longer exists — someone may have deleted it.',
    );
    expect(within(panel).getByRole('button', { name: 'Back to Activity types' })).toBeVisible();
  });
});

describe('ActivityTypesPage: create and edit (US5 scenarios 1 and 2, FR-013)', () => {
  it('creates "Webinar" with a default cost, allowed in a model (scenario 1)', async () => {
    const webinar = { ...PODCAST, id: 'AT-1008', name: 'Webinar' };
    const { user, api } = renderTypes({
      'POST /api/v1/activity-types': { status: 201, body: webinar },
    });

    await user.click(screen.getByRole('button', { name: 'New activity type' }));
    const form = await screen.findByRole('dialog', { name: 'New activity type' });
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'Webinar');
    const cost = within(form).getByRole('textbox', { name: /^Default planned cost/ });
    expect(cost).toHaveAccessibleDescription(
      expect.stringContaining('Offered when someone picks this type for an activity.'),
    );
    await user.type(cost, '5,000');
    const models = await within(form).findByRole('group', { name: /^Allowed in models/ });
    // Inactive models can't be newly chosen.
    expect(within(models).queryByRole('checkbox', { name: /Retail 0.9/ })).not.toBeInTheDocument();
    await user.click(within(models).getByRole('checkbox', { name: 'Lubricants 1.0' }));
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Activity type 'Webinar' created");
    expect(api.bodiesOf('POST /api/v1/activity-types')).toEqual([
      {
        name: 'Webinar',
        defaultPlannedCost: { amount: '5000.00', currency: 'USD' },
        modelIds: [LUBRICANTS.id],
      },
    ]);
  });

  it('shows every problem at once and saves nothing (FR-005)', async () => {
    const { user } = renderTypes({
      'POST /api/v1/activity-types': {
        status: 400,
        body: validationFailed([
          { field: 'name', message: "An activity type called 'Event' already exists." },
          {
            field: 'defaultPlannedCost',
            message: 'Enter an amount of 0 or more, with at most two decimals.',
          },
        ]),
      },
    });

    await user.click(screen.getByRole('button', { name: 'New activity type' }));
    const form = await screen.findByRole('dialog', { name: 'New activity type' });
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'event');
    await user.type(within(form).getByRole('textbox', { name: /^Default planned cost/ }), '-5');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('2 problems need your attention')).toBeInTheDocument();
    expect(
      within(form).getAllByText("An activity type called 'Event' already exists."),
    ).toHaveLength(2);
    expect(
      within(form).getAllByText('Enter an amount of 0 or more, with at most two decimals.'),
    ).toHaveLength(2);
  });

  it('says a changed default cost applies from now on, and saves only that (scenario 2)', async () => {
    const { user, api } = renderTypes({
      'PATCH /api/v1/activity-types/AT-1002': {
        body: { ...EVENT, defaultPlannedCost: { amount: '7500.00', currency: 'USD' } },
      },
    });

    await chooseAction('Event', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit activity type' });
    const cost = within(form).getByRole('textbox', { name: /^Default planned cost/ });
    expect(cost).toHaveValue('5,000.00');
    expect(form).not.toHaveTextContent('Applies when someone picks this type from now on.');
    await user.clear(cost);
    await user.type(cost, '7500');
    expect(form).toHaveTextContent('Applies when someone picks this type from now on.');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Activity type 'Event' saved");
    expect(api.bodiesOf('PATCH /api/v1/activity-types/AT-1002')).toEqual([
      { defaultPlannedCost: { amount: '7500.00', currency: 'USD' }, version: 3 },
    ]);
  });

  it('removes the default cost with null', async () => {
    const { user, api } = renderTypes({
      'PATCH /api/v1/activity-types/AT-1002': { body: { ...EVENT, defaultPlannedCost: null } },
    });

    await chooseAction('Event', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit activity type' });
    await user.clear(within(form).getByRole('textbox', { name: /^Default planned cost/ }));
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await findMessages();
    expect(api.bodiesOf('PATCH /api/v1/activity-types/AT-1002')).toEqual([
      { defaultPlannedCost: null, version: 3 },
    ]);
  });

  it('warns before a model stops allowing the type, then saves the new list (FR-012)', async () => {
    const warning =
      "In plans using 'Lubricants 1.0', 4 activities use 'Event'. They keep it; new activities can't choose it.";
    const { user, api } = renderTypes({
      'POST /api/v1/activity-types/AT-1002/impact': {
        body: { warnings: [{ kind: 'activityTypeRemoved', message: warning, count: 4 }] },
      },
      'PATCH /api/v1/activity-types/AT-1002': { body: { ...EVENT, models: [] } },
    });

    await chooseAction('Event', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit activity type' });
    const models = await within(form).findByRole('group', { name: /^Allowed in models/ });
    // The inactive model already allowing it stays, marked.
    expect(within(models).getByRole('checkbox', { name: 'Retail 0.9 (inactive)' })).toBeChecked();
    await user.click(within(models).getByRole('checkbox', { name: 'Lubricants 1.0' }));
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    const impact = await screen.findByRole('alertdialog', {
      name: "Save changes to activity type 'Event'?",
    });
    expect(impact).toHaveTextContent(warning);
    expect(within(impact).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(api.bodiesOf('PATCH /api/v1/activity-types/AT-1002')).toEqual([]);
    await user.click(within(impact).getByRole('button', { name: 'Save changes' }));

    expect(await findMessages()).toHaveTextContent("Activity type 'Event' saved");
    const change = { modelIds: [RETAIL.id], version: 3 };
    expect(api.bodiesOf('POST /api/v1/activity-types/AT-1002/impact')).toEqual([change]);
    expect(api.bodiesOf('PATCH /api/v1/activity-types/AT-1002')).toEqual([change]);
  });

  it('adds a model without an impact check', async () => {
    const { user, api } = renderTypes({
      'PATCH /api/v1/activity-types/AT-1006': { body: PODCAST },
    });

    await chooseAction('Podcast', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit activity type' });
    await user.click(await within(form).findByRole('checkbox', { name: 'Mobility 1.0' }));
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await findMessages();
    expect(api.bodiesOf('PATCH /api/v1/activity-types/AT-1006')).toEqual([
      { modelIds: ['MOD-1002'], version: 1 },
    ]);
    expect(api.requests.some((request) => request.path.endsWith('/impact'))).toBe(false);
  });

  it('keeps the typed values after a conflict and saves again over it (FR-007)', async () => {
    let patches = 0;
    const { user, api } = renderTypes({
      'PATCH /api/v1/activity-types/AT-1006': () => {
        patches += 1;
        return patches === 1
          ? { status: 409, body: versionConflict() }
          : { body: { ...PODCAST, name: 'Podcast series' } };
      },
      'GET /api/v1/activity-types/AT-1006': { body: { ...PODCAST, version: 4 } },
    });

    await chooseAction('Podcast', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit activity type' });
    const name = within(form).getByRole('textbox', { name: /^Name/ });
    await user.clear(name);
    await user.type(name, 'Podcast series');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this activity type on 14 Mar 2026 at 10:42',
    );
    expect(name).toHaveValue('Podcast series');
    await waitFor(() =>
      expect(
        api.requests.some((r) => r.method === 'GET' && r.path === '/api/v1/activity-types/AT-1006'),
      ).toBe(true),
    );
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Activity type 'Podcast series' saved");
    expect(api.bodiesOf('PATCH /api/v1/activity-types/AT-1006')).toEqual([
      { name: 'Podcast series', version: 1 },
      { name: 'Podcast series', version: 4 },
    ]);
  });
});

describe('ActivityTypesPage: status and delete (US5 scenario 5, FR-004)', () => {
  it('moves an unused type to the recycle bin and offers Undo', async () => {
    const { user } = renderTypes({
      'DELETE /api/v1/activity-types/AT-1006': {
        body: deleted('activity type', 'AT-1006', 'Podcast'),
      },
    });

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
    const { user, api } = renderTypes({
      'POST /api/v1/activity-types/AT-1002/status': { body: { ...EVENT, active: false } },
    });

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
    const { user } = renderTypes({
      'DELETE /api/v1/activity-types/AT-1006': { status: 409, body: inUse(message) },
    });

    await screen.findByRole('table');
    await chooseAction('Podcast', 'Delete', user);
    await user.click(await screen.findByRole('button', { name: 'Delete activity type' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'Podcast' is in use" });
    expect(dialog).toHaveTextContent(message);
  });

  it('reactivates an inactive type', async () => {
    const { user, api } = renderTypes(
      { 'POST /api/v1/activity-types/AT-1007/status': { body: { ...PRINT, active: true } } },
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
