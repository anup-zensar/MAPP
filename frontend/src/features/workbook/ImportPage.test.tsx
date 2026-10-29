import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import { plan } from '@/testing/fixtures/plans';
import { testUser } from '@/testing/fixtures/users';
import { importCheck, importDetail, importSummary } from '@/testing/fixtures/workbook';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { ImportPage } from './ImportPage';

function renderPage(answers: HttpHandler[]) {
  const api = mockApi([
    reply('GET /api/v1/plans/{planId}', {
      body: plan({ id: 'P-1001', name: 'FY2026 Lubricants Europe' }),
    }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<ImportPage />, {
    user: testUser(),
    route: '/plans/P-1001/import',
    path: '/plans/:planId/import',
  });
  return { ...rendered, api };
}

const XLSX = new File(['PK'], 'plan.xlsx', {
  type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
});

describe('012 FR-025 – FR-034: Import from Excel', () => {
  it('shows the steps and the earlier imports, then every problem of the check and imports the ready rows', async () => {
    const { user, api, router } = renderPage([
      reply('GET /api/v1/plans/{planId}/imports', { body: { items: [importSummary()], total: 1 } }),
      reply('POST /api/v1/plans/{planId}/imports/check', { body: importCheck() }),
      reply('POST /api/v1/plans/{planId}/imports', { status: 201, body: importDetail() }),
    ]);

    expect(await screen.findByRole('heading', { name: 'Import from Excel' })).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: 'plan.xlsx' })).toBeInTheDocument();
    await user.upload(screen.getByLabelText('Choose a file'), XLSX);
    await user.click(screen.getByRole('button', { name: 'Check file' }));

    expect(await screen.findByText('3 rows · 2 ready · 1 with problems')).toBeInTheDocument();
    expect(screen.getByText('1 to create · 1 to update')).toBeInTheDocument();
    const problems = screen.getByRole('table', { name: 'Problems' });
    expect(
      within(problems).getByText('Not a real date — write it as 2026-03-12'),
    ).toBeInTheDocument();
    expect(screen.getByText(/were ignored: Activities: Notes/)).toBeInTheDocument();
    await expectNoAxeViolations();
    await user.click(screen.getByRole('button', { name: 'Import 2 rows' }));

    expect(api.bodiesOf('POST /api/v1/plans/P-1001/imports')).toEqual([
      { fileName: 'plan.xlsx', content: 'UEs=' },
    ]);
    await screen.findByText('Another page');
    expect(router.state.location.pathname).toBe('/imports/IMP-1001');
  });

  it('shows the API refusal of a file (FR-028)', async () => {
    const { user } = renderPage([
      reply('GET /api/v1/plans/{planId}/imports', { body: { items: [], total: 0 } }),
      reply('POST /api/v1/plans/{planId}/imports/check', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need attention.',
            fields: [
              {
                field: 'file',
                message:
                  "The Activities sheet has no 'Activity type' column — download a fresh template",
              },
            ],
          },
        },
      }),
    ]);

    expect(
      await screen.findByText('No imports yet — import from a spreadsheet.'),
    ).toBeInTheDocument();
    await user.upload(screen.getByLabelText('Choose a file'), XLSX);
    await user.click(screen.getByRole('button', { name: 'Check file' }));
    expect(await screen.findByRole('alert')).toHaveTextContent("no 'Activity type' column");
  });

  it('says when an exported file holds what the plan holds (FR-027)', async () => {
    const { user } = renderPage([
      reply('GET /api/v1/plans/{planId}/imports', { body: { items: [], total: 0 } }),
      reply('POST /api/v1/plans/{planId}/imports/check', {
        body: importCheck({
          withProblems: 0,
          ready: 3,
          problems: [],
          ignoredColumns: [],
          needMoreFields: 0,
          nothingToChange: true,
          levels: importCheck().levels.map((l) => ({
            ...l,
            create: 0,
            update: 0,
            leftOut: 0,
            unchanged: 1,
          })),
        }),
      }),
    ]);

    await user.upload(await screen.findByLabelText('Choose a file'), XLSX);
    await user.click(screen.getByRole('button', { name: 'Check file' }));

    expect(await screen.findByText('Nothing to change')).toBeInTheDocument();
    expect(
      screen.getByText('The file holds what the plan holds — nothing to import.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Import/ })).not.toBeInTheDocument();
  });

  it('shows the error state when the plan cannot be read', async () => {
    renderPage([
      reply('GET /api/v1/plans/{planId}', {
        status: 404,
        body: { error: { code: 'not_found', message: 'This plan no longer exists.', fields: [] } },
      }),
    ]);

    expect(await screen.findByText('This plan no longer exists.')).toBeInTheDocument();
  });
});
