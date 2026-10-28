import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { lineItem } from '@/testing/fixtures/line-items';
import type { LineItem } from './api';
import { SwbsCard } from './SwbsCard';

const PLANNER = testUser({
  access: {
    roles: [],
    permissions: [{ key: 'plans.view', all: true, teams: [] }],
    administrators: [],
    noSelfApproval: true,
  },
});

function renderCard(item: LineItem, user = testUser()) {
  mockApi([]);
  renderWithProviders(<SwbsCard lineItem={item} />, { user });
  return screen.getByRole('region', { name: 'SWBS request' });
}

describe('030 FR-014, FR-001: the SWBS request card', () => {
  it.each([
    [
      'notRequested',
      'Not requested',
      "'Request SWBS code from GSAP' is off, so no code will be requested.",
    ],
    [
      'waitingForApproval',
      'Waiting for activity approval',
      'The request is sent once the activity is approved.',
    ],
    ['readyToSend', 'Ready to send to GSAP', 'Sent with the next run of the simulated GSAP.'],
  ] as const)(
    'shows %s as "%s" with its explanation and no code yet',
    (status, word, explanation) => {
      const card = renderCard(lineItem({ swbsStatus: status }), PLANNER);

      expect(within(card).getByText(word)).toBeInTheDocument();
      expect(within(card).getByText(explanation)).toBeInTheDocument();
      expect(within(card).getByText('Not received yet')).toBeInTheDocument();
      expect(within(card).queryByText('Simulated GSAP')).not.toBeInTheDocument();
    },
  );

  it('shows a received code with its dates and source', async () => {
    const card = renderCard(
      lineItem({
        id: 'L-1003',
        swbsStatus: 'received',
        swbsCode: 'SIM-DE03-0001',
        swbsRequestedAt: '2026-10-19T09:00:00Z',
        swbsReceivedAt: '2026-10-19T09:00:00Z',
      }),
    );

    expect(within(card).getByText('SWBS code received')).toBeInTheDocument();
    expect(within(card).getByText('SIM-DE03-0001')).toBeInTheDocument();
    expect(within(card).getByText('Simulated GSAP')).toBeInTheDocument();
    expect(within(card).getAllByText(/19 Oct 2026/)).toHaveLength(2);
    expect(within(card).getByText('HX1003')).toBeInTheDocument();
    expect(
      within(card).queryByRole('button', { name: 'Send to simulated GSAP now' }),
    ).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('030 FR-015: offers Send now only to a person who may import, for a line item ready to send', () => {
    const ready = lineItem({ swbsStatus: 'readyToSend' });
    const importer = renderCard(ready);
    expect(
      within(importer).getByRole('button', { name: 'Send to simulated GSAP now' }),
    ).toBeInTheDocument();
  });

  it('030 FR-015: hides Send now from a person who may not import', () => {
    const card = renderCard(lineItem({ swbsStatus: 'readyToSend' }), PLANNER);

    expect(
      within(card).queryByRole('button', { name: 'Send to simulated GSAP now' }),
    ).not.toBeInTheDocument();
  });
});
