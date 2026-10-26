import { screen, within } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { Outlet } from 'react-router';
import { ShellAttributeEditorPage, ShellAttributesPage } from '@/features/settings';
import { OptionsCard } from '@/features/settings/shell-attributes/OptionsCard';
import { SessionProvider } from '@/session/SessionProvider';
import { mockApi } from '../mock-api';
import type { MockedApi } from '../mock-api';
import { renderRoutes, renderWithProviders } from '../render';
import { reply } from '../replies';
import {
  ACTIVITY_ATTRIBUTES,
  BUSINESS_UNIT,
  FORM_ANSWERS,
  MEDIA_CHANNEL,
  page,
  shellOption,
  usedBy,
} from './shell-attributes';
import { testUser } from './users';

export function renderShellAttributesPage(route: string, answers: HttpHandler[] = []) {
  const api = mockApi([
    reply('GET /api/v1/shell-attributes', { body: page([BUSINESS_UNIT, ...ACTIVITY_ATTRIBUTES]) }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<ShellAttributesPage />, {
    route,
    user: testUser(),
    path: '/settings/shell-attributes',
  });
  return { ...rendered, api };
}

/** The names of the table's rows, section header rows included, top to bottom. */
export function rowNames(): string[] {
  const rows = within(screen.getByRole('table')).getAllByRole('row').slice(1);
  return rows.map((row) => {
    const header = within(row).queryByRole('rowheader') ?? within(row).getByRole('columnheader');
    // An attribute's name is its link; a "No help yet" chip may follow it (016 FR-048).
    return (within(header).queryByRole('link') ?? header).textContent ?? '';
  });
}

/** The editor at `route`, with the attribute page and the list as stand-ins to land on. */
export function renderAttributeEditor(route: string, answers: HttpHandler[] = []) {
  const api = mockApi([
    reply('GET /api/v1/shell-attributes', {
      body: page([BUSINESS_UNIT, ...ACTIVITY_ATTRIBUTES]),
    }),
    ...FORM_ANSWERS,
    ...answers,
  ]);
  const rendered = renderRoutes(
    [
      {
        // Like the app's RootLayout: the session around every page.
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/settings/shell-attributes/new',
            element: <ShellAttributeEditorPage mode="create" />,
          },
          {
            path: '/settings/shell-attributes/:attributeId/edit',
            element: <ShellAttributeEditorPage mode="edit" />,
          },
          { path: '/settings/shell-attributes/:attributeId', element: <p>Attribute page</p> },
          { path: '/settings/shell-attributes', element: <p>Attribute list</p> },
        ],
      },
    ],
    { route, user: testUser() },
  );
  return { ...rendered, api };
}

/** The options a select offers, without "Choose…". */
export function offered(name: RegExp): string[] {
  return within(screen.getByRole('combobox', { name }))
    .getAllByRole('option')
    .map((option) => option.textContent ?? '')
    .filter((text) => text !== 'Choose…');
}

export const TV = shellOption({
  id: 'SO-1101',
  label: 'TV',
  attributeId: 'SA-1010',
  usage: usedBy('Used by 7 activities', 7),
});
export const RADIO = shellOption({
  id: 'SO-1102',
  label: 'Radio',
  attributeId: 'SA-1010',
  position: 2,
});
export const PRINT = shellOption({
  id: 'SO-1103',
  label: 'Print',
  attributeId: 'SA-1010',
  position: 3,
  active: false,
});

export function renderOptionsCard(answers: HttpHandler[], attribute = MEDIA_CHANNEL) {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<OptionsCard attribute={attribute} />, {
    user: testUser(),
  });
  return { ...rendered, api };
}

/** The query of each options request, as `URLSearchParams`. */
export function optionQueries(api: MockedApi) {
  return api.requests
    .filter((request) => request.path === '/api/v1/shell-attributes/SA-1010/options')
    .map((request) => new URLSearchParams(request.search));
}
