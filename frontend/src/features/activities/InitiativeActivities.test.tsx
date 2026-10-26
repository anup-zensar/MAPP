import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { InitiativeActivities } from './InitiativeActivities';
import { launchFilmRow } from './testData';

describe('InitiativeActivities (feature 006, FR-001)', () => {
  it("lists the initiative's activities with their status, and offers Add activity", async () => {
    mockApi({
      'GET /api/v1/plans/P-1001/activities': {
        body: {
          items: [
            launchFilmRow(),
            launchFilmRow({ id: 'A-1050', name: 'Elsewhere', initiativeId: 'I-1002' }),
          ],
          total: 2,
        },
      },
    });
    renderWithProviders(<InitiativeActivities planId="P-1001" initiativeId="I-1001" />);

    const card = await screen.findByRole('region', { name: 'Activities' });
    const link = await within(card).findByRole('link', { name: 'Launch film' });
    expect(link).toHaveAttribute('href', '/plans/P-1001/activities/A-1042');
    expect(within(card).getByText('Created')).toBeVisible();
    expect(within(card).queryByText('Elsewhere')).not.toBeInTheDocument();
    expect(within(card).getByRole('link', { name: 'Add activity' })).toHaveAttribute(
      'href',
      '/plans/P-1001/initiatives/I-1001/activities/new',
    );
  });

  it('says when the initiative has no activities yet', async () => {
    mockApi({ 'GET /api/v1/plans/P-1001/activities': { body: { items: [], total: 0 } } });
    renderWithProviders(<InitiativeActivities planId="P-1001" initiativeId="I-1001" />);

    expect(await screen.findByText('No activities yet.')).toBeVisible();
  });
});
