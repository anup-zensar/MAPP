import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { FormEvent } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { Button } from './Button';

describe('Button', () => {
  it('is a button named by its label, which never submits a form by accident (FR-019)', () => {
    render(<Button variant="primary">Create plan</Button>);

    const button = screen.getByRole('button', { name: 'Create plan' });
    expect(button).toHaveAttribute('type', 'button');
  });

  it('submits its form when it is the form’s submit button', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn((event: FormEvent) => event.preventDefault());
    render(
      <form onSubmit={onSubmit}>
        <Button type="submit" variant="primary">
          Sign in
        </Button>
      </form>,
    );

    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });
});
