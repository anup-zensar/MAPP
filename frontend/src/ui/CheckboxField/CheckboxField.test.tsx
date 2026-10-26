import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { CheckboxField } from './CheckboxField';

function Example({ disabledReason }: { disabledReason?: string }) {
  const [checked, setChecked] = useState(false);
  return (
    <CheckboxField
      label="Strategic / AOP plan"
      name="aop"
      checked={checked}
      onChange={setChecked}
      hint="Part of the annual operating plan."
      error={checked ? undefined : 'Choose Yes or No.'}
      disabledReason={disabledReason}
    />
  );
}

describe('CheckboxField', () => {
  it('is a labelled tick box with its hint and problem (design system §7)', async () => {
    const { user } = renderWithProviders(<Example />);
    const box = screen.getByRole('checkbox', { name: 'Strategic / AOP plan' });

    expect(box).not.toBeChecked();
    expect(box).toHaveAccessibleDescription('Part of the annual operating plan. Choose Yes or No.');
    expect(box).toHaveAttribute('aria-invalid', 'true');

    await user.click(screen.getByText('Strategic / AOP plan'));

    expect(box).toBeChecked();
  });

  it('says why it is locked (design system §6.6)', () => {
    renderWithProviders(
      <Example disabledReason="Ongoing can't be undone — to plan with an end date, create a new activity." />,
    );

    const box = screen.getByRole('checkbox', { name: 'Strategic / AOP plan' });
    expect(box).toBeDisabled();
    expect(box).toHaveAccessibleDescription(
      "Part of the annual operating plan. Ongoing can't be undone — to plan with an end date, create a new activity. Choose Yes or No.",
    );
  });
});
