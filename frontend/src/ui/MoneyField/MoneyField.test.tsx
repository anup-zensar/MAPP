import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { MoneyField } from './MoneyField';

function Example({ start = '' }: { start?: string }) {
  const [value, setValue] = useState(start);
  return (
    <>
      <MoneyField
        label="Plan budget"
        name="budget"
        currency="USD"
        marking="Needed to publish"
        value={value}
        onChange={setValue}
      />
      <button type="button">Elsewhere</button>
    </>
  );
}

describe('MoneyField', () => {
  it('shows the currency inside the field and tells screen readers (design system §5)', () => {
    renderWithProviders(<Example />);

    const field = screen.getByRole('textbox', { name: 'Plan budget' });
    expect(field).toHaveAttribute('inputmode', 'decimal');
    expect(field).toHaveAccessibleDescription('USD Needed to publish');
    expect(screen.getByText('USD')).toBeInTheDocument();
  });

  it.each([
    ['12500', '12,500.00'],
    ['12,500.5', '12,500.50'],
    ['12.5k', '12,500.00'],
  ])('writes "%s" out as %s on leaving the field', async (typed, written) => {
    const { user } = renderWithProviders(<Example />);
    const field = screen.getByRole('textbox', { name: 'Plan budget' });

    await user.type(field, typed);
    await user.click(screen.getByRole('button', { name: 'Elsewhere' }));

    expect(field).toHaveValue(written);
    expect(screen.queryByText(/Rounded to/)).not.toBeInTheDocument();
  });

  it('rounds more than two decimals to the cent and says so (D-190)', async () => {
    const { user } = renderWithProviders(<Example />);
    const field = screen.getByRole('textbox', { name: 'Plan budget' });

    await user.type(field, '1234.567');
    await user.tab();

    expect(field).toHaveValue('1,234.57');
    expect(screen.getByText('Rounded to 1,234.57')).toBeInTheDocument();

    await user.type(field, '9');
    expect(screen.queryByText(/Rounded to/)).not.toBeInTheDocument();
  });

  it('keeps text that is not an amount as typed', async () => {
    const { user } = renderWithProviders(<Example start="twelve" />);
    const field = screen.getByRole('textbox', { name: 'Plan budget' });

    await user.click(field);
    await user.tab();

    expect(field).toHaveValue('twelve');
  });
});
