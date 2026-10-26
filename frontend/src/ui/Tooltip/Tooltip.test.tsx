import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Bell } from 'lucide-react';
import { Tooltip as RadixTooltip } from 'radix-ui';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
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
  afterEach(() => {
    vi.useRealTimers();
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

  it('stays closed when a mouse action gives the focus back to the control', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(
      <>
        <button type="button">Save</button>
        <Tooltip content="Notifications">
          <button type="button" aria-label="Notifications">
            <Bell />
          </button>
        </Tooltip>
      </>,
      { wrapper: TooltipProvider },
    );
    await user.click(screen.getByRole('button', { name: 'Save' }));

    // As a menu or dialog closed with the mouse does: the focus returns to its trigger.
    const bell = screen.getByRole('button', { name: 'Notifications' });
    act(() => bell.focus());

    expect(bell).toHaveFocus();
    await act(() => vi.advanceTimersByTimeAsync(50));
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });
});
