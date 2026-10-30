import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { LUBRICANTS, MOBILITY } from '@/testing/fixtures/settings';
import { chooseAction, renderModelsPage } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

describe('ModelsPage: create and edit (US1 scenarios 1, 4, 5, 6)', () => {
  it('creates a model and offers View in the message (scenario 1)', async () => {
    const created = { ...MOBILITY, id: 'MOD-1004', name: 'Fuels 1.0', usage: LUBRICANTS.usage };
    const { user, api, router } = renderModelsPage([
      reply('POST /api/v1/models', { status: 201, body: created }),
    ]);

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
    const { user } = renderModelsPage([
      reply('POST /api/v1/models', {
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
      }),
    ]);

    await user.click(screen.getByRole('button', { name: 'New model' }));
    const form = await screen.findByRole('dialog', { name: 'New model' });
    await user.type(within(form).getByRole('textbox', { name: /^INQ→MQL rate/ }), '0');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('3 problems need your attention')).toBeInTheDocument();
    expect(within(form).getByRole('textbox', { name: /^Name/ })).toHaveAccessibleDescription(
      'required Enter a name.',
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
    const { user, api } = renderModelsPage(
      [
        reply(
          'POST /api/v1/models/{modelId}/impact',
          {
            body: { warnings: [{ kind: 'goalsRecalculated', message: warning, count: 2 }] },
          },
          { modelId: 'MOD-1001' },
        ),
        reply(
          'PATCH /api/v1/models/{modelId}',
          {
            body: { ...LUBRICANTS, inqToMqlRate: '2.50', version: 4 },
          },
          { modelId: 'MOD-1001' },
        ),
      ],
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
    const { user, api } = renderModelsPage([
      reply(
        'POST /api/v1/models/{modelId}/impact',
        {
          body: { warnings: [{ kind: 'activityTypeRemoved', message: warning, count: 5 }] },
        },
        { modelId: 'MOD-1001' },
      ),
    ]);

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
    const { user, api } = renderModelsPage([
      reply(
        'PATCH /api/v1/models/{modelId}',
        {
          body: { ...LUBRICANTS, name: 'Lubricants 2.0', version: 4 },
        },
        { modelId: 'MOD-1001' },
      ),
    ]);

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
    const { user, api } = renderModelsPage([
      reply(
        'PATCH /api/v1/models/{modelId}',
        () => {
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
        { modelId: 'MOD-1001' },
      ),
      reply(
        'GET /api/v1/models/{modelId}',
        { body: { ...LUBRICANTS, version: 5 } },
        { modelId: 'MOD-1001' },
      ),
    ]);

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
    const { user } = renderModelsPage();

    await user.click(screen.getByRole('button', { name: 'New model' }));
    const form = await screen.findByRole('dialog', { name: 'New model' });
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'Fuels');
    await user.click(within(form).getByRole('button', { name: 'Cancel' }));

    expect(
      await screen.findByRole('alertdialog', { name: 'Discard changes?' }),
    ).toBeInTheDocument();
  });
});
