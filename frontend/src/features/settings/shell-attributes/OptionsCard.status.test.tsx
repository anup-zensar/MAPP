import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { page, shellAttribute } from '@/testing/fixtures/shell-attributes';
import { RADIO, renderOptionsCard, TV } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('OptionsCard (FR-018, FR-020, SC-007)', () => {
  it('moves an option and sets it inactive', async () => {
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV, RADIO]) },
        { attributeId: 'SA-1010' },
      ),
      reply(
        'POST /api/v1/shell-attribute-options/{optionId}/move',
        { body: { ...TV, position: 2 } },
        { optionId: 'SO-1101' },
      ),
      reply(
        'POST /api/v1/shell-attribute-options/{optionId}/status',
        { body: { ...RADIO, active: false } },
        { optionId: 'SO-1102' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for TV' }));
    expect(screen.queryByRole('menuitem', { name: 'Move up' })).not.toBeInTheDocument();
    await user.click(await screen.findByRole('menuitem', { name: 'Move down' }));
    await waitFor(() =>
      expect(api.bodiesOf('POST /api/v1/shell-attribute-options/SO-1101/move')).toEqual([
        { direction: 'down', version: 1 },
      ]),
    );
    expect(await screen.findByText("Moved 'TV' down.")).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Actions for Radio' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Set inactive' }));
    expect(await findMessages()).toHaveTextContent("Option 'Radio' set inactive");
  });

  it('offers Set inactive instead of deleting an option in use (FR-004)', async () => {
    const { user } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV, RADIO]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for TV' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'TV' is in use" });
    expect(dialog).toHaveTextContent('Used by 7 activities — set it inactive instead.');
    expect(dialog).toHaveTextContent(
      'Forms stop offering it; records that have it keep it as "(inactive)".',
    );
  });

  it('says an inactive option in use is already inactive', async () => {
    const { user } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([{ ...TV, active: false }]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for TV' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'TV' is in use" });
    expect(dialog).toHaveTextContent(
      "Used by 7 activities, so it can't be deleted. It's already inactive.",
    );
    expect(within(dialog).queryByRole('button', { name: 'Set inactive' })).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
  });

  it("says why the last option can't be deleted, and sends nothing (D-211, design system 6.6)", async () => {
    const only = {
      ...RADIO,
      locks: [
        {
          field: 'delete',
          reason: "'Media channel' needs at least one option, so 'Radio' can't be deleted.",
        },
      ],
    };
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([only]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for Radio' }));
    const remove = await screen.findByRole('menuitem', { name: /Delete/ });
    expect(remove).toHaveTextContent("'Media channel' needs at least one option");
    await user.click(remove);

    expect(await findMessages()).toHaveTextContent(
      "'Media channel' needs at least one option, so 'Radio' can't be deleted.",
    );
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(false);
  });

  it('deletes an unused option and offers Undo', async () => {
    const { user } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV, RADIO]) },
        { attributeId: 'SA-1010' },
      ),
      reply(
        'DELETE /api/v1/shell-attribute-options/{optionId}',
        {
          body: {
            recycleBinEntry: {
              id: 'BIN-1003',
              recordType: 'shellAttributeOption',
              recordTypeLabel: 'option',
              recordId: 'SO-1102',
              recordName: 'Radio',
              path: [{ id: 'SA-1010', name: 'Media channel', type: 'shellAttribute' }],
              parent: { id: 'SA-1010', name: 'Media channel', type: 'shellAttribute' },
              deletedAt: '2026-03-14T10:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [],
            },
          },
        },
        { optionId: 'SO-1102' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for Radio' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', { name: "Delete option 'Radio'?" });
    await user.click(within(dialog).getByRole('button', { name: 'Delete option' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Option 'Radio' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('shows where the options of a dependent attribute are linked', async () => {
    const region = shellAttribute({
      id: 'SA-1010',
      label: 'Region',
      parentAttribute: { id: 'SA-1009', name: 'Cluster', active: true },
    });
    renderOptionsCard(
      [
        reply(
          'GET /api/v1/shell-attributes/{attributeId}/options',
          {
            body: page([
              {
                ...TV,
                label: 'Retail – China',
                parentOptions: [
                  { id: 'SO-1', name: 'Asia', active: true },
                  { id: 'SO-2', name: 'Pacific', active: false },
                ],
              },
            ]),
          },
          { attributeId: 'SA-1010' },
        ),
      ],
      region,
    );

    expect(await screen.findByRole('columnheader', { name: 'Linked to' })).toBeInTheDocument();
    expect(screen.getByText('Asia, Pacific (inactive)')).toBeInTheDocument();
  });
});
