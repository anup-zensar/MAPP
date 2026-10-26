import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  ACTIVITY_TYPES,
  deleted,
  inUse,
  LUBRICANTS,
  MOBILITY,
  MODELS,
  RETAIL,
} from '@/testing/fixtures/settings';
import { ModelsPage } from './ModelsPage';

function renderModels(answers: Record<string, MockHandler> = {}, route = '/settings/models') {
  const api = mockApi({
    'GET /api/v1/models': { body: MODELS },
    'GET /api/v1/activity-types': { body: ACTIVITY_TYPES },
    ...answers,
  });
  return {
    ...renderWithProviders(<ModelsPage />, { route, user: testUser(), path: '/settings/models' }),
    api,
  };
}

function table() {
  return screen.getByRole('table');
}

function row(name: string): HTMLElement {
  const cell = within(table()).getByRole('rowheader', { name });
  return cell.closest('tr') as HTMLElement;
}

async function chooseAction(
  name: string,
  action: string,
  user: ReturnType<typeof renderModels>['user'],
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

describe('ModelsPage: the list (FR-002)', () => {
  it('shows the breadcrumb, the header and the active models with their key facts', async () => {
    renderModels();

    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    expect(screen.getByRole('heading', { level: 1, name: 'Models' })).toBeInTheDocument();
    expect(document.title).toBe('Models · MAPP');
    expect(screen.getByRole('button', { name: 'New model' })).toBeInTheDocument();

    const lubricants = row(await screen.findByText('Lubricants 1.0').then(() => 'Lubricants 1.0'));
    expect(lubricants).toHaveTextContent('2.00 % → 5.00 %');
    expect(lubricants).toHaveTextContent('$10,000.00');
    expect(lubricants).toHaveTextContent('Used by 2 plans');
    expect(within(lubricants).getByText('Active')).toBeInTheDocument();
    expect(row('Mobility 1.0')).toHaveTextContent('Not used yet');
    // Inactive models are behind the Inactive and All filters.
    expect(within(table()).queryByText('Retail 0.9')).not.toBeInTheDocument();
    expect(screen.getByText('2 of 3 models')).toBeInTheDocument();
  });

  it('filters by status and searches by name, keeping both in the address', async () => {
    const { user, router } = renderModels();
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Inactive 1' }));
    expect(within(table()).getByText('Retail 0.9')).toBeInTheDocument();
    expect(within(table()).queryByText('Lubricants 1.0')).not.toBeInTheDocument();
    expect(within(row('Retail 0.9')).getByText('Inactive')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'All 3' }));
    await user.type(screen.getByRole('searchbox', { name: 'Search models' }), 'mob');

    expect(
      within(table())
        .getAllByRole('rowheader')
        .map((cell) => cell.textContent),
    ).toEqual(['Mobility 1.0']);
    expect(screen.getByText('1 of 3 models')).toBeInTheDocument();
    expect(router.state.location.search).toBe('?status=all&q=mob');
  });

  it('says when nothing matches the filters, and clears them', async () => {
    const { user } = renderModels({}, '/settings/models?q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(within(table()).getAllByRole('rowheader')).toHaveLength(3);
  });

  it('invites to create the first model when there are none', async () => {
    renderModels({ 'GET /api/v1/models': { body: { items: [], total: 0 } } });

    expect(await screen.findByText('No models yet — create your first model')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'New model' })).toHaveLength(2);
  });

  it('says when the models cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderModels({
      'GET /api/v1/models': () => {
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
          : { body: MODELS };
      },
    });

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong.');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('table')).toBeInTheDocument();
  });
});

