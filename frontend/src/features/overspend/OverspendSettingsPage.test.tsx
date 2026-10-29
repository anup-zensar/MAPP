import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import { OverspendSettingsPage } from './OverspendSettingsPage';

const GET = 'GET /api/v1/overspend-configuration';
const PUT = 'PUT /api/v1/overspend-configuration';

function configuration(
  overrides: Partial<components['schemas']['OverspendConfiguration']> = {},
): components['schemas']['OverspendConfiguration'] {
  return {
    defaultMode: 'warn',
    version: 1,
    updatedAt: '2026-10-27T09:00:00Z',
    updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
    canChange: { available: true, reason: '' },
    ...overrides,
  };
}

function renderPage(handlers: Parameters<typeof mockApi>[0]) {
  const api = mockApi(handlers);
  return { api, ...renderWithProviders(<OverspendSettingsPage />, { user: testUser() }) };
}

describe('037 FR-006 FR-007: Settings › Overspend control', () => {
  it('shows a placeholder while the setting loads', async () => {
    renderPage([reply(GET, () => pending())]);

    expect(screen.getByRole('heading', { level: 1, name: 'Overspend control' })).toBeVisible();
    expect(screen.getByRole('status')).toHaveTextContent('Loading');
    await expectNoAxeViolations();
  });

  it('shows a failure with Try again', async () => {
    renderPage([reply(GET, { status: 500, body: apiError('internal_error', 'Something broke.') })]);

    expect(await screen.findByRole('alert')).toHaveTextContent('Something broke.');
    expect(screen.getByRole('button', { name: 'Try again' })).toBeVisible();
  });

  it('says existing budgets keep theirs, and saves Refuse with the version', async () => {
    const { api, user } = renderPage([
      reply(GET, { body: configuration() }),
      reply(PUT, { body: configuration({ defaultMode: 'refuse', version: 2 }) }),
    ]);

    expect(await screen.findByRole('radio', { name: /^Warn/ })).toBeChecked();
    expect(screen.getByText('Existing budgets keep their own setting.')).toBeVisible();
    expect(screen.getByText(/Last changed by Alex Morgan/)).toBeVisible();
    await expectNoAxeViolations();

    await user.click(screen.getByRole('radio', { name: /^Refuse/ }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Overspend control saved.')).toBeVisible();
    expect(api.bodiesOf(PUT)).toEqual([{ version: 1, defaultMode: 'refuse' }]);
  });

  it('is read-only with the reason for a person who may not change Settings', async () => {
    const reason = "Changing Settings isn't part of your role. Ask Alex Morgan.";
    renderPage([reply(GET, { body: configuration({ canChange: { available: false, reason } }) })]);

    expect(await screen.findByText(reason)).toBeVisible();
    expect(screen.getByRole('radio', { name: /^Warn/ })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('keeps the choice and says who changed it first on a version conflict', async () => {
    const { user } = renderPage([
      reply(GET, { body: configuration() }),
      reply(PUT, {
        status: 409,
        body: apiError('version_conflict', 'Someone changed it.', {
          conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-10-27T10:00:00Z' },
        }),
      }),
    ]);

    await user.click(await screen.findByRole('radio', { name: /^Refuse/ }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /Not saved — Daniel Okoro changed this setting/,
    );
    expect(screen.getByRole('radio', { name: /^Refuse/ })).toBeChecked();
  });
});
