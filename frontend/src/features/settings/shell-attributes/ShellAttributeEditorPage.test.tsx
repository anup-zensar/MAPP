import { screen, waitFor, within } from '@testing-library/react';
import { Outlet } from 'react-router';
import { describe, expect, it } from 'vitest';
import { SessionProvider } from '@/app/SessionProvider';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderRoutes } from '@/test/render';
import { testUser } from '@/test/users';
import { ShellAttributeEditorPage } from './ShellAttributeEditorPage';
import {
  ACTIVITY_ATTRIBUTES,
  BUSINESS_UNIT,
  BUSINESS_UNIT_LOCK,
  emptyForm,
  FORM_ANSWERS,
  LUBRICANTS,
  MEDIA_CHANNEL,
  page,
  PIR,
} from './testData';

const LIST = {
  'GET /api/shell-attributes': { body: page([BUSINESS_UNIT, ...ACTIVITY_ATTRIBUTES]) },
  ...FORM_ANSWERS,
};

/** The editor at `route`, with the attribute page and the list as stand-ins to land on. */
function renderEditor(route: string, answers: Record<string, MockHandler> = {}) {
  const api = mockApi({ ...LIST, ...answers });
  const rendered = renderRoutes(
    [
      {
        // Like the app's RootLayout: the session around every page.
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/settings/shell-attributes/new',
            element: <ShellAttributeEditorPage mode="create" />,
          },
          {
            path: '/settings/shell-attributes/:attributeId/edit',
            element: <ShellAttributeEditorPage mode="edit" />,
          },
          { path: '/settings/shell-attributes/:attributeId', element: <p>Attribute page</p> },
          { path: '/settings/shell-attributes', element: <p>Attribute list</p> },
        ],
      },
    ],
    { route, user: testUser() },
  );
  return { ...rendered, api };
}

/** The options a select offers, without "Choose…". */
function offered(name: RegExp): string[] {
  return within(screen.getByRole('combobox', { name }))
    .getAllByRole('option')
    .map((option) => option.textContent ?? '')
    .filter((text) => text !== 'Choose…');
}

