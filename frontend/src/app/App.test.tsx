import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { reply } from '@/testing/replies';
import { App } from './App';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('App', () => {
  it('opens the page of the current address, with the demo banner (FR-016)', async () => {
    // A first visit: nobody is signed in yet, so Home sends the visitor to the sign-in page.
    mockApi([
      reply('GET /api/v1/me', {
        status: 401,
        body: {
          error: { code: 'sign_in_required', message: 'Please sign in to continue.', fields: [] },
        },
      }),
      reply('GET /api/v1/auth/demo-people', { body: { password: 'mapp', people: [] } }),
    ]);

    render(<App />);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Sign in to MAPP' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Demo environment — sign-in and data are for demonstration only'),
    ).toBeInTheDocument();
    expect(screen.getByRole('main')).toBeInTheDocument();
  });
});
