import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { EVENT, PODCAST } from '@/testing/fixtures/activity-types';
import { LUBRICANTS, RETAIL, validationFailed, versionConflict } from '@/testing/fixtures/settings';
import { chooseAction, renderActivityTypesPage } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

describe('ActivityTypesPage: create and edit (US5 scenarios 1 and 2, FR-013)', () => {
  it('creates "Webinar" with a default cost, allowed in a model (scenario 1)', async () => {
    const webinar = { ...PODCAST, id: 'AT-1008', name: 'Webinar' };
    const { user, api } = renderActivityTypesPage([
      reply('POST /api/v1/activity-types', { status: 201, body: webinar }),
    ]);

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
    const { user } = renderActivityTypesPage([
      reply('POST /api/v1/activity-types', {
        status: 400,
        body: validationFailed([
          { field: 'name', message: "An activity type called 'Event' already exists." },
          {
            field: 'defaultPlannedCost',
            message: 'Enter an amount of 0 or more, with at most two decimals.',
          },
        ]),
      }),
    ]);

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
    const { user, api } = renderActivityTypesPage([
      reply(
        'PATCH /api/v1/activity-types/{activityTypeId}',
        {
          body: { ...EVENT, defaultPlannedCost: { amount: '7500.00', currency: 'USD' } },
        },
        { activityTypeId: 'AT-1002' },
      ),
    ]);

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
    const { user, api } = renderActivityTypesPage([
      reply(
        'PATCH /api/v1/activity-types/{activityTypeId}',
        { body: { ...EVENT, defaultPlannedCost: null } },
        { activityTypeId: 'AT-1002' },
      ),
    ]);

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
    const { user, api } = renderActivityTypesPage([
      reply(
        'POST /api/v1/activity-types/{activityTypeId}/impact',
        {
          body: { warnings: [{ kind: 'activityTypeRemoved', message: warning, count: 4 }] },
        },
        { activityTypeId: 'AT-1002' },
      ),
      reply(
        'PATCH /api/v1/activity-types/{activityTypeId}',
        { body: { ...EVENT, models: [] } },
        { activityTypeId: 'AT-1002' },
      ),
    ]);

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
    const { user, api } = renderActivityTypesPage([
      reply(
        'PATCH /api/v1/activity-types/{activityTypeId}',
        { body: PODCAST },
        { activityTypeId: 'AT-1006' },
      ),
    ]);

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
    const { user, api } = renderActivityTypesPage([
      reply(
        'PATCH /api/v1/activity-types/{activityTypeId}',
        () => {
          patches += 1;
          return patches === 1
            ? { status: 409, body: versionConflict() }
            : { body: { ...PODCAST, name: 'Podcast series' } };
        },
        { activityTypeId: 'AT-1006' },
      ),
      reply(
        'GET /api/v1/activity-types/{activityTypeId}',
        { body: { ...PODCAST, version: 4 } },
        { activityTypeId: 'AT-1006' },
      ),
    ]);

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
