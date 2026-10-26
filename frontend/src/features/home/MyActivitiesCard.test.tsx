import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import type { MyActivityGroup } from './homeApi';
import { MyActivitiesCard } from './MyActivitiesCard';
import { groupOf, myActivityOf, usd } from '@/testing/fixtures/home';

const GROUPS: MyActivityGroup[] = [
  groupOf('declined', [myActivityOf({ id: 'A-1043', name: 'Trade show' })]),
  groupOf('created', [
    myActivityOf({ id: 'A-1046', name: 'Spring radio', startDate: null, endDate: null }),
    myActivityOf({ id: 'A-1047', name: 'Dealer newsletter', ongoing: true, endDate: null }),
  ]),
  groupOf('submitted', [myActivityOf({ id: 'A-1045', name: 'Dealer summit' })]),
  groupOf('approved', [myActivityOf()]),
  groupOf('inProgress', [myActivityOf({ id: 'A-1048', name: 'Workshop tour' })]),
  groupOf('complete', [myActivityOf({ id: 'A-1049', name: 'Winter promo' })]),
];

const page = (groups: MyActivityGroup[]) => ({
  body: { groups, total: groups.reduce((sum, group) => sum + group.count, 0) },
});

function renderCard(answer: MockHandler = page(GROUPS)) {
  mockApi({ 'GET /api/v1/home/my-activities': answer });
  return renderWithProviders(<MyActivitiesCard />, { user: testUser(), path: '/' });
}

function card() {
  return screen.getByRole('region', { name: 'My activities' });
}

function rowOf(name: string): HTMLElement {
  return within(card()).getByRole('link', { name }).closest('li') as HTMLElement;
}

describe('MyActivitiesCard', () => {
  it('groups the activities by status in order, each with its chip and count, Complete folded (FR-014, US4 scenario 1)', async () => {
    renderCard();

    await within(card()).findByRole('link', { name: 'Trade show' });
    const toggles = within(card()).getAllByRole('button', { expanded: true });
    expect(toggles.map((toggle) => toggle.textContent)).toEqual([
      'Declined 1',
      'Created 2',
      'Submitted 1',
      'Approved 1',
      'In progress 1',
    ]);
    expect(within(card()).getByRole('button', { name: 'Complete 1' })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
    expect(within(card()).queryByRole('link', { name: 'Winter promo' })).not.toBeInTheDocument();
    expect(card()).toHaveTextContent('Activities you own, by status');
  });

  it('unfolds and folds a group on its button (FR-014)', async () => {
    const { user } = renderCard();

    await user.click(await within(card()).findByRole('button', { name: 'Complete 1' }));
    expect(within(card()).getByRole('link', { name: 'Winter promo' })).toBeInTheDocument();

    await user.click(within(card()).getByRole('button', { name: 'Declined 1' }));
    expect(within(card()).queryByRole('link', { name: 'Trade show' })).not.toBeInTheDocument();
  });

  it('shows each activity with its plan and campaign, dates, planned cost and completeness (FR-014)', async () => {
    renderCard();

    const name = await within(card()).findByRole('link', { name: 'Launch film' });
    expect(name).toHaveAttribute('href', '/plans/P-1001/activities/A-1042');
    const row = within(rowOf('Launch film'));
    expect(row.getByText('A-1042')).toBeInTheDocument();
    expect(row.getByText('FY2026 Lubricants Europe › Helix launch')).toBeInTheDocument();
    expect(row.getByText('01 Mar – 31 May 2026')).toBeInTheDocument();
    expect(row.getByText('$85,000.00')).toBeInTheDocument();
    expect(row.getByText('All filled in')).toBeInTheDocument();

    expect(within(rowOf('Spring radio')).getByText('No dates yet')).toBeInTheDocument();
    expect(
      within(rowOf('Dealer newsletter')).getByText('Ongoing from 01 Mar 2026'),
    ).toBeInTheDocument();
  });

  it('says what an activity still needs, and opens a missing field in its form (FR-014)', async () => {
    const { user, router } = renderCard(
      page([
        groupOf('created', [
          myActivityOf({
            plannedCost: null,
            missing: [{ field: 'region', label: 'Region' }],
            itemsBelow: 2,
          }),
        ]),
      ]),
    );

    await within(card()).findByRole('link', { name: 'Launch film' });
    const row = within(rowOf('Launch film'));
    expect(row.getByText('—')).toBeInTheDocument();
    expect(row.getByText('2 line items need attention')).toBeInTheDocument();

    await user.click(row.getByRole('button', { name: '1 field needed' }));
    await user.click(await screen.findByRole('button', { name: 'Region' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042/edit'),
    );
    expect(router.state.location.search).toBe('?field=region');
  });

  it('shows five activities of a group, then all of them on "View all (n)" (US4 scenario 2)', async () => {
    const many = Array.from({ length: 7 }, (_, index) =>
      myActivityOf({ id: `A-20${index}`, name: `Activity ${index + 1}` }),
    );
    const { user } = renderCard(page([groupOf('approved', many)]));

    await within(card()).findByRole('link', { name: 'Activity 5' });
    expect(within(card()).queryByRole('link', { name: 'Activity 6' })).not.toBeInTheDocument();

    await user.click(within(card()).getByRole('button', { name: 'View all (7)' }));
    expect(within(card()).getByRole('link', { name: 'Activity 7' })).toBeInTheDocument();

    await user.click(within(card()).getByRole('button', { name: 'Show fewer' }));
    expect(within(card()).queryByRole('link', { name: 'Activity 6' })).not.toBeInTheDocument();
  });

  it('is the place the "Activities I own" card leads to (D-174)', async () => {
    renderCard();

    await within(card()).findByRole('link', { name: 'Trade show' });
    expect(card().closest('#my-activities')).not.toBeNull();
  });

  it('sends a person without activities to the Plans page (FR-021)', async () => {
    renderCard({ body: { groups: [], total: 0 } });

    expect(
      await within(card()).findByText("You don't own any activities yet."),
    ).toBeInTheDocument();
    expect(within(card()).getByRole('link', { name: 'Go to Plans' })).toHaveAttribute(
      'href',
      '/plans',
    );
  });

  it('shows a placeholder while loading, and Try again when it fails (FR-022)', async () => {
    let calls = 0;
    const { user } = renderCard(() =>
      ++calls === 1
        ? { status: 500 }
        : page([groupOf('approved', [myActivityOf({ plannedCost: usd('1.00') })])]),
    );

    expect(within(card()).getByRole('status')).toHaveTextContent('Loading…');
    expect(await within(card()).findByRole('alert')).toHaveTextContent(
      "This section couldn't load",
    );
    await user.click(within(card()).getByRole('button', { name: 'Try again' }));

    expect(await within(card()).findByRole('link', { name: 'Launch film' })).toBeInTheDocument();
  });
});
