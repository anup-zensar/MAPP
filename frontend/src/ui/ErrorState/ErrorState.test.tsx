import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '@/api/errors';
import { ErrorState } from './ErrorState';

describe('ErrorState', () => {
  it('says plainly that the server cannot be reached, and offers Try again (FR-020)', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<ErrorState error={new ApiError('unreachable')} onRetry={onRetry} />);

    expect(screen.getByRole('alert')).toHaveTextContent(
      "SP&A One can't reach its server right now. Check that it is running and try again.",
    );

    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('shows the API’s own plain-language message (FR-020)', () => {
    const error = ApiError.fromBody(404, {
      error: {
        code: 'not_found',
        message: "This page or record doesn't exist.",
        fields: [],
      },
    });

    render(<ErrorState error={error} onRetry={vi.fn()} />);

    expect(screen.getByRole('alert')).toHaveTextContent("This page or record doesn't exist.");
  });

  it('never shows technical details of an unexpected error (FR-020)', () => {
    render(
      <ErrorState
        error={new TypeError("Cannot read properties of undefined (reading 'name')")}
        onRetry={vi.fn()}
      />,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong. Please try again.');
    expect(screen.queryByText(/Cannot read properties/)).not.toBeInTheDocument();
  });
});
