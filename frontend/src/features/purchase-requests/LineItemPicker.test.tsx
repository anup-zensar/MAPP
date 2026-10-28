import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { STAND_BUILD, lineItemOption, money } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { LineItemPicker } from './LineItemPicker';

const OPTIONS = 'GET /api/v1/purchase-requests/line-item-options';

const MEDIA_BOOKING = lineItemOption({
  lineItem: { ...STAND_BUILD, id: 'L-1005', name: 'Media booking', companyCode: 'NL02' },
  hxKey: 'HX1005',
  availableToRequest: money('30000.00', 'USD'),
  available: false,
  reason: 'One purchase request cannot charge two company codes (GB01 and NL02).',
});
const NOTHING_LEFT = lineItemOption({
  lineItem: { ...STAND_BUILD, id: 'L-1007', name: 'Station signage' },
  hxKey: 'HX1007',
  availableToRequest: money('0.00'),
  available: false,
  reason: 'Nothing left to request',
});

function renderPicker(answer: Answer<typeof OPTIONS>, inForm: string[] = ['L-1006']) {
  const api = mockApi([reply(OPTIONS, answer)]);
  const onChoose = vi.fn();
  const onOpenChange = vi.fn();
  return {
    api,
    onChoose,
    onOpenChange,
    ...renderWithProviders(
      <LineItemPicker
        open
        onOpenChange={onOpenChange}
        query={{ purchaseRequestId: 'PR-1002' }}
        inForm={inForm}
        onChoose={onChoose}
      />,
      { user: testUser() },
    ),
  };
}

describe('032 FR-010/FR-001: line item picker', () => {
  it('lists what each line item still has available, and why the others are unavailable', async () => {
    renderPicker({
      body: {
        items: [
          lineItemOption(),
          MEDIA_BOOKING,
          NOTHING_LEFT,
          lineItemOption({
            lineItem: STAND_BUILD,
            hxKey: 'HX1006',
            availableToRequest: money('3200.00'),
          }),
        ],
        total: 4,
      },
    });

    expect(await screen.findByText('Gift boxes (L-1011)')).toBeInTheDocument();
    expect(screen.getByText('Available to request: 10,000.00 USD')).toBeInTheDocument();
    expect(
      screen.getByText('One purchase request cannot charge two company codes (GB01 and NL02).'),
    ).toBeInTheDocument();
    expect(screen.getByText('Nothing left to request')).toBeInTheDocument();
    expect(screen.getByText('Already on this purchase request')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: /^Add line/ })).toHaveLength(1);
    await expectNoAxeViolations();
  });

  it('adds the chosen line item and closes', async () => {
    const { onChoose, onOpenChange, user } = renderPicker({
      body: { items: [lineItemOption()], total: 1 },
    });

    await user.click(await screen.findByRole('button', { name: 'Add line Gift boxes (L-1011)' }));

    expect(onChoose).toHaveBeenCalledWith(lineItemOption());
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('searches by name, ID or HX key', async () => {
    const { api, user } = renderPicker(({ url }) => {
      const search = new URL(url).searchParams.get('search');
      return { body: search ? { items: [], total: 0 } : { items: [lineItemOption()], total: 1 } };
    });
    await screen.findByText('Gift boxes (L-1011)');

    await user.type(screen.getByRole('searchbox', { name: 'Search line items' }), 'HX9999');

    expect(await screen.findByText("No line items match 'HX9999'.")).toBeInTheDocument();
    await waitFor(() =>
      expect(api.requests.map((request) => request.search)).toContain(
        '?purchaseRequestId=PR-1002&search=HX9999',
      ),
    );
  });

  it('shows a placeholder while the line items load', async () => {
    renderPicker(() => pending());

    expect(await screen.findByText('Looking for line items…')).toBeInTheDocument();
  });

  it('shows a failure with Try again', async () => {
    renderPicker({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByText("The line items couldn't be loaded.")).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('closes with Escape', async () => {
    const { onOpenChange, user } = renderPicker({ body: { items: [], total: 0 } });
    await screen.findByText('No line items to choose from.');

    await user.keyboard('{Escape}');

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
