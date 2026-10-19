import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { App } from './App';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('App', () => {
  it('opens the page of the current address, with the demo banner (FR-016)', async () => {
    // A first visit: nobody is signed in yet, so Home sends the visitor to the sign-in page.
    mockApi({
      'GET /api/me': {
        status: 401,
        body: { code: 'sign_in_required', message: 'Please sign in to continue.', fields: [] },
      },
      'GET /api/auth/demo-people': { body: { password: 'northstar', people: [] } },
    });

    render(<App />);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Welcome back' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Demo environment — sign-in and data are for demonstration only'),
    ).toBeInTheDocument();
    expect(screen.getByRole('main')).toBeInTheDocument();
  });
});
