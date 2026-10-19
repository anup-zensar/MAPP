import { fireEvent, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { ColumnChooser } from './ColumnChooser';
import { allWorkOptions } from './testData';

function Harness({ initial = ['id', 'status', 'owner'] }: { initial?: string[] }) {
  const [columns, setColumns] = useState(initial);
  const options = allWorkOptions();
  return (
    <ColumnChooser
      open
      onOpenChange={() => {}}
      groups={options.columnGroups}
      columns={columns}
      defaultColumns={options.defaultColumns}
      onChange={setColumns}
    />
  );
}

function shownInOrder(): string[] {
  const order = screen.getByRole('region', { name: 'Shown columns, in order' });
  return within(order)
    .getAllByRole('listitem')
    .map((item) => item.textContent ?? '');
}

describe('ColumnChooser (US3)', () => {
  it('offers the Standard group and one group per level (FR-018)', () => {
    renderWithProviders(<Harness />);

    const panel = screen.getByRole('dialog', { name: 'Columns' });
    expect(panel).toHaveAccessibleDescription('Choose and order the columns of the grid.');
    expect(
      within(panel)
        .getAllByRole('group')
        .map((group) => group.querySelector('legend')?.textContent),
    ).toEqual(['Standard', 'Plan', 'Campaign', 'Initiative', 'Activity', 'Line item']);
    const standard = within(panel).getByRole('group', { name: 'Standard' });
    expect(within(standard).getByRole('checkbox', { name: 'Planned cost' })).not.toBeChecked();
    expect(within(standard).getByRole('checkbox', { name: 'ID' })).toBeChecked();
  });

  it('locks Name and says why (FR-019)', () => {
    renderWithProviders(<Harness />);

    const name = screen.getByRole('checkbox', { name: 'Name' });
    expect(name).toBeChecked();
    expect(name).toBeDisabled();
    expect(name).toHaveAccessibleDescription(
      'The name column is always shown so you can tell rows apart',
    );
  });

  it('searches the columns of every group', async () => {
    const { user } = renderWithProviders(<Harness />);

    await user.type(screen.getByRole('searchbox', { name: 'Search columns' }), 'clus');

    const panel = screen.getByRole('dialog', { name: 'Columns' });
    expect(
      within(panel)
        .getAllByRole('group')
        .map((group) => group.querySelector('legend')?.textContent),
    ).toEqual(['Campaign']);
    expect(within(panel).getByRole('checkbox', { name: 'Cluster' })).toBeInTheDocument();
  });

  it('adds a Shell attribute column of the Campaign group with a tick, at the end (US3 scenario 1)', async () => {
    const { user } = renderWithProviders(<Harness />);

    await user.click(screen.getByRole('checkbox', { name: 'Cluster' }));

    expect(shownInOrder()).toEqual(['ID', 'Status', 'Owner', 'Cluster']);
    await user.click(screen.getByRole('checkbox', { name: 'Status' }));
    expect(shownInOrder()).toEqual(['ID', 'Owner', 'Cluster']);
  });

  it('moves columns with Move up and Move down (FR-021)', async () => {
    const { user } = renderWithProviders(<Harness />);

    expect(screen.queryByRole('button', { name: 'Move ID up' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Move Owner down' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Move Owner up' }));
    expect(shownInOrder()).toEqual(['ID', 'Owner', 'Status']);
    await user.click(screen.getByRole('button', { name: 'Move ID down' }));
    expect(shownInOrder()).toEqual(['Owner', 'ID', 'Status']);
  });

  it('moves a column by dragging it onto another (FR-021)', () => {
    renderWithProviders(<Harness />);

    const items = within(
      screen.getByRole('region', { name: 'Shown columns, in order' }),
    ).getAllByRole('listitem');
    fireEvent.dragStart(items[2]);
    fireEvent.dragOver(items[0]);
    fireEvent.drop(items[0]);

    expect(shownInOrder()).toEqual(['Owner', 'ID', 'Status']);
  });

  it('goes back to the default columns (FR-019)', async () => {
    const { user } = renderWithProviders(<Harness initial={['owner']} />);

    await user.click(screen.getByRole('button', { name: 'Reset to default columns' }));

    expect(shownInOrder()).toEqual([
      'ID',
      'Status',
      'Owner',
      'Start date',
      'End date',
      'Plan budget',
      'Planned cost',
      'Line items',
      'Unassigned planned cost',
    ]);
  });
});
