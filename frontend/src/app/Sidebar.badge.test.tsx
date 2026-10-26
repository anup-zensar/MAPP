import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { CountBadge } from '@/ui/CountBadge/CountBadge';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { Sidebar } from './Sidebar';

// Feature 008 fills the Approvals badge from its own folder; here it counts three.
vi.mock('@/features/approvals/ApprovalsNavBadge', () => ({
  ApprovalsNavBadge: ({ collapsed }: { collapsed: boolean }) => (
    <span data-collapsed={collapsed}>
      <CountBadge count={3} label="waiting for you" />
    </span>
  ),
}));

describe('Sidebar badges (feature 008 extension point)', () => {
  it('shows an item’s badge inside its link, so the count is part of its name', () => {
    renderWithProviders(<Sidebar />, { user: testUser() });

    expect(screen.getByRole('link', { name: 'Approvals 3 waiting for you' })).toHaveAttribute(
      'href',
      '/approvals',
    );
  });

  it('tells the badge when the sidebar is the icon rail', () => {
    const collapsed = testUser({
      preferences: { ...testUser().preferences, sidebarCollapsed: true },
    });
    renderWithProviders(<Sidebar />, { user: collapsed });

    expect(screen.getByText('3').closest('[data-collapsed]')).toHaveAttribute(
      'data-collapsed',
      'true',
    );
  });
});
