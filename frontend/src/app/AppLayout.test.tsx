import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import type { CurrentUser } from '@/session/useSession';
import { testUser } from '@/testing/fixtures/users';
import { AppLayout } from './AppLayout';

const PRIYA: CurrentUser = testUser();

function renderLayout(user: CurrentUser = PRIYA) {
  return renderWithProviders(
    <AppLayout>
      <h1>A page</h1>
    </AppLayout>,
    { user },
  );
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('AppLayout', () => {
  it('starts with "Skip to main content", the first thing the keyboard reaches (SC-006)', async () => {
    const { user } = renderLayout();

    await user.tab();

    const skipLink = screen.getByRole('link', { name: 'Skip to main content' });
    expect(skipLink).toHaveFocus();
    expect(skipLink).toHaveAttribute('href', '#main');
  });

  it('puts the page in the main region that the skip link leads to (FR-010)', () => {
    renderLayout();

    const main = screen.getByRole('main');
    expect(main).toHaveAttribute('id', 'main');
    expect(main).toHaveAttribute('tabindex', '-1');
    expect(main).toContainElement(screen.getByRole('heading', { name: 'A page' }));
  });

  it('shows the demo banner and the footer (FR-016, design system §2)', () => {
    renderLayout();

    expect(
      screen.getByText('Demo environment — sign-in and data are for demonstration only'),
    ).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toHaveTextContent('© MAPP — demo data');
  });

  it('shows the sidebar, and the menu button that narrow screens use instead (FR-010, FR-011)', () => {
    renderLayout();

    expect(screen.getByRole('navigation', { name: 'Main' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open menu' })).toBeInTheDocument();
  });

  it('shows the sidebar as the icon rail when the person collapsed it (FR-011)', () => {
    renderLayout({
      ...PRIYA,
      preferences: { theme: 'system', sidebarCollapsed: true, pinnedPlanIds: [] },
    });

    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument();
  });
});
