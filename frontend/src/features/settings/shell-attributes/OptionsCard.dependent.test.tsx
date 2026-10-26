import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  emptyForm,
  formAttribute,
  formOption,
  page,
  shellAttribute,
  shellOption,
} from '@/testing/fixtures/shell-attributes';
import { renderOptionsCard } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('OptionsCard: options of a dependent attribute (FR-022)', () => {
  const REGION = shellAttribute({
    id: 'SA-1010',
    label: 'Region',
    parentAttribute: { id: 'SA-1009', name: 'Cluster', active: true },
  });
  const CHINA = shellOption({
    id: 'SO-1101',
    label: 'Retail – China',
    attributeId: 'SA-1010',
    parentOptions: [
      { id: 'SO-1', name: 'Asia', active: true },
      { id: 'SO-2', name: 'Pacific', active: false },
    ],
  });
  /** The activity form: Cluster (Asia, Pacific inactive, Europe), then Region depending on it. */
  const FORM = {
    ...emptyForm('activity'),
    sections: [
      {
        name: 'Shell attributes',
        conditional: false,
        attributes: [
          formAttribute({
            id: 'SA-1009',
            label: 'Cluster',
            options: [
              formOption('SO-1', 'Asia'),
              formOption('SO-2', 'Pacific', [], false),
              formOption('SO-3', 'Europe'),
            ],
          }),
          formAttribute({
            id: 'SA-1010',
            label: 'Region',
            parentAttributeId: 'SA-1009',
            options: [formOption('SO-1101', 'Retail – China', ['SO-1', 'SO-2'])],
          }),
        ],
      },
    ],
  };

  it("adds an option linked to the parent's active options", async () => {
    const { user, api } = renderOptionsCard(
      [
        reply(
          'GET /api/v1/shell-attributes/{attributeId}/options',
          { body: page([CHINA]) },
          { attributeId: 'SA-1010' },
        ),
        reply('GET /api/v1/shell-attribute-forms/{level}', { body: FORM }, { level: 'activity' }),
        reply(
          'POST /api/v1/shell-attributes/{attributeId}/options',
          {
            status: 201,
            body: { ...CHINA, id: 'SO-1102', label: 'Retail – Japan' },
          },
          { attributeId: 'SA-1010' },
        ),
      ],
      REGION,
    );

    await user.click(await screen.findByRole('button', { name: 'Add option' }));
    const panel = await screen.findByRole('dialog', { name: 'Add option' });
    await user.type(within(panel).getByRole('textbox', { name: /Label/ }), 'Retail – Japan');
    const linkedTo = await within(panel).findByRole('group', { name: /^Linked to/ });
    expect(linkedTo).toHaveTextContent('Shown when one of these cluster options is chosen.');
    expect(within(linkedTo).queryByRole('checkbox', { name: /Pacific/ })).not.toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findAllByText("Link 'Retail – Japan' to at least one cluster option."),
    ).toHaveLength(2);
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1010/options')).toEqual([]);
    await user.click(within(linkedTo).getByRole('checkbox', { name: 'Asia' }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Option 'Retail – Japan' added");
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1010/options')).toEqual([
      { label: 'Retail – Japan', parentOptionIds: ['SO-1'] },
    ]);
  });

  it('changes only the links of an option, keeping an inactive link until it is unticked', async () => {
    const { user, api } = renderOptionsCard(
      [
        reply(
          'GET /api/v1/shell-attributes/{attributeId}/options',
          { body: page([CHINA]) },
          { attributeId: 'SA-1010' },
        ),
        reply('GET /api/v1/shell-attribute-forms/{level}', { body: FORM }, { level: 'activity' }),
        reply(
          'PATCH /api/v1/shell-attribute-options/{optionId}',
          {
            body: { ...CHINA, parentOptions: [{ id: 'SO-1', name: 'Asia', active: true }] },
          },
          { optionId: 'SO-1101' },
        ),
      ],
      REGION,
    );

    await user.click(await screen.findByRole('button', { name: 'Actions for Retail – China' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Edit' }));
    const panel = await screen.findByRole('dialog', { name: "Edit option 'Retail – China'" });
    const linkedTo = await within(panel).findByRole('group', { name: /^Linked to/ });
    expect(within(linkedTo).getByRole('checkbox', { name: 'Asia' })).toBeChecked();
    const pacific = within(linkedTo).getByRole('checkbox', { name: 'Pacific (inactive)' });
    expect(pacific).toBeChecked();
    await user.click(pacific);
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Option 'Retail – China' saved");
    expect(api.bodiesOf('PATCH /api/v1/shell-attribute-options/SO-1101')).toEqual([
      { parentOptionIds: ['SO-1'], version: 1 },
    ]);
  });
});
