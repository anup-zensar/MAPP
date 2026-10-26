import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ShellAttributesPage } from './ShellAttributesPage';
import {
  ACTIVITY_ATTRIBUTES,
  BRANDS,
  BUSINESS_UNIT,
  BUSINESS_UNIT_LOCK,
  MEDIA_CHANNEL,
  OLD_NOTES,
  page,
  PIR,
} from '@/testing/fixtures/shell-attributes';

const ALL = [BUSINESS_UNIT, ...ACTIVITY_ATTRIBUTES];
const LIST = 'GET /api/v1/shell-attributes';

function renderPage(route: string, answers: Record<string, MockHandler> = {}) {
  const api = mockApi({ [LIST]: { body: page(ALL) }, ...answers });
  const rendered = renderWithProviders(<ShellAttributesPage />, {
    route,
    user: testUser(),
    path: '/settings/shell-attributes',
  });
  return { ...rendered, api };
}

/** The names of the table's rows, section header rows included, top to bottom. */
function rowNames(): string[] {
  const rows = within(screen.getByRole('table')).getAllByRole('row').slice(1);
  return rows.map((row) => {
    const header = within(row).queryByRole('rowheader') ?? within(row).getByRole('columnheader');
    // An attribute's name is its link; a "No help yet" chip may follow it (016 FR-048).
    return (within(header).queryByRole('link') ?? header).textContent ?? '';
  });
}

