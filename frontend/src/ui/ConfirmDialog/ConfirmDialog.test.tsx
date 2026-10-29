import { render, screen, waitFor, within } from '@testing-library/react';
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
  it('asks the question, says what will happen and puts the focus on Cancel, so Enter never deletes by accident (FR-019, FR-024)', () => {
    render(<DeleteDialog onConfirm={vi.fn()} />);

    const dialog = screen.getByRole('alertdialog', { name: "Delete sample note 'Agenda'?" });
    expect(dialog).toHaveAccessibleDescription(TEXT);
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });

  it('confirms with a button that names the action (FR-019)', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(<DeleteDialog onConfirm={onConfirm} />);

    await user.click(screen.getByRole('button', { name: 'Delete sample note' }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('closes on Cancel and gives the focus back to the button that opened it (FR-019)', async () => {
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

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('button', { name: 'Delete note' })).toHaveFocus());
  });

  it('can name its cancel button after what it keeps, with the focus on it (plan "Shared hooks")', () => {
    render(
      <ConfirmDialog
        open
        onOpenChange={vi.fn()}
        title="Discard changes?"
        cancelLabel="Keep editing"
        confirmLabel="Discard changes"
        onConfirm={vi.fn()}
      >
        Your changes haven't been saved.
      </ConfirmDialog>,
    );

    expect(screen.getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
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

  it('038 FR-044: a dialog that only informs has one Close, which has the focus and closes it', async () => {
    const user = userEvent.setup();
    function Informing() {
      const [open, setOpen] = useState(true);
      return (
        <ConfirmDialog acknowledgeOnly open={open} onOpenChange={setOpen} title="Can't delete 'Q4'">
          It has allocations.
        </ConfirmDialog>
      );
    }
    render(<Informing />);

    const dialog = screen.getByRole('alertdialog', { name: "Can't delete 'Q4'" });
    const buttons = within(dialog).getAllByRole('button');
    expect(buttons.map((button) => button.textContent)).toEqual(['Close']);
    const close = within(dialog).getByRole('button', { name: 'Close' });
    expect(close).toHaveFocus();
    await user.click(close);
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
  });
});
