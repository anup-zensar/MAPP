import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { PlanTable } from './PlanTable';
import { seedList } from './testData';

function renderTable() {
  return renderWithProviders(<PlanTable plans={seedList().items} actionsFor={() => []} />, {
    user: testUser(),
    route: '/plans',
    path: '/plans',
  });
}

function row(name: string) {
  return screen.getByRole('link', { name }).closest('tr') as HTMLElement;
}

describe('PlanTable (FR-002)', () => {
  it('shows one row per plan with its facts', () => {
    renderTable();

    expect(screen.getAllByRole('columnheader').map((header) => header.textContent)).toEqual([
      'Name',
      'Year',
      'Status',
      'Completeness',
      'Owner',
      'Teams',
      'Business unit',
      'Budget used',
      'Activities approved',
      'Actions',
    ]);
    const europe = row('FY2026 Lubricants Europe');
    expect(europe).toHaveTextContent('P-1001');
    expect(europe).toHaveTextContent('80 %');
    expect(europe).toHaveTextContent('4 of 7 activities approved');
  });

  it('turns the bar red above 100 % and says when there is no budget', () => {
    renderTable();

    expect(
      within(row('FY2025 Lubricants Europe')).getByRole('progressbar', { name: 'Budget used' }),
    ).toHaveAttribute('aria-valuetext', '108 %');
    expect(within(row('FY2026 Mobility China')).getByText('No budget set')).toBeInTheDocument();
  });

  it('links completeness to the field to fill in (FR-017)', async () => {
    const { user, router } = renderTable();

    await user.click(
      within(row('FY2026 Mobility China')).getByRole('button', { name: '1 field needed' }),
    );
    await user.click(await screen.findByRole('button', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
  });
});
