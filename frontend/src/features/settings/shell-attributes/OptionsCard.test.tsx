import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { OptionsCard } from './OptionsCard';
import {
  emptyForm,
  formAttribute,
  formOption,
  MEDIA_CHANNEL,
  page,
  shellAttribute,
  shellOption,
  usedBy,
} from './testData';

const OPTIONS_PATH = 'GET /api/v1/shell-attributes/SA-1010/options';

const TV = shellOption({
  id: 'SO-1101',
  label: 'TV',
  attributeId: 'SA-1010',
  usage: usedBy('Used by 7 activities', 7),
});
const RADIO = shellOption({ id: 'SO-1102', label: 'Radio', attributeId: 'SA-1010', position: 2 });
const PRINT = shellOption({
  id: 'SO-1103',
  label: 'Print',
  attributeId: 'SA-1010',
  position: 3,
  active: false,
});

function renderCard(answers: Record<string, MockHandler>, attribute = MEDIA_CHANNEL) {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<OptionsCard attribute={attribute} />, {
    user: testUser(),
  });
  return { ...rendered, api };
}

/** The query of each options request, as `URLSearchParams`. */
function optionQueries(api: ReturnType<typeof mockApi>) {
  return api.requests
    .filter((request) => request.path === '/api/v1/shell-attributes/SA-1010/options')
    .map((request) => new URLSearchParams(request.search));
}

