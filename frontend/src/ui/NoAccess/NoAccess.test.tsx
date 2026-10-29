import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { NoAccess } from './NoAccess';

describe('NoAccess', () => {
  it('shows the server’s sentence, which names nothing of the record, and a way back to Home (feature 014, FR-008)', async () => {
    const { user, router } = renderWithProviders(
      <NoAccess message="You don't have access to this plan. Ask an administrator if you need it." />,
      { route: '/plans/P-1001' },
    );

    expect(screen.getByRole('heading', { level: 1, name: 'No access' })).toBeInTheDocument();
    expect(
      screen.getByText("You don't have access to this plan. Ask an administrator if you need it."),
    ).toBeInTheDocument();
    expect(document.title).toBe('No access · SP&A One');

    await user.click(screen.getByRole('link', { name: 'Go to Home' }));
    expect(router.state.location.pathname).toBe('/');
  });

  it('says so for a page the person’s roles do not open (FR-012)', () => {
    renderWithProviders(<NoAccess />);

    expect(
      screen.getByText("You don't have access to this page. Ask an administrator if you need it."),
    ).toBeInTheDocument();
  });
});
