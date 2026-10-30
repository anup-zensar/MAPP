import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  BIN_ENTRY,
  GIFT_BOXES,
  prLine,
  purchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import { DeletePurchaseRequestDialog } from './DeletePurchaseRequestDialog';

const DELETE = 'DELETE /api/v1/purchase-requests/{purchaseRequestId}';
const TWO_LINES = purchaseRequest({
  lines: [prLine(), prLine({ id: 'PRL-1003', position: 2, lineItem: GIFT_BOXES, hxKey: 'HX1011' })],
});

function Harness({ pr }: { pr: typeof TWO_LINES }) {
  const [open, setOpen] = useState(true);
  return <DeletePurchaseRequestDialog purchaseRequest={pr} open={open} onOpenChange={setOpen} />;
}

function renderDialog(pr: typeof TWO_LINES, answers: HttpHandler[] = []) {
  const api = mockApi(answers);
  return {
    api,
    ...renderWithProviders(<Harness pr={pr} />, {
      user: testUser(),
      route: '/purchase-requests/PR-1002',
      path: '/purchase-requests/:purchaseRequestId',
    }),
  };
}

describe('032 FR-040/FR-042/FR-055/FR-056: delete', () => {
  it('names the lines and the line items whose money is released (US5 sc.7)', async () => {
    renderDialog(TWO_LINES);

    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete purchase request 'PR for Stand build'?",
    });
    expect(dialog).toHaveTextContent(
      'Its 2 lines will be removed and the money reserved on line items L-1006 and L-1011 will be released. You can restore it from the recycle bin.',
    );
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await expectNoAxeViolations();
  });

  it('says "1 line" and "line item" for a single line', async () => {
    renderDialog(purchaseRequest());

    expect(await screen.findByRole('alertdialog')).toHaveTextContent(
      'Its 1 line will be removed and the money reserved on line item L-1006 will be released.',
    );
  });

  it('moves it to the recycle bin with Undo and opens Purchase requests (038 FR-013, H-3)', async () => {
    const { api, user, router } = renderDialog(purchaseRequest(), [
      reply(DELETE, { body: { recycleBinEntry: BIN_ENTRY } }),
    ]);

    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Delete purchase request' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Purchase request 'PR for Stand build' deleted");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/purchase-requests');
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(true);
  });

  it('shows the refusal in the dialog', async () => {
    const message = 'Purchase requests awaiting approval cannot be deleted — recall them first.';
    const { user } = renderDialog(purchaseRequest(), [
      reply(DELETE, { status: 409, body: apiError('not_allowed', message) }),
    ]);

    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Delete purchase request' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(message);
  });
});
