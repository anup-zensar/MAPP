import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { Button } from '../Button/Button';
import { SidePanel } from './SidePanel';

/** The panel with its open state kept like a page would keep it. */
function EditPanel({ description }: { description?: string }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        Edit
      </button>
      <SidePanel
        open={open}
        onOpenChange={setOpen}
        title="Edit sample folder"
        description={description}
        footer={
          <>
            <Button variant="primary">Save</Button>
            <Button onClick={() => setOpen(false)}>Cancel</Button>
          </>
        }
      >
        <label htmlFor="name">Name</label>
        <input id="name" />
      </SidePanel>
    </>
  );
}

describe('SidePanel', () => {
  // These tests render without a Tooltip.Provider on purpose: a Radix tooltip inside the panel
  // would need one, and must not exist, because it would swallow the first Escape (research R14).

  it('is a dialog named by its title, with its content and footer (FR-019)', () => {
    render(<EditPanel />);

    const panel = screen.getByRole('dialog', { name: 'Edit sample folder' });
    expect(panel).toContainElement(screen.getByRole('textbox', { name: 'Name' }));
    expect(panel).toContainElement(screen.getByRole('button', { name: 'Save' }));
    expect(panel).toContainElement(screen.getByRole('button', { name: 'Cancel' }));
  });

  it('is described by its description when it has one (FR-019)', () => {
    render(<EditPanel description="Sample folders exist only on this developer page." />);

    expect(screen.getByRole('dialog')).toHaveAccessibleDescription(
      'Sample folders exist only on this developer page.',
    );
  });

  it('has no description reference when it has no description (FR-019)', () => {
    render(<EditPanel />);

    expect(screen.getByRole('dialog')).not.toHaveAttribute('aria-describedby');
  });

  it('closes with its Close button (FR-019)', async () => {
    const user = userEvent.setup();
    render(<EditPanel />);

    await user.click(screen.getByRole('button', { name: 'Close' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('closes with one Escape (FR-019, research R14)', async () => {
    const user = userEvent.setup();
    render(<EditPanel />);
    await user.click(screen.getByRole('textbox', { name: 'Name' }));

    await user.keyboard('{Escape}');

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('gives the focus back to the button that opened it (FR-019)', async () => {
    const user = userEvent.setup();
    render(<EditPanel />);
    await user.click(screen.getByRole('button', { name: 'Close' }));

    await user.click(screen.getByRole('button', { name: 'Edit' }));
    await user.keyboard('{Escape}');

    await waitFor(() => expect(screen.getByRole('button', { name: 'Edit' })).toHaveFocus());
  });
});
