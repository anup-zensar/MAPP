import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { BUSINESS_UNIT_LOCK, MEDIA_CHANNEL, OLD_NOTES } from '@/testing/fixtures/shell-attributes';
import { renderShellAttributesPage } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('ShellAttributesPage: row actions (FR-004)', () => {
  async function choose(
    user: ReturnType<typeof renderShellAttributesPage>['user'],
    label: string,
    action: string,
  ) {
    await user.click(await screen.findByRole('button', { name: `Actions for ${label}` }));
    await user.click(await screen.findByRole('menuitem', { name: action }));
  }

  it('opens and edits from the menu', async () => {
    const { user, router } = renderShellAttributesPage('/settings/shell-attributes?level=activity');

    await choose(user, 'PIR', 'Edit');

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1011/edit'),
    );
  });

  it('deletes an unused attribute with its options after asking, and offers Undo', async () => {
    const { user, api } = renderShellAttributesPage('/settings/shell-attributes?level=activity', [
      reply(
        'DELETE /api/v1/shell-attributes/{attributeId}',
        {
          body: {
            recycleBinEntry: {
              id: 'BIN-1001',
              recordType: 'shellAttribute',
              recordTypeLabel: 'Shell attribute',
              recordId: 'SA-1011',
              recordName: 'PIR',
              path: [],
              parent: null,
              deletedAt: '2026-03-14T10:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [{ recordType: 'shellAttributeOption', label: 'options', count: 2 }],
            },
          },
        },
        { attributeId: 'SA-1011' },
      ),
    ]);

    await choose(user, 'PIR', 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete Shell attribute 'PIR'?",
    });
    expect(dialog).toHaveTextContent(
      'It moves to the recycle bin together with its 2 options; you can restore it.',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Delete Shell attribute' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Shell attribute 'PIR' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(true);
  });

  it('offers Set inactive instead of deleting an attribute in use (SC-004)', async () => {
    const { user, api } = renderShellAttributesPage('/settings/shell-attributes?level=activity', [
      reply(
        'POST /api/v1/shell-attributes/{attributeId}/status',
        { body: { ...MEDIA_CHANNEL, active: false } },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await choose(user, 'Media channel', 'Delete');
    const dialog = await screen.findByRole('alertdialog', { name: "'Media channel' is in use" });
    expect(dialog).toHaveTextContent(
      'Used by 12 activities (1 in the recycle bin) — set it inactive instead.',
    );
    expect(dialog).toHaveTextContent(
      'Forms stop offering it; values already on records stay visible as "(inactive)" and never block publishing or submitting.',
    );
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(await findMessages()).toHaveTextContent("Shell attribute 'Media channel' set inactive");
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1010/status')).toEqual([
      { active: false, version: 1 },
    ]);
  });

  it('switches to the in-use question when someone started using it meanwhile', async () => {
    const { user } = renderShellAttributesPage('/settings/shell-attributes?level=activity', [
      reply(
        'DELETE /api/v1/shell-attributes/{attributeId}',
        {
          status: 409,
          body: {
            error: {
              code: 'in_use',
              message: "'PIR' is in use (used by 1 activity) — set it inactive instead.",
              fields: [],
            },
          },
        },
        { attributeId: 'SA-1011' },
      ),
    ]);

    await choose(user, 'PIR', 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete Shell attribute' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'PIR' is in use" });
    expect(dialog).toHaveTextContent(
      "'PIR' is in use (used by 1 activity) — set it inactive instead.",
    );
  });

  it('reactivates an inactive attribute', async () => {
    const { user } = renderShellAttributesPage(
      '/settings/shell-attributes?level=activity&status=inactive',
      [
        reply(
          'POST /api/v1/shell-attributes/{attributeId}/status',
          { body: { ...OLD_NOTES, active: true } },
          { attributeId: 'SA-1013' },
        ),
      ],
    );

    await choose(user, 'Old notes', 'Reactivate');

    expect(await findMessages()).toHaveTextContent("Shell attribute 'Old notes' reactivated");
  });

  it('says why Business unit can be neither set inactive nor deleted (FR-023)', async () => {
    const { user, api } = renderShellAttributesPage('/settings/shell-attributes');

    await choose(user, 'Business unit', 'Set inactive');
    expect(await findMessages()).toHaveTextContent(BUSINESS_UNIT_LOCK);
    await choose(user, 'Business unit', 'Delete');

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.filter((request) => request.method !== 'GET')).toEqual([]);
  });
});
