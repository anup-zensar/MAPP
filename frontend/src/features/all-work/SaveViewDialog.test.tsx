import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import type { SavedView, ViewState } from './allWorkTypes';
import { SaveViewDialog } from './SaveViewDialog';
import { savedView, standard2026 } from '@/testing/fixtures/all-work';

const STATE: ViewState = {
  ...standard2026(),
  view: 'budget',
  filters: {
    ...standard2026().filters,
    plans: [{ id: 'P-1001', label: 'FY2026 Lubricants Europe' }],
  },
};

function renderDialog(
  answers: Record<string, MockHandler> = {},
  views: SavedView[] = [savedView()],
) {
  const api = mockApi({
    'GET /api/v1/saved-views': { body: { items: views, total: views.length } },
    ...answers,
  });
  const onSaved = vi.fn();
  const onOpenChange = vi.fn();
  const rendered = renderWithProviders(
    <SaveViewDialog onOpenChange={onOpenChange} views={views} state={STATE} onSaved={onSaved} />,
    { user: testUser() },
  );
  return { ...rendered, api, onSaved, onOpenChange };
}

describe('SaveViewDialog (US6)', () => {
  it('saves the state on screen under a name, as the default when ticked (US6 scenario 1, FR-029)', async () => {
    const saved = savedView({
      id: 'SV-1009',
      name: 'Lubricants by month',
      state: STATE,
      isDefault: true,
    });
    const { api, user, onSaved, onOpenChange } = renderDialog({
      'POST /api/v1/saved-views': { status: 201, body: saved },
    });

    const dialog = screen.getByRole('dialog', { name: 'Save current view' });
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Name' }),
      '  Lubricants by month ',
    );
    await user.click(within(dialog).getByRole('checkbox', { name: 'Use as my default view' }));
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(saved));
    expect(api.bodiesOf('POST /api/v1/saved-views')).toEqual([
      { name: 'Lubricants by month', state: STATE, isDefault: true },
    ]);
    // The dialog stays on screen here (onOpenChange is a stand-in), so the message is found by text.
    expect(await screen.findByText("View 'Lubricants by month' saved")).toBeInTheDocument();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('asks for a name and allows 60 characters at most, at the field and in the summary (FR-029)', async () => {
    const { api, user } = renderDialog();

    const dialog = screen.getByRole('dialog', { name: 'Save current view' });
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));
    const name = within(dialog).getByRole('textbox', { name: 'Name' });
    expect(name).toHaveAccessibleDescription(expect.stringContaining('Enter a name for the view.'));
    expect(within(dialog).getByRole('alert')).toHaveTextContent('1 problem needs your attention');

    await user.type(name, 'x'.repeat(61));
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));
    expect(name).toHaveAccessibleDescription(
      expect.stringContaining('Use 60 characters or fewer.'),
    );
    expect(api.bodiesOf('POST /api/v1/saved-views')).toEqual([]);
  });

  it('asks before replacing a view with the same name, ignoring letter case (FR-029)', async () => {
    const existing = savedView();
    const { api, user, onSaved } = renderDialog({
      'PATCH /api/v1/saved-views/SV-1001': { body: { ...existing, state: STATE } },
    });

    const dialog = screen.getByRole('dialog', { name: 'Save current view' });
    await user.type(within(dialog).getByRole('textbox', { name: 'Name' }), 'declined ACTIVITIES');
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));

    const question = await screen.findByRole('alertdialog', {
      name: "Replace your view 'Declined activities'?",
    });
    expect(question).toHaveTextContent('It will show what you see now.');
    await user.click(within(question).getByRole('button', { name: 'Replace view' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(api.bodiesOf('PATCH /api/v1/saved-views/SV-1001')).toEqual([{ state: STATE }]);
    expect(api.bodiesOf('POST /api/v1/saved-views')).toEqual([]);
    expect(await screen.findByText("View 'Declined activities' updated")).toBeInTheDocument();
  });

  it('keeps the name when the replace question is cancelled', async () => {
    const { user } = renderDialog();

    const dialog = screen.getByRole('dialog', { name: 'Save current view' });
    await user.type(within(dialog).getByRole('textbox', { name: 'Name' }), 'Declined activities');
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));
    await user.click(await screen.findByRole('button', { name: 'Cancel', hidden: false }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(within(dialog).getByRole('textbox', { name: 'Name' })).toHaveValue(
      'Declined activities',
    );
  });

  it('shows the API’s refusal at the field (FR-029)', async () => {
    const { user } = renderDialog({
      'POST /api/v1/saved-views': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: "You already have a view called 'Q4 review'.",
            fields: [{ field: 'name', message: "You already have a view called 'Q4 review'." }],
          },
        },
      },
    });

    const dialog = screen.getByRole('dialog', { name: 'Save current view' });
    await user.type(within(dialog).getByRole('textbox', { name: 'Name' }), 'Q4 review');
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));

    expect(
      await within(dialog).findByText("You already have a view called 'Q4 review'.", {
        selector: 'p',
      }),
    ).toBeInTheDocument();
  });
});
