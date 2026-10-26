import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { Answer } from '@/testing/replies';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { DEFAULT_COLUMNS } from '@/testing/fixtures/all-work-rows';
import { emptyResult, gridResult, standard2026 } from '@/testing/fixtures/all-work';
import { reply } from '@/testing/replies';
import type { AllWorkResult, ViewState } from './all-work-types';
import { ExportMenu } from './ExportMenu';

const XLSX_NAME = 'MAPP All work – Grid – 2026 – 18 Oct 2026.xlsx';

/** The API's file answer, named in Content-Disposition as the contract says. */
function fileAnswer(name: string, type: string): Response {
  return new Response('file', {
    headers: {
      'Content-Type': type,
      'Content-Disposition': `attachment; filename="${name.replaceAll('–', '-')}"; filename*=UTF-8''${encodeURIComponent(name)}`,
    },
  });
}

let saved: string[];

beforeEach(() => {
  saved = [];
  // jsdom has no object URLs and does not save files: the test notes the name instead.
  URL.createObjectURL = () => 'blob:export';
  URL.revokeObjectURL = () => {};
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    saved.push(this.download);
  });
});

afterEach(() => vi.restoreAllMocks());

function renderExport(
  state: ViewState,
  result: AllWorkResult | undefined,
  answer: Answer<'GET /api/v1/all-work/export'> = () =>
    fileAnswer(XLSX_NAME, 'application/octet-stream'),
) {
  const api = mockApi([reply('GET /api/v1/all-work/export', answer)]);
  const rendered = renderWithProviders(<ExportMenu state={state} result={result} />, {
    user: testUser(),
  });
  return { ...rendered, api };
}

function exportQuery(api: ReturnType<typeof mockApi>): URLSearchParams {
  const request = api.requests.find((candidate) => candidate.path === '/api/v1/all-work/export');
  return new URLSearchParams(request?.search);
}

describe('ExportMenu (US7)', () => {
  it('saves a spreadsheet of the view as filtered, under the API’s file name (US7 scenario 1, FR-032, FR-034)', async () => {
    const state: ViewState = {
      ...standard2026(),
      showUnassigned: false,
      filters: { ...standard2026().filters, activityStatuses: ['declined'] },
    };
    const { api, user } = renderExport(state, gridResult());

    await user.click(screen.getByRole('button', { name: 'Export' }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'Spreadsheet (.xlsx)',
      'CSV (.csv)',
    ]);
    await user.click(screen.getByRole('menuitem', { name: 'Spreadsheet (.xlsx)' }));

    const messages = await findMessages();
    expect(within(messages).getByText(`Exported '${XLSX_NAME}'`)).toBeInTheDocument();
    expect(saved).toEqual([XLSX_NAME]);
    const query = exportQuery(api);
    expect(query.get('format')).toBe('xlsx');
    expect(query.get('unassigned')).toBe('false');
    expect(query.get('year')).toBe('2026');
    expect(query.get('view')).toBe('grid');
    expect(query.getAll('activityStatus')).toEqual(['declined']);
    expect(query.getAll('column')).toEqual(DEFAULT_COLUMNS);
  });

  it('exports a CSV of Budget & cost with its period (US7 scenario 2, FR-032)', async () => {
    const name = 'MAPP All work – Budget & cost – 2026 – 18 Oct 2026.csv';
    const { api, user } = renderExport(
      { ...standard2026(), view: 'budget', period: 'quarter' },
      undefined,
      () => fileAnswer(name, 'text/csv'),
    );

    await user.click(screen.getByRole('button', { name: 'Export' }));
    await user.click(await screen.findByRole('menuitem', { name: 'CSV (.csv)' }));

    await waitFor(() => expect(saved).toEqual([name]));
    const query = exportQuery(api);
    expect(query.get('format')).toBe('csv');
    expect(query.get('view')).toBe('budget');
    expect(query.get('period')).toBe('quarter');
    expect(query.get('unassigned')).toBe('true');
  });

  it('says "Preparing your file…" while the file is written', async () => {
    let send: (response: Response) => void = () => {};
    const { user } = renderExport(
      standard2026(),
      gridResult(),
      () => new Promise<Response>((resolve) => (send = resolve)),
    );

    await user.click(screen.getByRole('button', { name: 'Export' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Spreadsheet (.xlsx)' }));

    const button = await screen.findByRole('button', { name: 'Preparing your file…' });
    expect(button).toHaveAttribute('aria-busy', 'true');
    send(fileAnswer(XLSX_NAME, 'application/octet-stream'));
    expect(await screen.findByRole('button', { name: 'Export' })).toBeInTheDocument();
  });

  it('says there is nothing to export, without asking the API (edge case)', async () => {
    const { api, user } = renderExport(standard2026(), emptyResult());

    await user.click(screen.getByRole('button', { name: 'Export' }));
    await user.click(await screen.findByRole('menuitem', { name: 'CSV (.csv)' }));

    const messages = await findMessages();
    expect(
      within(messages).getByText('Nothing to export — clear a filter first.'),
    ).toBeInTheDocument();
    expect(api.requests.filter((request) => request.path === '/api/v1/all-work/export')).toEqual(
      [],
    );
  });

  it('shows the API’s message when the file cannot be written', async () => {
    const { user } = renderExport(standard2026(), gridResult(), {
      status: 400,
      body: {
        error: {
          code: 'validation_failed',
          message: 'Choose a year from 2000 to 2100.',
          fields: [{ field: 'year', message: 'Choose a year from 2000 to 2100.' }],
        },
      },
    });

    await user.click(screen.getByRole('button', { name: 'Export' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Spreadsheet (.xlsx)' }));

    const messages = await findMessages();
    expect(within(messages).getByText('Choose a year from 2000 to 2100.')).toBeInTheDocument();
    expect(saved).toEqual([]);
  });
});
