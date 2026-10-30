import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import { bulkResult, outcome } from '@/testing/fixtures/bulk';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { BulkBar } from './BulkBar';
import type { SelectedRecord } from './bulk-types';
import { useSelection } from './useSelection';

const LAUNCH: SelectedRecord = {
  id: 'A-1001',
  level: 'activity',
  name: 'Launch film',
  planId: 'P-1001',
};
const PODCAST: SelectedRecord = {
  id: 'A-1002',
  level: 'activity',
  name: 'Podcast',
  planId: 'P-1001',
};
const HELIX: SelectedRecord = {
  id: 'C-1001',
  level: 'campaign',
  name: 'Helix launch',
  planId: 'P-1001',
};

function Harness({ records }: { records: SelectedRecord[] }) {
  const selection = useSelection();
  return (
    <>
      <button type="button" onClick={() => selection.select(records)}>
        Pick
      </button>
      <BulkBar selection={selection} matching={records} />
    </>
  );
}

async function renderBar(records: SelectedRecord[], answers: HttpHandler[] = []) {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<Harness records={records} />, { user: testUser() });
  await rendered.user.click(screen.getByRole('button', { name: 'Pick' }));
  return { ...rendered, api };
}

function bar() {
  return screen.getByRole('region', { name: 'Selection' });
}

describe('012 FR-012, FR-013, FR-021: Submit for approval and the results', () => {
  it('says how many go and why the others cannot, submits, and selects the failed items', async () => {
    const skipped = outcome({
      recordId: 'A-1002',
      name: 'Podcast',
      outcome: 'skipped',
      reasons: ['2 fields needed: Owner, Company code'],
    });
    const preview = bulkResult({
      action: 'submit',
      verb: 'submitted',
      counts: { done: 1, skipped: 1, failed: 0 },
      outcomes: [outcome(), skipped],
    });
    const { user, api } = await renderBar(
      [LAUNCH, PODCAST],
      [
        reply('POST /api/v1/bulk-actions/preview', { body: preview }),
        reply('POST /api/v1/bulk-actions', {
          status: 201,
          body: { ...preview, id: 'BLK-1001', preview: false },
        }),
      ],
    );

    await user.click(within(bar()).getByRole('button', { name: 'Submit for approval' }));

    const dialog = await screen.findByRole('dialog', { name: 'Submit for approval' });
    expect(
      await within(dialog).findByText('1 will be submitted · 1 is not ready'),
    ).toBeInTheDocument();
    expect(within(dialog).getByText('2 fields needed: Owner, Company code')).toBeInTheDocument();
    await expectNoAxeViolations(dialog);
    await user.click(within(dialog).getByRole('button', { name: 'Submit 1 activity' }));

    const panel = await screen.findByRole('dialog', { name: 'Results' });
    expect(within(panel).getByText('1 submitted · 1 skipped')).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/bulk-actions')).toEqual([
      { action: 'submit', recordIds: ['A-1001', 'A-1002'], comment: '', restartApproval: false },
    ]);
    await user.click(within(panel).getByRole('button', { name: 'Select failed items' }));
    expect(within(bar()).getByText('1 selected · 1 activity')).toBeInTheDocument();
  });

  it('offers only Close when nothing can be submitted', async () => {
    const { user } = await renderBar(
      [HELIX],
      [
        reply('POST /api/v1/bulk-actions/preview', {
          body: bulkResult({
            action: 'submit',
            verb: 'submitted',
            counts: { done: 0, skipped: 1, failed: 0 },
            outcomes: [
              outcome({
                recordId: 'C-1001',
                level: 'campaign',
                name: 'Helix launch',
                outcome: 'skipped',
                reasons: ['Only activities go through approval'],
              }),
            ],
          }),
        }),
      ],
    );

    await user.click(within(bar()).getByRole('button', { name: 'Submit for approval' }));

    const dialog = await screen.findByRole('dialog', { name: 'Submit for approval' });
    expect(
      await within(dialog).findByText('Nothing can be submitted — fix the problems first.'),
    ).toBeInTheDocument();
    expect(within(dialog).getByText('Only activities go through approval')).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Close' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

describe('012 FR-007, FR-014, FR-016: one level for Copy to…', () => {
  it('asks which level to keep, then lists the places and copies there', async () => {
    const { user, api } = await renderBar(
      [HELIX, LAUNCH],
      [
        reply('POST /api/v1/bulk-actions/targets', {
          body: {
            items: [
              {
                id: 'I-1003',
                level: 'initiative',
                name: 'Events',
                path: 'FY2026 Lubricants Europe › Rimula fleet › Events',
                planId: 'P-1001',
                year: 2026,
              },
            ],
            total: 1,
            note: null,
          },
        }),
        reply('POST /api/v1/bulk-actions/preview', {
          body: bulkResult({ action: 'copy', verb: 'copied' }),
        }),
        reply('POST /api/v1/bulk-actions', {
          status: 201,
          body: bulkResult({
            id: 'BLK-1',
            preview: false,
            action: 'copy',
            verb: 'copied',
            outcomes: [outcome({ newRecordId: 'A-1099' })],
          }),
        }),
      ],
    );

    await user.click(within(bar()).getByRole('button', { name: 'Copy to…' }));
    const choose = await screen.findByRole('dialog', { name: 'Copy works on one level at a time' });
    await user.click(within(choose).getByRole('radio', { name: /Keep only activities/ }));
    await user.click(within(choose).getByRole('button', { name: 'Continue' }));

    const dialog = await screen.findByRole('dialog', { name: 'Copy to…' });
    await user.click(await within(dialog).findByRole('radio', { name: /Rimula fleet › Events/ }));
    await user.click(await within(dialog).findByRole('button', { name: 'Copy 1 record' }));

    expect(await screen.findByText('1 copied')).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/bulk-actions')).toEqual([
      { action: 'copy', recordIds: ['A-1001'], targetId: 'I-1003', restartApproval: false },
    ]);
  });
});

