import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { DecimalField } from './DecimalField';

function Example({ decimals, suffix }: { decimals: 0 | 2 | 6; suffix?: string }) {
  const [value, setValue] = useState('');
  return (
    <DecimalField
      label="Rate"
      name="rate"
      decimals={decimals}
      suffix={suffix}
      value={value}
      onChange={setValue}
    />
  );
}

describe('DecimalField', () => {
  it('names its unit and asks for a decimal keyboard (feature 002)', () => {
    renderWithProviders(<Example decimals={2} suffix="%" />);

    const field = screen.getByRole('textbox', { name: 'Rate' });
    expect(field).toHaveAttribute('inputmode', 'decimal');
    expect(field).toHaveAccessibleDescription('%');
  });

  it('asks for a number keyboard for whole numbers and groups them on leaving', async () => {
    const { user } = renderWithProviders(<Example decimals={0} />);
    const field = screen.getByRole('textbox', { name: 'Rate' });
    expect(field).toHaveAttribute('inputmode', 'numeric');

    await user.type(field, '250000');
    await user.tab();

    expect(field).toHaveValue('250,000');
  });

  it('writes percentages with two decimals on leaving', async () => {
    const { user } = renderWithProviders(<Example decimals={2} suffix="%" />);
    const field = screen.getByRole('textbox', { name: 'Rate' });

    await user.type(field, '2');
    await user.tab();

    expect(field).toHaveValue('2.00');
  });

  it('never rounds: too many decimals stay as typed, and rates are left alone', async () => {
    const { user } = renderWithProviders(<Example decimals={2} />);
    const field = screen.getByRole('textbox', { name: 'Rate' });

    await user.type(field, '2.005');
    await user.tab();
    expect(field).toHaveValue('2.005');
  });

  it('keeps a rate as typed', async () => {
    const { user } = renderWithProviders(<Example decimals={6} suffix="per USD" />);
    const field = screen.getByRole('textbox', { name: 'Rate' });

    await user.type(field, '0.920000');
    await user.tab();
    expect(field).toHaveValue('0.920000');
    expect(field).toHaveAccessibleDescription('per USD');
  });
});
