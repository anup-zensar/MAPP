import { act, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/app/useSession';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { HomePage } from './HomePage';
import {
  HOME_ANSWERS,
  attentionItemOf,
  groupOf,
  logEntryOf,
  myActivityOf,
  summaryOf,
  waitingItemOf,
} from './testData';

/** The summary of a person who has nothing yet, as HOME_ANSWERS gives it. */
const HOME_EMPTY_SUMMARY = HOME_ANSWERS['GET /api/home/summary'].body as object;

const NO_TEAM_SENTENCE =
  "You're not in a team yet. Plans belong to teams — add yourself to one on the Teams page.";

function renderHome(answers: Record<string, MockHandler> = {}, user: CurrentUser = testUser()) {
  const api = mockApi({ ...HOME_ANSWERS, ...answers });
  const rendered = renderWithProviders(<HomePage />, { user, path: '/' });
  return { ...rendered, api };
}

function requestsTo(api: ReturnType<typeof mockApi>, path: string) {
  return api.requests.filter((request) => request.path === path);
}

describe('HomePage', () => {
  it('greets the person, with the scope switch and the Currency picker in the header (FR-001, US1 scenario 1, design system §2.4 E)', async () => {
    renderHome();

    expect(
      screen.getByRole('heading', { level: 1, name: 'Welcome back, Priya.' }),
    ).toBeInTheDocument();
    const header = screen.getByRole('banner');
    expect(within(header).getByRole('group', { name: 'Scope' })).toBeInTheDocument();
    expect(within(header).getByRole('combobox', { name: 'Currency' })).toHaveValue('USD');
    // Home has no create action: no primary button (design system §2.4 E).
    expect(header.querySelector('[class*="primary"]')).toBeNull();
    expect(
      await screen.findByRole('link', { name: "My teams' plans: 0 — open the Plans list" }),
    ).toBeInTheDocument();
  });

  it('tells a person without any role whom to ask, instead of empty cards (014 edge case "No role")', () => {
    renderHome(
      {},
      testUser({
        access: {
          roles: [],
          permissions: [],
          administrators: [
            { id: 'U-1001', name: 'Alex Morgan', initials: 'AM', jobTitle: 'Admin', active: true },
          ],
          noSelfApproval: true,
        },
      }),
    );

    expect(screen.getByText('No access to plans yet')).toBeInTheDocument();
    expect(
      screen.getByText(
        "You don't have access to any plans yet. Ask an administrator (Alex Morgan) to give you a role.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Summary' })).not.toBeInTheDocument();
    expect(screen.queryByRole('group', { name: 'Scope' })).not.toBeInTheDocument();
  });

  it('names the page "Home · NorthStar" in the browser tab', () => {
    renderHome();

    expect(document.title).toBe('Home · NorthStar');
  });

  it('starts on the scope the person chose last time (FR-003)', async () => {
    const { api } = renderHome(
      {},
      testUser({ preferences: { ...testUser().preferences, homeScope: 'everything' } }),
    );

    expect(
      await screen.findByRole('link', { name: 'All plans: 0 — open the Plans list' }),
    ).toBeInTheDocument();
    expect(requestsTo(api, '/api/home/summary').map((request) => request.search)).toEqual([
      '?scope=everything',
    ]);
  });

  it('shows Getting started with the no-team sentence to a person in no team (FR-020, US5 scenario 1)', async () => {
    renderHome({ 'GET /api/home/summary': { body: summaryOf({ gettingStarted: 'notInTeam' }) } });

    const card = await screen.findByRole('region', { name: 'Getting started' });
    expect(within(card).getAllByRole('listitem')).toHaveLength(5);
    expect(within(card).getByText(/You're not in a team yet/)).toHaveTextContent(NO_TEAM_SENTENCE);
  });

  it('shows Getting started without that sentence when no team of the person has a plan (FR-020)', async () => {
    renderHome({ 'GET /api/home/summary': { body: summaryOf({ gettingStarted: 'noTeamPlans' }) } });

    const card = await screen.findByRole('region', { name: 'Getting started' });
    expect(within(card).queryByText(/You're not in a team yet/)).not.toBeInTheDocument();
  });

  it('leaves Getting started out once a team of the person has a plan (US5 scenario 2)', async () => {
    renderHome({ 'GET /api/home/summary': { body: summaryOf() } });

    await screen.findByRole('link', { name: "My teams' plans: 1 — open the Plans list" });
    expect(screen.queryByRole('region', { name: 'Getting started' })).not.toBeInTheDocument();
  });

  it('asks for current figures each time Home is opened again (FR-019)', async () => {
    const { api, router } = renderHome();
    await screen.findByRole('link', { name: /My teams' plans/ });

    await act(() => router.navigate('/plans'));
    await screen.findByText('Another page');
    await act(() => router.navigate('/'));

    await waitFor(() => expect(requestsTo(api, '/api/home/summary')).toHaveLength(2));
  });

  it('keeps the page when the summary cannot load, with Try again in its place (FR-022)', async () => {
    renderHome({ 'GET /api/home/summary': { status: 500 } });

    const summary = await screen.findByRole('region', { name: 'Summary' });
    expect(await within(summary).findByRole('alert')).toHaveTextContent(
      "This section couldn't load",
    );
    expect(
      screen.getByRole('heading', { level: 1, name: 'Welcome back, Priya.' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Currency' })).toBeInTheDocument();
    // The other cards still show (FR-022).
    expect(await screen.findByText('None of your teams has a plan yet.')).toBeInTheDocument();
  });

  it('offers the My teams / Everything switch with its hint, on My teams by default (FR-003)', async () => {
    renderHome();

    const scope = screen.getByRole('group', { name: 'Scope' });
    expect(within(scope).getByRole('button', { name: 'My teams' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(within(scope).getByRole('button', { name: 'Everything' })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
    expect(
      screen.getByText(
        'Your own activities, what needs your attention and your approvals always cover all plans.',
      ),
    ).toBeInTheDocument();
    expect(await screen.findByRole('region', { name: 'Plans at a glance' })).toBeInTheDocument();
  });

  it('widens the plans cards to Everything at once and remembers the choice (FR-003, US2 scenario 4)', async () => {
    const { api, user } = renderHome({
      'PATCH /api/me/preferences': {
        body: { ...testUser().preferences, homeScope: 'everything' },
      },
    });
    await screen.findByRole('link', { name: "My teams' plans: 0 — open the Plans list" });

    await user.click(screen.getByRole('button', { name: 'Everything' }));

    expect(
      await screen.findByRole('link', { name: 'All plans: 0 — open the Plans list' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Live roll-up across all plans')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Everything' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ homeScope: 'everything' }]);
    expect(requestsTo(api, '/api/home/summary').map((request) => request.search)).toEqual([
      '?scope=myTeams',
      '?scope=everything',
    ]);
    expect(requestsTo(api, '/api/home/plans').map((request) => request.search)).toEqual([
      '?scope=myTeams',
      '?scope=everything',
    ]);
    await waitFor(() =>
      expect(requestsTo(api, '/api/home/recent-changes').map((request) => request.search)).toEqual([
        '?scope=myTeams',
        '?scope=everything',
      ]),
    );
    // The person's own work covers every plan whatever the scope: it is not asked again.
    expect(requestsTo(api, '/api/home/attention')).toHaveLength(1);
    expect(requestsTo(api, '/api/home/waiting')).toHaveLength(1);
    expect(requestsTo(api, '/api/home/my-activities')).toHaveLength(1);
  });

  it('shows what needs the person and what waits for their approval (US3)', async () => {
    renderHome({
      'GET /api/home/attention': {
        body: { items: [attentionItemOf()], total: 1 },
      },
      'GET /api/home/waiting': { body: { items: [waitingItemOf()], total: 1 } },
    });

    const attention = await screen.findByRole('region', { name: 'Needs your attention' });
    expect(await within(attention).findByRole('link', { name: 'Launch film' })).toBeInTheDocument();
    const waiting = screen.getByRole('region', { name: 'Waiting for my approval' });
    expect(
      await within(waiting).findByRole('link', { name: 'Review Dealer summit' }),
    ).toBeInTheDocument();
  });

  it('shows my activities and the recent changes of the plans in scope (US4)', async () => {
    renderHome({
      'GET /api/home/my-activities': {
        body: { groups: [groupOf('approved', [myActivityOf()])], total: 1 },
      },
      'GET /api/home/recent-changes': { body: { items: [logEntryOf()], total: 1 } },
    });

    const activities = await screen.findByRole('region', { name: 'My activities' });
    expect(
      await within(activities).findByRole('link', { name: 'Launch film' }),
    ).toBeInTheDocument();
    expect(activities.closest('#my-activities')).not.toBeNull();
    const changes = screen.getByRole('region', { name: 'Recent changes' });
    expect(await within(changes).findByRole('link', { name: 'Launch film' })).toBeInTheDocument();
  });

  it('gives a brand-new person an empty state in every card, never a blank one (FR-021, SC-006)', async () => {
    renderHome({
      'GET /api/home/summary': { body: { ...HOME_EMPTY_SUMMARY, gettingStarted: 'notInTeam' } },
    });

    expect(await screen.findByRole('region', { name: 'Getting started' })).toBeInTheDocument();
    expect(await screen.findByText('No plans in scope')).toBeInTheDocument();
    expect(await screen.findByText('None of your teams has a plan yet.')).toBeInTheDocument();
    expect(await screen.findByText('Nothing needs your attention right now.')).toBeInTheDocument();
    expect(
      await screen.findByText('Nothing is waiting for your approval right now.'),
    ).toBeInTheDocument();
    expect(await screen.findByText("You don't own any activities yet.")).toBeInTheDocument();
    expect(await screen.findByText('No changes yet in these plans.')).toBeInTheDocument();
  });

  it('keeps every other card when one card cannot load (FR-022)', async () => {
    renderHome({ 'GET /api/home/attention': { status: 500 } });

    const attention = await screen.findByRole('region', { name: 'Needs your attention' });
    expect(await within(attention).findByRole('alert')).toHaveTextContent(
      "This section couldn't load",
    );
    expect(
      await screen.findByText('Nothing is waiting for your approval right now.'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: "My teams' plans: 0 — open the Plans list" }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole('alert')).toHaveLength(1);
  });

  it('switches to Everything from the empty Plans at a glance (FR-021)', async () => {
    const { api, user } = renderHome({
      'PATCH /api/me/preferences': {
        body: { ...testUser().preferences, homeScope: 'everything' },
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Switch to Everything' }));

    expect(
      await screen.findByText('No plans yet — create the first one on the Plans page.'),
    ).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ homeScope: 'everything' }]);
  });

  it('says when the scope could not be saved, with Try again (FR-003)', async () => {
    const { user } = renderHome({ 'PATCH /api/me/preferences': { status: 500 } });
    await screen.findByRole('link', { name: /My teams' plans/ });

    await user.click(screen.getByRole('button', { name: 'Everything' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Your choice couldn't be saved.");
    expect(within(messages).getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});
