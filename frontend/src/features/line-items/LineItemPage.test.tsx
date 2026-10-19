import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { LineItemPage } from './LineItemPage';
import {
  approvedActivity,
  deleteResult,
  details,
  filmProduction,
  formAnswers,
  lineItem,
  notFound,
} from './testData';

const GET = 'GET /api/line-items/L-1001';
const DUPLICATE = 'POST /api/line-items/L-1001/duplicate';
const DELETE = 'DELETE /api/line-items/L-1001';

function renderPage(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({ ...formAnswers(), [GET]: { body: details() }, ...answers });
  return {
    api,
    ...renderWithProviders(<LineItemPage />, {
      route: '/line-items/L-1001',
      path: '/line-items/:lineItemId',
      user: testUser(),
    }),
  };
}

async function openMore(user: ReturnType<typeof renderPage>['user'], item: string) {
  await user.click(screen.getByRole('button', { name: 'More' }));
  await user.click(await screen.findByRole('menuitem', { name: item }));
}

describe('LineItemPage (US4, FR-026)', () => {
  it('shows where the line item lives, its chips and facts (scenario 1)', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Agency fee' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Agency fee · NorthStar');
    const breadcrumb = screen.getByLabelText('Breadcrumb');
    expect(within(breadcrumb).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1001',
    );
    expect(within(breadcrumb).getByText('Agency fee')).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText('LINE ITEM')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '3 fields needed' })).toBeInTheDocument();
    expect(screen.getByText('Waiting for activity approval')).toBeInTheDocument();
    expect(screen.getByText('L-1001')).toBeInTheDocument();
    expect(screen.getAllByText('Priya Raman').length).toBeGreaterThan(0);
    expect(screen.getAllByText('50,000.00 USD').length).toBeGreaterThan(0);
    expect(
      screen.getByText(
        'Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/line-items/L-1001/edit',
    );
    expect(screen.getByRole('button', { name: 'More' })).toBeInTheDocument();
    expect(screen.getByText('Mar 2026 · 16,666.66 USD')).toBeInTheDocument();
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    expect(screen.getByText('Filled by GSAP later')).toBeInTheDocument();
    expect(screen.getByText('From finance (coming later)').closest('details')).not.toHaveAttribute(
      'open',
    );
  });

  it('shows a planned cost in another currency with its USD amount', async () => {
    renderPage({ [GET]: { body: details({ lineItem: { ...filmProduction(), id: 'L-1001' } }) } });

    expect(
      (await screen.findAllByText('46,000.00 EUR = 50,000.00 USD at 0.92 EUR per USD')).length,
    ).toBeGreaterThan(0);
  });

  it('offers Duplicate, View history and Delete under More', async () => {
    const { user, router } = renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await user.click(screen.getByRole('button', { name: 'More' }));
    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['Duplicate', 'View history', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'View history' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/activity-log'));
    expect(router.state.location.search).toBe('?recordType=lineItem&recordId=L-1001');
  });

  it('duplicates and opens the copy (scenario 4, FR-029)', async () => {
    const { user, api, router } = renderPage({
      [DUPLICATE]: { status: 201, body: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) },
      'GET /api/line-items/L-2002': {
        body: details({ lineItem: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) }),
      },
    });
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Duplicate');

    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-2002'));
    expect(api.bodiesOf(DUPLICATE)).toEqual([{ restartApproval: false }]);
    expect(await screen.findByText("Line item 'Agency fee (copy)' created")).toBeInTheDocument();
  });

  it('asks before duplicating in an approved activity (FR-031)', async () => {
    const { user, api } = renderPage({
      [GET]: { body: details({ activity: approvedActivity() }) },
      [DUPLICATE]: { status: 201, body: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) },
      'GET /api/line-items/L-2002': {
        body: details({ lineItem: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) }),
      },
    });
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Duplicate');
    const dialog = await screen.findByRole('alertdialog', {
      name: 'This activity will need approval again',
    });
    expect(dialog).toHaveTextContent("Copy of 'Agency fee': 50,000.00 USD");
    expect(api.bodiesOf(DUPLICATE)).toHaveLength(0);

    await user.click(within(dialog).getByRole('button', { name: 'Save and restart approval' }));
    await waitFor(() => expect(api.bodiesOf(DUPLICATE)).toEqual([{ restartApproval: true }]));
  });

  it('deletes after the dialog and offers Undo (scenario 5, FR-030)', async () => {
    const { user, api, router } = renderPage({
      [DELETE]: { body: deleteResult() },
      'POST /api/recycle-bin/BIN-1001/undo': {
        body: {
          recordType: 'lineItem',
          recordTypeLabel: 'line item',
          recordId: 'L-1001',
          recordName: 'Agency fee',
          details: '',
        },
      },
    });
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    expect(dialog).toHaveTextContent(
      "The activity's unassigned planned cost goes up by 50,000.00 USD.",
    );
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    expect(dialog).not.toHaveTextContent('This activity will need approval again.');
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();

    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001'),
    );
    expect(router.state.location.hash).toBe('#line-items');
    expect(api.requests.find((r) => r.method === 'DELETE')?.search).toBe('');
    expect(
      await screen.findByText("Line item 'Agency fee' moved to the recycle bin"),
    ).toBeInTheDocument();
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(await screen.findByText("Line item 'Agency fee' restored")).toBeInTheDocument();
  });

  it('says that deleting restarts the approval of an approved activity (FR-031)', async () => {
    const { user, api } = renderPage({
      [GET]: { body: details({ activity: approvedActivity() }) },
      [DELETE]: { body: deleteResult() },
    });
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    expect(dialog).toHaveTextContent('This activity will need approval again.');
    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    await waitFor(() =>
      expect(api.requests.find((r) => r.method === 'DELETE')?.search).toBe('?restartApproval=true'),
    );
  });

  it.each([
    ['This line item no longer exists — someone may have deleted it.', 'Go to Plans', '/plans'],
    [
      'This activity was deleted — restore it from the recycle bin to keep working on its line items.',
      'Go to the recycle bin',
      '/recycle-bin',
    ],
  ])('says when it is gone: %s (D-154)', async (message, link, href) => {
    renderPage({ [GET]: notFound(message) });

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: link })).toHaveAttribute('href', href);
  });
});
