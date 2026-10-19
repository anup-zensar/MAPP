import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { TextField } from './TextField';
import type { TextFieldProps } from './TextField';

/** A TextField whose value is kept the way a form keeps it. */
function Field(props: Omit<TextFieldProps, 'value' | 'onChange'>) {
  const [value, setValue] = useState('');
  return (
    <>
      <TextField {...props} value={value} onChange={setValue} />
      <output>{value}</output>
    </>
  );
}

describe('TextField', () => {
  it('is named by its label and passes on what is typed (FR-019)', async () => {
    const user = userEvent.setup();
    render(<Field label="Name" name="name" />);

    await user.type(screen.getByRole('textbox', { name: 'Name' }), 'Ideas');

    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveValue('Ideas');
    expect(screen.getByRole('status')).toHaveTextContent('Ideas');
  });

  it('uses its name as the input id, so the error summary can link to it (FR-020)', () => {
    render(<Field label="E-mail" name="email" type="email" autoComplete="username" />);

    const input = screen.getByRole('textbox', { name: 'E-mail' });
    expect(input).toHaveAttribute('id', 'email');
    expect(input).toHaveAttribute('type', 'email');
    expect(input).toHaveAttribute('autocomplete', 'username');
  });

  it('shows the marking as a word and the hint, both read with the field (FR-019)', () => {
    render(
      <Field
        label="Name"
        name="name"
        marking="Required"
        hint="Up to 80 characters."
        maxLength={80}
      />,
    );

    expect(screen.getByText('Required')).toBeInTheDocument();
    const input = screen.getByRole('textbox', { name: 'Name' });
    expect(input).toHaveAccessibleDescription('Required Up to 80 characters.');
    expect(input).toHaveAttribute('maxLength', '80');
  });

  it('shows its problem under the field and marks the field as invalid (FR-020)', () => {
    render(<Field label="Name" name="name" error="Enter a name." />);

    const input = screen.getByRole('textbox', { name: 'Name' });
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAccessibleDescription('Enter a name.');
    expect(screen.getByText('Enter a name.')).toBeVisible();
  });

  it('is not marked invalid without a problem (FR-020)', () => {
    render(<Field label="Name" name="name" />);

    expect(screen.getByRole('textbox', { name: 'Name' })).not.toHaveAttribute('aria-invalid');
  });
});