describe('OptionsCard (FR-018, FR-020, SC-007)', () => {
  it('lists the options with their own Used by and status', async () => {
    renderCard({ [OPTIONS_PATH]: { body: page([TV, RADIO, PRINT]) } });

    const tv = await screen.findByRole('row', { name: /TV/ });
    expect(within(tv).getByText('Used by 7 activities')).toBeInTheDocument();
    expect(within(tv).getByText('Active')).toBeInTheDocument();
    const print = screen.getByRole('row', { name: /Print/ });
    expect(within(print).getByText('Inactive')).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: 'Linked to' })).not.toBeInTheDocument();
    expect(screen.queryByText(/Showing/)).not.toBeInTheDocument();
  });

  it('finds an option among 1,217 by any part of its label, asking the API', async () => {
    const match = shellOption({ id: 'SO-2220', label: 'CN04000220', attributeId: 'SA-1010' });
    const many = Array.from({ length: 100 }, (_, index) =>
      shellOption({
        id: `SO-${3000 + index}`,
        label: `CN0400${1000 + index}`,
        attributeId: 'SA-1010',
      }),
    );
    const { user, api } = renderCard({
      [OPTIONS_PATH]: (request) =>
        new URL(request.url).searchParams.get('q') === '0220'
          ? { body: page([match]) }
          : { body: page(many, 1217) },
    });

    expect(
      await screen.findByText('Showing 100 of 1,217 — search to find the others.'),
    ).toBeInTheDocument();
    await user.type(screen.getByRole('searchbox', { name: 'Search options' }), '0220');

    expect(await screen.findByRole('rowheader', { name: 'CN04000220' })).toBeInTheDocument();
    expect(screen.getAllByRole('rowheader')).toHaveLength(1);
    const last = optionQueries(api).at(-1);
    expect(last?.get('q')).toBe('0220');
    expect(last?.get('status')).toBe('all');
    expect(last?.get('limit')).toBe('100');
  });

  it('shows 100 more options at a time with Show more', async () => {
    const first = Array.from({ length: 100 }, (_, index) =>
      shellOption({
        id: `SO-${3000 + index}`,
        label: `Option ${index + 1}`,
        attributeId: 'SA-1010',
      }),
    );
    const second = Array.from({ length: 100 }, (_, index) =>
      shellOption({
        id: `SO-${4000 + index}`,
        label: `Option ${index + 101}`,
        attributeId: 'SA-1010',
      }),
    );
    const { user, api } = renderCard({
      [OPTIONS_PATH]: (request) =>
        new URL(request.url).searchParams.get('offset') === '100'
          ? { body: page(second, 1217) }
          : { body: page(first, 1217) },
    });

    await user.click(await screen.findByRole('button', { name: 'Show more' }));

    expect(
      await screen.findByText('Showing 200 of 1,217 — search to find the others.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('rowheader', { name: 'Option 200' })).toBeInTheDocument();
    expect(optionQueries(api).map((query) => query.get('offset'))).toEqual(['0', '100']);
  });

  it('filters by status and clears the filters when nothing matches', async () => {
    const { user, api } = renderCard({
      [OPTIONS_PATH]: (request) =>
        new URL(request.url).searchParams.get('status') === 'inactive'
          ? { body: page([]) }
          : { body: page([TV, RADIO, PRINT]) },
    });

    await screen.findByRole('rowheader', { name: 'TV' });
    await user.click(screen.getByRole('button', { name: 'Inactive' }));
    await user.click(await screen.findByRole('button', { name: 'Clear filters' }));

    expect(await screen.findByRole('rowheader', { name: 'Radio' })).toBeInTheDocument();
    expect(optionQueries(api).map((query) => query.get('status'))).toEqual([
      'all',
      'inactive',
      'all',
    ]);
  });

  it('adds an option, and shows the API problem under the label', async () => {
    let attempts = 0;
    const { user, api } = renderCard({
      [OPTIONS_PATH]: { body: page([TV]) },
      'POST /api/v1/shell-attributes/SA-1010/options': () => {
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
    });

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
    const { user, api } = renderCard({
      [OPTIONS_PATH]: { body: page([TV, RADIO]) },
      'GET /api/v1/shell-attribute-options/SO-1102': { body: { ...RADIO, version: 4 } },
      'PATCH /api/v1/shell-attribute-options/SO-1102': () => {
        attempts += 1;
        return attempts === 1
          ? {
              status: 409,
              body: {
                error: {
                  code: 'version_conflict',
                  message: 'Someone else changed this option.',
                  conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T14:32:00Z' },
                },
              },
            }
          : { body: { ...RADIO, label: 'Radio ads', version: 5 } };
      },
    });

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

  it('moves an option and sets it inactive', async () => {
    const { user, api } = renderCard({
      [OPTIONS_PATH]: { body: page([TV, RADIO]) },
      'POST /api/v1/shell-attribute-options/SO-1101/move': { body: { ...TV, position: 2 } },
      'POST /api/v1/shell-attribute-options/SO-1102/status': { body: { ...RADIO, active: false } },
    });

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
    const { user } = renderCard({ [OPTIONS_PATH]: { body: page([TV, RADIO]) } });

    await user.click(await screen.findByRole('button', { name: 'Actions for TV' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'TV' is in use" });
    expect(dialog).toHaveTextContent('Used by 7 activities — set it inactive instead.');
    expect(dialog).toHaveTextContent(
      'Forms stop offering it; records that have it keep it as "(inactive)".',
    );
  });

  it('says an inactive option in use is already inactive', async () => {
    const { user } = renderCard({
      [OPTIONS_PATH]: { body: page([{ ...TV, active: false }]) },
    });

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
    const { user, api } = renderCard({ [OPTIONS_PATH]: { body: page([only]) } });

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
    const { user } = renderCard({
      [OPTIONS_PATH]: { body: page([TV, RADIO]) },
      'DELETE /api/v1/shell-attribute-options/SO-1102': {
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
    });

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
    renderCard(
      {
        [OPTIONS_PATH]: {
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
      },
      region,
    );

    expect(await screen.findByRole('columnheader', { name: 'Linked to' })).toBeInTheDocument();
    expect(screen.getByText('Asia, Pacific (inactive)')).toBeInTheDocument();
  });
});

describe('OptionsCard: options from a spreadsheet (FR-031)', () => {
  it('opens the import of the attribute', async () => {
    const { user } = renderCard({ [OPTIONS_PATH]: { body: page([TV]) } });

    await user.click(await screen.findByRole('button', { name: 'Import from a spreadsheet' }));

    expect(
      await screen.findByRole('dialog', { name: 'Import options into Media channel' }),
    ).toBeInTheDocument();
  });

  it('downloads the template of the attribute', async () => {
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: () => 'blob:template',
    });
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: () => {} });
    const names: string[] = [];
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      names.push(this.download);
    });
    const { user } = renderCard({ [OPTIONS_PATH]: { body: page([TV]) } });

    await user.click(await screen.findByRole('button', { name: 'Download template' }));

    expect(names).toEqual(['Media channel – options template.csv']);
    click.mockRestore();
    Reflect.deleteProperty(URL, 'createObjectURL');
    Reflect.deleteProperty(URL, 'revokeObjectURL');
  });
});

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
    const { user, api } = renderCard(
      {
        [OPTIONS_PATH]: { body: page([CHINA]) },
        'GET /api/v1/shell-attribute-forms/activity': { body: FORM },
        'POST /api/v1/shell-attributes/SA-1010/options': {
          status: 201,
          body: { ...CHINA, id: 'SO-1102', label: 'Retail – Japan' },
        },
      },
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
    const { user, api } = renderCard(
      {
        [OPTIONS_PATH]: { body: page([CHINA]) },
        'GET /api/v1/shell-attribute-forms/activity': { body: FORM },
        'PATCH /api/v1/shell-attribute-options/SO-1101': {
          body: { ...CHINA, parentOptions: [{ id: 'SO-1', name: 'Asia', active: true }] },
        },
      },
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
