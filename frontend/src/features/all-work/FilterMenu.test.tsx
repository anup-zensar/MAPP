import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { FilterMenu } from './FilterMenu';
import type { FilterChoice, FilterMenuProps } from './FilterMenu';

const OWNERS: FilterChoice[] = [
  { value: 'U-1003', label: 'Daniel Okoro' },
  { value: 'U-1002', label: 'Priya Raman' },
  { value: 'U-1009', label: 'Tom Weber', inactive: true },
];

/** The menu with its values kept, as the filter bar keeps them. */
function Harness(props: Omit<FilterMenuProps, 'selected' | 'onChange'> & { initial?: string[] }) {
  const { initial = [], ...rest } = props;
  const [selected, setSelected] = useState(initial);
  return (
    <>
      <FilterMenu {...rest} selected={selected} onChange={setSelected} />
      <output aria-label="Chosen">{selected.join(',')}</output>
    </>
  );
}

function chosen() {
  return screen.getByRole('status', { name: 'Chosen' }).textContent;
}

describe('FilterMenu (FR-004, FR-009)', () => {
  it('lists the choices as check boxes that apply at once, inactive ones labelled', async () => {
    const { user } = renderWithProviders(
      <Harness label="Owner" choices={OWNERS} multiple summary={undefined} />,
    );

    await user.click(screen.getByRole('button', { name: 'Owner' }));
    const menu = await screen.findByRole('dialog', { name: 'Owner' });
    expect(
      within(menu)
        .getAllByRole('checkbox')
        .map((box) => box.closest('label')?.textContent),
    ).toEqual(['Daniel Okoro', 'Priya Raman', 'Tom Weber (inactive)']);

    await user.click(within(menu).getByRole('checkbox', { name: 'Daniel Okoro' }));
    await user.click(within(menu).getByRole('checkbox', { name: 'Priya Raman' }));
    expect(chosen()).toBe('U-1003,U-1002');
    await user.click(within(menu).getByRole('checkbox', { name: 'Daniel Okoro' }));
    expect(chosen()).toBe('U-1002');
  });

  it('offers one value with radio buttons for a single filter', async () => {
    const onChange = vi.fn();
    const { user } = renderWithProviders(
      <FilterMenu
        label="Show levels"
        multiple={false}
        choices={[
          { value: 'everything', label: 'Everything' },
          { value: 'completeBranches', label: 'Complete branches only' },
        ]}
        selected={['everything']}
        onChange={onChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Show levels' }));
    const menu = await screen.findByRole('dialog', { name: 'Show levels' });
    expect(within(menu).getByRole('radio', { name: 'Everything' })).toBeChecked();
    await user.click(within(menu).getByRole('radio', { name: 'Complete branches only' }));

    expect(onChange).toHaveBeenCalledWith(['completeBranches']);
  });

  it('ticks several values with the quick choice first (My teams)', async () => {
    const { user } = renderWithProviders(
      <Harness
        label="Team"
        multiple
        choices={[
          { value: 'T-1001', label: 'Lubricants Europe' },
          { value: 'T-1002', label: 'Mobility China' },
          { value: 'T-1003', label: 'Global Marketing Ops' },
        ]}
        quickChoice={{ label: 'My teams', values: ['T-1001', 'T-1003'] }}
        initial={['T-1002']}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Team' }));
    const menu = await screen.findByRole('dialog');
    const boxes = within(menu).getAllByRole('checkbox');
    expect(boxes[0]).toHaveAccessibleName('My teams');
    await user.click(nth(boxes, 0));
    expect(chosen()).toBe('T-1001,T-1003,T-1002');
    expect(within(menu).getByRole('checkbox', { name: 'My teams' })).toBeChecked();

    await user.click(within(menu).getByRole('checkbox', { name: 'My teams' }));
    expect(chosen()).toBe('T-1002');
  });

  it('searches long lists', async () => {
    const many = Array.from({ length: 12 }, (_, index) => ({
      value: `U-${index}`,
      label: index === 7 ? 'Sara Novak' : `Person ${index}`,
    }));
    const { user } = renderWithProviders(<Harness label="Owner" multiple choices={many} />);

    await user.click(screen.getByRole('button', { name: 'Owner' }));
    const menu = await screen.findByRole('dialog');
    await user.type(within(menu).getByRole('searchbox', { name: 'Search filters' }), 'nov');

    expect(within(menu).getAllByRole('checkbox')).toHaveLength(1);
    expect(within(menu).getByRole('checkbox', { name: 'Sara Novak' })).toBeInTheDocument();
  });

  it('keeps a chosen value that is no longer offered, so it can be removed', async () => {
    const { user } = renderWithProviders(
      <Harness label="Owner" multiple choices={OWNERS} initial={['U-0999']} />,
    );

    await user.click(screen.getByRole('button', { name: 'Owner' }));
    const menu = await screen.findByRole('dialog');
    await user.click(within(menu).getByRole('checkbox', { name: 'U-0999' }));

    expect(chosen()).toBe('');
  });
});
