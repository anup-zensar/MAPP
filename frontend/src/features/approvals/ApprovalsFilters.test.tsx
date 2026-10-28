import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { DANIEL, rowOf } from '@/testing/fixtures/approvals';
import { NO_FILTERS, applyFilters, filterChips, filterOptions } from './approval-filters';
import type { ApprovalFilterValues } from './approval-filters';
import { ApprovalsFilters } from './ApprovalsFilters';

const ROWS = [
  rowOf(),
  rowOf({
    activity: {
      id: 'A-1043',
      name: 'Spring radio',
      status: 'submitted',
      displayStatus: 'submitted',
    },
    path: [{ type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' }],
    owner: DANIEL,
    workflow: { id: 'WF-1002', name: 'China BC approval workflow', active: false },
    submittedAt: '2026-03-01T09:00:00Z',
  }),
  rowOf({
    activity: null,
    subject: {
      kind: 'purchaseRequest',
      label: 'Purchase request',
      id: 'PR-1001',
      name: 'Agency retainer',
      status: 'inReview',
      displayStatus: 'inReview',
    },
    submittedAt: '2026-03-02T09:00:00Z',
  }),
];

function Harness() {
  const [filters, setFilters] = useState<ApprovalFilterValues>(NO_FILTERS);
  const visible = applyFilters(ROWS, '', filters);
  return (
    <>
      <ApprovalsFilters filters={filters} options={filterOptions(ROWS)} onChange={setFilters} />
      <p>{visible.map((row) => row.subject.id).join(', ')}</p>
    </>
  );
}

describe('ApprovalsFilters (feature 008, FR-031)', () => {
  it('offers the plans, workflows and owners of the rows', async () => {
    const { user } = renderWithProviders(<Harness />, { user: testUser() });

    await user.click(screen.getByRole('button', { name: 'Filters' }));
    const popover = await screen.findByRole('dialog', { name: 'Filters' });
    const options = (name: string) =>
      within(within(popover).getByRole('combobox', { name }))
        .getAllByRole('option')
        .map((option) => option.textContent);

    expect(options('Plan')).toEqual([
      'Choose…',
      'FY2026 Lubricants Europe',
      'FY2026 Mobility China',
    ]);
    expect(options('Approval workflow')).toEqual([
      'Choose…',
      'China BC approval workflow (inactive)',
      'Lubricants EU approval',
    ]);
    expect(options('Owner')).toEqual(['Choose…', 'Daniel Okoro', 'Priya Raman']);
  });

  it('filters by record type, offering only the types of the rows (031 FR-026)', async () => {
    const { user } = renderWithProviders(<Harness />, { user: testUser() });

    await user.click(screen.getByRole('button', { name: 'Filters' }));
    const popover = await screen.findByRole('dialog', { name: 'Filters' });
    const recordType = within(popover).getByRole('combobox', { name: 'Record type' });
    expect(
      within(recordType)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual(['Choose…', 'Activity', 'Purchase request']);

    await user.selectOptions(recordType, 'purchaseRequest');

    expect(screen.getByText('PR-1001')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters (1)' })).toBeInTheDocument();
  });

  it('filters by the submitted dates and counts the filters in use', async () => {
    const { user } = renderWithProviders(<Harness />, { user: testUser() });

    await user.click(screen.getByRole('button', { name: 'Filters' }));
    const popover = await screen.findByRole('dialog', { name: 'Filters' });
    await user.type(within(popover).getByLabelText('Submitted from'), '2026-03-05');

    expect(screen.getByText('A-1042')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters (1)' })).toBeInTheDocument();

    await user.click(within(popover).getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('A-1042, A-1043, PR-1001')).toBeInTheDocument();
  });

  it('names each filter in use as a chip', () => {
    const chips = filterChips(
      {
        kind: 'purchaseRequest',
        planId: 'P-1002',
        workflowId: 'WF-1002',
        ownerId: 'U-1003',
        from: '2026-03-01',
        to: '',
      },
      filterOptions(ROWS),
      (date) => date,
    );

    expect(chips.map((chip) => chip.label)).toEqual([
      'Record type: Purchase request',
      'Plan: FY2026 Mobility China',
      'Approval workflow: China BC approval workflow (inactive)',
      'Owner: Daniel Okoro',
      'Submitted from: 2026-03-01',
    ]);
  });
});
