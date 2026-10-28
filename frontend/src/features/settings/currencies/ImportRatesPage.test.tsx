import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import type { Answer } from '@/testing/replies';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { expectNoAxeViolations } from '@/testing/axe';
import { parentOf } from '@/testing/dom';
import { validationFailed } from '@/testing/fixtures/settings';
import {
  IMPORTED,
  NO_IMPORTS,
  RECENT_IMPORTS_PAGE,
  REVIEW,
  REVIEW_NOTHING_NEW,
} from '@/testing/fixtures/rates';
import { reply } from '@/testing/replies';
import { ImportRatesPage } from './ImportRatesPage';
import { RecentImports } from './RecentImports';

const CSV = 'rate_type,from_currency,to_currency,valid_from,rate\nM,USD,EUR,2026-10-20,0.9230\n';

function rateFile(name = 'gsap_rates_2026-10-20.csv', content = CSV): File {
  return new File([content], name, { type: 'text/csv' });
}

function renderImport(answers: HttpHandler[] = []) {
  const api = mockApi([
    reply('POST /api/v1/exchange-rates/import-check', { body: REVIEW }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<ImportRatesPage />, {
    route: '/settings/currencies/import',
    path: '/settings/currencies/import',
    user: testUser(),
  });
  return { ...rendered, api };
}

function fileField(): HTMLInputElement {
  return screen.getByLabelText('GSAP daily rate extract (.csv)');
}

describe('ImportRatesPage (016 FR-006, US1-2)', () => {
  it('reviews the file before anything is saved: every outcome counted, rows with their numbers', async () => {
    const { user, api } = renderImport();

    expect(
      screen.getByRole('heading', { level: 1, name: 'Import GSAP rates' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Import GSAP rates · SP&A One');
    await user.upload(fileField(), rateFile());

    const review = await screen.findByRole('region', {
      name: "Review of 'gsap_rates_2026-10-20.csv'",
    });
    expect(api.bodiesOf('POST /api/v1/exchange-rates/import-check')).toEqual([
      { fileName: 'gsap_rates_2026-10-20.csv', content: btoa(CSV) },
    ]);
    const counts = within(review).getByRole('list', { name: 'Rows by outcome' });
    expect(
      within(counts)
        .getAllByRole('listitem')
        .map((item) => item.textContent),
    ).toEqual(['Problems1', 'Not used1', 'Replaces a rate2', 'To add3', 'Already present1']);
    // The rows that need a look are open at first, with their row numbers and why.
    expect(within(review).getByText('Problems (1)')).toBeInTheDocument();
    expect(
      within(review).getByText('Row 41: the rate must be above 0 with at most 6 decimals.'),
    ).toBeVisible();
    expect(within(review).getByText('CHF is not set up in Settings › Currencies.')).toBeVisible();
    expect(within(review).getByText('0.9214 → 0.923')).toBeVisible();
    expect(within(review).getByRole('rowheader', { name: '41' })).toBeInTheDocument();
    // Nothing is saved yet.
    expect(api.requests.some((r) => r.path === '/api/v1/exchange-rates/import')).toBe(false);
    expect(within(review).getByRole('button', { name: 'Import 5 rates' })).toBeInTheDocument();
  });

  it('imports the rows to add and replace in one step, then goes back to Currencies (FR-007)', async () => {
    const { user, api, router } = renderImport([
      reply('POST /api/v1/exchange-rates/import', { body: IMPORTED }),
    ]);
    await user.upload(fileField(), rateFile());

    await user.click(await screen.findByRole('button', { name: 'Import 5 rates' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/currencies'));
    expect(api.bodiesOf('POST /api/v1/exchange-rates/import')).toEqual([
      { fileName: 'gsap_rates_2026-10-20.csv', content: btoa(CSV) },
    ]);
    expect(
      await within(await findMessages()).findByText(
        "5 exchange rates imported from 'gsap_rates_2026-10-20.csv'.",
      ),
    ).toBeInTheDocument();
  });

  it('Cancel saves nothing and clears the review', async () => {
    const { user, api } = renderImport();
    await user.upload(fileField(), rateFile());
    await screen.findByRole('button', { name: 'Import 5 rates' });

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('region', { name: /^Review of/ })).not.toBeInTheDocument();
    expect(fileField()).toHaveValue('');
    expect(api.requests.some((r) => r.path === '/api/v1/exchange-rates/import')).toBe(false);
  });

  it('says when nothing in the file would change a rate', async () => {
    const { user } = renderImport([
      reply('POST /api/v1/exchange-rates/import-check', { body: REVIEW_NOTHING_NEW }),
    ]);
    await user.upload(fileField(), rateFile('gsap_rates_2026-10-19.csv'));

    expect(
      await screen.findByText('Nothing to import: every rate is already present or not used.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Import \d/ })).not.toBeInTheDocument();
  });

  it('takes only a .csv of up to 2 MB, before asking the API', async () => {
    const { api } = renderImport();

    fireEvent.change(fileField(), { target: { files: [rateFile('rates.xlsx')] } });
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Choose a .csv file: the GSAP daily rate extract.',
    );
    expect(fileField()).toHaveAttribute('aria-invalid', 'true');

    const big = rateFile();
    Object.defineProperty(big, 'size', { value: 3 * 1024 * 1024 });
    fireEvent.change(fileField(), { target: { files: [big] } });
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The file is larger than 2 MB. Split it and import each part.',
    );
    expect(api.requests.some((r) => r.path === '/api/v1/exchange-rates/import-check')).toBe(false);
  });

  it("shows the API's message for a file it can't read", async () => {
    const { user } = renderImport([
      reply('POST /api/v1/exchange-rates/import-check', {
        status: 400,
        body: validationFailed([
          { field: 'content', message: 'The file has no column "valid_from".' },
        ]),
      }),
    ]);

    await user.upload(fileField(), rateFile());

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The file has no column "valid_from".',
    );
  });

  describe('the sample file', () => {
    let saved: string[];

    beforeEach(() => {
      saved = [];
      // jsdom has no object URLs.
      URL.createObjectURL = () => 'blob:file-1';
      URL.revokeObjectURL = () => {};
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
        this: HTMLAnchorElement,
      ) {
        saved.push(this.download);
      });
    });

    afterEach(() => vi.restoreAllMocks());

    it('downloads a file of today’s rates', async () => {
      const { user } = renderImport([
        reply(
          'GET /api/v1/exchange-rates/import-sample',
          () =>
            new Response('rate_type,from_currency\n', {
              headers: {
                'Content-Type': 'text/csv',
                'Content-Disposition': 'attachment; filename="gsap_rates_sample.csv"',
              },
            }),
        ),
      ]);

      await user.click(screen.getByRole('button', { name: 'Download a sample file' }));

      await waitFor(() => expect(saved).toEqual(['gsap_rates_sample.csv']));
    });
  });

  it('has no axe violations with a review on screen (WCAG 2.2 AA)', async () => {
    const { user } = renderImport();
    await user.upload(fileField(), rateFile());
    await screen.findByRole('button', { name: 'Import 5 rates' });

    await expectNoAxeViolations();
  });
});

describe('RecentImports (016 FR-007, US1-3)', () => {
  function renderRecent(answer: Answer<'GET /api/v1/exchange-rates/imports'>) {
    mockApi([reply('GET /api/v1/exchange-rates/imports', answer)]);
    return renderWithProviders(<RecentImports />, { user: testUser() });
  }

  it('lists each file with who imported it, when, the dates of its rates and its counts', async () => {
    renderRecent({ body: RECENT_IMPORTS_PAGE });

    const row = parentOf(
      await screen.findByRole('rowheader', { name: 'gsap_rates_2026-10-20.csv' }),
    );
    expect(row).toHaveTextContent('Alex Morgan');
    expect(row).toHaveTextContent('21 Sep – 20 Oct 2026');
    expect(
      within(row)
        .getAllByRole('cell')
        .map((cell) => cell.textContent),
    ).toEqual(expect.arrayContaining(['87', '2', '1', '1', '1']));
    // Counts are numbers: right-aligned under their headers.
    const added = screen.getByRole('columnheader', { name: 'Added' });
    expect(within(row).getByText('87').className).toBe(added.className);
  });

  it('says when no file has been imported yet', async () => {
    renderRecent({ body: NO_IMPORTS });

    expect(await screen.findByText('No GSAP rate file has been imported yet.')).toBeInTheDocument();
  });

  it('has no axe violations (WCAG 2.2 AA)', async () => {
    renderRecent({ body: RECENT_IMPORTS_PAGE });
    await screen.findByRole('table');

    await expectNoAxeViolations();
  });
});
