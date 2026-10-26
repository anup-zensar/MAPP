import { screen } from '@testing-library/react';
import type { UserEvent } from '@testing-library/user-event';
import type { HttpHandler } from 'msw';
import type { ReactElement } from 'react';
import { ActivityTypesPage, ModelsPage } from '@/features/settings';
import { closestTo } from '../dom';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { TYPES } from './activity-types';
import { ACTIVITY_TYPES, MODELS } from './settings';
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

export function rowOf(name: string): HTMLElement {
  return closestTo(screen.getByRole('rowheader', { name }), 'tr');
}

/** Opens the Actions menu of the record `name` and chooses `action`. */
export async function chooseAction(name: string, action: string | RegExp, user: UserEvent) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

function renderSettingsPage(ui: ReactElement, path: string, route: string, answers: HttpHandler[]) {
  const api = mockApi(answers);
  return { ...renderWithProviders(ui, { route, user: testUser(), path }), api };
}
