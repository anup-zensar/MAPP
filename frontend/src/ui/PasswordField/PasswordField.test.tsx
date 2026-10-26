import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Tooltip as RadixTooltip } from 'radix-ui';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { describe, expect, it } from 'vitest';
import { PasswordField } from './PasswordField';

function TooltipProvider({ children }: { children: ReactNode }) {
  return <RadixTooltip.Provider delayDuration={0}>{children}</RadixTooltip.Provider>;
}

function Field({ error }: { error?: string }) {
  const [value, setValue] = useState('');
  return (
    <PasswordField
      label="Password"
      name="password"
      value={value}
      onChange={setValue}
      error={error}
      autoComplete="current-password"
    />
  );
}

describe('PasswordField', () => {
  it('hides what is typed (FR-001)', async () => {
    const user = userEvent.setup();
    render(<Field />, { wrapper: TooltipProvider });

    const input = screen.getByLabelText('Password');
    await user.type(input, 'mapp');

    expect(input).toHaveAttribute('type', 'password');
    expect(input).toHaveValue('mapp');
    expect(input).toHaveAttribute('autocomplete', 'current-password');
  });

  it('shows and hides the password with an eye button that keeps its name (FR-001)', async () => {
    const user = userEvent.setup();
    render(<Field />, { wrapper: TooltipProvider });
    const eye = screen.getByRole('button', { name: 'Show password' });
    expect(eye).toHaveAttribute('aria-pressed', 'false');

    await user.click(eye);

    expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'text');
    expect(screen.getByRole('button', { name: 'Show password' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );

    await user.click(eye);

    expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'password');
    expect(eye).toHaveAttribute('aria-pressed', 'false');
  });

  it('explains its icon-only eye button with a tooltip (FR-001)', async () => {
    const user = userEvent.setup();
    render(<Field />, { wrapper: TooltipProvider });

    await user.hover(screen.getByRole('button', { name: 'Show password' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Show password');
  });

  it('shows its problem like every other field (FR-004)', () => {
    render(<Field error="Enter the password" />, { wrapper: TooltipProvider });

    const input = screen.getByLabelText('Password');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAccessibleDescription('Enter the password');
  });
});
