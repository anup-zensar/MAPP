import { act, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import type { CurrentUser } from '@/session/useSession';
import { testUser } from '@/testing/fixtures/users';
import { NavDrawer } from './NavDrawer';

const PRIYA: CurrentUser = testUser({
  preferences: { ...testUser().preferences, sidebarCollapsed: true },
});

/** Pretends the window is narrower than 960 px; widen() makes it 960 px or wider. */
function fakeWindowWidth() {
  let wide = false;
  const listeners = new Set<() => void>();
  vi.stubGlobal('matchMedia', (query: string) => ({
    get matches() {
      return query === '(min-width: 960px)' && wide;
    },
    media: query,
    addEventListener: (_type: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_type: string, listener: () => void) => listeners.delete(listener),
  }));
  return {
    widen() {
      wide = true;
      act(() => listeners.forEach((listener) => listener()));
    },
  };
}

async function openDrawer() {
  const rendered = renderWithProviders(<NavDrawer />, { user: PRIYA, route: '/' });
  await rendered.user.click(screen.getByRole('button', { name: 'Open menu' }));
  return { ...rendered, drawer: screen.getByRole('dialog', { name: 'Menu' }) };
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('NavDrawer', () => {
  it('opens the whole sidebar as a drawer from the menu button (US2 scenario 4)', async () => {
    const { drawer } = await openDrawer();

    const navigation = within(drawer).getByRole('navigation', { name: 'Main' });
    expect(within(navigation).getAllByRole('link')).toHaveLength(11);
    // Always the expanded sidebar, even for a person who collapsed it on a wide screen.
    expect(within(navigation).getByText('Plans')).not.toHaveClass('visually-hidden');
    expect(within(drawer).queryByRole('button', { name: 'Collapse' })).not.toBeInTheDocument();
    expect(within(drawer).getByRole('button', { name: 'Close menu' })).toBeInTheDocument();
  });

  it('shows the logo, which leads Home, above the sections (plan "Layout")', async () => {
    const { drawer } = await openDrawer();

    expect(within(drawer).getByRole('link', { name: 'SP&A One home' })).toHaveAttribute(
      'href',
      '/',
    );
  });

  it('closes with one Escape, also with the focus on a section link (US2 scenario 4)', async () => {
    const { user, drawer } = await openDrawer();
    act(() => within(drawer).getByRole('link', { name: 'Settings' }).focus());

    await user.keyboard('{Escape}');

    expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('button', { name: 'Open menu' })).toHaveFocus());
  });

  it('closes with the Close menu button (US2 scenario 4)', async () => {
    const { user, drawer } = await openDrawer();

    await user.click(within(drawer).getByRole('button', { name: 'Close menu' }));

    expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
  });

  it('closes after choosing an item, which opens its page (US2 scenario 4)', async () => {
    const { user, router, drawer } = await openDrawer();

    await user.click(within(drawer).getByRole('link', { name: 'Plans' }));

    expect(router.state.location.pathname).toBe('/plans');
    expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
  });

  it('closes when the window becomes wide enough for the sidebar (plan "Layout")', async () => {
    const windowWidth = fakeWindowWidth();
    await openDrawer();

    windowWidth.widen();

    expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
  });

  it('explains the menu button with a tooltip (research R17)', async () => {
    const { user } = renderWithProviders(<NavDrawer />, { user: PRIYA });

    await user.hover(screen.getByRole('button', { name: 'Open menu' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Open menu');
  });
});
