import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { ConfirmDialog } from './ConfirmDialog';

const TEXT = 'It goes to the recycle bin. You can restore it.';

function DeleteDialog({ onConfirm, busy }: { onConfirm: () => void; busy?: boolean }) {
  const [open, setOpen] = useState(true);
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={setOpen}
      title="Delete sample note 'Agenda'?"
      confirmLabel="Delete sample note"
      onConfirm={onConfirm}
      tone="danger"
      busy={busy}
    >
      {TEXT}
    </ConfirmDialog>
  );
}

describe('ConfirmDialog', () => {
  it('asks the question and says what will happen (FR-019, FR-024)', () => {
    render(<DeleteDialog onConfirm={vi.fn()} />);

    const dialog = screen.getByRole('alertdialog', { name: "Delete sample note 'Agenda'?" });
    expect(dialog).toHaveAccessibleDescription(TEXT);
  });

  it('puts the focus on Cancel, so Enter never deletes by accident (FR-019)', () => {
    render(<DeleteDialog onConfirm={vi.fn()} />);

    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });

  it('confirms with a button that names the action (FR-019)', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(<DeleteDialog onConfirm={onConfirm} />);

    await user.click(screen.getByRole('button', { name: 'Delete sample note' }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('closes without doing anything on Cancel and on Escape (FR-019)', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const { unmount } = render(<DeleteDialog onConfirm={onConfirm} />);

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    unmount();

    render(<DeleteDialog onConfirm={onConfirm} />);
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();

    expect(onConfirm).not.toHaveBeenCalled();
  });

  it('gives the focus back to the button that opened it (FR-019)', async () => {
    const user = userEvent.setup();
    function Page() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>
            Delete note
          </button>
          <ConfirmDialog
            open={open}
            onOpenChange={setOpen}
            title="Delete sample note 'Agenda'?"
            confirmLabel="Delete sample note"
            onConfirm={() => setOpen(false)}
          >
            {TEXT}
          </ConfirmDialog>
        </>
      );
    }
    render(<Page />);

    await user.click(screen.getByRole('button', { name: 'Delete note' }));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(screen.getByRole('button', { name: 'Delete note' })).toHaveFocus());
  });

  it('ignores a second click while the action is running (FR-019)', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(<DeleteDialog onConfirm={onConfirm} busy />);

    const confirm = screen.getByRole('button', { name: 'Delete sample note' });
    await user.click(confirm);

    expect(onConfirm).not.toHaveBeenCalled();
    expect(confirm).toHaveAttribute('aria-busy', 'true');
    expect(confirm).toBeEnabled();
  });
});
