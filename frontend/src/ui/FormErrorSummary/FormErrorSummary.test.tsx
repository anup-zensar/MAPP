import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { FormErrorSummary } from './FormErrorSummary';

const BOTH_EMPTY = [
  { field: 'email', message: 'Enter your e-mail' },
  { field: 'password', message: 'Enter the password' },
];

function Form({ errors }: { errors: { field: string; message: string }[] }) {
  return (
    <form>
      <FormErrorSummary errors={errors} />
      <label htmlFor="email">E-mail</label>
      <input id="email" />
      <label htmlFor="password">Password</label>
      <input id="password" type="password" />
    </form>
  );
}

describe('FormErrorSummary', () => {
  it('lists every problem at once under a title that counts them (FR-020, FR-004)', () => {
    render(<Form errors={BOTH_EMPTY} />);

    expect(
      screen.getByRole('heading', { name: '2 problems need your attention' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Enter your e-mail' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Enter the password' })).toBeInTheDocument();
  });

  it('counts a single problem in the singular (FR-020)', () => {
    render(<Form errors={[nth(BOTH_EMPTY, 1)]} />);

    expect(
      screen.getByRole('heading', { name: '1 problem needs your attention' }),
    ).toBeInTheDocument();
  });

  it('takes the focus when it appears, so the problems are read out (FR-020)', () => {
    render(<Form errors={BOTH_EMPTY} />);

    expect(screen.getByRole('alert')).toHaveTextContent('2 problems need your attention');
    expect(screen.getByRole('alert').parentElement).toHaveFocus();
  });

  it('takes the focus again after the next failed attempt (FR-020)', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<Form errors={BOTH_EMPTY} />);
    await user.click(screen.getByRole('textbox', { name: 'E-mail' }));

    rerender(<Form errors={[nth(BOTH_EMPTY, 1)]} />);

    expect(screen.getByRole('alert').parentElement).toHaveFocus();
  });

  it('takes the person to the field when they choose a problem (FR-020)', async () => {
    const user = userEvent.setup();
    render(<Form errors={BOTH_EMPTY} />);

    await user.click(screen.getByRole('link', { name: 'Enter the password' }));

    expect(screen.getByLabelText('Password')).toHaveFocus();
  });

  it('finds fields whose input id differs from the field name (FR-020)', async () => {
    const user = userEvent.setup();
    render(
      <>
        <FormErrorSummary
          errors={[{ field: 'name', message: 'Enter a name.' }]}
          fieldId={(field) => `folder-${field}`}
        />
        <label htmlFor="folder-name">Name</label>
        <input id="folder-name" />
      </>,
    );

    await user.click(screen.getByRole('link', { name: 'Enter a name.' }));

    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveFocus();
  });

  it('shows nothing when there are no problems', () => {
    render(<Form errors={[]} />);

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
