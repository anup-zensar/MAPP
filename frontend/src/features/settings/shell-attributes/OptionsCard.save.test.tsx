import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { page } from '@/testing/fixtures/shell-attributes';
import { RADIO, renderOptionsCard, TV } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('OptionsCard (FR-018, FR-020, SC-007)', () => {
  it('adds an option, and shows the API problem under the label', async () => {
    let attempts = 0;
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV]) },
        { attributeId: 'SA-1010' },
      ),
      reply(
        'POST /api/v1/shell-attributes/{attributeId}/options',
        () => {
          attempts += 1;
          return attempts === 1
            ? {
                status: 400,
                body: {
                  error: {
                    code: 'validation_failed',
                    message: 'Check the highlighted fields.',
                    fields: [
                      { field: 'label', message: "'tv' is already an option of Media channel." },
                    ],
                  },
                },
              }
            : { status: 201, body: RADIO };
        },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Add option' }));
    const panel = await screen.findByRole('dialog', { name: 'Add option' });
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(within(panel).getAllByText('Enter a label.')).not.toHaveLength(0);
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1010/options')).toEqual([]);

    await user.type(within(panel).getByRole('textbox', { name: /Label/ }), 'tv');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(
      await within(panel).findAllByText("'tv' is already an option of Media channel."),
    ).not.toHaveLength(0);

    const label = within(panel).getByRole('textbox', { name: /Label/ });
    await user.clear(label);
    await user.type(label, 'Radio');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Option 'Radio' added");
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1010/options').at(-1)).toEqual({
      label: 'Radio',
    });
  });

  it('renames an option, and says who changed it first on a conflict (the latest version from GET, D-211)', async () => {
    let attempts = 0;
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV, RADIO]) },
        { attributeId: 'SA-1010' },
      ),
      reply(
        'GET /api/v1/shell-attribute-options/{optionId}',
        { body: { ...RADIO, version: 4 } },
        { optionId: 'SO-1102' },
      ),
      reply(
        'PATCH /api/v1/shell-attribute-options/{optionId}',
        () => {
          attempts += 1;
          return attempts === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Someone else changed this option.',
                    fields: [],
                    conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T14:32:00Z' },
                  },
                },
              }
            : { body: { ...RADIO, label: 'Radio ads', version: 5 } };
        },
        { optionId: 'SO-1102' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for Radio' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Edit' }));
    const panel = await screen.findByRole('dialog', { name: "Edit option 'Radio'" });
    await user.type(within(panel).getByRole('textbox', { name: /Label/ }), ' ads');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this option on 14 Mar 2026 at 14:32',
    );
    expect(within(panel).getByRole('textbox', { name: /Label/ })).toHaveValue('Radio ads');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Option 'Radio ads' saved");
    expect(api.bodiesOf('PATCH /api/v1/shell-attribute-options/SO-1102')).toEqual([
      { label: 'Radio ads', version: 1 },
      { label: 'Radio ads', version: 4 },
    ]);
  });
});
