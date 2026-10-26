import { screen } from '@testing-library/react';
import type { UserEvent } from '@testing-library/user-event';
import type { HttpHandler } from 'msw';
import type { ReactElement } from 'react';
import { ActivityTypesPage, CurrenciesPage, ModelsPage } from '@/features/settings';
import { closestTo } from '../dom';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { TYPES } from './activity-types';
import { EUR_RATES, NO_IMPORTS, NO_RATES } from './rates';
import { ACTIVITY_TYPES, CURRENCIES, MODELS } from './settings';
import { testUser } from './users';

/** The Models page for a signed-in person, with the models and activity types answered. */
export function renderModelsPage(answers: HttpHandler[] = [], route = '/settings/models') {
  return renderSettingsPage(<ModelsPage />, '/settings/models', route, [
    reply('GET /api/v1/models', { body: MODELS }),
    reply('GET /api/v1/activity-types', { body: ACTIVITY_TYPES }),
    ...answers,
  ]);
}

export function renderActivityTypesPage(
  answers: HttpHandler[] = [],
  route = '/settings/activity-types',
) {
  return renderSettingsPage(<ActivityTypesPage />, '/settings/activity-types', route, [
    reply('GET /api/v1/activity-types', { body: TYPES }),
    reply('GET /api/v1/models', { body: MODELS }),
    ...answers,
  ]);
}

export function renderCurrenciesPage(answers: HttpHandler[] = [], route = '/settings/currencies') {
  return renderSettingsPage(<CurrenciesPage />, '/settings/currencies', route, [
    reply('GET /api/v1/currencies', { body: CURRENCIES }),
    reply('GET /api/v1/exchange-rates/imports', { body: NO_IMPORTS }),
    reply('GET /api/v1/currencies/{code}/rates', { body: EUR_RATES }, { code: 'EUR' }),
    reply('GET /api/v1/currencies/{code}/rates', { body: NO_RATES }, { code: 'JPY' }),
    ...answers,
  ]);
}

export function rowOf(name: string): HTMLElement {
  return closestTo(screen.getByRole('rowheader', { name }), 'tr');
}

/** Opens the Actions menu of the record `name` and chooses the item whose name starts with `action`. */
export async function chooseAction(name: string, action: string, user: UserEvent) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  await user.click(
    await screen.findByRole('menuitem', { name: (itemName) => itemName.startsWith(action) }),
  );
}

function renderSettingsPage(ui: ReactElement, path: string, route: string, answers: HttpHandler[]) {
  const api = mockApi(answers);
  return { ...renderWithProviders(ui, { route, user: testUser(), path }), api };
}
