import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { components } from '@/api/generated/schema';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ImportOptionsDialog } from './ImportOptionsDialog';
import { shellAttribute } from '@/testing/fixtures/shell-attributes';

type ImportRow = components['schemas']['ImportRow'];

const COST_CENTRE = shellAttribute({ id: 'SA-1030', label: 'Cost centre', optionCount: 3 });
const CHECK = 'POST /api/v1/shell-attributes/SA-1030/options/import-check';
const IMPORT = 'POST /api/v1/shell-attributes/SA-1030/options/import';

const FILE_TEXT = 'Option\r\nCN04000220\r\n';

function row(number: number, label: string, message = '', parentLabel = ''): ImportRow {
  return { row: number, label, parentLabel, message };
}

/** The US6 file: 1,217 cost centres, one already an option, three with problems. */
const CHECKED = {
  totalRows: 1217,
  toAdd: Array.from({ length: 1214 }, (_, index) =>
    row(index + 2, `CN0400${String(index).padStart(4, '0')}`),
  ),
  alreadyPresent: [row(1216, 'CN04000001', 'Already an option — skipped.')],
  problems: [
    row(5, '', 'Empty — enter an option.'),
    row(9, 'X'.repeat(121), 'Longer than 120 characters.'),
    row(17, 'CN04000220', 'Appears twice in the file (also row 4).'),
  ],
};

function renderDialog(answers: Record<string, MockHandler> = {}, attribute = COST_CENTRE) {
  const onClose = vi.fn();
  const api = mockApi(answers);
  const rendered = renderWithProviders(
    <ImportOptionsDialog attribute={attribute} onClose={onClose} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose };
}

function fileField() {
  return screen.getByLabelText('Choose a file');
}

async function chooseFile(file: File) {
  fireEvent.change(fileField(), { target: { files: [file] } });
}

