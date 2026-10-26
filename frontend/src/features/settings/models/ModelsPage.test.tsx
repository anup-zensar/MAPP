import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MODELS } from '@/testing/fixtures/settings';
import { chooseAction, renderModelsPage, rowOf } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

function table() {
  return screen.getByRole('table');
}

describe('ModelsPage: the list (FR-002)', () => {
  it('shows the breadcrumb, the header and the active models with their key facts', async () => {
    renderModelsPage();

    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    expect(screen.getByRole('heading', { level: 1, name: 'Models' })).toBeInTheDocument();
    expect(document.title).toBe('Models · MAPP');
    expect(screen.getByRole('button', { name: 'New model' })).toBeInTheDocument();

    const lubricants = rowOf(
      await screen.findByText('Lubricants 1.0').then(() => 'Lubricants 1.0'),
    );
    expect(lubricants).toHaveTextContent('2.00 % → 5.00 %');
    expect(lubricants).toHaveTextContent('$10,000.00');
    expect(lubricants).toHaveTextContent('Used by 2 plans');
    expect(within(lubricants).getByText('Active')).toBeInTheDocument();
    expect(rowOf('Mobility 1.0')).toHaveTextContent('Not used yet');
    // Inactive models are behind the Inactive and All filters.
    expect(within(table()).queryByText('Retail 0.9')).not.toBeInTheDocument();
    expect(screen.getByText('2 of 3 models')).toBeInTheDocument();
  });

  it('filters by status and searches by name, keeping both in the address', async () => {
    const { user, router } = renderModelsPage();
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Inactive 1' }));
    expect(within(table()).getByText('Retail 0.9')).toBeInTheDocument();
    expect(within(table()).queryByText('Lubricants 1.0')).not.toBeInTheDocument();
    expect(within(rowOf('Retail 0.9')).getByText('Inactive')).toBeInTheDocument();

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
    const { user } = renderModelsPage([], '/settings/models?q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(within(table()).getAllByRole('rowheader')).toHaveLength(3);
  });

  it('invites to create the first model when there are none', async () => {
    renderModelsPage([reply('GET /api/v1/models', { body: { items: [], total: 0 } })]);

    expect(await screen.findByText('No models yet — create your first model')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'New model' })).toHaveLength(2);
  });

  it('says when the models cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderModelsPage([
      reply('GET /api/v1/models', () => {
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
      }),
    ]);

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong.');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('table')).toBeInTheDocument();
  });
});

describe('ModelsPage: details (US1 scenario 3)', () => {
  it('opens a model from its name: every field, the example, the plans using it and who changed it', async () => {
    const { user, router } = renderModelsPage();

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
    renderModelsPage([], '/settings/models?id=MOD-1003');

    const panel = await screen.findByRole('dialog', { name: 'Retail 0.9' });
    expect(within(panel).getByText('Print advert (inactive)')).toBeInTheDocument();
  });

  it('says when the model of the address no longer exists, with Back to Models', async () => {
    const { user, router } = renderModelsPage([], '/settings/models?id=MOD-9999');

    const panel = await screen.findByRole('dialog');
    expect(panel).toHaveTextContent('This model no longer exists — someone may have deleted it.');
    await user.click(within(panel).getByRole('button', { name: 'Back to Models' }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('Duplicate opens an unsaved new model named "<name> (copy)" (FR-009)', async () => {
    const { user } = renderModelsPage([], '/settings/models?id=MOD-1001');

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
