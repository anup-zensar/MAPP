import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { PublicLayout } from './PublicLayout';

function renderPage(withThemeSwitch?: boolean) {
  return renderWithProviders(
    <PublicLayout withThemeSwitch={withThemeSwitch}>
      <h1>Welcome back</h1>
      <button type="button">Sign in</button>
    </PublicLayout>,
  );
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('PublicLayout', () => {
  it('starts with "Skip to main content", the first thing the keyboard reaches (SC-006)', async () => {
    const { user } = renderPage();

    await user.tab();

    const skipLink = screen.getByRole('link', { name: 'Skip to main content' });
    expect(skipLink).toHaveFocus();
    expect(skipLink).toHaveAttribute('href', '#main');
  });

  it('shows the demo banner on the page (FR-016)', () => {
    renderPage();

    expect(
      screen.getByText('Demo environment — sign-in and data are for demonstration only'),
    ).toBeInTheDocument();
  });

  it('shows the MAPP logo in its header (FR-001)', () => {
    renderPage();

    expect(screen.getByRole('banner')).toHaveTextContent('MAPP');
  });

  it('has the light/dark switch in its header (FR-014, research R16)', () => {
    renderPage();

    expect(
      within(screen.getByRole('banner')).getByRole('button', { name: 'Switch to dark mode' }),
    ).toBeInTheDocument();
  });

  it('can leave the switch out, for the error page that shows when the app itself failed', () => {
    renderPage(false);

    expect(screen.queryByRole('button', { name: 'Switch to dark mode' })).not.toBeInTheDocument();
  });

  it('puts the page in the main region that the skip link leads to (SC-006)', () => {
    renderPage();

    const main = screen.getByRole('main');
    expect(main).toHaveAttribute('id', 'main');
    expect(main).toHaveAttribute('tabindex', '-1');
    expect(main).toContainElement(screen.getByRole('heading', { name: 'Welcome back' }));
  });
});
