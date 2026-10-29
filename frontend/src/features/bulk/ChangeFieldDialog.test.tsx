import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { bulkResult, outcome } from '@/testing/fixtures/bulk';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { ChangeFieldDialog } from './ChangeFieldDialog';

const FIELDS = {
  level: 'activity' as const,
  total: 2,
  notOffered: 'Names, IDs, money … are edited one record at a time.',
  items: [
    {
      key: 'endDate',
      label: 'End date',
      kind: 'date' as const,
      required: false,
      options: [],
      dependsOn: null,
      condition: null,
    },
    {
      key: 'attr.SA-1',
      label: 'PIR',
      kind: 'yesNo' as const,
      required: false,
      options: [],
      dependsOn: null,
      condition: null,
    },
  ],
};

describe('012 FR-009 – FR-011: Change field', () => {
  it('offers the fields, lists the approved activities and sends Change and restart approval', async () => {
    const preview = bulkResult({
      action: 'changeField',
      counts: { done: 1, skipped: 1, failed: 0 },
      reapproval: [
        {
          recordId: 'A-1002',
          level: 'activity',
          name: 'Fleet card',
          planId: 'P-1001',
          status: 'Approved',
        },
      ],
      outcomes: [
        outcome(),
        outcome({
          recordId: 'A-1002',
          name: 'Fleet card',
          outcome: 'skipped',
          reasons: ['Approved — left unchanged so it keeps its approval'],
        }),
      ],
    });
    const api = mockApi([
      reply('GET /api/v1/bulk-actions/fields', { body: FIELDS }),
      reply('POST /api/v1/bulk-actions/preview', { body: preview }),
      reply('POST /api/v1/bulk-actions', {
        status: 201,
        body: { ...preview, id: 'BLK-1', preview: false },
      }),
    ]);
    const onDone = vi.fn();
    const records = [
      { id: 'A-1001', level: 'activity' as const, name: 'Launch film', planId: 'P-1001' },
      { id: 'A-1002', level: 'activity' as const, name: 'Fleet card', planId: 'P-1001' },
    ];
    const { user } = renderWithProviders(
      <ChangeFieldDialog records={records} onClose={vi.fn()} onDone={onDone} />,
      {
        user: testUser(),
      },
    );

    const dialog = await screen.findByRole('dialog', { name: 'Change field' });
    expect(await within(dialog).findByText(/edited one record at a time/)).toBeInTheDocument();
    await user.selectOptions(within(dialog).getByRole('combobox', { name: /^Field/ }), 'End date');
    await user.type(within(dialog).getByLabelText('Value'), '2026-06-20');

    expect(
      await within(dialog).findByText(/These 1 activities are submitted or approved/),
    ).toBeInTheDocument();
    expect(
      within(dialog).getByText('Approved — left unchanged so it keeps its approval'),
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole('radio', { name: 'Change and restart approval' }));
    await user.click(await within(dialog).findByRole('button', { name: 'Change field' }));

    expect(api.bodiesOf('POST /api/v1/bulk-actions').at(-1)).toEqual({
      action: 'changeField',
      recordIds: ['A-1001', 'A-1002'],
      field: { key: 'endDate', value: { date: '2026-06-20' } },
      restartApproval: true,
    });
    expect(onDone).toHaveBeenCalled();
  });
});
