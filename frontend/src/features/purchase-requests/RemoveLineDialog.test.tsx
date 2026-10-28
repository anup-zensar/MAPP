import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { YES, money, no } from '@/testing/fixtures/purchase-requests';
import { renderWithProviders } from '@/testing/render';
import { RemoveLineButton } from './RemoveLineDialog';
import type { RemoveLineButtonProps } from './RemoveLineDialog';

function renderButton(props: Partial<RemoveLineButtonProps> = {}) {
  const onRemove = vi.fn();
  return {
    onRemove,
    ...renderWithProviders(
      <RemoveLineButton
        lineItemId="L-1006"
        reserved={money('3200.00')}
        availability={YES}
        isLast={false}
        onRemove={onRemove}
        {...props}
      />,
    ),
  };
}

describe('032 FR-015/FR-055: remove line', () => {
  it('asks first, naming the line item and the money released', async () => {
    const { onRemove, user } = renderButton();

    await user.click(screen.getByRole('button', { name: 'Remove line' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: 'Remove the line for line item L-1006 (3,200.00 EUR) from this purchase request?',
    });
    expect(dialog).toHaveTextContent(
      'The 3,200.00 EUR reserved on the line item will be released.',
    );
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Remove line' }));
    expect(onRemove).toHaveBeenCalledOnce();
  });

  it('changes nothing when cancelled', async () => {
    const { onRemove, user } = renderButton();

    await user.click(screen.getByRole('button', { name: 'Remove line' }));
    await user.click(await screen.findByRole('button', { name: 'Cancel' }));

    expect(onRemove).not.toHaveBeenCalled();
  });

  it('removes a line not saved yet without asking: nothing is reserved', async () => {
    const { onRemove, user } = renderButton({ reserved: null, availability: null });

    await user.click(screen.getByRole('button', { name: 'Remove line' }));

    expect(onRemove).toHaveBeenCalledOnce();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('is unavailable on an approved purchase request, with the reason', () => {
    const reason =
      'Lines of approved purchase requests cannot be removed — close the purchase request instead.';
    renderButton({ availability: no(reason) });

    const remove = screen.getByRole('button', { name: 'Remove line' });
    expect(remove).toBeDisabled();
    expect(remove).toHaveAccessibleDescription(reason);
  });

  it('refuses to remove the last line', () => {
    renderButton({ isLast: true });

    expect(screen.getByRole('button', { name: 'Remove line' })).toHaveAccessibleDescription(
      'A purchase request needs at least one line — delete the purchase request instead.',
    );
  });

  it('is left out when unavailable without a reason', () => {
    renderButton({ availability: no('') });

    expect(screen.queryByRole('button', { name: 'Remove line' })).not.toBeInTheDocument();
  });
});
