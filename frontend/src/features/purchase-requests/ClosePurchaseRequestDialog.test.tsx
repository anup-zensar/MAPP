import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  closePreview,
  purchaseRequest,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import { ClosePurchaseRequestDialog } from './ClosePurchaseRequestDialog';

const PREVIEW = 'GET /api/v1/purchase-requests/{purchaseRequestId}/close-preview';
const CLOSE = 'POST /api/v1/purchase-requests/{purchaseRequestId}/close';
const TITLE = "Close purchase request 'PR for Stand build'?";

function Harness() {
  const [open, setOpen] = useState(true);
  return open ? (
    <ClosePurchaseRequestDialog
      purchaseRequest={routedPurchaseRequest()}
      open={open}
      onOpenChange={setOpen}
    />
  ) : (
    <p>Dialog closed</p>
  );
}

function renderDialog(answers: HttpHandler[]) {
  const api = mockApi(answers);
  return { api, ...renderWithProviders(<Harness />, { user: testUser() }) };
}

describe('032 FR-041/FR-055: close', () => {
  it('shows per line the current, invoiced and released amounts before closing (US6 sc.6)', async () => {
    const { api, user } = renderDialog([
      reply(PREVIEW, { body: closePreview() }),
      reply(CLOSE, { body: purchaseRequest({ status: 'closed' }) }),
    ]);

    const dialog = await screen.findByRole('alertdialog', { name: TITLE });
    const row = await within(dialog).findByRole('row', { name: /L-1006 · HX1006/ });
    expect(row).toHaveTextContent('9,200.00 EUR');
    expect(row).toHaveTextContent('7,500.00 EUR');
    expect(row).toHaveTextContent('1,700.00 EUR');
    expect(dialog).toHaveTextContent('1,847.83 USD returns to the plan in total.');
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Close purchase request' }));

    expect(await findMessages()).toHaveTextContent("Purchase request 'PR for Stand build' closed");
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/close')).toEqual([{ version: 3 }]);
    expect(screen.getByText('Dialog closed')).toBeInTheDocument();
  });

  it('offers only Cancel while the preview loads', async () => {
    renderDialog([reply(PREVIEW, () => pending())]);

    const dialog = await screen.findByRole('alertdialog', { name: TITLE });
    expect(within(dialog).getByRole('status')).toHaveTextContent('Loading…');
    expect(within(dialog).queryByRole('button', { name: 'Close purchase request' })).toBeNull();
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
  });

  it('shows a failed preview with Try again', async () => {
    renderDialog([
      reply(PREVIEW, { status: 500, body: apiError('internal_error', 'Something went wrong.') }),
    ]);

    const dialog = await screen.findByRole('alertdialog');
    expect(await within(dialog).findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says why a purchase request cannot be closed now, with no Close button', async () => {
    const reason = 'Wait until the change of this purchase request is approved or declined.';
    renderDialog([reply(PREVIEW, { body: closePreview({ available: false, reason, lines: [] }) })]);

    const dialog = await screen.findByRole('alertdialog');
    expect(await within(dialog).findByText(reason)).toBeInTheDocument();
    expect(within(dialog).queryByRole('button', { name: 'Close purchase request' })).toBeNull();
  });

  it('keeps the dialog open with the refusal when the close is refused', async () => {
    const message = 'Not closed — Daniel Okoro changed this purchase request at 10:15';
    const { user } = renderDialog([
      reply(PREVIEW, { body: closePreview() }),
      reply(CLOSE, { status: 409, body: apiError('version_conflict', message) }),
    ]);

    const dialog = await screen.findByRole('alertdialog');
    await user.click(await within(dialog).findByRole('button', { name: 'Close purchase request' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(message);
  });
});