describe('ShellAttributesPage: list (FR-016 – FR-019)', () => {
  it('shows the plan level first, with the level counts and the header (FR-002)', async () => {
    renderPage('/settings/shell-attributes');

    expect(await screen.findByRole('link', { name: 'Business unit' })).toHaveAttribute(
      'href',
      '/settings/shell-attributes/SA-1001',
    );
    expect(screen.getByRole('heading', { level: 1, name: 'Shell attributes' })).toBeInTheDocument();
    expect(
      screen.getByText(
        "Shell's own fields on each level's form. Changes apply to every form straight away.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent(
      'Settings›Shell attributes',
    );
    const levels = screen.getByRole('group', { name: 'Level' });
    expect(within(levels).getByRole('button', { name: 'Plan 1' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    // Active attributes only, as the status filter says.
    expect(within(levels).getByRole('button', { name: 'Activity 3' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'New Shell attribute' })).toHaveAttribute(
      'href',
      '/settings/shell-attributes/new?level=plan',
    );
  });

  it('switches level and groups the attributes by section in the order of the form (FR-019)', async () => {
    const { user, router } = renderPage('/settings/shell-attributes?status=all');

    await user.click(await screen.findByRole('button', { name: 'Activity 4' }));

    expect(router.state.location.search).toBe('?status=all&level=activity');
    expect(rowNames()).toEqual([
      'Shell attributes',
      'Media channel',
      'PIR',
      'Lubricants attributes',
      'Brands',
      'Old notes',
    ]);
    const media = screen.getByRole('row', { name: /Media channel/ });
    expect(within(media).getByText('Single choice')).toBeInTheDocument();
    expect(within(media).getByText('Needed to submit')).toBeInTheDocument();
    expect(within(media).getByText('4')).toBeInTheDocument();
    expect(within(media).getByText('Always shown')).toBeInTheDocument();
    expect(
      within(media).getByText('Used by 12 activities (1 in the recycle bin)'),
    ).toBeInTheDocument();
    expect(within(media).getByText('Active')).toBeInTheDocument();
    const brands = screen.getByRole('row', { name: /Brands/ });
    expect(within(brands).getByText('Multiple choice with % split')).toBeInTheDocument();
    expect(
      within(brands).getByText('Shown when the business unit includes Lubricants'),
    ).toBeInTheDocument();
    const notes = screen.getByRole('row', { name: /Old notes/ });
    expect(within(notes).getByText('—')).toBeInTheDocument();
    expect(within(notes).getByText('Inactive')).toBeInTheDocument();
    expect(screen.getByText('4 of 4 Shell attributes')).toBeInTheDocument();
  });

  it('searches label or section and filters by status, keeping both in the address', async () => {
    const { user, router } = renderPage('/settings/shell-attributes?level=activity');

    expect(await screen.findByText('3 of 4 Shell attributes')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Old notes' })).not.toBeInTheDocument();

    await user.type(screen.getByRole('searchbox', { name: 'Search label or section' }), 'lubri');
    expect(rowNames()).toEqual(['Lubricants attributes', 'Brands']);
    expect(router.state.location.search).toBe('?level=activity&q=lubri');

    await user.click(screen.getByRole('button', { name: 'Inactive 1' }));
    expect(rowNames()).toEqual(['Lubricants attributes', 'Old notes']);
  });

  it('says when nothing matches, and Clear filters shows every attribute again', async () => {
    const { user } = renderPage('/settings/shell-attributes?level=activity&q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(screen.getByText('4 of 4 Shell attributes')).toBeInTheDocument();
  });

  it('invites to create the first attribute of an empty level', async () => {
    renderPage('/settings/shell-attributes?level=campaign');

    expect(
      await screen.findByText('No campaign attributes yet — create the first one'),
    ).toBeInTheDocument();
    const links = screen.getAllByRole('link', { name: 'New Shell attribute' });
    expect(links[1]).toHaveAttribute('href', '/settings/shell-attributes/new?level=campaign');
  });

  it('shows loading, then a failure with Try again', async () => {
    let calls = 0;
    const { user } = renderPage('/settings/shell-attributes', {
      [LIST]: () => {
        calls += 1;
        return calls === 1
          ? {
              status: 500,
              body: { error: { code: 'internal_error', message: 'Something went wrong.' } },
            }
          : { body: page(ALL) };
      },
    });

    expect(screen.getByText('Loading…')).toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('link', { name: 'Business unit' })).toBeInTheDocument();
  });
});

describe('ShellAttributesPage: order (FR-019, SC-009)', () => {
  /** The level's new order after moving PIR above Media channel, as the API answers it. */
  const PIR_FIRST = page([
    { ...PIR, position: 1, version: 2 },
    { ...MEDIA_CHANNEL, position: 2, version: 2 },
    BRANDS,
    OLD_NOTES,
  ]);

  it('moves an attribute with the keyboard, keeps the focus on its button and says so', async () => {
    const { user, api } = renderPage('/settings/shell-attributes?level=activity&status=all', {
      'POST /api/v1/shell-attributes/SA-1011/move': { body: PIR_FIRST },
    });

    const up = await screen.findByRole('button', { name: "Move 'PIR' up" });
    up.focus();
    await user.keyboard('{Enter}');

    await waitFor(() =>
      expect(rowNames()).toEqual([
        'Shell attributes',
        'PIR',
        'Media channel',
        'Lubricants attributes',
        'Brands',
        'Old notes',
      ]),
    );
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1011/move')).toEqual([
      { what: 'attribute', direction: 'up', version: 1 },
    ]);
    await waitFor(() =>
      expect(screen.getByRole('button', { name: "Move 'PIR' up" })).toHaveFocus(),
    );
    expect(screen.getByText("Moved 'PIR' up.")).toHaveAttribute('role', 'status');
  });

  it('keeps edge buttons focusable and says why they do nothing', async () => {
    const { user, api } = renderPage('/settings/shell-attributes?level=activity&status=all');

    const up = await screen.findByRole('button', { name: "Move 'Media channel' up" });
    expect(up).toHaveAttribute('aria-disabled', 'true');
    await user.hover(up);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      "'Media channel' is already first in its section.",
    );
    await user.click(up);
    expect(screen.getByRole('button', { name: "Move 'Old notes' down" })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
    expect(
      screen.getByRole('button', { name: "Move section 'Shell attributes' up" }),
    ).toHaveAttribute('aria-disabled', 'true');
    expect(api.requests.filter((request) => request.method === 'POST')).toEqual([]);
  });

  it('explains the order buttons while searching or filtering', async () => {
    const { user, api } = renderPage('/settings/shell-attributes?level=activity');

    const down = await screen.findByRole('button', { name: "Move 'Media channel' down" });
    expect(down).toHaveAttribute('aria-disabled', 'true');
    // Keyboard focus, which shows the tooltip: Tab from the button just before it.
    screen.getByRole('button', { name: "Move 'Media channel' up" }).focus();
    await user.tab();
    expect(down).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Clear the search and filters to change the order.',
    );
    await user.keyboard('{Enter}');
    expect(api.requests.filter((request) => request.method === 'POST')).toEqual([]);
  });

  it('moves a whole section', async () => {
    const { user, api } = renderPage('/settings/shell-attributes?level=activity&status=all', {
      'POST /api/v1/shell-attributes/SA-1012/move': {
        body: page([BRANDS, OLD_NOTES, MEDIA_CHANNEL, PIR]),
      },
    });

    await user.click(
      await screen.findByRole('button', { name: "Move section 'Lubricants attributes' up" }),
    );

    await waitFor(() => expect(rowNames()[0]).toBe('Lubricants attributes'));
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1012/move')).toEqual([
      { what: 'section', direction: 'up', version: 1 },
    ]);
    expect(screen.getByText("Moved 'Lubricants attributes' up.")).toHaveAttribute('role', 'status');
  });

  it('shows why the API refused a move', async () => {
    const { user } = renderPage('/settings/shell-attributes?level=activity&status=all', {
      'POST /api/v1/shell-attributes/SA-1013/move': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the move.',
            fields: [{ field: 'direction', message: 'Old notes must stay below Brands.' }],
          },
        },
      },
    });

    await user.click(await screen.findByRole('button', { name: "Move 'Old notes' up" }));

    expect(await findMessages()).toHaveTextContent('Old notes must stay below Brands.');
  });

  it('shows the lock of Business unit on its order buttons (FR-023)', async () => {
    const { user } = renderPage('/settings/shell-attributes');

    const down = await screen.findByRole('button', { name: "Move 'Business unit' down" });
    expect(down).toHaveAttribute('aria-disabled', 'true');
    // Keyboard focus, which shows the tooltip: Tab from the button just before it.
    screen.getByRole('button', { name: "Move 'Business unit' up" }).focus();
    await user.tab();
    expect(down).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(BUSINESS_UNIT_LOCK);
  });
});

