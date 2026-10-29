import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { MEDIA_CHANNEL } from '@/testing/fixtures/shell-attributes';
import { offered, renderAttributeEditor } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('ShellAttributeEditorPage: create (FR-016 – FR-019)', () => {
  it('creates "Media reach" with four options and opens it (US2-1)', async () => {
    const { user, api, router } = renderAttributeEditor(
      '/settings/shell-attributes/new?level=activity',
      [
        reply('POST /api/v1/shell-attributes', {
          status: 201,
          body: { ...MEDIA_CHANNEL, id: 'SA-1020', label: 'Media reach' },
        }),
      ],
    );

    expect(
      await screen.findByRole('heading', { level: 1, name: 'New Shell attribute' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('New Shell attribute · SP&A One');
    expect(screen.getByRole('combobox', { name: /Level/ })).toHaveValue('activity');
    expect(screen.getByRole('combobox', { name: /Section/ })).toHaveValue('Shell attributes');

    await user.type(screen.getByRole('textbox', { name: /^Label/ }), 'Media reach');
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
    expect(await findMessages()).toHaveTextContent("Shell attribute 'Media reach' created");
    expect(api.bodiesOf('POST /api/v1/shell-attributes')).toEqual([
      {
        label: 'Media reach',
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
    const { user } = renderAttributeEditor('/settings/shell-attributes/new');

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
    const { user } = renderAttributeEditor('/settings/shell-attributes/new?level=plan');

    await user.selectOptions(await screen.findByRole('combobox', { name: /Type/ }), 'yesNo');

    expect(screen.queryByRole('combobox', { name: /Marking/ })).not.toBeInTheDocument();
    expect(
      screen.getByText('Always optional; shown as a tick box that defaults to No.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add option' })).not.toBeInTheDocument();
  });

  it('shows every problem it can see at once, without asking the API (US2-2)', async () => {
    const { user, api } = renderAttributeEditor('/settings/shell-attributes/new?level=activity');

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
    expect(api.bodiesOf('POST /api/v1/shell-attributes')).toEqual([]);
  });

  it("shows the API's problems together, at the fields (US2-2)", async () => {
    const { user } = renderAttributeEditor('/settings/shell-attributes/new?level=activity', [
      reply('POST /api/v1/shell-attributes', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [
              {
                field: 'label',
                message: "The Activity form already has a Shell attribute called 'Reach'.",
              },
              { field: 'options[0].label', message: 'Use 120 characters or fewer.' },
            ],
          },
        },
      }),
    ]);

    await user.type(await screen.findByRole('textbox', { name: /^Label/ }), 'Reach');
    await user.selectOptions(screen.getByRole('combobox', { name: /Type/ }), 'singleChoice');
    await user.click(screen.getByRole('button', { name: 'Add option' }));
    await user.keyboard('Yes');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('2 problems need your attention')).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^Label/ })).toHaveAccessibleDescription(
      expect.stringContaining("The Activity form already has a Shell attribute called 'Reach'."),
    );
    expect(screen.getByRole('textbox', { name: 'Option 1' })).toHaveAccessibleDescription(
      'Use 120 characters or fewer.',
    );
  });

  it('asks before leaving with unsaved changes', async () => {
    const { user, router } = renderAttributeEditor('/settings/shell-attributes/new?level=activity');

    await user.type(await screen.findByRole('textbox', { name: /^Label/ }), 'Media');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(dialog).getByRole('button', { name: 'Discard changes' }));

    await waitFor(() => expect(router.state.location.search).toBe('?level=activity'));
    expect(screen.getByText('Attribute list')).toBeInTheDocument();
  });
});
