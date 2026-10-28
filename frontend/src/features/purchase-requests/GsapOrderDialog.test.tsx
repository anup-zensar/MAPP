import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  GIFT_BOXES,
  money,
  orderedPurchaseRequest,
  prLine,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { GsapOrderButton } from './GsapOrderButton';

const RUN = 'POST /api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order';
const ACTION = 'Create GSAP PR and PO (simulated)';

const GIFT_PR = routedPurchaseRequest({
  lines: [
    prLine({
      id: 'PRL-1003',
      lineItem: GIFT_BOXES,
      hxKey: 'HX1011',
      currency: 'USD',
      amount: money('10000.00', 'USD'),
      gsapPending: true,
    }),
  ],
});

function renderRun(pr: typeof GIFT_PR, answer: Answer<typeof RUN>) {
  const api = mockApi([
    reply(RUN, answer, { purchaseRequestId: 'PR-1002' }),
    // A 403 makes the session reload the person's access (014 FR-020).
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
  return {
    api,
    ...renderWithProviders(<GsapOrderButton purchaseRequest={pr} />, { user: testUser() }),
  };
}

describe('033 FR-001/FR-006/FR-010: Create GSAP PR and PO (simulated)', () => {
  it('confirms what the simulated GSAP will create, then sends the version seen (US1 sc.1–2)', async () => {
    const { api, user } = renderRun(GIFT_PR, { body: orderedPurchaseRequest() });

    await user.click(screen.getByRole('button', { name: ACTION }));
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Create GSAP PR and PO (simulated)?',
    });
    expect(dialog).toHaveTextContent(
      'The simulated GSAP will create a GSAP PR and a purchase order with 1 line (10,000.00 USD) for PR-1002.',
    );
    expect(
      within(dialog).getByRole('list', { name: 'Lines sent to the simulated GSAP' }),
    ).toHaveTextContent('Gift boxes (L-1011) · HX1011 · 10,000.00 USD');
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Create' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent(
      'Simulated GSAP created GSAP PR SIM-0010001001 and PO SIM-4500001001',
    );
    expect(
      api.bodiesOf('POST /api/v1/simulated-gsap/purchase-requests/PR-1002/purchase-order'),
    ).toEqual([{ version: 3 }]);
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('ignores a second click while the run is under way', async () => {
    const { api, user } = renderRun(GIFT_PR, () => pending());

    await user.click(screen.getByRole('button', { name: ACTION }));
    const dialog = await screen.findByRole('alertdialog');
    const create = within(dialog).getByRole('button', { name: 'Create' });
    await user.click(create);
    await user.click(create);

    expect(create).toHaveAttribute('aria-busy', 'true');
    expect(api.requests.filter((request) => request.method === 'POST')).toHaveLength(1);
  });

  it.each([
    [403, apiError('forbidden', "Importing GSAP extracts isn't part of your role.")],
    [
      404,
      apiError('not_found', "This purchase request doesn't exist or you don't have access to it."),
    ],
    [
      409,
      apiError(
        'not_allowed',
        'Wait until the change of this purchase request is approved or declined.',
      ),
    ],
  ] as const)('shows the %s refusal in the dialog', async (status, body) => {
    const { user } = renderRun(GIFT_PR, { status, body });

    await user.click(screen.getByRole('button', { name: ACTION }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Create' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(body.error.message);
  });
});

describe('033 FR-007: top-up', () => {
  it('says which PO lines change and which are added, and confirms with Update PO', async () => {
    const ordered = orderedPurchaseRequest();
    const [first] = ordered.lines;
    if (!first) {
      throw new Error('The fixture has a line.');
    }
    const changed = routedPurchaseRequest({
      lines: [
        { ...first, amount: money('9500.00'), gsapPending: true },
        prLine({
          id: 'PRL-1004',
          position: 2,
          lineItem: GIFT_BOXES,
          hxKey: 'HX1011',
          gsapPending: true,
        }),
      ],
    });
    const { user } = renderRun(changed, { body: ordered });

    await user.click(screen.getByRole('button', { name: ACTION }));
    const dialog = await screen.findByRole('alertdialog', { name: 'Update the PO (simulated)?' });
    expect(dialog).toHaveTextContent(
      'The simulated GSAP will update PO SIM-4500001001: 1 line changed, 1 line added (top-up).',
    );
    expect(within(dialog).getByRole('button', { name: 'Update PO' })).toBeInTheDocument();
    await expectNoAxeViolations();
  });
});
