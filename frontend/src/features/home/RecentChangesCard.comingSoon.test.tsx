import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { COMING_SOON_SECTIONS } from '@/app/navigation';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { RecentChangesCard } from './RecentChangesCard';

describe('RecentChangesCard while the Activity log is not built', () => {
  it('says Recent changes are coming soon and asks the API nothing (feature 011 set aside)', () => {
    // The test setup fails a test that makes a request without a canned answer.
    const api = mockApi({});
    expect(COMING_SOON_SECTIONS.map((section) => section.path)).toContain('/activity-log');

    renderWithProviders(<RecentChangesCard scope="myTeams" />, { user: testUser(), path: '/' });

    const card = screen.getByRole('region', { name: 'Recent changes' });
    expect(within(card).getByText('Coming soon')).toBeInTheDocument();
    expect(
      within(card).getByText('Recent changes will show here once the Activity log is built.'),
    ).toBeInTheDocument();
    expect(within(card).queryByRole('link', { name: 'View activity log' })).not.toBeInTheDocument();
    expect(api.requests).toEqual([]);
  });
});