describe('ShellAttributeEditorPage: create (FR-016 – FR-019)', () => {
  it('creates "Media channel" with four options and opens it (US2-1)', async () => {
    const { user, api, router } = renderEditor('/settings/shell-attributes/new?level=activity', {
      'POST /api/shell-attributes': { status: 201, body: { ...MEDIA_CHANNEL, id: 'SA-1020' } },
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'New Shell attribute' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('New Shell attribute · MAPP');
    expect(screen.getByRole('combobox', { name: /Level/ })).toHaveValue('activity');
    expect(screen.getByRole('combobox', { name: /Section/ })).toHaveValue('Shell attributes');

    await user.type(screen.getByRole('textbox', { name: /^Label/ }), 'Media channel');
    await user.selectOptions(screen.getByRole('combobox', { name: /Type/ }), 'singleChoice');
    await user.selectOptions(screen.getByRole('combobox', { name: /Marking/ }), 'neededToSubmit');
    for (const label of ['TV', 'Radio', 'Print', 'Online']) {
      await user.click(screen.getByRole('button', { name: 'Add option' }));
      await waitFor(() =>
        expect(screen.getAllByRole('textbox', { name: /^Option/ }).at(-1)).toHaveFocus(),
      );
      await user.keyboard(label);
    }
    await user.click(screen.getByRole('button', { name: "Move 'Online' up" }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1020'),
    );
    expect(await findMessages()).toHaveTextContent("Shell attribute 'Media channel' created");
    expect(api.bodiesOf('POST /api/shell-attributes')).toEqual([
      {
        label: 'Media channel',
        helpText: '',
        level: 'activity',
        type: 'singleChoice',
        marking: 'neededToSubmit',
        section: 'Shell attributes',
        parentAttributeId: '',
        conditionOptionIds: [],
        options: [{ label: 'TV' }, { label: 'Radio' }, { label: 'Online' }, { label: 'Print' }],
      },
    ]);
  });

  it('offers markings and types by level (FR-017)', async () => {
    const { user } = renderEditor('/settings/shell-attributes/new');

    await screen.findByRole('heading', { level: 1, name: 'New Shell attribute' });
    expect(offered(/^Marking/)).toEqual(['Optional', 'Needed to publish']);
    expect(offered(/^Type/)).not.toContain('Multiple choice with % split');

    await user.selectOptions(screen.getByRole('combobox', { name: /Level/ }), 'lineItem');
    expect(offered(/^Marking/)).toEqual(['Optional', 'Needed to submit']);
    expect(offered(/^Type/)).toContain('Multiple choice with % split');
    await user.selectOptions(screen.getByRole('combobox', { name: /Type/ }), 'multipleChoiceSplit');
    expect(
      screen.getByText(
        "People split the record's cost over the chosen values; the shares must add up to 100 %.",
      ),
    ).toBeInTheDocument();
  });

  it('fixes yes/no attributes as Optional, with the hint (FR-017)', async () => {
    const { user } = renderEditor('/settings/shell-attributes/new?level=plan');

    await user.selectOptions(await screen.findByRole('combobox', { name: /Type/ }), 'yesNo');

    expect(screen.queryByRole('combobox', { name: /Marking/ })).not.toBeInTheDocument();
    expect(
      screen.getByText('Always optional; shown as a tick box that defaults to No.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add option' })).not.toBeInTheDocument();
  });

  it('shows every problem it can see at once, without asking the API (US2-2)', async () => {
    const { user, api } = renderEditor('/settings/shell-attributes/new?level=activity');

    await user.selectOptions(await screen.findByRole('combobox', { name: /Type/ }), 'singleChoice');
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(await screen.findByText('2 problems need your attention')).toBeInTheDocument();
    expect(screen.getAllByText('Enter a label.')).toHaveLength(2);
    expect(screen.getAllByText('Add at least one option.')).toHaveLength(2);

    for (const label of ['TV', 'tv']) {
      await user.click(screen.getByRole('button', { name: 'Add option' }));
      await user.keyboard(label);
    }
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findAllByText("'tv' appears twice.")).toHaveLength(2);
    expect(screen.getAllByText('Enter a label.')).toHaveLength(2);
    expect(api.bodiesOf('POST /api/shell-attributes')).toEqual([]);
  });

  it("shows the API's problems together, at the fields (US2-2)", async () => {
    const { user } = renderEditor('/settings/shell-attributes/new?level=activity', {
      'POST /api/shell-attributes': {
        status: 400,
        body: {
          code: 'validation_failed',
          message: 'Check the highlighted fields.',
          fields: [
            {
              field: 'label',
              message: "The Activity form already has a Shell attribute called 'PIR'.",
            },
            { field: 'options[0].label', message: 'Use 120 characters or fewer.' },
          ],
        },
      },
    });

    await user.type(await screen.findByRole('textbox', { name: /^Label/ }), 'PIR');
    await user.selectOptions(screen.getByRole('combobox', { name: /Type/ }), 'singleChoice');
    await user.click(screen.getByRole('button', { name: 'Add option' }));
    await user.keyboard('Yes');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('2 problems need your attention')).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^Label/ })).toHaveAccessibleDescription(
      expect.stringContaining("The Activity form already has a Shell attribute called 'PIR'."),
    );
    expect(screen.getByRole('textbox', { name: 'Option 1' })).toHaveAccessibleDescription(
      'Use 120 characters or fewer.',
    );
  });

  it('asks before leaving with unsaved changes', async () => {
    const { user, router } = renderEditor('/settings/shell-attributes/new?level=activity');

    await user.type(await screen.findByRole('textbox', { name: /^Label/ }), 'Media');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(dialog).getByRole('button', { name: 'Discard changes' }));

    await waitFor(() => expect(router.state.location.search).toBe('?level=activity'));
    expect(screen.getByText('Attribute list')).toBeInTheDocument();
  });
});

