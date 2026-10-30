import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import {
  BIN_ENTRY,
  purchaseRequest,
  purchaseRequestApproval,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { VIEWER } from '@/testing/fixtures/viewer';
import { mockApi } from '@/testing/mock-api';
import type { RecordedRequest } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { CurrentUser } from '@/session/useSession';
import { PurchaseRequestPage } from './PurchaseRequestPage';

const GONE = "This purchase request doesn't exist or you don't have access to it.";
const RECORD = '/api/v1/purchase-requests/PR-1002';

function renderPage(user: CurrentUser) {
  let deleted = false;
  const api = mockApi([
    ...HISTORY_ANSWERS,
    reply('GET /api/v1/shell-attribute-forms/{level}', {
      body: {
        level: 'purchaseRequest',
        businessUnitAttributeId: null,
        businessUnitOptions: [],
        sections: [],
      },
    }),
    reply('GET /api/v1/purchase-requests/{purchaseRequestId}/approval', () =>
      deleted
        ? { status: 404, body: apiError('not_found', GONE) }
        : { body: purchaseRequestApproval() },
    ),
    reply('GET /api/v1/purchase-requests/{purchaseRequestId}', () =>
      deleted ? { status: 404, body: apiError('not_found', GONE) } : { body: purchaseRequest() },
    ),
    reply('DELETE /api/v1/purchase-requests/{purchaseRequestId}', () => {
      deleted = true;
      return { body: { recycleBinEntry: BIN_ENTRY } };
    }),
  ]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestPage />, {
      route: '/purchase-requests/PR-1002',
      path: '/purchase-requests/:purchaseRequestId',
      user,
    }),
  };
}

/** The requests made for the deleted record after its DELETE: none may be made. */
function askedAgain(requests: RecordedRequest[]): string[] {
  const at = requests.findIndex((r) => r.method === 'DELETE');
  return requests
    .slice(at + 1)
    .filter((r) => r.method === 'GET' && r.path.startsWith(RECORD))
    .map((r) => r.path);
}

async function deleteFromPage(user: ReturnType<typeof renderPage>['user']) {
  await user.click(await screen.findByRole('button', { name: 'More' }));
  await user.click(await screen.findByRole('menuitem', { name: 'Delete purchase request' }));
  const dialog = await screen.findByRole('alertdialog');
  await user.click(within(dialog).getByRole('button', { name: 'Delete purchase request' }));
}

describe('038 FR-013 (H-3): delete a purchase request from its page', () => {
  it('lands on Purchase requests with Undo and never asks for the deleted record', async () => {
    const { api, user, router } = renderPage(testUser());

    await deleteFromPage(user);

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Purchase request 'PR for Stand build'");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    await vi.waitFor(() => expect(router.state.location.pathname).toBe('/purchase-requests'));
    expect(screen.queryByText(GONE)).not.toBeInTheDocument();
    expect(askedAgain(api.requests)).toEqual([]);
  });

  it("038 FR-013 (L-3): never asks for the deleted record's history after the delete", async () => {
    const { api, user, router } = renderPage(testUser());
    await screen.findByRole('heading', { level: 1, name: 'PR for Stand build' });

    await deleteFromPage(user);

    await findMessages();
    await vi.waitFor(() => expect(router.state.location.pathname).toBe('/purchase-requests'));
    const at = api.requests.findIndex((r) => r.method === 'DELETE');
    const after = api.requests.slice(at + 1).map((r) => `${r.method} ${r.path}${r.search}`);
    expect(after.filter((request) => request.includes('PR-1002'))).toEqual([]);
  });

  it('lands on the line item when the person may not see the Purchase requests page', async () => {
    const { user, router } = renderPage(VIEWER);

    await deleteFromPage(user);

    expect(await findMessages()).toHaveTextContent("Purchase request 'PR for Stand build'");
    await vi.waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1006'));
  });
});