describe('ShellAttributesPage: row actions (FR-004)', () => {
  async function choose(
    user: ReturnType<typeof renderPage>['user'],
    label: string,
    action: string,
  ) {
    await user.click(await screen.findByRole('button', { name: `Actions for ${label}` }));
    await user.click(await screen.findByRole('menuitem', { name: action }));
  }

  it('opens and edits from the menu', async () => {
    const { user, router } = renderPage('/settings/shell-attributes?level=activity');

    await choose(user, 'PIR', 'Edit');

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/shell-attributes/SA-1011/edit'),
    );
  });

  it('deletes an unused attribute with its options after asking, and offers Undo', async () => {
    const { user, api } = renderPage('/settings/shell-attributes?level=activity', {
      'DELETE /api/v1/shell-attributes/SA-1011': {
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
    });

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
    const { user, api } = renderPage('/settings/shell-attributes?level=activity', {
      'POST /api/v1/shell-attributes/SA-1010/status': { body: { ...MEDIA_CHANNEL, active: false } },
    });

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
    const { user } = renderPage('/settings/shell-attributes?level=activity', {
      'DELETE /api/v1/shell-attributes/SA-1011': {
        status: 409,
        body: {
          error: {
            code: 'in_use',
            message: "'PIR' is in use (used by 1 activity) — set it inactive instead.",
          },
        },
      },
    });

    await choose(user, 'PIR', 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete Shell attribute' }));

    const dialog = await screen.findByRole('alertdialog', { name: "'PIR' is in use" });
    expect(dialog).toHaveTextContent(
      "'PIR' is in use (used by 1 activity) — set it inactive instead.",
    );
  });

  it('reactivates an inactive attribute', async () => {
    const { user } = renderPage('/settings/shell-attributes?level=activity&status=inactive', {
      'POST /api/v1/shell-attributes/SA-1013/status': { body: { ...OLD_NOTES, active: true } },
    });

    await choose(user, 'Old notes', 'Reactivate');

    expect(await findMessages()).toHaveTextContent("Shell attribute 'Old notes' reactivated");
  });

  it('says why Business unit can be neither set inactive nor deleted (FR-023)', async () => {
    const { user, api } = renderPage('/settings/shell-attributes');

    await choose(user, 'Business unit', 'Set inactive');
    expect(await findMessages()).toHaveTextContent(BUSINESS_UNIT_LOCK);
    await choose(user, 'Business unit', 'Delete');

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.filter((request) => request.method !== 'GET')).toEqual([]);
  });
});

describe('ShellAttributesPage: attributes without help (016 FR-048)', () => {
  const WITH_HELP = [
    { ...BUSINESS_UNIT, helpText: 'The Shell business the plan belongs to.' },
    { ...MEDIA_CHANNEL, helpText: 'Where the activity reaches its audience.' },
    PIR,
    { ...BRANDS, helpText: 'The brands the activity promotes.' },
    OLD_NOTES,
  ];

  it('marks each attribute without help with "No help yet"', async () => {
    renderPage('/settings/shell-attributes?level=activity', { [LIST]: { body: page(WITH_HELP) } });
    await screen.findByRole('table');

    const pir = screen.getByRole('rowheader', { name: /^PIR/ });
    expect(within(pir).getByText('No help yet')).toBeInTheDocument();
    const media = screen.getByRole('rowheader', { name: /^Media channel/ });
    expect(within(media).queryByText('No help yet')).not.toBeInTheDocument();
  });

  it('lists only them from the Field help link, on the first level that has any', async () => {
    const { user, router } = renderPage('/settings/shell-attributes?help=missing', {
      [LIST]: { body: page(WITH_HELP) },
    });
    await screen.findByRole('table');

    // The plan level has none: the activity level opens, with PIR only (Old notes is inactive).
    expect(rowNames()).toEqual(['Shell attributes', 'PIR']);
    expect(screen.getByRole('button', { name: /^No help yet/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    );

    await user.click(screen.getByRole('button', { name: /^Any help/ }));

    // The list stays on the level it showed, now with every active attribute.
    expect(router.state.location.search).toBe('?level=activity');
    expect(rowNames()).toEqual([
      'Shell attributes',
      'Media channel',
      'PIR',
      'Lubricants attributes',
      'Brands',
    ]);
  });
});