describe('ImportOptionsDialog (US6, FR-031)', () => {
  it('says how the file is made and that nothing is added until confirmed', () => {
    renderDialog();

    const dialog = screen.getByRole('dialog', { name: 'Import options into Cost centre' });
    expect(dialog).toHaveAccessibleDescription(
      'Download the template, fill one row per option, then upload it. Nothing is added until you confirm.',
    );
    expect(fileField()).toHaveAttribute('accept', '.csv,.xlsx');
    expect(
      within(dialog)
        .getAllByRole('button')
        .map((button) => button.textContent),
    ).toEqual(['Cancel']);
  });

  it('lists every row by outcome with its row number, then adds the new ones (US6-1, US6-2)', async () => {
    const { user, api, onClose } = renderDialog({
      [CHECK]: { body: CHECKED },
      [IMPORT]: { body: { added: 1214, alreadyPresent: 1, problems: 3 } },
    });

    await chooseFile(new File([FILE_TEXT], 'cost-centres.csv', { type: 'text/csv' }));

    const dialog = screen.getByRole('dialog');
    expect(await within(dialog).findByText('1,214 options to add')).toBeInTheDocument();
    expect(within(dialog).getByText('1 already present (skipped)')).toBeInTheDocument();
    // The problems are unfolded, each with its row number and what is wrong.
    const problems = within(dialog).getByText('3 rows with problems').closest('details');
    expect(problems).toHaveAttribute('open');
    expect(problems).toHaveTextContent('Row 5—Empty — enter an option.');
    expect(problems).toHaveTextContent('Row 9');
    expect(problems).toHaveTextContent('Longer than 120 characters.');
    expect(problems).toHaveTextContent('Row 17CN04000220Appears twice in the file (also row 4).');
    // The rows to add show the first 100, then how many more.
    const toAdd = within(dialog).getByText('1,214 options to add').closest('details');
    expect(toAdd).not.toHaveAttribute('open');
    await user.click(within(dialog).getByText('1,214 options to add'));
    expect(within(toAdd as HTMLElement).getAllByRole('listitem')).toHaveLength(100);
    expect(toAdd).toHaveTextContent('and 1,114 more');

    const file = { fileName: 'cost-centres.csv', content: btoa(FILE_TEXT) };
    expect(api.bodiesOf(CHECK)).toEqual([file]);
    expect(api.bodiesOf(IMPORT)).toEqual([]);
    await user.click(within(dialog).getByRole('button', { name: 'Add 1,214 options' }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    // The dialog stays in this test (onClose is a stand-in), so the message is found by its text.
    expect(await screen.findAllByText('1,214 options added to Cost centre')).not.toHaveLength(0);
    expect(api.bodiesOf(IMPORT)).toEqual([file]);
  });

  it('adds nothing on Cancel', async () => {
    const { user, api, onClose } = renderDialog({ [CHECK]: { body: CHECKED } });

    await chooseFile(new File([FILE_TEXT], 'cost-centres.csv'));
    await screen.findByText('1,214 options to add');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onClose).toHaveBeenCalled();
    expect(api.bodiesOf(IMPORT)).toEqual([]);
  });

  it('says when there is nothing to add, offering Cancel only', async () => {
    renderDialog({
      [CHECK]: { body: { ...CHECKED, toAdd: [], totalRows: 4 } },
    });

    await chooseFile(new File([FILE_TEXT], 'cost-centres.csv'));

    expect(
      await screen.findByText('Nothing to add — every row is already an option or has a problem.'),
    ).toBeInTheDocument();
    expect(screen.getByText('0 options to add')).toBeInTheDocument();
    expect(
      within(screen.getByRole('dialog'))
        .getAllByRole('button')
        .map((button) => button.textContent),
    ).toEqual(['Cancel']);
  });

  it('refuses other file types and files over 700 KB before sending them', async () => {
    const { api } = renderDialog();

    await chooseFile(new File(['x'], 'cost-centres.xls'));
    expect(
      screen.getByText(
        'Choose a .csv or .xlsx file. An older .xls file can be saved as .xlsx in Excel.',
      ),
    ).toBeInTheDocument();
    expect(fileField()).toHaveAttribute('aria-invalid', 'true');

    await chooseFile(new File([new Uint8Array(700 * 1024 + 1)], 'cost-centres.csv'));
    expect(
      screen.getByText('The file is larger than 700 KB — split it into smaller files.'),
    ).toBeInTheDocument();
    expect(fileField()).toHaveAccessibleDescription(
      'The file is larger than 700 KB — split it into smaller files.',
    );
    expect(api.requests).toEqual([]);
  });

  it("shows the API's problem with the file under the field", async () => {
    renderDialog({
      [CHECK]: {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [
              {
                field: 'content',
                message: 'The file has 6,000 rows — import at most 5,000 rows at a time.',
              },
            ],
          },
        },
      },
    });

    await chooseFile(new File([FILE_TEXT], 'cost-centres.xlsx'));

    expect(
      await screen.findByText('The file has 6,000 rows — import at most 5,000 rows at a time.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Add/ })).not.toBeInTheDocument();
  });

  it('shows the parent option of each row for a dependent attribute', async () => {
    const region = shellAttribute({
      id: 'SA-1030',
      label: 'Region',
      parentAttribute: { id: 'SA-1003', name: 'Cluster', active: true },
    });
    renderDialog(
      {
        [CHECK]: {
          body: {
            totalRows: 2,
            toAdd: [row(2, 'RETAIL – JAPAN', '', 'Retail')],
            alreadyPresent: [],
            problems: [
              row(3, 'B2B – MARS', "Parent option 'Space' not found in cluster.", 'Space'),
            ],
          },
        },
      },
      region,
    );

    expect(screen.getByRole('dialog')).toHaveAccessibleDescription(
      'Download the template, fill one row per option, with its cluster option in the second column, then upload it. Nothing is added until you confirm.',
    );
    await chooseFile(new File([FILE_TEXT], 'regions.csv'));

    const problems = (await screen.findByText('1 row with problems')).closest('details');
    expect(problems).toHaveTextContent(
      "Row 3B2B – MARS (Space)Parent option 'Space' not found in cluster.",
    );
    expect(screen.getByRole('button', { name: 'Add 1 option' })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText('0 already present (skipped)')).toBeVisible());
  });
});
