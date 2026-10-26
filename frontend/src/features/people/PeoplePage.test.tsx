import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { PeoplePage } from './PeoplePage';

const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-10T14:32:00Z',
  updatedBy: { id: 'U-1002', name: 'Priya Raman' },
};
const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true };
const CHINA = { id: 'T-1002', name: 'Mobility China', colour: 'teal', active: true };

function person(
  id: string,
  name: string,
  jobTitle: string,
  teams: (typeof EUROPE)[],
  active = true,
) {
  const email = `${name.toLowerCase().replace(' ', '.')}@shell.com`;
  const initials = name
    .split(' ')
    .map((part) => part[0])
    .join('');
  return { id, name, email, jobTitle, initials, active, teams, ...META };
}

const ALEX = person('U-1001', 'Alex Morgan', 'Head of marketing', [EUROPE]);
const PRIYA = person('U-1002', 'Priya Raman', 'Marketer', [EUROPE]);
const DANIEL = person('U-1003', 'Daniel Okoro', '', [CHINA]);
const SARA = person('U-1004', 'Sara Novak', 'Local finance advisor', [CHINA]);
const JANE = person('U-1006', 'Jane Doe', 'Marketer', [], false);
const PEOPLE = { items: [ALEX, DANIEL, JANE, PRIYA, SARA], total: 5 };

const ALEX_DETAILS = {
  ...ALEX,
  usage: [
    { role: 'owner', recordType: 'plan', label: 'plans', count: 2 },
    { role: 'owner', recordType: 'activity', label: 'activities', count: 5 },
    { role: 'approver', recordType: 'approvalWorkflow', label: 'approval workflow', count: 1 },
  ],
};

function team(ref: typeof EUROPE) {
  return { ...ref, description: '', members: [], plans: [], ...META };
}
const TEAMS = { items: [team(EUROPE), team(CHINA)], total: 2 };

/** Someone who may not manage access can't read a person's roles: the Roles line is left out. */
const ROLES_FORBIDDEN = {
  status: 403,
  body: { error: { code: 'forbidden', message: "You don't have access to this.", fields: [] } },
};
const ROLES_OF_EVERYONE = Object.fromEntries(
  ['U-1001', 'U-1002', 'U-1004', 'U-1006', 'U-1007'].map((id) => [
    `GET /api/v1/people/${id}/roles`,
    ROLES_FORBIDDEN,
  ]),
);

/** The page at /settings/people (or one person's address) with the answers every test needs. */
function renderPeople(answers: Record<string, MockHandler> = {}, route = '/settings/people') {
  const api = mockApi({
    'GET /api/v1/me': { body: testUser() },
    'GET /api/v1/people': { body: PEOPLE },
    'GET /api/v1/teams': { body: TEAMS },
    ...ROLES_OF_EVERYONE,
    ...answers,
  });
  return {
    api,
    ...renderWithProviders(<PeoplePage />, {
      user: testUser(),
      route,
      path: '/settings/people/:personId?',
    }),
  };
}

async function openActions(user: ReturnType<typeof renderPeople>['user'], name: string) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  return screen.getByRole('menu');
}

async function chooseAction(
  user: ReturnType<typeof renderPeople>['user'],
  name: string,
  action: string,
) {
  const menu = await openActions(user, name);
  await user.click(within(menu).getByRole('menuitem', { name: action }));
}

function rowOf(name: string) {
  return screen.getByRole('row', { name: new RegExp(`^${name}`) });
}

/** The value next to a label of the details' fact list. */
function factOf(panel: HTMLElement, label: string) {
  const term = within(panel).getByText(label, { selector: 'dt' });
  return term.nextElementSibling as HTMLElement;
}

const VALIDATION = (fields: { field: string; message: string }[]) => ({
  status: 400,
  body: { error: { code: 'validation_failed', message: 'Check the highlighted fields.', fields } },
});

