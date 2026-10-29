import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { BUSINESS_UNIT, FORM_ANSWERS, page } from '@/testing/fixtures/shell-attributes';
import { reply } from '@/testing/replies';

/** What the Shell attribute pages ask the API when they open. */
const ANSWERS = [
  reply('GET /api/v1/shell-attributes', { body: page([BUSINESS_UNIT]) }),
  reply(
    'GET /api/v1/shell-attributes/{attributeId}',
    { body: BUSINESS_UNIT },
    { attributeId: 'SA-1001' },
  ),
  reply(
    'GET /api/v1/shell-attributes/{attributeId}/options',
    { body: page([]) },
    { attributeId: 'SA-1001' },
  ),
  ...FORM_ANSWERS,
];

// The addresses of the Shell attribute pages (feature 002, US2; plan "Routes and navigation").
describe('Shell attribute routes (feature 002)', () => {
  it.each([
    ['/settings/shell-attributes', 'Shell attributes', 'Shell attributes'],
    ['/settings/shell-attributes/new', 'New Shell attribute', 'New Shell attribute'],
    ['/settings/shell-attributes/SA-1001', 'Business unit', 'Business unit · Shell attributes'],
    [
      '/settings/shell-attributes/SA-1001/edit',
      "Edit Shell attribute 'Business unit'",
      'Edit Business unit',
    ],
  ])('opens %s: "%s", with Settings current in the sidebar', async (path, heading, title) => {
    renderApp(path, ANSWERS);

    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument();
    await waitFor(() => expect(document.title).toBe(`${title} · SP&A One`));
    expectCurrentSection('Settings');
  });
});
