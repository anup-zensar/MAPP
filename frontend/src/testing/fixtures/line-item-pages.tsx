/** The line-item screens of feature 007 drawn for their tests, with the answers they ask for. */
import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { LineItemCreatePage, LineItemsSection } from '@/features/line-items';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import type { Answer } from '../replies';
import { activityRecord } from './line-item-financials';
import { formAnswers, formOptions } from './line-item-forms';
import { testUser } from './users';

export const CREATE_ROUTE = '/activities/A-1001/line-items/new';
export const CREATE = 'POST /api/v1/activities/A-1001/line-items';

export function renderCreatePage(answers: HttpHandler[] = []) {
  const api = mockApi([
    ...formAnswers(),
    reply(
      'GET /api/v1/activities/{activityId}/line-items/form-options',
      { body: formOptions() },
      { activityId: 'A-1001' },
    ),
    ...answers,
  ]);
  return {
    api,
    ...renderWithProviders(<LineItemCreatePage />, {
      route: CREATE_ROUTE,
      path: '/activities/:activityId/line-items/new',
      user: testUser(),
    }),
  };
}

/** A button outside the dialogs, found quickly in the long form. */
export function formButton(name: string) {
  return screen.getByText(name, { selector: 'button' });
}

export function findNameField() {
  return screen.findByLabelText('Line item name');
}

/** The Line items section of "Launch film" with its GSAP figures, `state` given to its address. */
export function renderLineItemsSection(
  answer: Answer<'GET /api/v1/activities/{activityId}/line-items'>,
  state?: unknown,
  financials: Answer<'GET /api/v1/financials/records/{recordId}'> = { body: activityRecord() },
) {
  mockApi([
    reply('GET /api/v1/activities/{activityId}/line-items', answer, { activityId: 'A-1001' }),
    reply('GET /api/v1/financials/records/{recordId}', financials, { recordId: 'A-1001' }),
  ]);
  return renderWithProviders(<LineItemsSection activityId="A-1001" />, {
    route: { pathname: '/plans/P-1001/activities/A-1001', state },
    user: testUser(),
  });
}
