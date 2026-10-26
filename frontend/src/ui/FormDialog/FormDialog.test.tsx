import { screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { Button } from '../Button/Button';
import { TextField } from '../TextField/TextField';
import { FormDialog } from './FormDialog';

function Example({
  onSubmit,
  busy = false,
  withField = true,
}: {
  onSubmit: () => void;
  busy?: boolean;
  withField?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('FY2026 Lubricants Europe (copy)');
  return (
    <>
      <Button onClick={() => setOpen(true)}>Copy</Button>
      <FormDialog
        open={open}
        onOpenChange={setOpen}
        title="Copy plan 'FY2026 Lubricants Europe'"
        submitLabel="Copy plan"
        onSubmit={onSubmit}
        busy={busy}
        hideCancel={!withField}
      >
        {withField ? (
          <TextField label="Name of the copy" name="copyName" value={name} onChange={setName} />
        ) : (
          <p>Nothing to fill in.</p>
        )}
      </FormDialog>
    </>
  );
}

describe('FormDialog', () => {
  it('focuses the first field, submits with Enter and gives the focus back (feature 004)', async () => {
    const onSubmit = vi.fn();
    const { user } = renderWithProviders(<Example onSubmit={onSubmit} />);

    await user.click(screen.getByRole('button', { name: 'Copy' }));
    const dialog = screen.getByRole('dialog', { name: "Copy plan 'FY2026 Lubricants Europe'" });
    expect(dialog).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: 'Name of the copy' })).toHaveFocus();

    await user.keyboard('{Enter}');
    expect(onSubmit).toHaveBeenCalledOnce();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Copy' })).toHaveFocus());
  });

  it('ignores a second submit while busy', async () => {
    const onSubmit = vi.fn();
    const { user } = renderWithProviders(<Example onSubmit={onSubmit} busy />);

    await user.click(screen.getByRole('button', { name: 'Copy' }));
    await user.click(screen.getByRole('button', { name: 'Copy plan' }));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Copy plan' })).toHaveAttribute('aria-busy', 'true');
  });

  it('focuses the submit button when there is no field, and can hide Cancel', async () => {
    const { user } = renderWithProviders(<Example onSubmit={vi.fn()} withField={false} />);

    await user.click(screen.getByRole('button', { name: 'Copy' }));

    expect(screen.getByRole('button', { name: 'Copy plan' })).toHaveFocus();
    expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
  });
});
