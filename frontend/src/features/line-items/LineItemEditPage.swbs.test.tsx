import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { details, lineItem } from '@/testing/fixtures/line-items';
import { formAnswers, formOptions } from '@/testing/fixtures/line-item-forms';
import { reply } from '@/testing/replies';
import type { LineItem } from './api';
import { LineItemEditPage } from './LineItemEditPage';

function renderEdit(item: LineItem) {
  mockApi([
    ...formAnswers(),
    reply(
      'GET /api/v1/line-items/{lineItemId}',
      { body: details({ lineItem: item }) },
      { lineItemId: 'L-1001' },
    ),
    reply(
      'GET /api/v1/activities/{activityId}/line-items/form-options',
      { body: formOptions() },
      { activityId: 'A-1001' },
    ),
  ]);
  renderWithProviders(<LineItemEditPage />, {
    route: '/line-items/L-1001/edit',
    path: '/line-items/:lineItemId/edit',
    user: testUser(),
  });
}

describe('030 FR-004, FR-018: the request stays on once the code is received', () => {
  it('locks the request and says why, and shows the received code', async () => {
    renderEdit(
      lineItem({
        swbsStatus: 'received',
        swbsCode: 'SIM-DE03-0001',
        swbsRequestedAt: '2026-10-19T09:00:00Z',
        swbsReceivedAt: '2026-10-19T09:00:00Z',
      }),
    );

    const request = await screen.findByLabelText('Request SWBS code from GSAP');
    expect(request).toBeChecked();
    expect(request).toBeDisabled();
    expect(request).toHaveAccessibleDescription(
      expect.stringContaining(
        'This line item already has its SWBS code from the simulated GSAP, so the request stays on.',
      ),
    );
    expect(screen.getByText('SIM-DE03-0001')).toBeInTheDocument();
  });

  it('leaves the request free to change before a code arrives', async () => {
    renderEdit(lineItem());

    const request = await screen.findByLabelText('Request SWBS code from GSAP');
    expect(request).toBeEnabled();
    expect(
      screen.getByText(
        'Filled by the simulated GSAP once the activity is approved and the request is sent',
      ),
    ).toBeInTheDocument();
  });
});
