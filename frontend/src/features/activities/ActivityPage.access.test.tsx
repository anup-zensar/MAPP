import { configure, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { approvalOf } from '@/testing/fixtures/approvals';
import { activityAttributeForm, editForm } from '@/testing/fixtures/activity-forms';
import { launchFilm } from '@/testing/fixtures/activities';
import { activityRecord } from '@/testing/fixtures/line-item-financials';
import { currentRates } from '@/testing/fixtures/line-item-forms';
import { activity, launchFilmSection } from '@/testing/fixtures/line-items';
import { plan } from '@/testing/fixtures/plans';
import { activitySection } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import {
  ALLOWED,
  refused,
  VIEW_ONLY_NOTE,
  VIEWER,
  VIEWER_ACTIVITY_ACCESS,
  VIEWER_LINE_ITEM_ACCESS,
} from '@/testing/fixtures/viewer';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { Activity } from './api';
import { ActivityFormPage } from './ActivityFormPage';
import { ActivityPage } from './ActivityPage';

configure({ asyncUtilTimeout: 10_000 });

const FORM_REFUSED = apiError(
  'forbidden',
  "Changing activities and line items isn't part of your role (Viewer). Ask an administrator if you need it.",
);

function openPage(access: Activity['access'], person: CurrentUser, formAllowed: boolean) {
  const section = launchFilmSection({ activity: activity({ id: 'A-1042' }) });
  mockApi([
    reply('GET /api/v1/activities/{activityId}', { body: launchFilm({ access }) }),
    reply('GET /api/v1/activities/{activityId}/approval', { body: approvalOf() }),
    reply('GET /api/v1/exchange-rates/current', { body: currentRates() }),
    reply('GET /api/v1/activities/{activityId}/line-items', {
      body: formAllowed ? section : { ...section, access: VIEWER_LINE_ITEM_ACCESS },
    }),
    reply('GET /api/v1/financials/records/{recordId}', {
      body: activityRecord({ record: { type: 'activity', id: 'A-1042', name: 'Launch film' } }),
    }),
    reply(
      'GET /api/v1/activities/{activityId}/form',
      formAllowed ? { body: editForm() } : { status: 403, body: FORM_REFUSED },
    ),
    reply('GET /api/v1/plans/{planId}', { body: plan() }),
    reply('GET /api/v1/shell-attribute-forms/{level}', { body: activityAttributeForm() }),
    reply('GET /api/v1/activities/{activityId}/purchase-requests', {
      body: activitySection({ activityId: 'A-1042', lineItems: [] }),
    }),
  ]);
  return renderWithProviders(<ActivityPage />, {
    route: '/plans/P-1001/activities/A-1042',
    path: '/plans/:planId/activities/:activityId',
    user: person,
  });
}

describe('014 FR-011: the activity page follows the reader’s access', () => {
  it('shows a Viewer one View only note and no change action', async () => {
    const { user } = openPage(VIEWER_ACTIVITY_ACCESS, VIEWER, false);

    await screen.findByRole('heading', { level: 1, name: 'Launch film' });
    expect(screen.getByRole('note')).toHaveTextContent(VIEW_ONLY_NOTE);
    expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Submit for approval' })).not.toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Ready to submit' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual([
      'View history',
    ]);
    await user.keyboard('{Escape}');
    await expectNoAxeViolations();
  });

  it('shows a Viewer the Shell attributes and goals, which do not need the edit form (bug)', async () => {
    const { user } = openPage(VIEWER_ACTIVITY_ACCESS, VIEWER, false);
    await screen.findByRole('heading', { level: 1, name: 'Launch film' });

    await user.click(screen.getByRole('tab', { name: 'Details' }));

    const attributes = await screen.findByRole('region', { name: 'Shell attributes' });
    expect(await within(attributes).findAllByRole('heading')).not.toHaveLength(0);
    expect(within(attributes).queryByRole('alert')).not.toBeInTheDocument();
  });

  it('lists Submit for approval in More, disabled with its reason, for someone who may edit but not submit', async () => {
    const access = {
      ...VIEWER_ACTIVITY_ACCESS,
      edit: ALLOWED,
      submit: refused('Submitting activities', 'Editor'),
    };
    const { user } = openPage(access, testUser(), true);
    await screen.findByRole('heading', { level: 1, name: 'Launch film' });

    expect(screen.getByRole('link', { name: 'Edit' })).toBeVisible();
    expect(screen.queryByRole('note')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    expect(
      screen.getByRole('menuitem', { name: 'Submit for approval' }),
    ).toHaveAccessibleDescription(
      "Submitting activities isn't part of your role (Editor). Ask an administrator if you need it.",
    );
  });

  it('shows the no-access page at the edit address when the API refuses the form (FR-008)', async () => {
    mockApi([
      reply('GET /api/v1/activities/{activityId}', {
        body: launchFilm({ access: VIEWER_ACTIVITY_ACCESS }),
      }),
      reply('GET /api/v1/activities/{activityId}/form', { status: 403, body: FORM_REFUSED }),
      reply('GET /api/v1/me', { body: VIEWER }),
    ]);
    renderWithProviders(<ActivityFormPage mode="edit" />, {
      route: '/plans/P-1001/activities/A-1042/edit',
      path: '/plans/:planId/activities/:activityId/edit',
      user: VIEWER,
    });

    expect(await screen.findByRole('heading', { level: 1, name: 'No access' })).toBeVisible();
    expect(screen.getByText(FORM_REFUSED.error.message)).toBeVisible();
  });
});
