import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { importDetail, importSummary } from '@/testing/fixtures/workbook';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import { ImportResultPage } from './ImportResultPage';
import { problemsCsv } from './files';

function renderPage(answers: HttpHandler[]) {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<ImportResultPage />, {
    user: testUser(),
    route: '/imports/IMP-1001',
    path: '/imports/:importId',
  });
  return { ...rendered, api };
}

describe('012 FR-035, FR-036: import results and Undo import', () => {
  it('shows what the import did and undoes it after one confirmation', async () => {
    const undone = importDetail({
      import: importSummary({
        status: 'undone',
        undoneAt: '2026-10-18T10:00:00Z',
        undoneBy: { id: 'U-1002', name: 'Priya Raman' },
      }),
      undo: null,
      records: importDetail().records.map((r) => ({ ...r, exists: r.action === 'updated' })),
    });
    let done = false;
    const { user, api } = renderPage([
      reply('GET /api/v1/imports/{importId}', () => ({ body: done ? undone : importDetail() })),
      reply('POST /api/v1/imports/{importId}/undo', () => {
        done = true;
        return { body: undone };
      }),
    ]);

    expect(await screen.findByText('Created 1 campaign.')).toBeInTheDocument();
    expect(screen.getByText('Updated 1 activity.')).toBeInTheDocument();
    expect(screen.getByText('1 of 3 rows were left out.')).toBeInTheDocument();
    await expectNoAxeViolations();
    await user.click(screen.getByRole('button', { name: 'Undo import' }));

    const dialog = await screen.findByRole('alertdialog', { name: "Undo import of 'plan.xlsx'?" });
    expect(
      within(dialog).getByText(/The 1 campaign it created move to the recycle bin/),
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Undo import' }));

    expect(
      await screen.findByText('Import undone — the records are in the recycle bin'),
    ).toBeInTheDocument();
    expect(await screen.findByText(/Undone by Priya Raman/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Undo import' })).not.toBeInTheDocument();
    expect(api.requests.some((r) => r.path === '/api/v1/imports/IMP-1001/undo')).toBe(true);
  });

  it('shows the skeleton while the import loads', async () => {
    renderPage([reply('GET /api/v1/imports/{importId}', () => pending())]);

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(screen.queryByRole('button', { name: 'Undo import' })).not.toBeInTheDocument();
  });

  it('shows the no-access page when the API refuses the import', async () => {
    const message = "You don't have access to this plan. Ask an administrator if you need it.";
    renderPage([
      reply('GET /api/v1/imports/{importId}', {
        status: 403,
        body: apiError('forbidden', message),
      }),
      reply('GET /api/v1/me', { body: testUser() }),
    ]);

    expect(await screen.findByRole('heading', { level: 1, name: 'No access' })).toBeInTheDocument();
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument();
  });

  it('shows the API message for an import that no longer exists', async () => {
    renderPage([
      reply('GET /api/v1/imports/{importId}', {
        status: 404,
        body: {
          error: { code: 'not_found', message: 'This import no longer exists.', fields: [] },
        },
      }),
    ]);

    expect(await screen.findByText('This import no longer exists.')).toBeInTheDocument();
  });
});

describe('012 FR-031: Download problem list', () => {
  it('writes every problem as a CSV row, quoting commas and defusing formulas', () => {
    const csv = problemsCsv([
      { sheet: 'Activities', row: 4, column: 'Name', value: '=SUM(A1)', reason: 'Bad, very' },
    ]);

    expect(csv.split('\n').slice(0, 2)).toEqual([
      `${String.fromCharCode(0xfeff)}Sheet,Row,Column,Value found,Reason`,
      `Activities,4,Name,'=SUM(A1),"Bad, very"`,
    ]);
  });
});
