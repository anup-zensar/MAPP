import { configure, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { launchFilm, PEOPLE } from '@/testing/fixtures/activities';
import { activityAttributeForm, newForm } from '@/testing/fixtures/activity-forms';
import { fieldHelpReply, helpOf } from '@/testing/fixtures/field-help-texts';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';
import { ActivityFigures } from './ActivityFigures';
import { ActivityFormPage } from './ActivityFormPage';
import { ActivityHeader } from './ActivityHeader';
import { ActivityDetails, ActivityGoals } from './ActivitySections';
import { SplitLinks } from './SplitLinks';

// The whole form renders: on a busy computer the first render can take a while.
configure({ asyncUtilTimeout: 10_000 });

const FORM: [string, string][] = [
  ['Activity name', 'activity.name'],
  ['External name', 'activity.externalName'],
  ['Campaign, Initiative', 'activity.initiative'],
  ['Activity type', 'activity.activityType'],
  ['Owner', 'activity.owner'],
  ['Approval workflow', 'activity.approvalWorkflow'],
  ['Start and end dates', 'activity.dates'],
  ['Ongoing (no end date)', 'activity.ongoing'],
  ['Planned cost', 'activity.plannedCost'],
  ['INQ', 'activity.inq'],
  ['MQL', 'activity.mql'],
  ['Revenue', 'activity.revenue'],
  ['Set later stages myself', 'activity.setLaterStagesMyself'],
  ['Description', 'activity.description'],
];

const PAGE: [string, string][] = [
  ['Activity name', 'activity.name'],
  ['External name', 'activity.externalName'],
  ['Campaign', 'activity.initiative'],
  ['Activity type', 'activity.activityType'],
  ['Owner', 'activity.owner'],
  ['Start date', 'activity.dates'],
  ['Approval workflow', 'activity.approvalWorkflow'],
  ['Description', 'activity.description'],
  ['INQ', 'activity.inq'],
  ['Unassigned planned cost', 'activity.unassignedPlannedCost'],
  ['Status', 'activity.status'],
  ['Completeness', 'common.completeness'],
  ['Split from / Split into', 'activity.splitFrom'],
];

const keysOf = (rows: [string, string][]) => rows.map(([, key]) => key);

async function expectOneTipEach(rows: [string, string][]) {
  for (const [label] of rows) {
    const tips = await screen.findAllByRole('button', { name: `About ${label}` });
    expect(tips, label).toHaveLength(1);
  }
}

describe('016 FR-047: the activity screens show the Field help of every labelled field', () => {
  it('gives each field of the activity form one ⓘ, and none for an emptied text', async () => {
    mockApi([
      reply(
        'GET /api/v1/initiatives/{initiativeId}/activity-form',
        { body: newForm() },
        { initiativeId: 'I-1001' },
      ),
      reply(
        'GET /api/v1/shell-attribute-forms/{level}',
        { body: activityAttributeForm() },
        { level: 'activity' },
      ),
      reply('GET /api/v1/people', { body: PEOPLE }),
      fieldHelpReply({ ...helpOf(keysOf(FORM)), 'activity.externalName': '' }),
    ]);
    renderWithProviders(<ActivityFormPage mode="new" />, {
      route: '/plans/P-1001/initiatives/I-1001/activities/new',
      path: '/plans/:planId/initiatives/:initiativeId/activities/new',
      user: testUser(),
    });

    await expectOneTipEach(FORM.filter(([label]) => label !== 'External name'));
    expect(screen.queryByRole('button', { name: 'About External name' })).toBeNull();
  });

  it('gives each value of the activity page one ⓘ', async () => {
    mockApi([fieldHelpReply(helpOf([...keysOf(PAGE), 'activity.plannedCost']))]);
    const activity = launchFilm({
      splitFrom: { id: 'A-1012', name: 'UK launch event', planId: 'P-1001', deleted: false },
    });
    renderWithProviders(
      <>
        <ActivityHeader
          activity={activity}
          approval={undefined}
          moreActions={[]}
          onOpenReady={() => {}}
          onShowProblems={() => {}}
        />
        <SplitLinks activity={activity} />
        <ActivityFigures activity={activity} />
        <ActivityDetails activity={activity} />
        <ActivityGoals activity={activity} />
      </>,
      { user: testUser() },
    );

    await expectOneTipEach(PAGE);
    const details = screen.getByRole('region', { name: 'Details' });
    expect(within(details).getByRole('button', { name: 'About Planned cost' })).toBeVisible();
  });
});
