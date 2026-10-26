import { configure, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { launchFilmSection } from '@/testing/fixtures/line-items';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import type { Activity } from './api';
import { ActivityFormPage } from './ActivityFormPage';
import {
  activityAttributeForm,
  editForm,
  launchFilm,
  newForm,
  PEOPLE,
} from '@/testing/fixtures/activities';

// The whole form renders in each test: on a busy computer the first render can take a while.
configure({ asyncUtilTimeout: 10_000 });

function openNewForm(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    'GET /api/v1/initiatives/I-1001/activity-form': { body: newForm() },
    'GET /api/v1/people': { body: PEOPLE },
    'GET /api/v1/shell-attribute-forms/activity': { body: activityAttributeForm() },
    'POST /api/v1/initiatives/I-1001/activities': { status: 201, body: launchFilm() },
    ...answers,
  });
  const rendered = renderWithProviders(<ActivityFormPage mode="new" />, {
    route: '/plans/P-1001/initiatives/I-1001/activities/new',
    path: '/plans/:planId/initiatives/:initiativeId/activities/new',
    user: testUser(),
  });
  return { ...rendered, api };
}

// The edit form's dates say what a change does to the line items' spreads (feature 007's
// SpreadEffectNote), which reads the activity's line items: here it has none.
const NO_LINE_ITEMS = {
  'GET /api/v1/activities/A-1042/line-items': { body: launchFilmSection({ items: [] }) },
};

function openEditForm(activity: Activity) {
  const api = mockApi({
    'GET /api/v1/activities/A-1042': { body: activity },
    'GET /api/v1/activities/A-1042/form': { body: editForm() },
    'GET /api/v1/people': { body: PEOPLE },
    'GET /api/v1/shell-attribute-forms/activity': { body: activityAttributeForm() },
    'PATCH /api/v1/activities/A-1042': { body: activity },
    ...NO_LINE_ITEMS,
  });
  const rendered = renderWithProviders(<ActivityFormPage mode="edit" />, {
    route: '/plans/P-1001/activities/A-1042/edit',
    path: '/plans/:planId/activities/:activityId/edit',
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('ActivityFormPage, Shell attributes and goals (feature 006, US3)', () => {
  it("shows the attributes every activity needs and the plan's business-unit group, saying why (FR-014)", async () => {
    openNewForm();

    expect(await screen.findByLabelText('Cluster')).toBeVisible();
    expect(screen.getByRole('heading', { level: 2, name: 'Shell attributes' })).toBeVisible();
    expect(screen.getByRole('heading', { level: 2, name: 'Lubricants' })).toBeVisible();
    expect(
      screen.getByText("Shown because the plan's business unit includes Lubricants."),
    ).toBeVisible();
    expect(screen.queryByRole('heading', { name: 'Mobility' })).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Fleet card')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'Goals' })).toBeVisible();
  });

  it('says that more attributes appear once the plan has a business unit, with a link to the plan (FR-014)', async () => {
    openNewForm({
      'GET /api/v1/initiatives/I-1001/activity-form': {
        body: newForm({
          context: { ...newForm().context, businessUnits: [] },
        }),
      },
    });

    expect(
      await screen.findByText(/More Shell attributes appear once the plan's business unit/),
    ).toBeVisible();
    expect(screen.getByRole('link', { name: 'Open the plan' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(screen.queryByLabelText('Shell Helix')).not.toBeInTheDocument();
  });

  it('splits Country / OU evenly, shows a total over 100 % and still saves (FR-016, FR-017)', async () => {
    const { user, api } = openNewForm();

    const split = (await screen.findAllByRole('group', { name: /Country \/ OU/ }))[0];
    for (const country of ['Germany', 'France', 'Netherlands']) {
      await user.click(within(split).getByRole('checkbox', { name: country }));
    }
    expect(within(split).getByLabelText('Share of Germany')).toHaveValue('33.33');
    expect(within(split).getByLabelText('Share of France')).toHaveValue('33.33');
    expect(within(split).getByLabelText('Share of Netherlands')).toHaveValue('33.34');
    expect(within(split).getByText('100.00 % — complete')).toBeVisible();

    const germany = within(split).getByLabelText('Share of Germany');
    await user.clear(germany);
    await user.type(germany, '50.00');
    expect(within(split).getByText('116.67 % — 16.67 % too much')).toBeVisible();

    await user.type(screen.getByLabelText('Activity name'), 'Trade show');
    await user.selectOptions(screen.getByLabelText('Activity type'), 'AT-1003');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Activity 'Launch film' created")).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/activities')).toEqual([
      expect.objectContaining({
        attributes: {
          'SA-1013': [
            { optionId: 'SO-DE', share: '50.00' },
            { optionId: 'SO-FR', share: '33.33' },
            { optionId: 'SO-NL', share: '33.34' },
          ],
        },
      }),
    ]);
  });

  it('locks a dependent attribute until its parent is chosen (FR-016)', async () => {
    openNewForm();

    expect(await screen.findByLabelText('Region')).toBeDisabled();
    expect(screen.getByText('Choose a cluster first.')).toBeVisible();
  });

  it('lists the dependent values a save would clear, then saves (FR-016)', async () => {
    const { user, api } = openEditForm(
      launchFilm({ attributes: { 'SA-CLUSTER': 'SO-RETAIL', 'SA-REGION': 'SO-CHINA' } }),
    );

    await user.selectOptions(await screen.findByLabelText('Cluster'), 'B2B');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const question = await screen.findByRole('alertdialog', {
      name: 'Clear values that no longer fit?',
    });
    expect(question).toHaveTextContent('Region: RETAIL – CHINA');
    expect(api.bodiesOf('PATCH /api/v1/activities/A-1042')).toEqual([]);
    await user.click(within(question).getByRole('button', { name: 'Clear and save' }));

    expect(await screen.findByText("Activity 'Launch film' saved")).toBeVisible();
    expect(api.bodiesOf('PATCH /api/v1/activities/A-1042')).toEqual([
      expect.objectContaining({
        attributes: { 'SA-CLUSTER': 'SO-B2B', 'SA-REGION': 'SO-CHINA' },
      }),
    ]);
  });

  it('sends the values of hidden groups back unchanged (FR-015)', async () => {
    const { user, api } = openEditForm(launchFilm({ attributes: { 'SA-FLEET': 'FC-2207' } }));

    await user.type(await screen.findByLabelText('Description'), 'Shot in Lisbon');
    expect(screen.queryByLabelText('Fleet card')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Activity 'Launch film' saved")).toBeVisible();
    expect(api.bodiesOf('PATCH /api/v1/activities/A-1042')).toEqual([
      expect.objectContaining({ attributes: { 'SA-FLEET': 'FC-2207' } }),
    ]);
  });

  it('sends INQ and keeps MQL and revenue to the model unless typed (FR-018)', async () => {
    const { user, api } = openNewForm();

    await user.type(await screen.findByLabelText('Activity name'), 'Trade show');
    await user.selectOptions(screen.getByLabelText('Activity type'), 'AT-1003');
    await user.type(screen.getByLabelText('INQ'), '250,000');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Activity 'Launch film' created")).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/activities')).toEqual([
      expect.objectContaining({
        goals: { inq: 250000, setLaterStagesMyself: false, mql: null, revenue: null },
      }),
    ]);
  });
});
