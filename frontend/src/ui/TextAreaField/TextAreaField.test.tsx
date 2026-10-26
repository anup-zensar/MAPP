import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { TextAreaField } from './TextAreaField';

function Example({ start = '', error }: { start?: string; error?: string }) {
  const [value, setValue] = useState(start);
  return (
    <TextAreaField
      label="Description"
      name="description"
      marking="Required"
      hint="What the plan is for."
      tip="Shown on the plan page."
      maxLength={100}
      rows={3}
      value={value}
      onChange={setValue}
      error={error}
    />
  );
}

describe('TextAreaField', () => {
  it('is a labelled multi-line field with its marking, hint and ⓘ (design system §7)', async () => {
    const { user } = renderWithProviders(<Example error="Enter a description." />);
    const field = screen.getByRole('textbox', { name: 'Description' });

    expect(field).toHaveAttribute('rows', '3');
    expect(field).toHaveAccessibleDescription(
      'Required What the plan is for. Enter a description.',
    );
    expect(screen.getByRole('button', { name: 'About Description' })).toBeInTheDocument();

    await user.type(field, 'Line one{Enter}Line two');
    expect(field).toHaveValue('Line one\nLine two');
  });

  it('counts down in the last 50 characters and never cuts typing off', async () => {
    const { user } = renderWithProviders(<Example start={'x'.repeat(49)} />);
    const field = screen.getByRole('textbox', { name: 'Description' });

    expect(screen.queryByText(/characters left/)).not.toBeInTheDocument();

    await user.type(field, 'x');
    expect(screen.getByText('50 characters left')).toBeInTheDocument();

    await user.type(field, 'x'.repeat(50));
    expect(screen.getByText('0 characters left')).toBeInTheDocument();

    await user.type(field, 'xx');
    expect(screen.getByText('2 characters too many')).toBeInTheDocument();
    expect(field).toHaveValue('x'.repeat(102));
  });
});