describe('PeoplePage (feature 003, US1)', () => {
  it('lists the active people first, and everyone under All (FR-001)', async () => {
    const { user } = renderPeople();

    expect(await screen.findByText('4 of 5 people')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Active 4' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.queryByRole('row', { name: /^Jane Doe/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'All 5' }));
    expect(screen.getByText('5 of 5 people')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'People' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    const priya = rowOf('Priya Raman');
    expect(within(priya).getByRole('link', { name: 'Priya Raman (you)' })).toHaveAttribute(
      'href',
      '/settings/people/U-1002',
    );
    expect(priya).toHaveTextContent('priya.raman@shell.com');
    expect(priya).toHaveTextContent('Lubricants Europe');
    expect(priya).toHaveTextContent('Active');
    expect(rowOf('Daniel Okoro')).toHaveTextContent('—');
    const jane = rowOf('Jane Doe');
    expect(jane).toHaveTextContent('Jane Doe (inactive)');
    expect(jane).toHaveTextContent('No team');
    expect(jane).toHaveTextContent('Inactive');
  });

  it('searches name, e-mail and job title, filters by status, and clears the filters', async () => {
    const { user } = renderPeople();
    const search = await screen.findByRole('searchbox', {
      name: 'Search name, e-mail or job title',
    });

    await user.type(search, 'finance');
    expect(screen.getByText('1 of 5 people')).toBeInTheDocument();
    expect(rowOf('Sara Novak')).toBeInTheDocument();

    await user.clear(search);
    await user.type(search, 'okoro@');
    expect(rowOf('Daniel Okoro')).toBeInTheDocument();
    expect(screen.getAllByRole('row')).toHaveLength(2);

    await user.clear(search);
    await user.type(search, 'ALEX');
    expect(rowOf('Alex Morgan')).toBeInTheDocument();

    await user.clear(search);
    await user.click(screen.getByRole('button', { name: /^Inactive/ }));
    expect(screen.getByText('1 of 5 people')).toBeInTheDocument();
    expect(rowOf('Jane Doe')).toBeInTheDocument();

    await user.type(search, 'sara');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('5 of 5 people')).toBeInTheDocument();
  });

  it('shows how to add the first person when there is none', async () => {
    renderPeople({ 'GET /api/v1/people': { body: { items: [], total: 0 } } });

    expect(await screen.findByText('No people yet — add the first person')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'New person' })).toHaveLength(2);
  });

  it('says when the people could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderPeople({
      'GET /api/v1/people': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: PEOPLE };
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('4 of 5 people')).toBeInTheDocument();
  });

  it('shows every problem at once, then creates the person with View (US1-1, US1-2)', async () => {
    let posts = 0;
    const created = { ...person('U-1007', 'Jane Smith', 'Marketer', [EUROPE]), usage: [] };
    const { api, user, router } = renderPeople({
      'POST /api/v1/people': () => {
        posts += 1;
        if (posts === 1) {
          return VALIDATION([
            { field: 'name', message: 'Enter a name.' },
            { field: 'email', message: 'Enter a valid e-mail address.' },
          ]);
        }
        if (posts === 2) {
          return VALIDATION([
            {
              field: 'email',
              message:
                'jane.doe@shell.com already belongs to Jane Doe (inactive) — reactivate them instead.',
            },
          ]);
        }
        return { status: 201, body: created };
      },
      'GET /api/v1/people/U-1007': { body: created },
    });

    await user.click(await screen.findByRole('button', { name: 'New person' }));
    const panel = await screen.findByRole('dialog', { name: 'New person' });
    expect(
      within(panel).getByText(
        'Their sign-in name. They can sign in straight away with the demo password.',
      ),
    ).toBeInTheDocument();
    await user.type(within(panel).getByRole('textbox', { name: /E-mail/ }), 'jane');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Enter a name.')).toHaveLength(2);
    expect(within(panel).getAllByText('Enter a valid e-mail address.')).toHaveLength(2);

    await user.type(within(panel).getByRole('textbox', { name: /Name/ }), 'Jane Smith');
    await user.type(within(panel).getByRole('textbox', { name: /E-mail/ }), '.doe@shell.com');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(
      await within(panel).findAllByText(
        'jane.doe@shell.com already belongs to Jane Doe (inactive) — reactivate them instead.',
      ),
    ).toHaveLength(2);
    expect(within(panel).queryByText('Enter a name.')).not.toBeInTheDocument();

    const email = within(panel).getByRole('textbox', { name: /E-mail/ });
    await user.clear(email);
    await user.type(email, 'jane.smith@shell.com');
    await user.type(within(panel).getByRole('textbox', { name: /Job title/ }), 'Marketer');
    await user.type(await within(panel).findByRole('combobox', { name: 'Teams' }), 'europe');
    await user.keyboard('{Enter}');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Person 'Jane Smith' created")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people')[2]).toEqual({
      name: 'Jane Smith',
      email: 'jane.smith@shell.com',
      jobTitle: 'Marketer',
      teamIds: ['T-1001'],
    });
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/people/U-1007'));
  });

  it('shows a person with their teams, what they own and approve, and the record line (US1-3)', async () => {
    const { user, router } = renderPeople(
      {
        'GET /api/v1/people/U-1001': { body: ALEX_DETAILS },
        'GET /api/v1/people/U-1001/roles': {
          body: {
            person: { id: 'U-1001', name: 'Alex Morgan', active: true },
            teams: [EUROPE],
            roles: [
              {
                id: 'RA-1001',
                role: { id: 'RL-1003', name: 'Planner', builtIn: 'planner' },
                scope: { type: 'myTeams', teams: [EUROPE], text: 'My teams' },
                scopeEmpty: false,
              },
            ],
            permissions: [],
            version: 1,
          },
        },
      },
      '/settings/people/U-1001',
    );

    const panel = await screen.findByRole('dialog', { name: 'Alex Morgan' });
    expect(await within(panel).findByText('alex.morgan@shell.com')).toBeInTheDocument();
    expect(within(panel).getByText('Head of marketing')).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    // Two plain labels (functional review S-02): what they own, and where they approve.
    expect(factOf(panel, 'Owns')).toHaveTextContent('2 plans and 5 activities');
    expect(factOf(panel, 'Approves in')).toHaveTextContent('1 approval workflow');
    expect(await within(panel).findByText('Planner · My teams')).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'Change roles' })).toHaveAttribute(
      'href',
      '/settings/access?tab=people&person=U-1001',
    );
    expect(
      within(panel).getByText(/^Created by Alex Morgan on .* · Last changed by Priya Raman on /),
    ).toBeInTheDocument();
    // The one panel footer: Edit, then ⋯ with Deactivate and its reason.
    expect(within(panel).getByRole('button', { name: 'Edit' })).toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Actions for Alex Morgan' }));
    expect(
      within(screen.getByRole('menu')).getByRole('menuitem', { name: 'Deactivate' }),
    ).toHaveAccessibleDescription('People are never deleted, so the history keeps their name.');
    await user.keyboard('{Escape}');

    await user.click(within(panel).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/people'));
  });

  it('says in the panel when the person does not exist', async () => {
    renderPeople(
      {
        'GET /api/v1/people/U-1999': {
          status: 404,
          body: { error: { code: 'not_found', message: "This person doesn't exist.", fields: [] } },
        },
      },
      '/settings/people/U-1999',
    );

    const panel = await screen.findByRole('dialog', { name: 'Person' });
    expect(await within(panel).findByText("This person doesn't exist.")).toBeInTheDocument();
  });

  it('edits a person and says who saved first after a conflict (US1-4)', async () => {
    let patches = 0;
    const { api, user } = renderPeople({
      'PATCH /api/v1/people/U-1001': () => {
        patches += 1;
        return patches === 1
          ? {
              status: 409,
              body: {
                error: {
                  code: 'version_conflict',
                  message: 'Not saved — Daniel Okoro changed this person.',
                  fields: [],
                  conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-10T14:32:00Z' },
                },
              },
            }
          : { body: { ...ALEX_DETAILS, jobTitle: 'Marketing director', version: 3 } };
      },
      'GET /api/v1/people/U-1001': { body: { ...ALEX_DETAILS, version: 2 } },
    });

    await chooseAction(user, 'Alex Morgan', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    expect(
      within(panel).getByText('Their sign-in name — the next sign-in uses the new e-mail.'),
    ).toBeInTheDocument();
    const jobTitle = within(panel).getByRole('textbox', { name: /Job title/ });
    await user.clear(jobTitle);
    await user.type(jobTitle, 'Marketing director');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findByText(
        'Not saved — Daniel Okoro changed this person on 10 Mar 2026 at 14:32',
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(api.requests.some((request) => request.path === '/api/v1/people/U-1001')).toBe(true),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Person 'Alex Morgan' saved")).toBeInTheDocument();
    // Only the changed field and the version (D-210).
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([
      { jobTitle: 'Marketing director', version: 1 },
      { jobTitle: 'Marketing director', version: 2 },
    ]);
    expect(api.bodiesOf('POST /api/v1/people/U-1001/impact')).toEqual([]);
  });

  it('checks the impact of removing a team; Keep editing sends nothing, Save changes saves', async () => {
    const warning =
      'Alex Morgan owns 1 plan that uses Lubricants Europe: FY2026 Lubricants Europe. Alex will no longer belong to its teams, so its owner will need attention.';
    const { api, user } = renderPeople({
      'POST /api/v1/people/U-1001/impact': {
        body: { warnings: [{ kind: 'ownerNeedsAttention', message: warning, count: 1 }] },
      },
      'PATCH /api/v1/people/U-1001': { body: { ...ALEX_DETAILS, teams: [], version: 2 } },
    });

    await chooseAction(user, 'Alex Morgan', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    await user.click(
      await within(panel).findByRole('button', { name: 'Remove Lubricants Europe' }),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: 'Some plans will need a new owner',
    });
    expect(dialog).toHaveTextContent(warning);
    await user.click(within(dialog).getByRole('button', { name: 'Keep editing' }));
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([]);

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    await user.click(
      within(
        await screen.findByRole('alertdialog', { name: 'Some plans will need a new owner' }),
      ).getByRole('button', { name: 'Save changes' }),
    );

    expect(await screen.findByText("Person 'Alex Morgan' saved")).toBeInTheDocument();
    const body = { teamIds: [], version: 1 };
    expect(api.bodiesOf('POST /api/v1/people/U-1001/impact')).toEqual([body, body]);
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([body]);
  });

  it('shows the impact check problems at the fields', async () => {
    const { api, user } = renderPeople({
      'POST /api/v1/people/U-1001/impact': VALIDATION([
        { field: 'name', message: 'Enter a name.' },
      ]),
    });

    await chooseAction(user, 'Alex Morgan', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    await user.clear(within(panel).getByRole('textbox', { name: /Name/ }));
    await user.click(
      await within(panel).findByRole('button', { name: 'Remove Lubricants Europe' }),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Enter a name.')).toHaveLength(2);
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([]);
  });

  it('asks before deactivating, with what the person owns and Cancel focused (US1-5)', async () => {
    const { api, user } = renderPeople({
      'GET /api/v1/people/U-1001': { body: ALEX_DETAILS },
      'POST /api/v1/people/U-1001/status': { body: { ...ALEX_DETAILS, active: false, version: 2 } },
    });

    await chooseAction(user, 'Alex Morgan', 'Deactivate');
    const dialog = await screen.findByRole('alertdialog', { name: 'Deactivate Alex Morgan?' });
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(
      await within(dialog).findByText(
        "Alex owns 2 plans and 5 activities and approves in 1 approval workflow; these keep the name, shown as 'Alex Morgan (inactive)'.",
      ),
    ).toBeInTheDocument();
    expect(dialog).toHaveTextContent("Alex won't be able to sign in or be chosen in people lists.");
    expect(dialog).toHaveTextContent('Their teams are kept.');

    await user.click(within(dialog).getByRole('button', { name: 'Deactivate person' }));

    expect(await screen.findByText("Person 'Alex Morgan' deactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1001/status')).toEqual([
      { active: false, version: 1 },
    ]);
  });

  it('sends nothing when the deactivation is cancelled', async () => {
    const { api, user } = renderPeople({ 'GET /api/v1/people/U-1001': { body: ALEX_DETAILS } });

    await chooseAction(user, 'Alex Morgan', 'Deactivate');
    const dialog = await screen.findByRole('alertdialog', { name: 'Deactivate Alex Morgan?' });
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1001/status')).toEqual([]);
  });

  it("explains that you can't deactivate yourself (US1-6)", async () => {
    const { api, user } = renderPeople();

    const menu = await openActions(user, 'Priya Raman');
    const deactivate = within(menu).getByRole('menuitem', { name: 'Deactivate' });
    expect(deactivate).toHaveAccessibleDescription(
      "You can't deactivate yourself — ask someone else to do it.",
    );
    await user.click(deactivate);

    expect(
      await screen.findByText("You can't deactivate yourself — ask someone else to do it."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'POST')).toBe(false);
  });

  it('keeps Deactivate in the details for yourself, saying why it does nothing (FR-006)', async () => {
    const { api, user } = renderPeople(
      { 'GET /api/v1/people/U-1002': { body: { ...PRIYA, usage: [] } } },
      '/settings/people/U-1002',
    );

    const panel = await screen.findByRole('dialog', { name: 'Priya Raman' });
    expect(await within(panel).findByText('Nothing yet')).toBeInTheDocument();
    expect(factOf(panel, 'Approves in')).toHaveTextContent('No approval workflows');
    await user.click(within(panel).getByRole('button', { name: 'Actions for Priya Raman' }));
    const deactivate = within(screen.getByRole('menu')).getByRole('menuitem', {
      name: 'Deactivate',
    });
    expect(deactivate).toHaveAccessibleDescription(
      "You can't deactivate yourself — ask someone else to do it.",
    );
    await user.click(deactivate);

    expect(
      await screen.findByText("You can't deactivate yourself — ask someone else to do it."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'POST')).toBe(false);
  });

  it('deactivates someone who owns nothing at once, with Undo (design system §6.7)', async () => {
    const saraDetails = { ...SARA, usage: [] };
    let posts = 0;
    const { api, user } = renderPeople({
      'GET /api/v1/people/U-1004': { body: saraDetails },
      'POST /api/v1/people/U-1004/status': () => {
        posts += 1;
        return posts === 1
          ? { body: { ...saraDetails, active: false, version: 2 } }
          : { body: { ...saraDetails, active: true, version: 3 } };
      },
    });

    await chooseAction(user, 'Sara Novak', 'Deactivate');

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    const messages = await findMessages();
    expect(
      await within(messages).findByText("Person 'Sara Novak' deactivated"),
    ).toBeInTheDocument();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    expect(await screen.findByText("Person 'Sara Novak' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1004/status')).toEqual([
      { active: false, version: 1 },
      { active: true, version: 2 },
    ]);
  });

  it('reactivates an inactive person (US1-7)', async () => {
    const { api, user } = renderPeople(
      {
        'POST /api/v1/people/U-1006/status': {
          body: { ...JANE, active: true, version: 2, usage: [] },
        },
      },
      '/settings/people?status=inactive',
    );

    await chooseAction(user, 'Jane Doe', 'Reactivate');

    expect(await screen.findByText("Person 'Jane Doe' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1006/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });

  it('says why a status change was refused', async () => {
    const { user } = renderPeople(
      {
        'POST /api/v1/people/U-1006/status': {
          status: 404,
          body: { error: { code: 'not_found', message: "This person doesn't exist.", fields: [] } },
        },
      },
      '/settings/people?status=inactive',
    );

    await chooseAction(user, 'Jane Doe', 'Reactivate');

    expect(await screen.findByText("This person doesn't exist.")).toBeInTheDocument();
  });

  it('has no Delete; Deactivate sits in its place with the tip (US1-8, FR-008)', async () => {
    const { user } = renderPeople();

    const menu = await openActions(user, 'Alex Morgan');

    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((item) => item.firstChild?.textContent),
    ).toEqual(['View', 'Edit', 'Deactivate']);
    expect(within(menu).queryByRole('menuitem', { name: /Delete/ })).not.toBeInTheDocument();
    expect(within(menu).getByRole('menuitem', { name: 'Deactivate' })).toHaveAccessibleDescription(
      'People are never deleted, so the history keeps their name.',
    );
  });
});