describe('ModelsPage: details (US1 scenario 3)', () => {
  it('opens a model from its name: every field, the example, the plans using it and who changed it', async () => {
    const { user, router } = renderModels();

    await user.click(await screen.findByRole('button', { name: 'Lubricants 1.0' }));

    const panel = await screen.findByRole('dialog', { name: 'Lubricants 1.0' });
    expect(router.state.location.search).toBe('?id=MOD-1001');
    expect(panel).toHaveTextContent('INQ→MQL rate2.00 %');
    expect(panel).toHaveTextContent('MQL→deal rate5.00 %');
    expect(panel).toHaveTextContent('Average deal size$10,000.00');
    expect(within(panel).getByText('Event')).toBeInTheDocument();
    expect(within(panel).getByText('Used by 2 plans')).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(
      within(panel).getByText('250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD revenue'),
    ).toBeInTheDocument();
    expect(
      within(panel).getByText(
        'Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026',
      ),
    ).toBeInTheDocument();

    await user.click(within(panel).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(router.state.location.search).toBe(''));
  });

  it('shows an inactive allowed type as "(inactive)"', async () => {
    renderModels({}, '/settings/models?id=MOD-1003');

    const panel = await screen.findByRole('dialog', { name: 'Retail 0.9' });
    expect(within(panel).getByText('Print advert (inactive)')).toBeInTheDocument();
  });

  it('says when the model of the address no longer exists, with Back to Models', async () => {
    const { user, router } = renderModels({}, '/settings/models?id=MOD-9999');

    const panel = await screen.findByRole('dialog');
    expect(panel).toHaveTextContent('This model no longer exists — someone may have deleted it.');
    await user.click(within(panel).getByRole('button', { name: 'Back to Models' }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('Duplicate opens an unsaved new model named "<name> (copy)" (FR-009)', async () => {
    const { user } = renderModels({}, '/settings/models?id=MOD-1001');

    await screen.findByRole('dialog', { name: 'Lubricants 1.0' });
    await chooseAction('Lubricants 1.0', 'Duplicate', user);

    const form = await screen.findByRole('dialog', { name: 'New model' });
    expect(within(form).getByRole('textbox', { name: /^Name/ })).toHaveValue(
      'Lubricants 1.0 (copy)',
    );
    expect(within(form).getByRole('textbox', { name: /^INQ→MQL rate/ })).toHaveValue('2.00');
    expect(within(form).getByRole('checkbox', { name: 'Event' })).toBeChecked();
  });
});

describe('ModelsPage: create and edit (US1 scenarios 1, 4, 5, 6)', () => {
  it('creates a model and offers View in the message (scenario 1)', async () => {
    const created = { ...MOBILITY, id: 'MOD-1004', name: 'Fuels 1.0', usage: LUBRICANTS.usage };
    const { user, api, router } = renderModels({
      'POST /api/v1/models': { status: 201, body: created },
    });

    await user.click(screen.getByRole('button', { name: 'New model' }));
    const form = await screen.findByRole('dialog', { name: 'New model' });
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'Fuels 1.0');
    await user.type(within(form).getByRole('textbox', { name: /^INQ→MQL rate/ }), '2');
    await user.type(within(form).getByRole('textbox', { name: /^MQL→deal rate/ }), '5');
    await user.type(within(form).getByRole('textbox', { name: /^Average deal size/ }), '10,000');
    await user.click(await within(form).findByRole('checkbox', { name: 'Event' }));
    await user.click(within(form).getByRole('checkbox', { name: 'Webinar' }));
    expect(
      within(form).getByText('250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD revenue'),
    ).toBeInTheDocument();
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Model 'Fuels 1.0' created");
    expect(api.bodiesOf('POST /api/v1/models')).toEqual([
      {
        name: 'Fuels 1.0',
        inqToMqlRate: '2.00',
        mqlToDealRate: '5.00',
        averageDealSize: { amount: '10000.00', currency: 'USD' },
        activityTypeIds: ['AT-1002', 'AT-1005'],
      },
    ]);
    expect(screen.queryByRole('dialog', { name: 'New model' })).not.toBeInTheDocument();

    await user.click(within(messages).getByRole('button', { name: 'View' }));
    expect(router.state.location.search).toBe('?id=MOD-1004');
  });

  it('shows every problem at once and saves nothing (scenario 5)', async () => {
    const { user } = renderModels({
      'POST /api/v1/models': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [
              { field: 'name', message: 'Enter a name.' },
              {
                field: 'inqToMqlRate',
                message: 'Enter a rate above 0 % and at most 100 %, with at most two decimals.',
              },
              { field: 'activityTypeIds', message: 'Tick at least one activity type.' },
            ],
          },
        },
      },
    });

    await user.click(screen.getByRole('button', { name: 'New model' }));
    const form = await screen.findByRole('dialog', { name: 'New model' });
    await user.type(within(form).getByRole('textbox', { name: /^INQ→MQL rate/ }), '0');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('3 problems need your attention')).toBeInTheDocument();
    expect(within(form).getByRole('textbox', { name: /^Name/ })).toHaveAccessibleDescription(
      'Required Enter a name.',
    );
    expect(within(form).getByRole('textbox', { name: /^INQ→MQL rate/ })).toHaveAttribute(
      'aria-invalid',
      'true',
    );
    expect(within(form).getAllByText('Tick at least one activity type.')).toHaveLength(2);
  });

  it('asks before a change that recalculates goals, and saves only the changed fields (scenario 4)', async () => {
    const warning =
      "Used by 2 plans (31 activities). Their calculated MQL and revenue goals will be recalculated; typed goals ('Set later stages myself') are kept.";
    const { user, api } = renderModels(
      {
        'POST /api/v1/models/MOD-1001/impact': {
          body: { warnings: [{ kind: 'goalsRecalculated', message: warning, count: 2 }] },
        },
        'PATCH /api/v1/models/MOD-1001': {
          body: { ...LUBRICANTS, inqToMqlRate: '2.50', version: 4 },
        },
      },
      '/settings/models?id=MOD-1001',
    );

    const details = await screen.findByRole('dialog', { name: 'Lubricants 1.0' });
    await user.click(within(details).getByRole('button', { name: 'Edit' }));
    const form = await screen.findByRole('dialog', { name: 'Edit model' });
    const rate = within(form).getByRole('textbox', { name: /^INQ→MQL rate/ });
    await user.clear(rate);
    await user.type(rate, '2.5');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    const impact = await screen.findByRole('alertdialog', {
      name: "Save changes to model 'Lubricants 1.0'?",
    });
    expect(impact).toHaveTextContent(warning);
    expect(within(impact).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(api.bodiesOf('PATCH /api/v1/models/MOD-1001')).toEqual([]);

    await user.click(within(impact).getByRole('button', { name: 'Save changes' }));

    // The details are open again, so the message is looked for by its text.
    expect(await screen.findAllByText("Model 'Lubricants 1.0' saved")).not.toHaveLength(0);
    expect(api.bodiesOf('POST /api/v1/models/MOD-1001/impact')).toEqual([
      { inqToMqlRate: '2.50', version: 3 },
    ]);
    expect(api.bodiesOf('PATCH /api/v1/models/MOD-1001')).toEqual([
      { inqToMqlRate: '2.50', version: 3 },
    ]);
    // Back to the details.
    expect(await screen.findByRole('dialog', { name: 'Lubricants 1.0' })).toBeInTheDocument();
  });

  it('warns before an allowed activity type is removed; Keep editing saves nothing (scenario 6)', async () => {
    const warning = "5 activities use 'Event'. They keep it; new activities can't choose it.";
    const { user, api } = renderModels({
      'POST /api/v1/models/MOD-1001/impact': {
        body: { warnings: [{ kind: 'activityTypeRemoved', message: warning, count: 5 }] },
      },
    });

    await chooseAction('Lubricants 1.0', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit model' });
    await user.click(await within(form).findByRole('checkbox', { name: 'Event' }));
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    const impact = await screen.findByRole('alertdialog');
    expect(impact).toHaveTextContent(warning);
    await user.click(within(impact).getByRole('button', { name: 'Keep editing' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(within(form).getByRole('checkbox', { name: 'Event' })).not.toBeChecked();
    expect(api.bodiesOf('POST /api/v1/models/MOD-1001/impact')).toEqual([
      { activityTypeIds: ['AT-1005'], version: 3 },
    ]);
    expect(api.requests.some((request) => request.method === 'PATCH')).toBe(false);
  });

  it('renames without an impact check', async () => {
    const { user, api } = renderModels({
      'PATCH /api/v1/models/MOD-1001': {
        body: { ...LUBRICANTS, name: 'Lubricants 2.0', version: 4 },
      },
    });

    await chooseAction('Lubricants 1.0', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit model' });
    const name = within(form).getByRole('textbox', { name: /^Name/ });
    await user.clear(name);
    await user.type(name, 'Lubricants 2.0');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Model 'Lubricants 2.0' saved");
    expect(api.bodiesOf('PATCH /api/v1/models/MOD-1001')).toEqual([
      { name: 'Lubricants 2.0', version: 3 },
    ]);
  });

  it('keeps the typed values after a conflict, names who saved first and saves again over it (FR-007)', async () => {
    let patches = 0;
    const { user, api } = renderModels({
      'PATCH /api/v1/models/MOD-1001': () => {
        patches += 1;
        return patches === 1
          ? {
              status: 409,
              body: {
                error: {
                  code: 'version_conflict',
                  message: 'Someone else changed this model.',
                  fields: [],
                  conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T10:42:00Z' },
                },
              },
            }
          : { body: { ...LUBRICANTS, name: 'Lubricants X', version: 6 } };
      },
      'GET /api/v1/models/MOD-1001': { body: { ...LUBRICANTS, version: 5 } },
    });

    await chooseAction('Lubricants 1.0', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit model' });
    const name = within(form).getByRole('textbox', { name: /^Name/ });
    await user.clear(name);
    await user.type(name, 'Lubricants X');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this model on 14 Mar 2026 at 10:42',
    );
    expect(name).toHaveValue('Lubricants X');
    await waitFor(() =>
      expect(
        api.requests.some((r) => r.method === 'GET' && r.path === '/api/v1/models/MOD-1001'),
      ).toBe(true),
    );

    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Model 'Lubricants X' saved");
    expect(api.bodiesOf('PATCH /api/v1/models/MOD-1001')).toEqual([
      { name: 'Lubricants X', version: 3 },
      { name: 'Lubricants X', version: 5 },
    ]);
  });

  it('asks before discarding typed changes', async () => {
    const { user } = renderModels();

    await user.click(screen.getByRole('button', { name: 'New model' }));
    const form = await screen.findByRole('dialog', { name: 'New model' });
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'Fuels');
    await user.click(within(form).getByRole('button', { name: 'Cancel' }));

    expect(
      await screen.findByRole('alertdialog', { name: 'Discard changes?' }),
    ).toBeInTheDocument();
  });
});

describe('ModelsPage: status and delete (US1 scenario 7, FR-004)', () => {
  it('moves an unused model to the recycle bin and offers Undo', async () => {
    const { user, api } = renderModels({
      'DELETE /api/v1/models/MOD-1002': { body: deleted('model', 'MOD-1002', 'Mobility 1.0') },
    });

    await screen.findByRole('table');
    await chooseAction('Mobility 1.0', 'Delete', user);

    const dialog = await screen.findByRole('alertdialog', { name: "Delete model 'Mobility 1.0'?" });
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Delete model' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Model 'Mobility 1.0' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    expect(api.requests.filter((r) => r.method === 'DELETE')).toHaveLength(1);
  });

  it('offers Set inactive instead of deleting a model in use', async () => {
    const { user, api } = renderModels({
      'POST /api/v1/models/MOD-1001/status': { body: { ...LUBRICANTS, active: false, version: 4 } },
    });

    await screen.findByRole('table');
    await chooseAction('Lubricants 1.0', 'Delete', user);

    const dialog = await screen.findByRole('alertdialog', { name: "'Lubricants 1.0' is in use" });
    expect(dialog).toHaveTextContent(
      "Used by 2 plans — set it inactive instead. New plans can't choose an inactive model; plans using it keep it and still calculate goals.",
    );
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(await findMessages()).toHaveTextContent("Model 'Lubricants 1.0' set inactive");
    expect(api.bodiesOf('POST /api/v1/models/MOD-1001/status')).toEqual([
      { active: false, version: 3 },
    ]);
    expect(api.requests.some((r) => r.method === 'DELETE')).toBe(false);
  });

  it('switches to the in-use dialog when the API finds the model in use', async () => {
    const message = "'Mobility 1.0' is in use (used by 1 plan) — set it inactive instead.";
    const { user } = renderModels({
      'DELETE /api/v1/models/MOD-1002': { status: 409, body: inUse(message) },
    });

    await screen.findByRole('table');
    await chooseAction('Mobility 1.0', 'Delete', user);
    await user.click(await screen.findByRole('button', { name: 'Delete model' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'Mobility 1.0' is in use" });
    expect(dialog).toHaveTextContent(message);
    expect(within(dialog).getByRole('button', { name: 'Set inactive' })).toBeInTheDocument();
  });

  it('says an inactive model in use is already inactive, with Close only', async () => {
    const inactiveInUse = { ...LUBRICANTS, active: false };
    const { user } = renderModels(
      { 'GET /api/v1/models': { body: { items: [inactiveInUse], total: 1 } } },
      '/settings/models?status=inactive',
    );

    await screen.findByRole('table');
    await chooseAction('Lubricants 1.0', 'Delete', user);

    const dialog = await screen.findByRole('alertdialog');
    expect(dialog).toHaveTextContent(
      "Used by 2 plans, so it can't be deleted. It's already inactive.",
    );
    expect(
      within(dialog)
        .getAllByRole('button')
        .map((b) => b.textContent),
    ).toEqual(['Close']);
  });

  it('reactivates an inactive model', async () => {
    const { user, api } = renderModels(
      { 'POST /api/v1/models/MOD-1003/status': { body: { ...RETAIL, active: true, version: 2 } } },
      '/settings/models?status=inactive',
    );

    await screen.findByRole('table');
    await chooseAction('Retail 0.9', 'Reactivate', user);

    expect(await findMessages()).toHaveTextContent("Model 'Retail 0.9' reactivated");
    expect(api.bodiesOf('POST /api/v1/models/MOD-1003/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });
});
