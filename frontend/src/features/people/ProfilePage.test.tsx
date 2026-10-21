import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { useSession } from '@/app/useSession';
import type { CurrentUser } from '@/app/useSession';
import { useFormatters } from '@/lib/useFormatters';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { ProfilePage } from './ProfilePage';

const USD = { code: 'USD', name: 'US dollar', active: true };
const EUR = { code: 'EUR', name: 'Euro', active: true };
const CURRENCIES = { items: [EUR, USD], total: 2 };

const PRIYA_AS_PERSON = {
  id: 'U-1002',
  name: 'Priya Raman',
  email: 'priya.raman@northstar.demo',
  jobTitle: 'Marketer',
  initials: 'PR',
  active: true,
  teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true }],
  usage: [],
  version: 2,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-10T14:32:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

const VALIDATION = (fields: { field: string; message: string }[]) => ({
  status: 400,
  body: { code: 'validation_failed', message: 'Check the highlighted fields.', fields },
});

/** What the rest of the app sees of the session: the account's name and a date on another page. */
function ElsewhereInTheApp() {
  const { user } = useSession();
  const { date } = useFormatters();
  return <p>{`Account: ${user?.name} (${user?.initials}) · a date: ${date('2026-03-12')}`}</p>;
}

/**
 * My profile at /profile as `person`, with the answers every test needs (GET /api/me: the
 * session is asked again after every change).
 */
function renderProfile(answers: Record<string, MockHandler> = {}, person = testUser()) {
  const api = mockApi({
    'GET /api/me': { body: person },
    'GET /api/me/display-currencies': { body: CURRENCIES },
    ...answers,
  });
  return {
    api,
    ...renderWithProviders(
      <>
        <ProfilePage />
        <ElsewhereInTheApp />
      </>,
      { user: person, route: '/profile', path: '/profile' },
    ),
  };
}

function card(name: string) {
  return screen.getByRole('region', { name });
}

