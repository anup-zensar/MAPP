import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { ALEX_RUN } from '@/testing/fixtures/swbs-runs';
import { apiError, reply } from '@/testing/replies';
import { SendNowButton } from './SendNowButton';

const SEND = 'POST /api/v1/simulated-gsap/swbs-runs';

function renderButton(answer: Parameters<typeof reply<typeof SEND>>[1]) {
  const api = mockApi([reply(SEND, answer)]);
  return {
    api,
    ...renderWithProviders(<SendNowButton lineItemId="L-1006" lineItemName="Stand build" />, {
      user: testUser(),
    }),
  };
}

describe('030 FR-015: send one line item to the simulated GSAP now', () => {
  it('asks first, then sends only this line item', async () => {
    const { api, user } = renderButton({ status: 201, body: { ...ALEX_RUN, sent: 1 } });

    await user.click(screen.getByRole('button', { name: 'Send to simulated GSAP now' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Send Stand build to simulated GSAP now?',
    });
    expect(dialog).toHaveTextContent(
      "It receives its SWBS code at once. Codes can't be changed or removed in MAPP.",
    );
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Send to simulated GSAP now' }));

    expect(await screen.findByRole('button', { name: 'Send to simulated GSAP now' })).toBeVisible();
    expect(api.bodiesOf(SEND)).toEqual([{ lineItemIds: ['L-1006'] }]);
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('says so when the line item is no longer ready to send', async () => {
    const message =
      "Not sent — these line items aren't ready to send to the simulated GSAP: Stand build (L-1006). Reload to see their SWBS status.";
    const { user } = renderButton({ status: 409, body: apiError('not_allowed', message) });

    await user.click(screen.getByRole('button', { name: 'Send to simulated GSAP now' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Send to simulated GSAP now' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(message);
  });
});
