import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  BUSINESS_UNIT,
  BUSINESS_UNIT_LOCK,
  FORM_ANSWERS,
  MEDIA_CHANNEL,
  OLD_NOTES,
  page,
  PIR,
  shellOption,
} from '@/testing/fixtures/shell-attributes';
import { looseReply } from '@/testing/replies';
import { ShellAttributePage } from './ShellAttributePage';

const TV = shellOption({ id: 'SO-1101', label: 'TV', attributeId: 'SA-1010' });

/** The attribute page; the preview asks for the level's form (FORM_ANSWERS). */
function renderPage(id: string, answers: HttpHandler[]) {
  const api = mockApi([...FORM_ANSWERS, ...answers]);
  const rendered = renderWithProviders(<ShellAttributePage />, {
    route: `/settings/shell-attributes/${id}`,
    user: testUser(),
    path: '/settings/shell-attributes/:attributeId',
  });
  return { ...rendered, api };
}

describe('ShellAttributePage (FR-020)', () => {
  it('shows the settings with where the attribute is used, and its options', async () => {
    renderPage('SA-1010', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: MEDIA_CHANNEL },
        { attributeId: 'SA-1010' },
      ),
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Media channel' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Media channel · Shell attributes · MAPP');
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(breadcrumb).toHaveTextContent('Settings›Shell attributes›Activity form›Media channel');
    expect(within(breadcrumb).getByRole('link', { name: 'Activity form' })).toHaveAttribute(
      'href',
      '/settings/shell-attributes?level=activity',
    );
    expect(screen.getByText('ACTIVITY')).toBeInTheDocument();
    expect(screen.getByText('SA-1010')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/settings/shell-attributes/SA-1010/edit',
    );

    const settings = screen.getByRole('region', { name: 'Settings' });
    const facts = within(settings)
      .getAllByRole('definition')
      .map((value) => value.textContent);
    expect(facts).toEqual([
      'Activity',
      'Single choice',
      'Needed to submit',
      'Shell attributes',
      '—',
      'None',
      'Always shown',
      'Used by 12 activities (1 in the recycle bin)',
    ]);
    expect(settings).toHaveTextContent(
      'Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026',
    );

    const options = screen.getByRole('region', { name: 'Options' });
    expect(await within(options).findByRole('rowheader', { name: 'TV' })).toBeInTheDocument();
    expect(within(options).getByText('Not used yet')).toBeInTheDocument();

    // The preview of the Activity form, with this attribute on it (FR-020).
    const preview = screen.getByRole('region', { name: 'Preview — Activity form' });
    expect(
      await within(preview).findByRole('combobox', { name: /Media channel/ }),
    ).toBeInTheDocument();
    expect(preview).toHaveTextContent('Needed to submit');
  });

  it('shows what the attribute depends on and when it is shown (US3)', async () => {
    renderPage('SA-1011', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        {
          body: {
            ...PIR,
            parentAttribute: { id: 'SA-1010', name: 'Media channel', active: true },
            condition: [{ id: 'SO-BU-LUB', name: 'Lubricants', active: true }],
          },
        },
        { attributeId: 'SA-1011' },
      ),
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([]) },
        { attributeId: 'SA-1011' },
      ),
    ]);

    const settings = await screen.findByRole('region', { name: 'Settings' });
    const facts = within(settings)
      .getAllByRole('definition')
      .map((value) => value.textContent);
    expect(facts).toContain('Media channel');
    expect(facts).toContain('Lubricants');
  });

  it('says that types without options have none', async () => {
    renderPage('SA-1013', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: OLD_NOTES },
        { attributeId: 'SA-1013' },
      ),
    ]);

    const options = await screen.findByRole('region', { name: 'Options' });
    expect(options).toHaveTextContent('Long text attributes have no options.');
    expect(screen.getByText('Inactive')).toBeInTheDocument();
  });

  it('shows each lock once, with its reason (FR-021, FR-023)', async () => {
    renderPage('SA-1001', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: BUSINESS_UNIT },
        { attributeId: 'SA-1001' },
      ),
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([]) },
        { attributeId: 'SA-1001' },
      ),
    ]);

    expect(await screen.findAllByText(BUSINESS_UNIT_LOCK)).toHaveLength(1);
  });

  it('says when the attribute no longer exists, with the way back', async () => {
    renderPage('SA-9999', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        {
          status: 404,
          body: { error: { code: 'not_found', message: 'This Shell attribute no longer exists.' } },
        },
        { attributeId: 'SA-9999' },
      ),
    ]);

    expect(
      await screen.findByText(
        'This Shell attribute no longer exists — someone may have deleted it.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to Shell attributes' })).toHaveAttribute(
      'href',
      '/settings/shell-attributes',
    );
  });

  it('shows a failure with Try again', async () => {
    renderPage('SA-1010', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        {
          status: 500,
          body: {
            error: { code: 'internal_error', message: 'Something went wrong. Please try again.' },
          },
        },
        { attributeId: 'SA-1010' },
      ),
    ]);

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('deletes an unused attribute and goes back to its level, with Undo (FR-004)', async () => {
    const { user, router } = renderPage('SA-1011', [
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}',
        { body: { ...PIR, optionCount: 0 } },
        { attributeId: 'SA-1011' },
      ),
      looseReply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([]) },
        { attributeId: 'SA-1011' },
      ),
      looseReply(
        'DELETE /api/v1/shell-attributes/{attributeId}',
        {
          body: {
            recycleBinEntry: {
              id: 'BIN-1002',
              recordType: 'shellAttribute',
              recordTypeLabel: 'Shell attribute',
              recordId: 'SA-1011',
              recordName: 'PIR',
              path: [],
              parent: null,
              deletedAt: '2026-03-14T10:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [],
            },
          },
        },
        { attributeId: 'SA-1011' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for PIR' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete Shell attribute 'PIR'?",
    });
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    await user.click(within(dialog).getByRole('button', { name: 'Delete Shell attribute' }));

    await waitFor(() => expect(router.state.location.search).toBe('?level=activity'));
    expect(await findMessages()).toHaveTextContent(
      "Shell attribute 'PIR' moved to the recycle bin",
    );
  });
});
