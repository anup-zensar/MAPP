import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { deleted, inUse, LUBRICANTS, RETAIL } from '@/testing/fixtures/settings';
import { chooseAction, renderModelsPage } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

describe('ModelsPage: status and delete (US1 scenario 7, FR-004)', () => {
  it('moves an unused model to the recycle bin and offers Undo', async () => {
    const { user, api } = renderModelsPage([
      reply(
        'DELETE /api/v1/models/{modelId}',
        { body: deleted('model', 'MOD-1002', 'Mobility 1.0') },
        { modelId: 'MOD-1002' },
      ),
    ]);

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
    const { user, api } = renderModelsPage([
      reply(
        'POST /api/v1/models/{modelId}/status',
        { body: { ...LUBRICANTS, active: false, version: 4 } },
        { modelId: 'MOD-1001' },
      ),
    ]);

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
    const { user } = renderModelsPage([
      reply(
        'DELETE /api/v1/models/{modelId}',
        { status: 409, body: inUse(message) },
        { modelId: 'MOD-1002' },
      ),
    ]);

    await screen.findByRole('table');
    await chooseAction('Mobility 1.0', 'Delete', user);
    await user.click(await screen.findByRole('button', { name: 'Delete model' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'Mobility 1.0' is in use" });
    expect(dialog).toHaveTextContent(message);
    expect(within(dialog).getByRole('button', { name: 'Set inactive' })).toBeInTheDocument();
  });

  it('says an inactive model in use is already inactive, with Close only', async () => {
    const inactiveInUse = { ...LUBRICANTS, active: false };
    const { user } = renderModelsPage(
      [reply('GET /api/v1/models', { body: { items: [inactiveInUse], total: 1 } })],
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
    const { user, api } = renderModelsPage(
      [
        reply(
          'POST /api/v1/models/{modelId}/status',
          { body: { ...RETAIL, active: true, version: 2 } },
          { modelId: 'MOD-1003' },
        ),
      ],
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