describe('012 FR-019, FR-020: Delete and Undo', () => {
  it('names what goes, deletes to the recycle bin and undoes it in one step', async () => {
    const summary = {
      selected: { campaigns: 1, initiatives: 0, activities: 1 },
      below: { campaigns: 0, initiatives: 1, activities: 2, lineItems: 3 },
      submittedOrApproved: 1,
    };
    const preview = bulkResult({
      action: 'delete',
      verb: 'deleted',
      counts: { done: 2, skipped: 0, failed: 0 },
      deleteSummary: summary,
    });
    const { user, api } = await renderBar(
      [HELIX, LAUNCH],
      [
        reply('POST /api/v1/bulk-actions/preview', { body: preview }),
        reply('POST /api/v1/bulk-actions', {
          status: 201,
          body: { ...preview, id: 'BLK-1003', preview: false },
        }),
        reply('POST /api/v1/bulk-actions/{bulkActionId}/undo', {
          body: { bulkActionId: 'BLK-1003', restored: [] },
        }),
      ],
    );

    await user.click(within(bar()).getByRole('button', { name: 'Delete' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: 'Delete 1 campaign, 1 activity?',
    });
    expect(
      within(dialog).getByText('With them go 1 initiative, 2 activities and 3 line items.'),
    ).toBeInTheDocument();
    expect(within(dialog).getByText('1 of them are submitted or approved.')).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Delete 2 records' }));

    expect(await screen.findByText('2 records moved to the recycle bin')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Undo' }));
    expect(await screen.findByText('0 records restored')).toBeInTheDocument();
    expect(api.requests.map((r) => r.path)).toContain('/api/v1/bulk-actions/BLK-1003/undo');
  });

  it('refuses more than 1,000 records with the limit (FR-006)', async () => {
    const many = Array.from({ length: 1001 }, (_, i) => ({ ...LAUNCH, id: `A-${2000 + i}` }));
    const { user } = await renderBar(many);

    expect(within(bar()).getByText(/handle up to 1,000 records/)).toBeInTheDocument();
    await user.click(within(bar()).getByRole('button', { name: 'Change owner' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
