import { screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { campaignForm } from '@/test/attributeForms';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import {
  SETTINGS_KEYS,
  useAttributeForm,
  useCurrencies,
  useFinanceBudgets,
  useModels,
} from './settingsQueries';

function FormLevel() {
  const form = useAttributeForm('campaign');
  return <p>{form.data ? `sections: ${form.data.sections.length}` : 'loading'}</p>;
}

function Lists() {
  const models = useModels();
  const currencies = useCurrencies();
  const budgets = useFinanceBudgets(2026);
  return (
    <p>
      {models.data && currencies.data && budgets.data
        ? `models ${models.data.total}, currencies ${currencies.data.total}`
        : 'loading'}
    </p>
  );
}

describe('settingsQueries (feature 002)', () => {
  it('names its query keys so other features can refresh Settings data', () => {
    expect(SETTINGS_KEYS.models).toEqual(['models']);
    expect(SETTINGS_KEYS.shellAttributeForm('activity')).toEqual([
      'shell-attribute-form',
      'activity',
    ]);
    expect(SETTINGS_KEYS.financeBudgets(2026)).toEqual(['finance-budgets', 2026]);
  });

  it('reads the lists from the API', async () => {
    const api = mockApi({
      'GET /api/v1/models': { body: { items: [], total: 2 } },
      'GET /api/v1/currencies': { body: { items: [], total: 5 } },
      'GET /api/v1/finance-budgets': { body: { year: 2026, years: [2026], items: [] } },
    });
    renderWithProviders(<Lists />);

    expect(await screen.findByText('models 2, currencies 5')).toBeInTheDocument();
    expect(api.requests.find((request) => request.path === '/api/v1/finance-budgets')?.search).toBe(
      '?year=2026',
    );
  });

  it('asks for the attribute form again every time a form opens (SC-008)', async () => {
    const api = mockApi({ 'GET /api/v1/shell-attribute-forms/campaign': { body: campaignForm() } });
    const { user } = renderWithProviders(<OpenAndClose />);
    const asked = () =>
      api.requests.filter((request) => request.path === '/api/v1/shell-attribute-forms/campaign');

    expect(await screen.findByText('sections: 2')).toBeInTheDocument();
    expect(asked()).toHaveLength(1);

    await user.click(screen.getByRole('button', { name: 'Close the form' }));
    await user.click(screen.getByRole('button', { name: 'Open the form' }));

    await waitFor(() => expect(asked()).toHaveLength(2));
  });
});

/** A form that is closed and opened again in the same session. */
function OpenAndClose() {
  const [open, setOpen] = useState(true);
  return (
    <>
      <button type="button" onClick={() => setOpen(!open)}>
        {open ? 'Close the form' : 'Open the form'}
      </button>
      {open && <FormLevel />}
    </>
  );
}
