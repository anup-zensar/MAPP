import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Bell } from 'lucide-react';
import { Tooltip as RadixTooltip } from 'radix-ui';
import type { ReactNode } from 'react';
import { describe, expect, it } from 'vitest';
import { Tooltip } from './Tooltip';

// The app puts one Tooltip.Provider around everything (App.tsx); tests need one as well.
function TooltipProvider({ children }: { children: ReactNode }) {
  return <RadixTooltip.Provider delayDuration={0}>{children}</RadixTooltip.Provider>;
}

function renderBell() {
  return render(
    <Tooltip content="Notifications">
      <button type="button" aria-label="Notifications">
        <Bell />
      </button>
    </Tooltip>,
    { wrapper: TooltipProvider },
  );
}

describe('Tooltip', () => {
  it('shows its text when the pointer rests on the control (FR-019)', async () => {
    const user = userEvent.setup();
    renderBell();

    await user.hover(screen.getByRole('button', { name: 'Notifications' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Notifications');
  });

  it('shows its text on keyboard focus and describes the control (FR-019)', async () => {
    const user = userEvent.setup();
    renderBell();

    await user.tab();

    const button = screen.getByRole('button', { name: 'Notifications' });
    expect(button).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Notifications');
    expect(button).toHaveAccessibleDescription('Notifications');
  });

  it('closes on Escape', async () => {
    const user = userEvent.setup();
    renderBell();
    await user.tab();
    await screen.findByRole('tooltip');

    await user.keyboard('{Escape}');

    await waitFor(() => expect(screen.queryByRole('tooltip')).not.toBeInTheDocument());
  });
});
