import { configure, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ActivityFormPage } from './ActivityFormPage';
import {
  activityAttributeForm,
  launchFilm,
  newForm,
  PEOPLE,
  validationFailed,
} from '@/testing/fixtures/activities';

// The whole form renders in each test: on a busy computer the first render can take a while.
configure({ asyncUtilTimeout: 10_000 });

const NEW_PATH = '/plans/:planId/initiatives/:initiativeId/activities/new';
const NEW_ROUTE = '/plans/P-1001/initiatives/I-1001/activities/new';

function openNewForm(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    'GET /api/v1/initiatives/I-1001/activity-form': { body: newForm() },
    'GET /api/v1/shell-attribute-forms/activity': { body: activityAttributeForm() },
    'GET /api/v1/people': { body: PEOPLE },
    ...answers,
  });
  const rendered = renderWithProviders(<ActivityFormPage mode="new" />, {
    route: NEW_ROUTE,
    path: NEW_PATH,
    user: testUser(),
  });
  return { ...rendered, api };
}

async function nameField() {
  return screen.findByLabelText('Activity name');
}

describe('ActivityFormPage, new activity (feature 006, US1)', () => {
  it('shows the fields with their markings as words and the defaults labelled (FR-001, FR-002, FR-008, FR-009)', async () => {
    openNewForm();

    expect(await nameField()).toHaveAccessibleDescription('Required');
    expect(screen.getByRole('heading', { level: 1, name: 'New activity' })).toBeVisible();
    // The breadcrumb says where it lives; the count line says how far the form is (P-04).
    expect(screen.getByText(/of \d+ needed fields filled$/)).toBeVisible();
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(within(breadcrumb).getByRole('link', { name: 'Digital & social' })).toHaveAttribute(
      'href',
      '/plans/P-1001/initiatives/I-1001',
    );
    for (const section of ['Basics', 'Dates & cost', 'More details']) {
      expect(screen.getByRole('heading', { level: 2, name: section })).toBeVisible();
    }
    expect(screen.getByLabelText('Activity type')).toHaveAccessibleDescription('Required');
    const campaign = screen.getByText('Campaign');
    expect(campaign.nextElementSibling).toHaveTextContent('Helix launch');
    expect(screen.getByText('Initiative').nextElementSibling).toHaveTextContent('Digital & social');
    expect(screen.getByText('Change it later with Move')).toBeVisible();
    expect(await screen.findByLabelText('Owner')).toHaveAccessibleDescription(
      'Needed to submit From the initiative',
    );
    expect(screen.getByLabelText('Start date')).toHaveValue('2026-01-01');
    expect(screen.getByLabelText('End date')).toHaveValue('2026-06-30');
    expect(screen.getAllByText('From the initiative')).toHaveLength(2);
    expect(
      screen.getByText("Allowed: 01 Jan 2026 – 30 Jun 2026 (the initiative's dates)"),
    ).toBeVisible();
    expect(screen.getByLabelText('Approval workflow')).toHaveValue('WF-1001');
    expect(screen.getByText('The only active approval workflow')).toBeVisible();
    expect(screen.getByText('Save the activity to add its line items')).toBeVisible();
  });

  it('offers the allowed types with their default costs, and Event fills its 25,000.00 (FR-007)', async () => {
    const { user } = openNewForm();
    const type = await screen.findByLabelText('Activity type');

    expect(
      within(type)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual([
      'Choose…',
      'Digital campaign · default 10,000.00 USD',
      'Event · default 25,000.00 USD',
      'Sponsorship',
    ]);
    await user.selectOptions(type, 'Event · default 25,000.00 USD');

    expect(screen.getByLabelText('Planned cost')).toHaveValue('25,000.00');
    expect(screen.getByText('From the activity type')).toBeVisible();

    // Another type replaces a default it filled in itself, without asking.
    await user.selectOptions(type, 'Digital campaign · default 10,000.00 USD');
    expect(screen.getByLabelText('Planned cost')).toHaveValue('10,000.00');
  });

  it('creates the activity, says so and opens its page (FR-003, FR-004)', async () => {
    const { user, api, router } = openNewForm({
      'POST /api/v1/initiatives/I-1001/activities': {
        status: 201,
        body: launchFilm({ id: 'A-1043', name: 'Trade show' }),
      },
    });

    await user.type(await nameField(), 'Trade show');
    await user.selectOptions(screen.getByLabelText('Activity type'), 'AT-1001');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Activity 'Trade show' created")).toBeVisible();
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1043'),
    );
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/activities')).toEqual([
      {
        name: 'Trade show',
        externalName: '',
        activityTypeId: 'AT-1001',
        ownerId: 'U-1002',
        startDate: '2026-01-01',
        endDate: '2026-06-30',
        ongoing: false,
        plannedCost: { amount: '25000.00', currency: 'USD' },
        approvalWorkflowId: 'WF-1001',
        description: '',
        attributes: {},
        goals: { inq: null, setLaterStagesMyself: false, mql: null, revenue: null },
      },
    ]);
  });

  it('shows every problem at once in the summary and under the fields, and sends nothing (FR-006, FR-009, FR-011)', async () => {
    const { user, api } = openNewForm();

    await user.type(await nameField(), 'launch film ');
    fireEvent.change(screen.getByLabelText('Start date'), { target: { value: '2026-05-31' } });
    fireEvent.change(screen.getByLabelText('End date'), { target: { value: '2026-03-01' } });
    await user.type(screen.getByLabelText('Planned cost'), '-5');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const summary = await screen.findByRole('alert');
    expect(summary).toHaveTextContent('4 problems need your attention');
    for (const message of [
      "'Digital & social' already has an activity called 'Launch film'.",
      'Choose an activity type.',
      'End date must be on or after the start date (31 May 2026).',
      "Planned cost can't be negative.",
    ]) {
      expect(within(summary).getByRole('link', { name: message })).toBeVisible();
    }
    expect(screen.getByLabelText('Activity name')).toHaveAccessibleDescription(
      "Required 'Digital & social' already has an activity called 'Launch film'.",
    );
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/activities')).toEqual([]);

    fireEvent.change(screen.getByLabelText('Start date'), { target: { value: '2025-12-01' } });
    fireEvent.change(screen.getByLabelText('End date'), { target: { value: '2026-07-31' } });
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(
      await screen.findAllByText(
        "Dates must be inside the initiative's dates (01 Jan 2026 – 30 Jun 2026).",
      ),
    ).not.toHaveLength(0);
  });

  it('shows the problems the API finds under the fields', async () => {
    const { user } = openNewForm({
      'POST /api/v1/initiatives/I-1001/activities': validationFailed([
        { field: 'ownerId', message: 'Choose an active person.' },
      ]),
    });

    await user.type(await nameField(), 'Trade show');
    await user.selectOptions(screen.getByLabelText('Activity type'), 'AT-1003');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const summary = await screen.findByRole('alert');
    expect(within(summary).getByRole('link', { name: 'Choose an active person.' })).toHaveAttribute(
      'href',
      '#field-ownerId',
    );
  });

  it('clears the end date when Ongoing is ticked, warns, and can be unticked (FR-010)', async () => {
    const { user } = openNewForm();
    const ongoing = await screen.findByLabelText('Ongoing (no end date)');

    await user.click(ongoing);
    expect(screen.getByLabelText('End date')).toHaveValue('');
    expect(screen.getByLabelText('End date')).toBeDisabled();
    expect(screen.getByText("Ongoing can't be undone after saving.")).toBeVisible();

    await user.click(ongoing);
    expect(screen.getByLabelText('End date')).toBeEnabled();
    expect(screen.queryByText("Ongoing can't be undone after saving.")).not.toBeInTheDocument();
  });

  it('opens a fresh form after Save and add another, with View in the message', async () => {
    const { user, router } = openNewForm({
      'POST /api/v1/initiatives/I-1001/activities': {
        status: 201,
        body: launchFilm({ id: 'A-1043', name: 'Trade show' }),
      },
    });

    await user.type(await nameField(), 'Trade show');
    await user.selectOptions(screen.getByLabelText('Activity type'), 'AT-1003');
    await user.click(screen.getByRole('button', { name: 'Save and add another' }));

    expect(await screen.findByText("Activity 'Trade show' created")).toBeVisible();
    await waitFor(() => expect(screen.getByLabelText('Activity name')).toHaveValue(''));
    expect(router.state.location.pathname).toBe(NEW_ROUTE);

    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1043'),
    );
  });

  it('asks before Cancel throws typed values away', async () => {
    const { user, router } = openNewForm();

    await user.type(await nameField(), 'Trade show');
    await user.click(screen.getByRole('link', { name: 'Cancel' }));

    const question = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(question).getByRole('button', { name: 'Keep editing' }));
    expect(screen.getByLabelText('Activity name')).toHaveValue('Trade show');

    await user.click(screen.getByRole('link', { name: 'Cancel' }));
    await user.click(await screen.findByRole('button', { name: 'Discard changes' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });

  it('says when the model has no activity types (edge case)', async () => {
    openNewForm({
      'GET /api/v1/initiatives/I-1001/activity-form': { body: newForm({ activityTypes: [] }) },
    });

    const notice = await screen.findByRole('note');
    expect(notice).toHaveTextContent(
      "The model 'Lubricants 1.0' has no activity types yet — an administrator adds them in Settings. Open Settings",
    );
    expect(within(notice).getByRole('link', { name: 'Open Settings' })).toHaveAttribute(
      'href',
      '/settings/activity-types',
    );
  });

  it('says when the initiative no longer exists', async () => {
    openNewForm({
      'GET /api/v1/initiatives/I-1001/activity-form': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This initiative no longer exists — someone may have deleted it.',
            fields: [],
          },
        },
      },
    });

    expect(
      await screen.findByText('This initiative no longer exists — someone may have deleted it.'),
    ).toBeVisible();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });

  it('offers Try again when the form cannot be loaded', async () => {
    let calls = 0;
    openNewForm({
      'GET /api/v1/initiatives/I-1001/activity-form': () => {
        calls += 1;
        return calls === 1
          ? {
              status: 500,
              body: { error: { code: 'internal_error', message: 'Broken', fields: [] } },
            }
          : { body: newForm() };
      },
    });

    const retry = await screen.findByRole('button', { name: 'Try again' });
    fireEvent.click(retry);

    expect(await nameField()).toBeVisible();
  });
});
