import { configure, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { launchFilmSection } from '@/features/line-items/testData';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { Activity } from './activitiesApi';
import { ActivityFormPage } from './ActivityFormPage';
import { activityAttributeForm, editForm, launchFilm, PEOPLE } from './testData';

// The whole form renders in each test: on a busy computer the first render can take a while.
configure({ asyncUtilTimeout: 10_000 });

const EDIT_PATH = '/plans/:planId/activities/:activityId/edit';
const EDIT_ROUTE = '/plans/P-1001/activities/A-1042/edit';

const CONFLICT = {
  status: 409,
  body: {
    code: 'version_conflict',
    message: 'Someone else changed this activity.',
    fields: [],
    conflict: { changedBy: 'Alex Morgan', changedAt: '2026-03-12T10:42:00Z' },
  },
};

// The edit form's dates say what a change does to the line items' spreads (feature 007's
// SpreadEffectNote), which reads the activity's line items: here it has none.
const NO_LINE_ITEMS = {
  'GET /api/activities/A-1042/line-items': { body: launchFilmSection({ items: [] }) },
};

function openEditForm(
  activity: Activity,
  answers: Record<string, MockHandler> = {},
  route = EDIT_ROUTE,
) {
  const api = mockApi({
    'GET /api/activities/A-1042': { body: activity },
    'GET /api/activities/A-1042/form': { body: editForm() },
    'GET /api/shell-attribute-forms/activity': { body: activityAttributeForm() },
    'GET /api/people': { body: PEOPLE },
    'PATCH /api/activities/A-1042': { body: activity },
    ...NO_LINE_ITEMS,
    ...answers,
  });
  const rendered = renderWithProviders(<ActivityFormPage mode="edit" />, {
    route,
    path: EDIT_PATH,
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('ActivityFormPage, edit (feature 006, US4)', () => {
  it('opens with the activity and saves it with the version it opened with (FR-026)', async () => {
    let reads = 0;
    const { user, api, router, queryClient } = openEditForm(launchFilm(), {
      'GET /api/activities/A-1042': () => {
        reads += 1;
        return { body: launchFilm({ version: reads === 1 ? 3 : 4 }) };
      },
    });

    const description = await screen.findByLabelText('Description');
    expect(
      screen.getByRole('heading', { level: 1, name: "Edit activity 'Launch film'" }),
    ).toBeVisible();
    expect(document.title).toBe('Edit Launch film · NorthStar');
    expect(screen.getByLabelText('Activity name')).toHaveValue('Launch film');
    expect(screen.getByLabelText('Planned cost')).toHaveValue('120,000.00');
    expect(screen.queryByText('From the initiative')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Save and add another' })).toBeNull();
    expect(screen.queryByText('Save the activity to add its line items')).toBeNull();

    // Someone else's change arrives in the background: the form keeps what it opened with.
    await queryClient.invalidateQueries({ queryKey: ['activity'] });
    await waitFor(() => expect(reads).toBe(2));

    await user.type(description, 'Shot in Lisbon');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Activity 'Launch film' saved")).toBeVisible();
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042'),
    );
    expect(api.bodiesOf('PATCH /api/activities/A-1042')).toEqual([
      expect.objectContaining({
        name: 'Launch film',
        description: 'Shot in Lisbon',
        plannedCost: { amount: '120000.00', currency: 'USD' },
        version: 3,
        restartApproval: false,
      }),
    ]);
  });

  it('keeps the typed values after a conflict and saves over it with the newer version (FR-026)', async () => {
    let saves = 0;
    const { user, api } = openEditForm(launchFilm(), {
      'GET /api/activities/A-1042': () => ({ body: launchFilm({ version: saves === 0 ? 3 : 4 }) }),
      'PATCH /api/activities/A-1042': () => {
        saves += 1;
        return saves === 1 ? CONFLICT : { body: launchFilm({ description: 'Mine', version: 5 }) };
      },
    });

    await user.type(await screen.findByLabelText('Description'), 'Mine');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByText(
        'Not saved — Alex Morgan changed this activity on 12 Mar 2026 at 10:42',
      ),
    ).toBeVisible();
    expect(screen.getByLabelText('Description')).toHaveValue('Mine');

    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(await screen.findByText("Activity 'Launch film' saved")).toBeVisible();
    expect(
      api.bodiesOf('PATCH /api/activities/A-1042').map((body) => (body as Activity).version),
    ).toEqual([3, 4]);
  });

  it('asks before a cost change sends a submitted activity back for approval (FR-027)', async () => {
    const { user, api } = openEditForm(
      launchFilm({ status: 'submitted', displayStatus: 'submitted' }),
    );

    const cost = await screen.findByLabelText('Planned cost');
    await user.clear(cost);
    await user.type(cost, '150,000.00');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const question = await screen.findByRole('alertdialog', {
      name: 'This activity will need approval again',
    });
    expect(question).toHaveTextContent(
      "Saving returns 'Launch film' to Created; it must be submitted for approval again.",
    );
    expect(question).toHaveTextContent('Planned cost: 120,000.00 USD → 150,000.00 USD');
    await user.click(within(question).getByRole('button', { name: 'Keep editing' }));
    expect(api.bodiesOf('PATCH /api/activities/A-1042')).toEqual([]);
    expect(screen.getByLabelText('Planned cost')).toHaveValue('150,000.00');

    await user.click(screen.getByRole('button', { name: 'Save' }));
    await user.click(await screen.findByRole('button', { name: 'Save and restart approval' }));

    expect(await screen.findByText("Activity 'Launch film' saved")).toBeVisible();
    expect(api.bodiesOf('PATCH /api/activities/A-1042')).toEqual([
      expect.objectContaining({
        plannedCost: { amount: '150000.00', currency: 'USD' },
        restartApproval: true,
      }),
    ]);
  });

  it('saves other changes of an approved activity without asking (FR-028)', async () => {
    const { user, api } = openEditForm(
      launchFilm({ status: 'approved', displayStatus: 'approved' }),
    );

    await user.type(await screen.findByLabelText('Description'), 'Shot in Lisbon');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Activity 'Launch film' saved")).toBeVisible();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/activities/A-1042')).toEqual([
      expect.objectContaining({ restartApproval: false }),
    ]);
  });

  it('locks a saved Ongoing and the workflow of a submitted activity, saying why (FR-010, FR-012)', async () => {
    openEditForm(
      launchFilm({
        status: 'submitted',
        displayStatus: 'submitted',
        ongoing: true,
        endDate: null,
      }),
    );

    const ongoing = await screen.findByLabelText('Ongoing (no end date)');
    expect(ongoing).toBeDisabled();
    expect(
      screen.getByText(
        "Ongoing can't be undone — to plan with an end date, create a new activity.",
      ),
    ).toBeVisible();
    expect(screen.queryByText("Ongoing can't be undone after saving.")).not.toBeInTheDocument();
    expect(screen.getByLabelText('Approval workflow')).toBeDisabled();
    expect(
      screen.getByText("It can't change while the activity is submitted or approved."),
    ).toBeVisible();
  });

  it('asks whether a new type should bring its default planned cost (FR-007)', async () => {
    const { user } = openEditForm(launchFilm());

    const type = await screen.findByLabelText('Activity type');
    await user.selectOptions(type, 'AT-1002');
    const question = await screen.findByRole('alertdialog', {
      name: "Use the default planned cost of 10,000.00 USD for 'Digital campaign'?",
    });
    await user.click(within(question).getByRole('button', { name: 'Keep 120,000.00 USD' }));
    expect(screen.getByLabelText('Planned cost')).toHaveValue('120,000.00');

    await user.selectOptions(type, 'AT-1001');
    await user.click(await screen.findByRole('button', { name: 'Use default' }));
    expect(screen.getByLabelText('Planned cost')).toHaveValue('25,000.00');
    expect(screen.getByText('From the activity type')).toBeVisible();
  });

  it('focuses the field named in the address (?field=ownerId)', async () => {
    openEditForm(launchFilm(), {}, `${EDIT_ROUTE}?field=ownerId`);

    const owner = await screen.findByLabelText('Owner');
    await waitFor(() => expect(owner).toHaveFocus());
  });

  it('says when the activity no longer exists', async () => {
    openEditForm(launchFilm(), {
      'GET /api/activities/A-1042': {
        status: 404,
        body: { code: 'not_found', message: 'Gone', fields: [] },
      },
    });

    expect(
      await screen.findByText('This activity no longer exists — someone may have deleted it.'),
    ).toBeVisible();
  });
});
