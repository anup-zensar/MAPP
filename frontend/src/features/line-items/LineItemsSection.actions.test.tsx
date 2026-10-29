import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { launchFilmSection } from '@/testing/fixtures/line-items';
import { deleteResult } from '@/testing/fixtures/line-item-forms';
import { activityRecord, lineItemRecord } from '@/testing/fixtures/line-item-financials';
import { renderLineItemsSection } from '@/testing/fixtures/line-item-pages';
import { apiError, reply } from '@/testing/replies';
import { LineItemsSection } from './LineItemsSection';

describe('LineItemsSection, row actions (US1, US4)', () => {
  it('highlights the row just created (FR-010)', async () => {
    renderLineItemsSection({ body: launchFilmSection() }, { highlightLineItemId: 'L-1002' });

    const link = await screen.findByRole('link', { name: 'Film production' });
    expect(link.closest('tr')?.className).toMatch(/highlight/);
    expect(screen.getByRole('link', { name: 'Agency fee' }).closest('tr')?.className).not.toMatch(
      /highlight/,
    );
  });

  it('offers View, Edit, Duplicate and Delete in each row', async () => {
    const { user, router } = renderLineItemsSection({ body: launchFilmSection() });

    await user.click(await screen.findByRole('button', { name: 'Actions for Agency fee' }));
    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['View', 'Edit', 'Duplicate', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'Edit' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1001/edit'));
  });

  it('deletes from the row and offers Undo, staying on the activity (FR-030)', async () => {
    mockApi([
      reply(
        'GET /api/v1/activities/{activityId}/line-items',
        { body: launchFilmSection() },
        { activityId: 'A-1001' },
      ),
      reply(
        'GET /api/v1/financials/records/{recordId}',
        { body: activityRecord() },
        { recordId: 'A-1001' },
      ),
      reply(
        'GET /api/v1/financials/records/{recordId}',
        { body: lineItemRecord() },
        { recordId: 'L-1001' },
      ),
      reply(
        'DELETE /api/v1/line-items/{lineItemId}',
        { body: deleteResult() },
        { lineItemId: 'L-1001' },
      ),
    ]);
    const { user, router } = renderWithProviders(<LineItemsSection activityId="A-1001" />, {
      route: '/plans/P-1001/activities/A-1001',
      user: testUser(),
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Agency fee' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    // The GSAP transactions linked to it stay (015 FR-029).
    expect(
      await within(dialog).findByText(/^2 GSAP transactions are linked to it\./),
    ).toBeVisible();
    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    expect(
      await screen.findByText("Line item 'Agency fee' deleted"),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001');
  });

  it('opens a needed field from the completeness chip (FR-028)', async () => {
    const { user, router } = renderLineItemsSection({ body: launchFilmSection() });

    await user.click(await screen.findByRole('button', { name: '3 fields needed' }));
    await user.click(await screen.findByRole('button', { name: 'Finance budget item' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1001/edit'));
    expect(router.state.location.search).toBe('?field=financeBudgetItemId');
  });
});

describe('032 FR-019: delete refused while a PR draws on it', () => {
  it('shows the refusal naming the purchase request and keeps the row', async () => {
    const refusal =
      "Can't delete — purchase request PR-1002 draws on line item L-1001. Delete or close the purchase request first.";
    mockApi([
      reply(
        'GET /api/v1/activities/{activityId}/line-items',
        { body: launchFilmSection() },
        { activityId: 'A-1001' },
      ),
      reply(
        'GET /api/v1/financials/records/{recordId}',
        { body: activityRecord() },
        { recordId: 'A-1001' },
      ),
      reply(
        'GET /api/v1/financials/records/{recordId}',
        { body: lineItemRecord() },
        { recordId: 'L-1001' },
      ),
      reply(
        'DELETE /api/v1/line-items/{lineItemId}',
        { status: 409, body: apiError('in_use', refusal) },
        { lineItemId: 'L-1001' },
      ),
    ]);
    const { user } = renderWithProviders(<LineItemsSection activityId="A-1001" />, {
      route: '/plans/P-1001/activities/A-1001',
      user: testUser(),
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Agency fee' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(await screen.findByRole('button', { name: 'Delete line item' }));

    expect(await screen.findByText(refusal)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Agency fee' })).toBeVisible();
  });
});