describe('ProfilePage (feature 003, US4)', () => {
  it('shows the details with the e-mail lock, the preferences and My teams (US4-1, FR-021)', async () => {
    renderProfile();

    expect(screen.getByRole('heading', { level: 1, name: 'My profile' })).toBeInTheDocument();
    expect(document.title).toBe('My profile · NorthStar');

    const details = card('Your details');
    expect(details).toHaveTextContent('Priya Raman');
    expect(details).toHaveTextContent('Marketer');
    expect(details).toHaveTextContent('priya.raman@northstar.demo');
    expect(
      within(details).getByText(
        'Your e-mail is your sign-in name; it is changed in Settings › People.',
      ),
    ).toBeInTheDocument();
    expect(within(details).getByRole('button', { name: 'Edit details' })).toBeInTheDocument();

    const preferences = card('Display preferences');
    expect(await within(preferences).findByText('USD — US dollar')).toBeInTheDocument();
    expect(within(preferences).getByText('12 Mar 2026 (DD MMM YYYY)')).toBeInTheDocument();
    expect(
      within(preferences).getByText(
        "Amounts are shown in your display currency with today's rate in Settings; amounts typed in other currencies stay as they are.",
      ),
    ).toBeInTheDocument();
    expect(
      within(preferences).getByRole('button', { name: 'Edit preferences' }),
    ).toBeInTheDocument();

    const teams = card('My teams');
    expect(within(teams).getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    expect(
      within(teams).getByRole('link', { name: 'Manage teams in Settings › Teams' }),
    ).toHaveAttribute('href', '/settings/teams');
  });

  it('saves a new name and job title; the account shows them at once (US4-2, FR-022)', async () => {
    const renamed: CurrentUser = testUser({
      name: 'Priya Iyer',
      initials: 'PI',
      jobTitle: 'Senior marketer',
      version: 2,
    });
    const { api, user } = renderProfile({
      'PATCH /api/me': { body: renamed },
      'GET /api/me': { body: renamed },
    });

    await user.click(screen.getByRole('button', { name: 'Edit details' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    const name = within(panel).getByRole('textbox', { name: /Name/ });
    expect(name).toHaveValue('Priya Raman');
    await user.clear(name);
    await user.type(name, 'Priya Iyer');
    const jobTitle = within(panel).getByRole('textbox', { name: /Job title/ });
    await user.clear(jobTitle);
    await user.type(jobTitle, 'Senior marketer');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Your details saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/me')).toEqual([
      { name: 'Priya Iyer', jobTitle: 'Senior marketer', version: 1 },
    ]);
    expect(screen.queryByRole('dialog', { name: 'Edit your details' })).not.toBeInTheDocument();
    expect(screen.getByText('Account: Priya Iyer (PI) · a date: 12 Mar 2026')).toBeInTheDocument();
    expect(card('Your details')).toHaveTextContent('Senior marketer');
  });

  it('shows "Enter a name" at the field and in the summary when the name is empty (US4-2, SC-007)', async () => {
    const { user } = renderProfile({
      'PATCH /api/me': VALIDATION([{ field: 'name', message: 'Enter a name.' }]),
    });

    await user.click(screen.getByRole('button', { name: 'Edit details' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    await user.clear(within(panel).getByRole('textbox', { name: /Name/ }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Enter a name.')).toHaveLength(2);
    expect(screen.queryByText('Your details saved')).not.toBeInTheDocument();
    expect(screen.getByText(/^Account: Priya Raman \(PR\)/)).toBeInTheDocument();
  });

  it('keeps typed values after a conflict and saves over it with the newer version (FR-022)', async () => {
    let patches = 0;
    const saved = testUser({ jobTitle: 'Brand lead', version: 3 });
    const { api, user } = renderProfile({
      'PATCH /api/me': () => {
        patches += 1;
        return patches === 1
          ? {
              status: 409,
              body: {
                code: 'version_conflict',
                message: 'Not saved — Alex Morgan changed this person.',
                fields: [],
                conflict: { changedBy: 'Alex Morgan', changedAt: '2026-03-10T14:32:00Z' },
              },
            }
          : { body: saved };
      },
      'GET /api/me': () => ({ body: patches === 1 ? testUser({ version: 2 }) : saved }),
      'GET /api/people/U-1002': { body: PRIYA_AS_PERSON },
    });

    await user.click(screen.getByRole('button', { name: 'Edit details' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    const jobTitle = within(panel).getByRole('textbox', { name: /Job title/ });
    await user.clear(jobTitle);
    await user.type(jobTitle, 'Brand lead');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findByText(/^Not saved — Alex Morgan changed this profile on/),
    ).toBeInTheDocument();
    expect(jobTitle).toHaveValue('Brand lead');

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(await screen.findByText('Your details saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/me')).toEqual([
      { name: 'Priya Raman', jobTitle: 'Brand lead', version: 1 },
      { name: 'Priya Raman', jobTitle: 'Brand lead', version: 2 },
    ]);
  });

  it('saves the display currency and date format together; dates follow at once and every query is asked again (US4-3, US4-4, FR-023, FR-024, SC-006)', async () => {
    const preferences = {
      theme: 'system',
      sidebarCollapsed: false,
      displayCurrency: 'EUR',
      dateFormat: 'DD/MM/YYYY',
      pinnedPlanIds: [] as string[],
    } as const;
    const { api, user, queryClient } = renderProfile({
      'PATCH /api/me/preferences': { body: preferences },
      'GET /api/me': { body: testUser({ preferences }) },
    });
    // A list of another page, kept in the cache: its amounts must come again, converted.
    queryClient.setQueryData(['plans'], { items: [], total: 0 });
    await within(card('Display preferences')).findByText('USD — US dollar');

    await user.click(screen.getByRole('button', { name: 'Edit preferences' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit display preferences' });
    const currency = await within(panel).findByRole('combobox', { name: 'Display currency' });
    expect(currency).toHaveDisplayValue('USD — US dollar');
    const dateFormat = within(panel).getByRole('combobox', { name: 'Date format' });
    expect(
      within(dateFormat)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual([
      'Choose…',
      '12 Mar 2026 (DD MMM YYYY)',
      '12/03/2026 (DD/MM/YYYY)',
      '03/12/2026 (MM/DD/YYYY)',
      '2026-03-12 (YYYY-MM-DD)',
    ]);
    await user.selectOptions(currency, 'EUR — Euro');
    await user.selectOptions(dateFormat, '12/03/2026 (DD/MM/YYYY)');
    const currencyAsks = api.bodiesOf('GET /api/me/display-currencies').length;
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Your preferences saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([
      { displayCurrency: 'EUR', dateFormat: 'DD/MM/YYYY' },
    ]);
    const shown = card('Display preferences');
    expect(within(shown).getByText('EUR — Euro')).toBeInTheDocument();
    expect(within(shown).getByText('12/03/2026 (DD/MM/YYYY)')).toBeInTheDocument();
    expect(screen.getByText('Account: Priya Raman (PR) · a date: 12/03/2026')).toBeInTheDocument();
    expect(queryClient.getQueryState(['plans'])?.isInvalidated).toBe(true);
    await waitFor(() =>
      expect(api.bodiesOf('GET /api/me/display-currencies')).toHaveLength(currencyAsks + 1),
    );
  });

  it('shows every problem of the preferences at its field at once (SC-007)', async () => {
    const { user } = renderProfile({
      'PATCH /api/me/preferences': VALIDATION([
        { field: 'displayCurrency', message: 'Choose one of the active currencies.' },
        { field: 'dateFormat', message: 'Choose one of the date formats.' },
      ]),
    });

    await user.click(screen.getByRole('button', { name: 'Edit preferences' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit display preferences' });
    await within(panel).findByRole('combobox', { name: 'Display currency' });
    await user.selectOptions(within(panel).getByRole('combobox', { name: 'Date format' }), '');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Choose one of the date formats.')).toHaveLength(2);
    expect(within(panel).getAllByText('Choose one of the active currencies.')).toHaveLength(2);
    expect(screen.queryByText('Your preferences saved')).not.toBeInTheDocument();
  });

  it('shows a grey field while the currencies load, and Try again when they fail (plan "States")', async () => {
    let failing = true;
    const { user } = renderProfile({
      'GET /api/me/display-currencies': () =>
        failing
          ? {
              status: 500,
              body: { code: 'internal_error', message: 'Something went wrong. Please try again.' },
            }
          : { body: CURRENCIES },
    });
    // Without the currencies' names, the card shows the code from the session.
    expect(card('Display preferences')).toHaveTextContent('USD');

    await user.click(screen.getByRole('button', { name: 'Edit preferences' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit display preferences' });
    expect(
      await within(panel).findByText('Something went wrong. Please try again.'),
    ).toBeInTheDocument();
    expect(within(panel).getByRole('combobox', { name: 'Date format' })).toBeInTheDocument();

    failing = false;
    await user.click(within(panel).getByRole('button', { name: 'Try again' }));
    expect(
      await within(panel).findByRole('combobox', { name: 'Display currency' }),
    ).toHaveDisplayValue('USD — US dollar');
  });

  it('keeps a display currency that was set inactive, labelled "(inactive)" (edge case, D-063)', async () => {
    renderProfile(
      {
        'GET /api/me/display-currencies': {
          body: { items: [EUR, { code: 'GBP', name: 'Pound sterling', active: false }, USD] },
        },
      },
      testUser({ preferences: { ...testUser().preferences, displayCurrency: 'GBP' } }),
    );

    expect(
      await within(card('Display preferences')).findByText('GBP — Pound sterling (inactive)'),
    ).toBeInTheDocument();
  });

  it('says how to join a team when the person is in none (US4-5)', () => {
    renderProfile({}, testUser({ teams: [] }));

    const teams = card('My teams');
    expect(teams).toHaveTextContent(
      "You're not in any team yet, so you can't own a plan. Teams are managed in Settings › Teams.",
    );
    expect(within(teams).getByRole('link', { name: 'Settings › Teams' })).toHaveAttribute(
      'href',
      '/settings/teams',
    );
  });

  it('uses USD, 12 Mar 2026 and no teams when an older answer leaves them out (D-191)', async () => {
    renderProfile(
      {},
      testUser({
        version: undefined,
        teams: undefined,
        preferences: { theme: 'system', sidebarCollapsed: false, pinnedPlanIds: [] },
      }),
    );

    const preferences = card('Display preferences');
    expect(await within(preferences).findByText('USD — US dollar')).toBeInTheDocument();
    expect(within(preferences).getByText('12 Mar 2026 (DD MMM YYYY)')).toBeInTheDocument();
    expect(card('My teams')).toHaveTextContent("You're not in any team yet");
  });
});
