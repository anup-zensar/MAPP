import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RouteErrorPage } from './RouteErrorPage';

/** A page with a programming mistake. */
function BrokenPage(): never {
  throw new TypeError("Cannot read properties of undefined (reading 'name')");
}

function renderBrokenPage() {
  // React reports the thrown error on the console; the test only looks at the screen.
  vi.spyOn(console, 'error').mockImplementation(() => {});
  const router = createMemoryRouter([
    { path: '/', element: <BrokenPage />, errorElement: <RouteErrorPage /> },
  ]);
  render(<RouterProvider router={router} />);
}

describe('RouteErrorPage', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows a friendly message instead of a blank page, never the technical one (FR-020)', () => {
    renderBrokenPage();

    expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong. Please try again.');
    expect(screen.queryByText(/Cannot read properties/)).not.toBeInTheDocument();
  });

  it('keeps the demo banner, like every page (FR-016)', () => {
    renderBrokenPage();

    expect(
      screen.getByText('Demo environment — sign-in and data are for demonstration only'),
    ).toBeInTheDocument();
  });

  it('works without the session, which may be what failed: no theme switch (FR-020)', () => {
    renderBrokenPage();

    expect(screen.queryByRole('button', { name: /^Switch to/ })).not.toBeInTheDocument();
  });

  it('reloads the page on Try again (FR-020)', async () => {
    const user = userEvent.setup();
    const reload = vi.fn();
    vi.stubGlobal('location', { ...window.location, reload });
    renderBrokenPage();

    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(reload).toHaveBeenCalledTimes(1);
  });
});
