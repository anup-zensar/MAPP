import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  BUSINESS_UNIT,
  BUSINESS_UNIT_LOCK,
  MEDIA_CHANNEL,
  PIR,
} from '@/testing/fixtures/shell-attributes';
import { renderAttributeEditor } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

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
    renderAttributeEditor('/settings/shell-attributes/SA-1010/edit', [
      reply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: locked },
        { attributeId: 'SA-1010' },
      ),
    ]);

    expect(
      await screen.findByRole('heading', {
        level: 1,
        name: "Edit Shell attribute 'Media channel'",
      }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Edit Media channel · SP&A One');
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
    renderAttributeEditor('/settings/shell-attributes/SA-1001/edit', [
      reply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: BUSINESS_UNIT },
        { attributeId: 'SA-1001' },
      ),
    ]);

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
    const { user, api, router } = renderAttributeEditor('/settings/shell-attributes/SA-1011/edit', [
      reply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: PIR },
        { attributeId: 'SA-1011' },
      ),
      reply(
        'POST /api/v1/shell-attributes/{attributeId}/impact',
        {
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
        { attributeId: 'SA-1011' },
      ),
      reply(
        'PATCH /api/v1/shell-attributes/{attributeId}',
        { body: { ...PIR, marking: 'neededToSubmit' } },
        { attributeId: 'SA-1011' },
      ),
    ]);

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
    expect(api.bodiesOf('PATCH /api/v1/shell-attributes/SA-1011')).toEqual([]);
    await user.click(within(dialog).getByRole('button', { name: 'Save changes' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1011'),
    );
    expect(await findMessages()).toHaveTextContent("Shell attribute 'PIR' saved");
    const change = { marking: 'neededToSubmit', version: 1 };
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1011/impact')).toEqual([change]);
    expect(api.bodiesOf('PATCH /api/v1/shell-attributes/SA-1011')).toEqual([change]);
  });

  it('keeps the typed values after a conflict and saves over it on purpose (FR-007)', async () => {
    let gets = 0;
    let patches = 0;
    const { user, api } = renderAttributeEditor('/settings/shell-attributes/SA-1011/edit', [
      reply(
        'GET /api/v1/shell-attributes/{attributeId}',
        () => {
          gets += 1;
          return { body: gets === 1 ? PIR : { ...PIR, label: 'PIR code', version: 3 } };
        },
        { attributeId: 'SA-1011' },
      ),
      reply(
        'PATCH /api/v1/shell-attributes/{attributeId}',
        () => {
          patches += 1;
          return patches === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Someone else changed this Shell attribute.',
                    fields: [],
                    conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T14:32:00Z' },
                  },
                },
              }
            : { body: { ...PIR, label: 'PIR number', version: 4 } };
        },
        { attributeId: 'SA-1011' },
      ),
    ]);

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
    expect(api.bodiesOf('PATCH /api/v1/shell-attributes/SA-1011')).toEqual([
      { label: 'PIR number', version: 1 },
      { label: 'PIR number', version: 3 },
    ]);
  });

  it('says when the attribute no longer exists', async () => {
    renderAttributeEditor('/settings/shell-attributes/SA-9999/edit', [
      reply(
        'GET /api/v1/shell-attributes/{attributeId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This Shell attribute no longer exists.',
              fields: [],
            },
          },
        },
        { attributeId: 'SA-9999' },
      ),
    ]);

    expect(
      await screen.findByText(
        'This Shell attribute no longer exists — someone may have deleted it.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to Shell attributes' })).toBeInTheDocument();
  });
});
