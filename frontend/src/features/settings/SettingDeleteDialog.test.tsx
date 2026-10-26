import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '@/api/errors';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { deleted, NOT_USED, USED_BY_TWO_PLANS } from '@/testing/fixtures/settings';
import { SettingDeleteDialog } from './SettingDeleteDialog';
import type { SettingDeleteDialogProps } from './SettingDeleteDialog';

function renderDialog(props: Partial<SettingDeleteDialogProps> = {}) {
  const handlers = {
    remove: vi.fn(async () => deleted('currency', 'JPY', 'Japanese yen')),
    setInactive: vi.fn(async () => true),
    onDeleted: vi.fn(),
    onClose: vi.fn(),
  };
  const rendered = renderWithProviders(<Page {...handlers} {...props} />, { user: testUser() });
  return { ...rendered, ...handlers, ...props };
}

/** A page that shows the dialog until it closes, as the Settings pages do. */
function Page({ onClose, ...props }: Partial<SettingDeleteDialogProps>) {
  const [open, setOpen] = useState(true);
  return open ? (
    <SettingDeleteDialog
      kind="currency"
      name="Japanese yen"
      usage={NOT_USED}
      active
      invalidate={[['currencies']]}
      remove={vi.fn()}
      setInactive={vi.fn()}
      {...props}
      onClose={() => {
        onClose?.();
        setOpen(false);
      }}
    />
  ) : null;
}

describe('SettingDeleteDialog (FR-003, FR-004)', () => {
  it('asks before deleting an unused record, then offers Undo', async () => {
    const { user, remove, onDeleted, onClose } = renderDialog();

    const dialog = screen.getByRole('alertdialog', { name: "Delete currency 'Japanese yen'?" });
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Delete currency' }));

    expect(remove).toHaveBeenCalledOnce();
    expect(onDeleted).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Currency 'Japanese yen' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('offers Set inactive for a record in use, with what that does', async () => {
    const { user, setInactive, remove, onClose } = renderDialog({ usage: USED_BY_TWO_PLANS });

    const dialog = screen.getByRole('alertdialog', { name: "'Japanese yen' is in use" });
    expect(dialog).toHaveTextContent(
      "Used by 2 plans — set it inactive instead. Line items and people can't choose an inactive currency; those using it keep it and amounts still convert.",
    );
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(setInactive).toHaveBeenCalledOnce();
    expect(remove).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('says an inactive record in use is already inactive, with Close only', async () => {
    const { user, onClose } = renderDialog({ usage: USED_BY_TWO_PLANS, active: false });

    const dialog = screen.getByRole('alertdialog', { name: "'Japanese yen' is in use" });
    expect(dialog).toHaveTextContent(
      "Used by 2 plans, so it can't be deleted. It's already inactive.",
    );
    expect(
      within(dialog)
        .getAllByRole('button')
        .map((button) => button.textContent),
    ).toEqual(['Close']);
    await user.click(within(dialog).getByRole('button', { name: 'Close' }));

    expect(onClose).toHaveBeenCalled();
  });

  it("switches to the in-use state with the API's message when the API refuses the delete", async () => {
    const message = "'Japanese yen' is in use (used by 1 line item) — set it inactive instead.";
    const { user, onClose } = renderDialog({
      remove: vi.fn(async () => {
        throw new ApiError('inUse', { status: 409, message });
      }),
    });

    await user.click(screen.getByRole('button', { name: 'Delete currency' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'Japanese yen' is in use" });
    expect(dialog).toHaveTextContent(message);
    expect(within(dialog).getByRole('button', { name: 'Set inactive' })).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('closes and says why when the delete fails otherwise, with Try again', async () => {
    const { user, onClose } = renderDialog({
      remove: vi.fn(async () => {
        throw new ApiError('unreachable');
      }),
    });

    await user.click(screen.getByRole('button', { name: 'Delete currency' }));

    expect(onClose).toHaveBeenCalled();
    const messages = await findMessages();
    expect(messages).toHaveTextContent("MAPP can't reach its server right now.");
    expect(within(messages).getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});
