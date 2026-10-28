import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { configuration, no } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import { PurchaseRequestSettingsPage } from './PurchaseRequestSettingsPage';

const GET = 'GET /api/v1/purchase-request-configuration';
const PUT = 'PUT /api/v1/purchase-request-configuration';

function renderPage(handlers: Parameters<typeof mockApi>[0]) {
  const api = mockApi(handlers);
  return { api, ...renderWithProviders(<PurchaseRequestSettingsPage />, { user: testUser() }) };
}

describe('032 FR-004: Settings › Purchase requests', () => {
  it('shows a placeholder while the configuration loads', async () => {
    renderPage([reply(GET, () => pending())]);

    expect(screen.getByRole('heading', { level: 1, name: 'Purchase requests' })).toBeVisible();
    expect(screen.getByRole('status')).toHaveTextContent('Loading');
    await expectNoAxeViolations();
  });

  it('shows a failure with Try again', async () => {
    renderPage([reply(GET, { status: 500, body: apiError('internal_error', 'Something broke.') })]);

    expect(await screen.findByRole('alert')).toHaveTextContent('Something broke.');
    expect(screen.getByRole('button', { name: 'Try again' })).toBeVisible();
  });

  it('says so when the API refuses the person', async () => {
    renderPage([
      reply('GET /api/v1/me', { body: testUser() }),
      reply(GET, {
        status: 403,
        body: apiError('forbidden', "Changing Settings isn't part of your role."),
      }),
    ]);

    expect(await screen.findByText("Changing Settings isn't part of your role.")).toBeVisible();
  });

  it('shows the values and saves a change with the version', async () => {
    const saved = configuration({ purchaseSupport: false, version: 2 });
    const { api, user } = renderPage([
      reply(GET, { body: configuration() }),
      reply(PUT, { body: saved }),
    ]);

    const support = await screen.findByRole('checkbox', { name: 'Purchase Support' });
    expect(support).toBeChecked();
    expect(screen.getByRole('radio', { name: 'Allowed with modification approval' })).toBeChecked();
    expect(screen.getByText(/Last changed by Alex Morgan/)).toBeVisible();
    await expectNoAxeViolations();

    await user.click(support);
    await user.click(screen.getByRole('radio', { name: 'Allowed without approval' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Purchase request settings saved')).toBeVisible();
    expect(api.bodiesOf(PUT)).toEqual([
      {
        version: 1,
        purchaseSupport: false,
        linesOnApprovedPurchaseRequests: 'withoutApproval',
      },
    ]);
  });

  it('is read-only with the reason for a person who may not change Settings', async () => {
    const reason = "Changing Settings isn't part of your role. Ask Alex Morgan.";
    renderPage([reply(GET, { body: configuration({ canChange: no(reason) }) })]);

    expect(await screen.findByText(reason)).toBeVisible();
    expect(screen.getByText('On')).toBeVisible();
    expect(screen.getByText('Allowed with modification approval')).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('keeps the entries and says who changed it first on a version conflict', async () => {
    const { user } = renderPage([
      reply(GET, { body: configuration() }),
      reply(PUT, {
        status: 409,
        body: apiError('version_conflict', 'Someone changed it.', {
          conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-10-27T10:00:00Z' },
        }),
      }),
    ]);

    await user.click(await screen.findByRole('checkbox', { name: 'Purchase Support' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /Not saved — Daniel Okoro changed this purchase request configuration/,
    );
    expect(screen.getByRole('checkbox', { name: 'Purchase Support' })).not.toBeChecked();
  });
});
