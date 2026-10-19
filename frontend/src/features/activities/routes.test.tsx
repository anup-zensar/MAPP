import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';

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
vi.mock('@/features/approvals/ApprovalSection', () => ({
  ApprovalSection: ({ activity }: { activity: { id: string } }) => (
    <section aria-label="Approval">{`Approval of ${activity.id}`}</section>
  ),
}));
vi.mock('@/features/approvals/ApprovalStatusAction', () => ({
  ApprovalStatusAction: () => <button type="button">Submit for approval</button>,
}));

// Feature 006's addresses (build guide §5, D-194). Feature 006 keeps this file up to date as it
// replaces the stub pages (their own requests go into renderApp's answers).
describe('Activity routes (feature 006)', () => {
  it.each([
    ['/plans/P-1001/initiatives/I-1001/activities/new', 'New activity'],
    ['/plans/P-1001/activities/A-1001', 'Activity'],
    ['/plans/P-1001/activities/A-1001/edit', 'Edit activity'],
  ])('opens %s: "%s", with Plans current in the sidebar', async (path, title) => {
    renderApp(path);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
    expectCurrentSection('Plans');
  });

  it('opens the short address /activities/:activityId (D-194)', async () => {
    renderApp('/activities/A-1001');

    expect(await screen.findByRole('heading', { level: 1, name: 'Activity' })).toBeInTheDocument();
  });

  it('places the line items (007) and the approval (008) on the activity page', async () => {
    renderApp('/plans/P-1001/activities/A-1001');

    const lineItems = await screen.findByRole('region', { name: 'Line items' });
    expect(lineItems).toHaveTextContent('Line items of A-1001');
    expect(lineItems.closest('#line-items')).not.toBeNull();
    const approval = screen.getByRole('region', { name: 'Approval' });
    expect(approval).toHaveTextContent('Approval of A-1001');
    expect(approval.closest('#approval')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Submit for approval' })).toBeInTheDocument();
  });

  it('places the spread effect note (007) on the edit form', async () => {
    renderApp('/plans/P-1001/activities/A-1001/edit');

    expect(await screen.findByText('Spread effect for A-1001')).toBeInTheDocument();
  });
});
