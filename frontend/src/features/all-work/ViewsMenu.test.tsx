import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { savedView } from '@/testing/fixtures/all-work';
import type { ViewsMenuProps } from './ViewsMenu';
import { ViewsMenu } from './ViewsMenu';

const DECLINED = savedView();
const BY_MONTH = savedView({ id: 'SV-1002', name: 'Lubricants by month', isDefault: true });

function renderMenu(props: Partial<ViewsMenuProps> = {}) {
  const handlers = {
    onApply: vi.fn(),
    onSave: vi.fn(),
    onUpdate: vi.fn(),
    onSaveAsNew: vi.fn(),
    onManage: vi.fn(),
    onStandard: vi.fn(),
  };
  const rendered = renderWithProviders(
    <ViewsMenu
      views={[DECLINED, BY_MONTH]}
      active={null}
      modified={false}
      {...handlers}
      {...props}
    />,
  );
  return { ...rendered, ...handlers };
}

describe('ViewsMenu (US6)', () => {
  it('lists the saved views, the active one ticked and the default one tagged (FR-030, FR-031)', async () => {
    const { user } = renderMenu({ active: DECLINED });

    await user.click(screen.getByRole('button', { name: 'View: Declined activities' }));

    const views = await screen.findAllByRole('menuitemcheckbox');
    expect(views.map((item) => item.textContent)).toEqual([
      'Declined activities',
      'Lubricants by monthDefault',
    ]);
    expect(views[0]).toHaveAttribute('aria-checked', 'true');
    expect(views[1]).toHaveAttribute('aria-checked', 'false');
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual([
      'Save current view…',
      'Manage views…',
      'Standard view',
    ]);
  });

  it('applies a view once the menu has closed (FR-030)', async () => {
    const { user, onApply } = renderMenu();

    await user.click(screen.getByRole('button', { name: 'Views' }));
    await user.click(await screen.findByRole('menuitemcheckbox', { name: /Lubricants by month/ }));

    await waitFor(() => expect(onApply).toHaveBeenCalledWith(BY_MONTH));
  });

  it('shows Modified and offers Update view and Save as new view (US6 scenario 2, FR-030)', async () => {
    const { user, onUpdate, onSaveAsNew } = renderMenu({ active: DECLINED, modified: true });

    const button = screen.getByRole('button', { name: 'View: Declined activities Modified' });
    await user.click(button);
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual([
      'Save current view…',
      'Update view',
      'Save as new view…',
      'Manage views…',
      'Standard view',
    ]);
    await user.click(screen.getByRole('menuitem', { name: 'Update view' }));
    await waitFor(() => expect(onUpdate).toHaveBeenCalled());

    await user.click(button);
    await user.click(await screen.findByRole('menuitem', { name: 'Save as new view…' }));
    await waitFor(() => expect(onSaveAsNew).toHaveBeenCalled());
  });

  it('explains how to save a view when there is none', async () => {
    const { user, onSave } = renderMenu({ views: [] });

    await user.click(screen.getByRole('button', { name: 'Views' }));

    expect(
      await screen.findByText(
        'No saved views yet — set up filters and columns, then choose Save current view.',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('menuitem', { name: 'Manage views…' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('menuitem', { name: 'Save current view…' }));
    await waitFor(() => expect(onSave).toHaveBeenCalled());
  });

  it('goes back to the standard view', async () => {
    const { user, onStandard, onManage } = renderMenu();

    await user.click(screen.getByRole('button', { name: 'Views' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Standard view' }));
    await waitFor(() => expect(onStandard).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: 'Views' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Manage views…' }));
    await waitFor(() => expect(onManage).toHaveBeenCalled());
  });
});
