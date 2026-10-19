import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Tooltip as RadixTooltip } from 'radix-ui';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { ConfirmDialog } from '../ConfirmDialog/ConfirmDialog';
import { ActionsMenu } from './ActionsMenu';

function TooltipProvider({ children }: { children: ReactNode }) {
  return <RadixTooltip.Provider delayDuration={0}>{children}</RadixTooltip.Provider>;
}

/** A table row's menu whose Delete opens a confirmation, as on the sample page. */
function FolderRow({ onEdit = vi.fn() }: { onEdit?: () => void }) {
  const [confirming, setConfirming] = useState(false);
  return (
    <>
      <ActionsMenu
        label="Ideas"
        items={[
          { label: 'Edit', onSelect: onEdit },
          { label: 'Add note', onSelect: vi.fn() },
          {
            label: 'Delete',
            tone: 'danger',
            separatorBefore: true,
            onSelect: () => setConfirming(true),
          },
        ]}
      />
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title="Delete sample folder 'Ideas'?"
        confirmLabel="Delete sample folder"
        tone="danger"
        onConfirm={() => setConfirming(false)}
      >
        It goes to the recycle bin. You can restore it.
      </ConfirmDialog>
    </>
  );
}

function menuButton() {
  return screen.getByRole('button', { name: 'Actions for Ideas' });
}

describe('ActionsMenu', () => {
  it('is a "⋯" button named after its record (FR-019)', () => {
    render(<FolderRow />, { wrapper: TooltipProvider });

    expect(menuButton()).toHaveAttribute('aria-haspopup', 'menu');
  });

  it('lists the actions in order, with Delete last after a separator (FR-019)', async () => {
    const user = userEvent.setup();
    render(<FolderRow />, { wrapper: TooltipProvider });

    await user.click(menuButton());

    const menu = screen.getByRole('menu');
    const items = screen.getAllByRole('menuitem').map((item) => item.textContent);
    expect(items).toEqual(['Edit', 'Add note', 'Delete']);
    const separator = screen.getByRole('separator');
    expect(menu).toContainElement(separator);
    expect(separator.nextElementSibling).toHaveTextContent('Delete');
  });

  it('runs the chosen action once the menu has closed (FR-019)', async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn(() => {
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });
    render(<FolderRow onEdit={onEdit} />, { wrapper: TooltipProvider });

    await user.click(menuButton());
    await user.click(screen.getByRole('menuitem', { name: 'Edit' }));

    await waitFor(() => expect(onEdit).toHaveBeenCalledTimes(1));
  });

  it('works with the keyboard alone (FR-019, SC-006)', async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    render(<FolderRow onEdit={onEdit} />, { wrapper: TooltipProvider });

    await user.tab();
    expect(menuButton()).toHaveFocus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('menu')).toBeInTheDocument();

    await user.keyboard('{Escape}');
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    expect(menuButton()).toHaveFocus();

    await user.keyboard('{Enter}');
    await user.keyboard('{Enter}');
    await waitFor(() => expect(onEdit).toHaveBeenCalledTimes(1));
  });

  it('opens a dialog from the menu and gives the focus back to "⋯" when it closes (FR-019)', async () => {
    const user = userEvent.setup();
    render(<FolderRow />, { wrapper: TooltipProvider });

    await user.click(menuButton());
    await user.click(screen.getByRole('menuitem', { name: 'Delete' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete sample folder 'Ideas'?",
    });
    await waitFor(() => expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus());
    expect(dialog).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    await waitFor(() => expect(menuButton()).toHaveFocus());
  });
});
