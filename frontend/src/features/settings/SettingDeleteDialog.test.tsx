import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '@/api/errors';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { NOT_USED } from '@/testing/fixtures/settings';
import { SettingDeleteDialog } from './SettingDeleteDialog';
import type { SettingDeleteDialogProps } from './SettingDeleteDialog';

function renderDialog(props: Partial<SettingDeleteDialogProps> = {}) {
  const onClose = vi.fn();
  const rendered = renderWithProviders(<Page onClose={onClose} {...props} />, { user: testUser() });
  return { ...rendered, onClose };
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
