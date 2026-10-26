import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { approvalOf } from '@/testing/fixtures/approvals';
import { activityRecord } from '@/testing/fixtures/line-items';
import { expectCurrentSection, renderApp } from '@/testing/app';
import {
  activityAttributeForm,
  editForm,
  launchFilm,
  newForm,
  PEOPLE,
} from '@/testing/fixtures/activities';
import { reply } from '@/testing/replies';

// The parts features 007 and 008 show on the activity pages are tested by those features; here
// they stand in, so those features never break these tests.
vi.mock('@/features/line-items/LineItemsSection', () => ({
  LineItemsSection: ({ activityId }: { activityId: string }) => (
    <section aria-label="Line items">{`Line items of ${activityId}`}</section>
  ),
}));
vi.mock('@/features/line-items/SpreadEffectNote', () => ({
  SpreadEffectNote: ({ activityId }: { activityId: string }) => (
    <p>{`Spread effect for ${activityId}`}</p>
  ),
}));
// The re-approval question's spread sentence (007, D-150), which reads the activity's line items.
vi.mock('@/features/line-items/useSpreadEffectText', () => ({
  useSpreadEffectText: () => null,
}));
vi.mock('@/features/approvals/ApprovalSection', () => ({
  ApprovalSection: ({ activity }: { activity: { id: string } }) => (
    <section aria-label="Approval">{`Approval of ${activity.id}`}</section>
  ),
}));
vi.mock('@/features/approvals/SubmitButton', () => ({
  SubmitButton: () => <button type="button">Submit for approval</button>,
}));

/** The requests of each page. */
const ANSWERS: HttpHandler[] = [
  ...HISTORY_ANSWERS,
  reply(
    'GET /api/v1/initiatives/{initiativeId}/activity-form',
    { body: newForm() },
    { initiativeId: 'I-1001' },
  ),
  reply('GET /api/v1/people', { body: PEOPLE }),
  reply(
    'GET /api/v1/activities/{activityId}',
    { body: launchFilm({ id: 'A-1001' }) },
    { activityId: 'A-1001' },
  ),
  reply('GET /api/v1/activities/{activityId}/form', { body: editForm() }, { activityId: 'A-1001' }),
  reply(
    'GET /api/v1/activities/{activityId}/approval',
    {
      body: approvalOf({
        activityId: 'A-1001',
        status: 'created',
        displayStatus: 'created',
        rounds: [],
      }),
    },
    { activityId: 'A-1001' },
  ),
  reply(
    'GET /api/v1/shell-attribute-forms/{level}',
    { body: activityAttributeForm() },
    { level: 'activity' },
  ),
  reply(
    'GET /api/v1/financials/records/{recordId}',
    { body: activityRecord() },
    { recordId: 'A-1001' },
  ),
];

// Feature 006's addresses (build guide §5, D-194). Feature 006 keeps this file up to date as it
// replaces the stub pages (their own requests go into renderApp's answers).
describe('Activity routes (feature 006)', () => {
  it.each([
    ['/plans/P-1001/initiatives/I-1001/activities/new', 'New activity', 'New activity'],
    ['/plans/P-1001/activities/A-1001', 'Launch film', 'Launch film'],
    ['/plans/P-1001/activities/A-1001/edit', 'Edit activity', 'Edit Launch film'],
  ])('opens %s: "%s", with Plans current in the sidebar', async (path, heading, tab) => {
    renderApp(path, ANSWERS);

    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument();
    await waitFor(() => expect(document.title).toBe(`${tab} · MAPP`));
    expectCurrentSection('Plans');
  });

  it('opens the short address /activities/:activityId at the page under its plan, keeping the #part (D-194)', async () => {
    const { router } = renderApp('/activities/A-1001#line-items', ANSWERS);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Launch film' }),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001');
    expect(router.state.location.hash).toBe('#line-items');
  });

  it('places the line items (007) and the approval (008) on the activity page', async () => {
    const { user } = renderApp('/plans/P-1001/activities/A-1001', ANSWERS);

    const lineItems = await screen.findByRole('region', { name: 'Line items' });
    expect(lineItems).toHaveTextContent('Line items of A-1001');
    expect(lineItems.closest('#line-items')).not.toBeNull();
    await user.click(screen.getByRole('tab', { name: 'Approval' }));
    const approval = await screen.findByRole('region', { name: 'Approval' });
    expect(approval).toHaveTextContent('Approval of A-1001');
    expect(approval.closest('#approval')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
  });

  it('places the spread effect note (007) on the edit form', async () => {
    renderApp('/plans/P-1001/activities/A-1001/edit', ANSWERS);

    expect(await screen.findByText('Spread effect for A-1001')).toBeInTheDocument();
  });
});
