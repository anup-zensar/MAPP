import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { testUser } from '@/testing/fixtures/users';
import { checkResult } from '@/testing/fixtures/financials';
import { ImportDialog } from './ImportDialog';

const CHECK = 'POST /api/v1/financials/imports/check';
const IMPORT = 'POST /api/v1/financials/imports';
const CSV =
  'type,document_number,posting_date,amount,currency\nPO,4500001009,2026-10-01,100.00,USD\n';

function renderDialog(answers: Record<string, MockHandler> = {}) {
  const onClose = vi.fn();
  const api = mockApi({ [CHECK]: { body: checkResult() }, ...answers });
  const rendered = renderWithProviders(<ImportDialog onClose={onClose} />, { user: testUser() });
  return { ...rendered, api, onClose };
}

const csvFile = (name = 'gsap-2026-10-19.csv', content = CSV) =>
  new File([content], name, { type: 'text/csv' });

describe('ImportDialog (015 US4, FR-004 – FR-009)', () => {
  it('explains the format, checks the chosen file and imports it (US4 scenarios 1, 2, 7)', async () => {
    const { api, user, queryClient, onClose } = renderDialog({
      [IMPORT]: {
        status: 201,
        body: {
          import: {
            id: 'FIM-1002',
            source: 'file',
            feedName: 'Simulated GSAP extract',
            fileName: 'gsap-2026-10-19.csv',
            importedAt: '2026-10-20T09:00:00Z',
            importedBy: { id: 'U-1001', name: 'Alex Morgan' },
            counts: {
              rows: 4,
              new: 2,
              updated: 1,
              unchanged: 1,
              refused: 0,
              notLinked: 1,
              relinked: 2,
            },
          },
          problems: [],
          problemsTotal: 0,
          notLinked: [],
          notLinkedTotal: 0,
        },
      },
    });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');

    const dialog = await screen.findByRole('dialog', { name: 'Import GSAP extract' });
    expect(within(dialog).getByText(/at most 700 KB and 5,000 rows/)).toBeInTheDocument();
    expect(
      within(dialog).getByText('type, document_number, posting_date, amount, currency'),
    ).toBeInTheDocument();
    expect(
      within(dialog).getByRole('button', { name: 'Download sample file' }),
    ).toBeInTheDocument();

    await user.upload(within(dialog).getByLabelText(/GSAP extract file/), csvFile());

    expect(
      await within(dialog).findByText(
        '4 rows: 2 new, 1 updated, 1 unchanged, 0 refused · 1 not linked',
      ),
    ).toBeInTheDocument();
    const [checked] = api.bodiesOf(CHECK) as { fileName: string; content: string }[];
    expect(checked.fileName).toBe('gsap-2026-10-19.csv');
    expect(atob(checked.content)).toBe(CSV);
    const notLinked = within(dialog).getByRole('region', { name: 'Not linked (1)' });
    expect(notLinked).toHaveTextContent(
      'Row 5 · PO 4500001009 / 10: No line item for HX key HX1099 — check the tracking key in GSAP.',
    );

    await user.click(within(dialog).getByRole('button', { name: 'Import 4 rows' }));

    expect(
      (
        await screen.findAllByText(
          "Imported 'gsap-2026-10-19.csv': 2 new, 1 updated · 1 not linked. 2 earlier transactions are now linked",
        )
      )[0],
    ).toBeInTheDocument();
    expect(api.bodiesOf(IMPORT)).toEqual([checked]);
    expect(onClose).toHaveBeenCalled();
    await waitFor(() =>
      expect(invalidate.mock.calls.map(([filters]) => filters?.queryKey)).toEqual(
        expect.arrayContaining([['financials'], ['all-work'], ['home']]),
      ),
    );
  });

  it('lists every row problem at once and imports the other rows (US4 scenario 3)', async () => {
    const { user } = renderDialog({
      [CHECK]: {
        body: checkResult({
          counts: {
            rows: 4,
            new: 2,
            updated: 0,
            unchanged: 0,
            refused: 2,
            notLinked: 0,
            relinked: 0,
          },
          problems: [
            {
              row: 3,
              column: 'amount',
              message: 'Amount must be a number like 1234.50 (a minus sign for credits).',
            },
            {
              row: 5,
              column: 'currency',
              message:
                "Currency 'XYZ' isn't set up in Settings — an administrator adds it under Currencies.",
            },
          ],
          problemsTotal: 2,
          notLinked: [],
          notLinkedTotal: 0,
          unusedColumns: ['cost_centre'],
        }),
      },
    });

    const dialog = await screen.findByRole('dialog');
    await user.upload(within(dialog).getByLabelText(/GSAP extract file/), csvFile());

    const problems = await within(dialog).findByRole('region', { name: 'Problems (2)' });
    expect(
      within(problems)
        .getAllByRole('listitem')
        .map((item) => item.textContent),
    ).toEqual([
      'Row 3 · amount: Amount must be a number like 1234.50 (a minus sign for credits).',
      "Row 5 · currency: Currency 'XYZ' isn't set up in Settings — an administrator adds it under Currencies.",
    ]);
    expect(within(dialog).getByText('Columns not used: cost_centre')).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: 'Import 2 rows' })).toBeInTheDocument();
  });

  it('offers nothing to import when every row is refused', async () => {
    const { api, user } = renderDialog({
      [CHECK]: {
        body: checkResult({
          counts: {
            rows: 1,
            new: 0,
            updated: 0,
            unchanged: 0,
            refused: 1,
            notLinked: 0,
            relinked: 0,
          },
          problems: [
            { row: 2, column: 'type', message: 'Type must be PO, INVOICE, PAYMENT or ACCRUAL.' },
          ],
          problemsTotal: 1,
          notLinked: [],
          notLinkedTotal: 0,
        }),
      },
    });

    const dialog = await screen.findByRole('dialog');
    await user.upload(within(dialog).getByLabelText(/GSAP extract file/), csvFile());

    expect(
      await within(dialog).findByText('Nothing to import — fix the rows and choose the file again'),
    ).toBeInTheDocument();
    expect(within(dialog).queryByRole('button', { name: /^Import/ })).toBeNull();
    await user.click(within(dialog).getByRole('button', { name: 'Choose another file' }));
    expect(within(dialog).getByLabelText(/GSAP extract file/)).toBeInTheDocument();
    expect(api.bodiesOf(IMPORT)).toEqual([]);
  });

  it('shows a whole-file refusal at the file field', async () => {
    const { user } = renderDialog({
      [CHECK]: {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the file.',
            fields: [
              {
                field: 'content',
                message: 'The first row must name the columns. Missing: type, amount.',
              },
            ],
          },
        },
      },
    });

    const dialog = await screen.findByRole('dialog');
    const input = within(dialog).getByLabelText(/GSAP extract file/);
    await user.upload(input, csvFile());

    expect(
      await within(dialog).findByText(
        'The first row must name the columns. Missing: type, amount.',
      ),
    ).toBeInTheDocument();
    expect(input).toHaveAttribute('aria-invalid', 'true');
  });

  it('asks for a file before checking, and refuses one above 700 KB', async () => {
    const { api, user } = renderDialog();

    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: 'Check file' }));
    expect(
      within(dialog).getByText('Choose a .csv file — GSAP extracts are imported as CSV.'),
    ).toBeInTheDocument();

    await user.upload(
      within(dialog).getByLabelText(/GSAP extract file/),
      csvFile('big.csv', 'x'.repeat(700 * 1024 + 1)),
    );
    expect(
      await within(dialog).findByText(
        'The file is larger than 700 KB — split it into smaller files.',
      ),
    ).toBeInTheDocument();
    expect(api.bodiesOf(CHECK)).toEqual([]);
  });
});
