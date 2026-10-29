import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ACTIVITY_TYPES, LUBRICANTS, PRINT, ref, RETAIL } from '@/testing/fixtures/settings';
import { reply } from '@/testing/replies';
import { ModelForm } from './ModelForm';
import type { ModelFormMode } from './model-values';

function renderForm(mode: ModelFormMode, answers: HttpHandler[] = []) {
  const onClose = vi.fn();
  const api = mockApi([reply('GET /api/v1/activity-types', { body: ACTIVITY_TYPES }), ...answers]);
  const rendered = renderWithProviders(
    <ModelForm mode={mode} onView={vi.fn()} onClose={onClose} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose };
}

function field(name: RegExp) {
  return screen.getByRole('textbox', { name });
}

describe('ModelForm (FR-009, FR-010)', () => {
  it('names every field with its marking and explains the allowed activity types', async () => {
    renderForm({ kind: 'new' });

    const form = screen.getByRole('dialog', { name: 'New model' });
    expect(field(/^Name/)).toHaveAccessibleDescription('required');
    expect(field(/^INQ→MQL rate/)).toBeInTheDocument();
    expect(field(/^MQL→deal rate/)).toBeInTheDocument();
    expect(field(/^Average deal size/)).toHaveAccessibleDescription(/USD/);
    const types = await within(form).findByRole('group', { name: /Allowed activity types/ });
    expect(types).toHaveTextContent('Activities in plans using this model can be of these types.');
    // Inactive types are not offered for new choices.
    expect(within(types).queryByRole('checkbox', { name: /Print advert/ })).not.toBeInTheDocument();
  });

  it('shows the worked example live while the values are typed (US1 scenario 2)', async () => {
    const { user } = renderForm({ kind: 'new' });

    expect(
      screen.getByText('Fill in both rates and the average deal size to see the example.'),
    ).toBeInTheDocument();
    await user.type(field(/^INQ→MQL rate/), '2');
    await user.type(field(/^MQL→deal rate/), '5');
    await user.type(field(/^Average deal size/), '10k');

    expect(
      screen.getByText('250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD revenue'),
    ).toBeInTheDocument();

    await user.clear(field(/^MQL→deal rate/));
    await user.type(field(/^MQL→deal rate/), '10');
    expect(
      screen.getByText('250,000 INQ → 5,000 MQL → 500 deals → 5,000,000.00 USD revenue'),
    ).toBeInTheDocument();
  });

  it('keeps an inactive type the model already allows, labelled "(inactive)"', async () => {
    renderForm({ kind: 'edit', model: { ...RETAIL, activityTypes: [ref(PRINT)] } });

    expect(await screen.findByRole('checkbox', { name: 'Print advert (inactive)' })).toBeChecked();
  });

  it('copies only the active allowed types into a duplicate', async () => {
    renderForm({
      kind: 'duplicate',
      model: { ...LUBRICANTS, activityTypes: [...LUBRICANTS.activityTypes, ref(PRINT)] },
    });

    expect(field(/^Name/)).toHaveValue('Lubricants 1.0 (copy)');
    expect(await screen.findByRole('checkbox', { name: 'Event' })).toBeChecked();
    expect(screen.getByRole('checkbox', { name: 'Webinar' })).toBeChecked();
    expect(screen.queryByRole('checkbox', { name: /Print advert/ })).not.toBeInTheDocument();
  });

  it('closes without a request when nothing changed', async () => {
    const { user, api, onClose } = renderForm({ kind: 'edit', model: LUBRICANTS });

    await screen.findByRole('checkbox', { name: 'Event' });
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(onClose).toHaveBeenCalled();
    expect(api.requests.filter((request) => request.method !== 'GET')).toEqual([]);
  });

  it('says when the activity types cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderForm({ kind: 'new' }, [
      reply('GET /api/v1/activity-types', () => {
        calls += 1;
        return calls === 1 ? Promise.reject(new TypeError('offline')) : { body: ACTIVITY_TYPES };
      }),
    ]);

    expect(await screen.findByText(/can't reach its server/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('checkbox', { name: 'Event' })).toBeInTheDocument();
  });
});
