import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { Sidebar } from './Sidebar';
import styles from './Sidebar.module.css';
import type { CurrentUser } from './useSession';
import { testUser } from '@/test/users';

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
      'Approvals',
      'All work',
      'Financials',
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
    const api = mockApi({
      'PATCH /api/me/preferences': { body: { theme: 'system', sidebarCollapsed: true } },
    });
    const { user } = renderWithProviders(<Sidebar />, { user: PRIYA });

    await user.click(screen.getByRole('button', { name: 'Collapse' }));

    expect(await screen.findByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ sidebarCollapsed: true }]);
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
    expect(plans).toHaveClass(styles.item);
    expect(plans).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Home' })).toHaveClass(styles.item);
  });

  it('shows each item’s name in a tooltip in the icon rail (FR-011)', async () => {
    const { user } = renderWithProviders(<Sidebar />, { user: COLLAPSED });

    await user.hover(screen.getByRole('link', { name: 'Recycle bin' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Recycle bin');
  });

  it('expands again and saves that for the person (FR-011)', async () => {
    const api = mockApi({
      'PATCH /api/me/preferences': { body: { theme: 'system', sidebarCollapsed: false } },
    });
    const { user } = renderWithProviders(<Sidebar />, { user: COLLAPSED });

    await user.click(screen.getByRole('button', { name: 'Expand sidebar' }));

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Collapse' })).toBeInTheDocument(),
    );
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ sidebarCollapsed: false }]);
  });
});
