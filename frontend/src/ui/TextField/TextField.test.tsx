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

  it('marks a field required to save with an asterisk and reads "required" and the hint with it (FR-019, 038 FR-063)', () => {
    render(
      <Field
        label="Name"
        name="name"
        marking="Required"
        hint="Up to 80 characters."
        maxLength={80}
      />,
    );

    expect(screen.getByText('*')).toHaveAttribute('aria-hidden', 'true');
    const input = screen.getByRole('textbox', { name: 'Name' });
    expect(input).toHaveAttribute('aria-required', 'true');
    expect(input).toHaveAccessibleDescription('required Up to 80 characters.');
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

  it('explains the field in an ⓘ next to the label (feature 002)', () => {
    render(<Field label="Region" name="region" tip="The region the money is spent in." />);

    expect(screen.getByRole('button', { name: 'About Region' })).toBeInTheDocument();
  });

  it('offers suggestions that can still be ignored (feature 002, budget names)', () => {
    render(<Field label="Budget name" name="budgetName" suggestions={['Europe', 'Asia']} />);

    // An input with suggestions is a combobox for screen readers.
    const input = screen.getByRole('combobox', { name: 'Budget name' });
    const list = document.getElementById(input.getAttribute('list') ?? '');
    expect(list?.tagName).toBe('DATALIST');
    expect([...(list?.querySelectorAll('option') ?? [])].map((option) => option.value)).toEqual([
      'Europe',
      'Asia',
    ]);
  });
});
