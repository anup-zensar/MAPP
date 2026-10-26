import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  ACTIVITY_ATTRIBUTES,
  BRANDS,
  BUSINESS_UNIT,
  MEDIA_CHANNEL,
  OLD_NOTES,
  page,
  PIR,
} from '@/testing/fixtures/shell-attributes';
import { renderShellAttributesPage, rowNames } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

const ALL = [BUSINESS_UNIT, ...ACTIVITY_ATTRIBUTES];

describe('ShellAttributesPage: list (FR-016 – FR-019)', () => {
  it('shows the plan level first, with the level counts and the header (FR-002)', async () => {
    renderShellAttributesPage('/settings/shell-attributes');

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
    const { user, router } = renderShellAttributesPage('/settings/shell-attributes?status=all');

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
    const { user, router } = renderShellAttributesPage('/settings/shell-attributes?level=activity');

    expect(await screen.findByText('3 of 4 Shell attributes')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Old notes' })).not.toBeInTheDocument();

    await user.type(screen.getByRole('searchbox', { name: 'Search label or section' }), 'lubri');
    expect(rowNames()).toEqual(['Lubricants attributes', 'Brands']);
    expect(router.state.location.search).toBe('?level=activity&q=lubri');

    await user.click(screen.getByRole('button', { name: 'Inactive 1' }));
    expect(rowNames()).toEqual(['Lubricants attributes', 'Old notes']);
  });

  it('says when nothing matches, and Clear filters shows every attribute again', async () => {
    const { user } = renderShellAttributesPage('/settings/shell-attributes?level=activity&q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(screen.getByText('4 of 4 Shell attributes')).toBeInTheDocument();
  });

  it('invites to create the first attribute of an empty level', async () => {
    renderShellAttributesPage('/settings/shell-attributes?level=campaign');

    expect(
      await screen.findByText('No campaign attributes yet — create the first one'),
    ).toBeInTheDocument();
    const links = screen.getAllByRole('link', { name: 'New Shell attribute' });
    expect(links[1]).toHaveAttribute('href', '/settings/shell-attributes/new?level=campaign');
  });

  it('shows loading, then a failure with Try again', async () => {
    let calls = 0;
    const { user } = renderShellAttributesPage('/settings/shell-attributes', [
      reply('GET /api/v1/shell-attributes', () => {
        calls += 1;
        return calls === 1
          ? {
              status: 500,
              body: {
                error: { code: 'internal_error', message: 'Something went wrong.', fields: [] },
              },
            }
          : { body: page(ALL) };
      }),
    ]);

    expect(screen.getByText('Loading…')).toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('link', { name: 'Business unit' })).toBeInTheDocument();
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
    renderShellAttributesPage('/settings/shell-attributes?level=activity', [
      reply('GET /api/v1/shell-attributes', { body: page(WITH_HELP) }),
    ]);
    await screen.findByRole('table');

    const pir = screen.getByRole('rowheader', { name: /^PIR/ });
    expect(within(pir).getByText('No help yet')).toBeInTheDocument();
    const media = screen.getByRole('rowheader', { name: /^Media channel/ });
    expect(within(media).queryByText('No help yet')).not.toBeInTheDocument();
  });

  it('lists only them from the Field help link, on the first level that has any', async () => {
    const { user, router } = renderShellAttributesPage('/settings/shell-attributes?help=missing', [
      reply('GET /api/v1/shell-attributes', { body: page(WITH_HELP) }),
    ]);
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