describe('ShellAttributeEditorPage: edit (FR-021)', () => {
  it('shows locked fields with their reason instead of a control', async () => {
    const locked = {
      ...MEDIA_CHANNEL,
      locks: [
        {
          field: 'type',
          reason:
            "Type can't change once records use this attribute — create a new attribute instead.",
        },
        {
          field: 'level',
          reason:
            "Level can't change once records use this attribute — create a new attribute instead.",
        },
      ],
    };
    renderEditor('/settings/shell-attributes/SA-1010/edit', {
      'GET /api/shell-attributes/SA-1010': { body: locked },
    });

    expect(
      await screen.findByRole('heading', {
        level: 1,
        name: "Edit Shell attribute 'Media channel'",
      }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Edit Media channel · MAPP');
    expect(screen.queryByRole('combobox', { name: /Type/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: /Level/ })).not.toBeInTheDocument();
    expect(
      screen.getByText(
        "Type can't change once records use this attribute — create a new attribute instead.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('Single choice')).toBeInTheDocument();
    // Options are managed on the attribute page.
    expect(screen.queryByRole('button', { name: 'Add option' })).not.toBeInTheDocument();
  });

  it('shows the Business unit lock on its section (FR-023)', async () => {
    renderEditor('/settings/shell-attributes/SA-1001/edit', {
      'GET /api/shell-attributes/SA-1001': { body: BUSINESS_UNIT },
    });

    await screen.findByRole('heading', { level: 1, name: "Edit Shell attribute 'Business unit'" });
    expect(screen.queryByRole('combobox', { name: /Section/ })).not.toBeInTheDocument();
    // Level, Type, Section, Depends on and the condition.
    await waitFor(() => expect(screen.getAllByText(BUSINESS_UNIT_LOCK)).toHaveLength(5));
    expect(screen.queryByRole('combobox', { name: /Depends on/ })).not.toBeInTheDocument();
    expect(
      screen.queryByRole('group', { name: /Shown when the plan's business unit includes/ }),
    ).not.toBeInTheDocument();
  });

  it('warns how many records become incomplete before saving a new marking (US2-5)', async () => {
    const { user, api, router } = renderEditor('/settings/shell-attributes/SA-1011/edit', {
      'GET /api/shell-attributes/SA-1011': { body: PIR },
      'POST /api/shell-attributes/SA-1011/impact': {
        body: {
          warnings: [
            {
              kind: 'becomesIncomplete',
              count: 7,
              message:
                '7 activities have no value yet and will show as incomplete. Submitted and approved activities keep their status.',
            },
          ],
        },
      },
      'PATCH /api/shell-attributes/SA-1011': { body: { ...PIR, marking: 'neededToSubmit' } },
    });

    await user.selectOptions(
      await screen.findByRole('combobox', { name: /Marking/ }),
      'neededToSubmit',
    );
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "Save changes to Shell attribute 'PIR'?",
    });
    expect(dialog).toHaveTextContent(
      '7 activities have no value yet and will show as incomplete. Submitted and approved activities keep their status.',
    );
    expect(within(dialog).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(api.bodiesOf('PATCH /api/shell-attributes/SA-1011')).toEqual([]);
    await user.click(within(dialog).getByRole('button', { name: 'Save changes' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1011'),
    );
    expect(await findMessages()).toHaveTextContent("Shell attribute 'PIR' saved");
    const change = { marking: 'neededToSubmit', version: 1 };
    expect(api.bodiesOf('POST /api/shell-attributes/SA-1011/impact')).toEqual([change]);
    expect(api.bodiesOf('PATCH /api/shell-attributes/SA-1011')).toEqual([change]);
  });

  it('keeps the typed values after a conflict and saves over it on purpose (FR-007)', async () => {
    let gets = 0;
    let patches = 0;
    const { user, api } = renderEditor('/settings/shell-attributes/SA-1011/edit', {
      'GET /api/shell-attributes/SA-1011': () => {
        gets += 1;
        return { body: gets === 1 ? PIR : { ...PIR, label: 'PIR code', version: 3 } };
      },
      'PATCH /api/shell-attributes/SA-1011': () => {
        patches += 1;
        return patches === 1
          ? {
              status: 409,
              body: {
                code: 'version_conflict',
                message: 'Someone else changed this Shell attribute.',
                conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T14:32:00Z' },
              },
            }
          : { body: { ...PIR, label: 'PIR number', version: 4 } };
      },
    });

    const label = await screen.findByRole('textbox', { name: /^Label/ });
    await user.clear(label);
    await user.type(label, 'PIR number');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this Shell attribute on 14 Mar 2026 at 14:32',
    );
    expect(screen.getByRole('textbox', { name: /^Label/ })).toHaveValue('PIR number');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Shell attribute 'PIR number' saved");
    expect(api.bodiesOf('PATCH /api/shell-attributes/SA-1011')).toEqual([
      { label: 'PIR number', version: 1 },
      { label: 'PIR number', version: 3 },
    ]);
  });

  it('says when the attribute no longer exists', async () => {
    renderEditor('/settings/shell-attributes/SA-9999/edit', {
      'GET /api/shell-attributes/SA-9999': {
        status: 404,
        body: { code: 'not_found', message: 'This Shell attribute no longer exists.' },
      },
    });

    expect(
      await screen.findByText(
        'This Shell attribute no longer exists — someone may have deleted it.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to Shell attributes' })).toBeInTheDocument();
  });
});

describe('ShellAttributeEditorPage: depends on and condition (US3, FR-022, FR-023)', () => {
  it('links each option of a dependent attribute, listing every unlinked one at once (US3-2)', async () => {
    const { user, api, router } = renderEditor('/settings/shell-attributes/new?level=activity', {
      'POST /api/shell-attributes': {
        status: 201,
        body: { ...MEDIA_CHANNEL, id: 'SA-1020', label: 'Spot type' },
      },
    });

    await user.type(await screen.findByRole('textbox', { name: /^Label/ }), 'Spot type');
    await user.selectOptions(screen.getByRole('combobox', { name: /Type/ }), 'singleChoice');
    const dependsOn = await screen.findByRole('combobox', { name: /Depends on/ });
    // Only single and multiple choice attributes placed before it (not the % split Brands).
    expect(offered(/^Depends on/)).toEqual(['None', 'Media channel', 'PIR']);
    expect(dependsOn).toHaveValue('none');
    expect(dependsOn).toHaveAccessibleDescription(
      'Its options are filtered by the value chosen for that attribute.',
    );
    await user.selectOptions(dependsOn, 'SA-1010');
    for (const label of ['TV spot', 'Radio spot']) {
      await user.click(screen.getByRole('button', { name: 'Add option' }));
      await waitFor(() =>
        expect(screen.getAllByRole('textbox', { name: /^Option/ }).at(-1)).toHaveFocus(),
      );
      await user.keyboard(label);
    }
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('2 problems need your attention')).toBeInTheDocument();
    expect(
      screen.getAllByText("Link 'TV spot' to at least one media channel option."),
    ).toHaveLength(2);
    expect(
      screen.getAllByText("Link 'Radio spot' to at least one media channel option."),
    ).toHaveLength(2);
    expect(api.bodiesOf('POST /api/shell-attributes')).toEqual([]);

    const first = screen.getByRole('group', { name: 'Option 1' });
    expect(first).toHaveTextContent('Shown when one of these media channel options is chosen.');
    // An inactive parent option can't be newly linked.
    expect(within(first).queryByRole('checkbox', { name: /Print/ })).not.toBeInTheDocument();
    await user.click(within(first).getByRole('checkbox', { name: 'TV' }));
    const second = screen.getByRole('group', { name: 'Option 2' });
    await user.click(within(second).getByRole('checkbox', { name: 'Radio' }));
    await user.click(within(second).getByRole('checkbox', { name: 'Online' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1020'),
    );
    expect(api.bodiesOf('POST /api/shell-attributes')).toEqual([
      {
        label: 'Spot type',
        helpText: '',
        level: 'activity',
        type: 'singleChoice',
        marking: 'optional',
        section: 'Shell attributes',
        parentAttributeId: 'SA-1010',
        conditionOptionIds: [],
        options: [
          { label: 'TV spot', parentOptionIds: ['SO-1101'] },
          { label: 'Radio spot', parentOptionIds: ['SO-1102', 'SO-1104'] },
        ],
      },
    ]);
  });

  it('shows an attribute only for chosen business units (FR-023)', async () => {
    const { user, api } = renderEditor('/settings/shell-attributes/new?level=activity', {
      'POST /api/shell-attributes': { status: 201, body: { ...PIR, id: 'SA-1021' } },
    });

    await user.type(await screen.findByRole('textbox', { name: /^Label/ }), 'Lubricants code');
    await user.selectOptions(screen.getByRole('combobox', { name: /Type/ }), 'text');
    const condition = await screen.findByRole('group', {
      name: /Shown when the plan's business unit includes/,
    });
    expect(condition).toHaveTextContent('Leave all unticked to show it always.');
    // Only choice attributes depend on others.
    expect(screen.queryByRole('combobox', { name: /Depends on/ })).not.toBeInTheDocument();
    await user.click(within(condition).getByRole('checkbox', { name: 'Lubricants' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(api.bodiesOf('POST /api/shell-attributes')).toHaveLength(1));
    expect(api.bodiesOf('POST /api/shell-attributes')[0]).toMatchObject({
      type: 'text',
      parentAttributeId: '',
      conditionOptionIds: [LUBRICANTS],
      options: [],
    });
  });

  it('says why no condition can be set while there is no Business unit attribute', async () => {
    renderEditor('/settings/shell-attributes/new?level=plan', {
      'GET /api/shell-attribute-forms/plan': {
        body: { ...emptyForm('plan'), businessUnitAttributeId: null, businessUnitOptions: [] },
      },
    });

    expect(
      await screen.findByText(
        "There is no Business unit attribute yet, so conditions can't be set.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('group', { name: /Shown when the plan's business unit includes/ }),
    ).not.toBeInTheDocument();
  });

  it('links every active option when an attribute starts depending on another (FR-022)', async () => {
    const { user, api, router } = renderEditor('/settings/shell-attributes/SA-1011/edit', {
      'GET /api/shell-attributes/SA-1011': { body: PIR },
      'PATCH /api/shell-attributes/SA-1011': {
        body: { ...PIR, parentAttribute: { id: 'SA-1010', name: 'Media channel', active: true } },
      },
    });

    const dependsOn = await screen.findByRole('combobox', { name: /Depends on/ });
    // PIR keeps its place, after Media channel only.
    expect(offered(/^Depends on/)).toEqual(['None', 'Media channel']);
    await user.selectOptions(dependsOn, 'SA-1010');
    const yes = await screen.findByRole('group', { name: 'Yes' });
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('2 problems need your attention')).toBeInTheDocument();
    expect(screen.getAllByText("Link 'Yes' to at least one media channel option.")).toHaveLength(2);
    expect(screen.getAllByText("Link 'No' to at least one media channel option.")).toHaveLength(2);
    await user.click(within(yes).getByRole('checkbox', { name: 'TV' }));
    await user.click(
      within(screen.getByRole('group', { name: 'No' })).getByRole('checkbox', { name: 'Radio' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1011'),
    );
    expect(api.bodiesOf('PATCH /api/shell-attributes/SA-1011')).toEqual([
      {
        parentAttributeId: 'SA-1010',
        optionLinks: [
          { optionId: 'SO-1201', parentOptionIds: ['SO-1101'] },
          { optionId: 'SO-1202', parentOptionIds: ['SO-1102'] },
        ],
        version: 1,
      },
    ]);
  });

  it('removes a dependency and a condition, sending only those changes', async () => {
    const region = {
      ...PIR,
      parentAttribute: { id: 'SA-1010', name: 'Media channel', active: true },
      condition: [{ id: LUBRICANTS, name: 'Lubricants', active: true }],
    };
    const { user, api } = renderEditor('/settings/shell-attributes/SA-1011/edit', {
      'GET /api/shell-attributes/SA-1011': { body: region },
      'PATCH /api/shell-attributes/SA-1011': { body: PIR },
    });

    const dependsOn = await screen.findByRole('combobox', { name: /Depends on/ });
    expect(dependsOn).toHaveValue('SA-1010');
    await user.selectOptions(dependsOn, 'none');
    const condition = screen.getByRole('group', {
      name: /Shown when the plan's business unit includes/,
    });
    expect(within(condition).getByRole('checkbox', { name: 'Lubricants' })).toBeChecked();
    await user.click(within(condition).getByRole('checkbox', { name: 'Lubricants' }));
    // No new parent: no option links to fill in.
    expect(screen.queryByRole('group', { name: 'Yes' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(api.bodiesOf('PATCH /api/shell-attributes/SA-1011')).toEqual([
        { parentAttributeId: '', conditionOptionIds: [], version: 1 },
      ]),
    );
  });
});
