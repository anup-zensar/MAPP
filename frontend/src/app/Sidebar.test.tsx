import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import type { Permission } from '@/session/access';
import type { CurrentUser } from '@/session/useSession';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';
import { Sidebar } from './Sidebar';
import styles from './Sidebar.module.css';

const PRIYA: CurrentUser = testUser();

const COLLAPSED: CurrentUser = {
  ...PRIYA,
  preferences: { ...PRIYA.preferences, sidebarCollapsed: true },
};

function mainNavigation() {
  return screen.getByRole('navigation', { name: 'Main' });
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('Sidebar', () => {
  it('lists the sections in order, with Settings, Recycle bin and Activity log under More (FR-010)', () => {
    renderWithProviders(<Sidebar />, { user: PRIYA });

    const links = within(mainNavigation()).getAllByRole('link');
    // Trimmed: a link with a badge (Approvals, feature 008) has a space before it.
    expect(links.map((link) => link.textContent?.trim())).toEqual([
      'Home',
      'Plans',
      'Global budgets',
      'Approvals',
      'All work',
      'Financials',
      'Purchase requests',
      'Dashboards',
      'Settings',
      'Recycle bin',
      'Activity log',
    ]);
    const more = within(mainNavigation()).getByRole('list', { name: 'More' });
    expect(
      within(more)
        .getAllByRole('link')
        .map((link) => link.textContent?.trim()),
    ).toEqual(['Settings', 'Recycle bin', 'Activity log']);
    expect(within(mainNavigation()).getByRole('link', { name: 'Plans' })).toHaveAttribute(
      'href',
      '/plans',
    );
  });

  it('highlights the section on screen (US2 scenario 1, FR-011)', () => {
    renderWithProviders(<Sidebar />, { user: PRIYA, route: '/plans' });

    expect(screen.getByRole('link', { name: 'Plans' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Home' })).not.toHaveAttribute('aria-current');
  });

  it('highlights Home on Home only (FR-011)', () => {
    renderWithProviders(<Sidebar />, { user: PRIYA, route: '/' });

    expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Plans' })).not.toHaveAttribute('aria-current');
  });

  it('collapses and saves that for the person (FR-011, US2 scenario 3)', async () => {
    const api = mockApi([
      reply('PATCH /api/v1/me/preferences', {
        body: { ...PRIYA.preferences, theme: 'system', sidebarCollapsed: true },
      }),
    ]);
    const { user } = renderWithProviders(<Sidebar />, { user: PRIYA });

    await user.click(screen.getByRole('button', { name: 'Collapse' }));

    expect(await screen.findByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([{ sidebarCollapsed: true }]);
  });

  it('collapses to an icon rail whose links keep their names (FR-011)', () => {
    renderWithProviders(<Sidebar />, { user: COLLAPSED });

    const plans = screen.getByRole('link', { name: 'Plans' });
    // The label stays in the page for screen readers; it is only hidden on screen.
    expect(within(plans).getByText('Plans')).toHaveClass('visually-hidden');
    expect(screen.getByText('More')).toHaveClass('visually-hidden');
    expect(screen.queryByRole('button', { name: 'Collapse' })).not.toBeInTheDocument();
  });

  it('keeps the item style and the highlight in the icon rail, where each link has a tooltip (FR-011)', () => {
    renderWithProviders(<Sidebar />, { user: COLLAPSED, route: '/plans' });

    const plans = screen.getByRole('link', { name: 'Plans' });
    expect(plans).toHaveClass(String(styles.item));
    expect(plans).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Home' })).toHaveClass(String(styles.item));
  });

  it('shows each item’s name in a tooltip in the icon rail (FR-011)', async () => {
    const { user } = renderWithProviders(<Sidebar />, { user: COLLAPSED });

    await user.hover(screen.getByRole('link', { name: 'Recycle bin' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Recycle bin');
  });

  it('expands again and saves that for the person (FR-011)', async () => {
    const api = mockApi([
      reply('PATCH /api/v1/me/preferences', {
        body: { ...PRIYA.preferences, theme: 'system', sidebarCollapsed: false },
      }),
    ]);
    const { user } = renderWithProviders(<Sidebar />, { user: COLLAPSED });

    await user.click(screen.getByRole('button', { name: 'Expand sidebar' }));

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Collapse' })).toBeInTheDocument(),
    );
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([{ sidebarCollapsed: false }]);
  });
});

function withPermissions(keys: Permission[]): CurrentUser {
  return testUser({
    access: {
      roles: [],
      permissions: keys.map((key) => ({ key, all: true, teams: [] })),
      administrators: [],
      noSelfApproval: true,
    },
  });
}

describe('032 FR-044/FR-049: Purchase requests in the sidebar only with purchaseRequests.list', () => {
  it('shows Purchase requests after Financials to a requisitioner', () => {
    renderWithProviders(<Sidebar />, {
      user: withPermissions(['plans.view', 'purchaseRequests.list', 'finance.import']),
    });

    expect(screen.getByRole('link', { name: 'Purchase requests' })).toHaveAttribute(
      'href',
      '/purchase-requests',
    );
  });

  it('leaves it out for a planner who raises purchase requests but may not see the page', () => {
    renderWithProviders(<Sidebar />, {
      user: withPermissions(['plans.view', 'purchaseRequests.edit', 'purchaseRequests.submit']),
    });

    expect(screen.queryByRole('link', { name: 'Purchase requests' })).not.toBeInTheDocument();
  });
});
