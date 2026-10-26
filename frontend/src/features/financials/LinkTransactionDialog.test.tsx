import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { testUser } from '@/testing/users';
import { LinkTransactionDialog } from './LinkTransactionDialog';
import { FT_1004, manuallyLinked, STATION_SIGNAGE, transactionOf } from './testData';

const LINK = 'POST /api/v1/financials/transactions/FT-1004/link';

function renderDialog(answers: Record<string, MockHandler> = {}) {
  const onClose = vi.fn();
  const api = mockApi({
    'GET /api/v1/line-items/L-1007': { body: STATION_SIGNAGE },
    [LINK]: { body: manuallyLinked() },
    ...answers,
  });
  const rendered = renderWithProviders(
    <LinkTransactionDialog transaction={transactionOf(FT_1004)} onClose={onClose} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose };
}

describe('LinkTransactionDialog (015 US7, FR-026)', () => {
  it('previews the line item, then links it with the version (US7 scenario 1)', async () => {
    const { api, user, onClose } = renderDialog();

    const dialog = await screen.findByRole('dialog', {
      name: "Link 'PO 4500001004 / 10' to a line item",
    });
    const field = within(dialog).getByRole('textbox', { name: /Line item ID/ });
    expect(field).toHaveFocus();
    await user.type(field, 'l-1007');

    expect(
      await within(dialog).findByText(
        'Station signage (L-1007) · Grand opening Shanghai · FY2026 Mobility China',
      ),
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Link' }));

    expect(
      (await screen.findAllByText("'PO 4500001004 / 10' linked to 'Station signage (L-1007)'"))[0],
    ).toBeInTheDocument();
    expect(api.bodiesOf(LINK)).toEqual([{ lineItemId: 'L-1007', version: 1 }]);
    expect(onClose).toHaveBeenCalled();
  });

  it('says when no line item has the ID, and checks the ID before asking', async () => {
    const { api, user } = renderDialog({
      'GET /api/v1/line-items/L-9999': {
        status: 404,
        body: { error: { code: 'not_found', message: 'This line item no longer exists.' } },
      },
    });

    const dialog = await screen.findByRole('dialog');
    const field = within(dialog).getByRole('textbox', { name: /Line item ID/ });
    await user.type(field, 'L-9999');
    expect(await within(dialog).findByText('No line item L-9999.')).toBeInTheDocument();

    await user.clear(field);
    await user.type(field, '1007');
    await user.click(within(dialog).getByRole('button', { name: 'Link' }));
    expect(within(dialog).getByText('Line item ID must look like L-1003.')).toBeInTheDocument();
    expect(api.bodiesOf(LINK)).toEqual([]);
  });

  it("shows the API's refusal at the field (US7 scenario 4)", async () => {
    const { user, onClose } = renderDialog({
      'GET /api/v1/line-items/L-1005': {
        status: 404,
        body: { error: { code: 'not_found', message: 'This line item no longer exists.' } },
      },
      [LINK]: {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the line item.',
            fields: [
              {
                field: 'lineItemId',
                message: 'Line item L-1005 is in the recycle bin — restore it first.',
              },
            ],
          },
        },
      },
    });

    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByRole('textbox', { name: /Line item ID/ }), 'L-1005');
    await user.click(within(dialog).getByRole('button', { name: 'Link' }));

    expect(
      await within(dialog).findByText('Line item L-1005 is in the recycle bin — restore it first.'),
    ).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('names who changed the transaction meanwhile (D-030)', async () => {
    const { user } = renderDialog({
      [LINK]: {
        status: 409,
        body: {
          error: {
            code: 'version_conflict',
            message: 'Someone else changed this GSAP transaction.',
            conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-10-20T09:30:00Z' },
          },
        },
      },
    });

    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByRole('textbox', { name: /Line item ID/ }), 'L-1007');
    await user.click(within(dialog).getByRole('button', { name: 'Link' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      /^Not saved — Daniel Okoro changed this GSAP transaction (at|on) /,
    );
  });
});
