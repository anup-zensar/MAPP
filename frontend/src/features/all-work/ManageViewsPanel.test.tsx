import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ManageViewsPanel } from './ManageViewsPanel';
import { savedView } from '@/testing/fixtures/all-work';

const DECLINED = savedView();
const BY_MONTH = savedView({ id: 'SV-1002', name: 'Lubricants by month', isDefault: true });
const VIEWS = [DECLINED, BY_MONTH];

function renderPanel(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    'GET /api/v1/saved-views': { body: { items: VIEWS, total: 2 } },
    ...answers,
  });
  const rendered = renderWithProviders(
    <ManageViewsPanel open onOpenChange={() => {}} views={VIEWS} />,
    { user: testUser() },
  );
  return { ...rendered, api };
}

function panel() {
  return screen.getByRole('dialog', { name: 'Your saved views' });
}

describe('ManageViewsPanel (US6, FR-031)', () => {
  it('lists the views with their actions and the default tagged', () => {
    renderPanel();

    const items = within(panel()).getAllByRole('listitem');
    expect(items.map((item) => item.firstElementChild?.textContent)).toEqual([
      'Declined activities',
      'Lubricants by monthDefault',
    ]);
    expect(
      within(items[0])
        .getAllByRole('button')
        .map((button) => button.textContent),
    ).toEqual([
      'Set as default Declined activities',
      'Rename Declined activities',
      'Delete Declined activities',
    ]);
    expect(
      within(items[1]).getByRole('button', { name: 'Remove default Lubricants by month' }),
    ).toBeVisible();
  });

  it('sets and removes the default view (FR-031)', async () => {
    const { api, user } = renderPanel({
      'PATCH /api/v1/saved-views/SV-1001': { body: { ...DECLINED, isDefault: true } },
      'PATCH /api/v1/saved-views/SV-1002': { body: { ...BY_MONTH, isDefault: false } },
    });

    await user.click(
      within(panel()).getByRole('button', { name: 'Set as default Declined activities' }),
    );
    expect(
      await screen.findByText("'Declined activities' is now your default view"),
    ).toBeInTheDocument();
    await user.click(
      within(panel()).getByRole('button', { name: 'Remove default Lubricants by month' }),
    );
    expect(
      await screen.findByText("'Lubricants by month' is no longer your default view"),
    ).toBeInTheDocument();

    expect(api.bodiesOf('PATCH /api/v1/saved-views/SV-1001')).toEqual([{ isDefault: true }]);
    expect(api.bodiesOf('PATCH /api/v1/saved-views/SV-1002')).toEqual([{ isDefault: false }]);
  });

  it('renames a view, refusing a name the person already uses (FR-029, FR-031)', async () => {
    const { api, user } = renderPanel({
      'PATCH /api/v1/saved-views/SV-1001': { body: { ...DECLINED, name: 'Declined in Europe' } },
    });

    await user.click(within(panel()).getByRole('button', { name: 'Rename Declined activities' }));
    const dialog = await screen.findByRole('dialog', { name: 'Rename view' });
    const name = within(dialog).getByRole('textbox', { name: 'Name' });
    expect(name).toHaveValue('Declined activities');
    await user.clear(name);
    await user.type(name, 'LUBRICANTS by month');
    await user.click(within(dialog).getByRole('button', { name: 'Rename view' }));
    expect(name).toHaveAccessibleDescription(
      expect.stringContaining("You already have a view called 'LUBRICANTS by month'."),
    );

    await user.clear(name);
    await user.type(name, 'Declined in Europe');
    await user.click(within(dialog).getByRole('button', { name: 'Rename view' }));

    expect(await screen.findByText("View 'Declined in Europe' renamed")).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/saved-views/SV-1001')).toEqual([
      { name: 'Declined in Europe' },
    ]);
  });

  it('deletes a view after asking, and Undo saves it again (FR-031)', async () => {
    const restored = { ...DECLINED, id: 'SV-1003' };
    const { api, user } = renderPanel({
      'DELETE /api/v1/saved-views/SV-1001': { body: DECLINED },
      'POST /api/v1/saved-views': { status: 201, body: restored },
    });

    await user.click(within(panel()).getByRole('button', { name: 'Delete Declined activities' }));
    const question = await screen.findByRole('alertdialog', {
      name: "Delete view 'Declined activities'?",
    });
    expect(question).toHaveTextContent(
      'Only your saved view is deleted; no plans or records change.',
    );
    await user.click(within(question).getByRole('button', { name: 'Delete view' }));

    expect(await screen.findByText("View 'Declined activities' deleted")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'Undo', hidden: true }));

    expect(await screen.findByText("View 'Declined activities' restored")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/saved-views')).toEqual([
      { name: DECLINED.name, state: DECLINED.state, isDefault: false },
    ]);
  });

  it('says when a view was deleted in another tab (plan "States")', async () => {
    const { user } = renderPanel({
      'PATCH /api/v1/saved-views/SV-1001': {
        status: 404,
        body: { error: { code: 'not_found', message: 'Not found' } },
      },
    });

    await user.click(
      within(panel()).getByRole('button', { name: 'Set as default Declined activities' }),
    );

    expect(
      await screen.findByText(
        'This view no longer exists — it may have been deleted in another tab.',
      ),
    ).toBeInTheDocument();
  });
});
