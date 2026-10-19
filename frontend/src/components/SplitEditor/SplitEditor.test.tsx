import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { SplitEditor } from './SplitEditor';
import type { SplitShare } from './SplitEditor';

const BRANDS = [
  { id: 'SO-1', label: 'Shell Helix', active: true },
  { id: 'SO-2', label: 'Shell Rimula', active: true },
  { id: 'SO-3', label: 'Shell Tellus', active: true },
  { id: 'SO-4', label: 'Pennzoil', active: false },
];

function Example({ start = [] }: { start?: SplitShare[] }) {
  const [value, setValue] = useState(start);
  return (
    <>
      <SplitEditor
        label="Brand split"
        name="attributes.SA-1013"
        marking="Needed to submit"
        options={BRANDS}
        value={value}
        onChange={setValue}
      />
      <p>value: {JSON.stringify(value)}</p>
    </>
  );
}

describe('SplitEditor', () => {
  it('splits evenly whenever a value is chosen or removed, and says so (R10)', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(screen.getByRole('checkbox', { name: 'Shell Helix' }));
    await user.click(screen.getByRole('checkbox', { name: 'Shell Rimula' }));
    await user.click(screen.getByRole('checkbox', { name: 'Shell Tellus' }));

    expect(screen.getByRole('textbox', { name: 'Share of Shell Helix' })).toHaveValue('33.33');
    expect(screen.getByRole('textbox', { name: 'Share of Shell Tellus' })).toHaveValue('33.34');
    expect(screen.getByText('100.00 % — complete')).toBeInTheDocument();
    expect(screen.getByText('Shares split evenly again.')).toBeInTheDocument();

    await user.click(screen.getByRole('checkbox', { name: 'Shell Rimula' }));
    expect(screen.getByRole('textbox', { name: 'Share of Shell Helix' })).toHaveValue('50.00');
  });

  it('keeps a running total that says what is left or too much (FR-027)', async () => {
    const { user } = renderWithProviders(
      <Example
        start={[
          { optionId: 'SO-1', share: '60.00' },
          { optionId: 'SO-2', share: '30.00' },
        ]}
      />,
    );

    expect(screen.getByText('90.00 % — 10.00 % left')).toBeInTheDocument();

    const second = screen.getByRole('textbox', { name: 'Share of Shell Rimula' });
    await user.clear(second);
    await user.type(second, '45');
    expect(screen.getByText('105.00 % — 5.00 % too much')).toBeInTheDocument();
    expect(screen.queryByText('Shares split evenly again.')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Split evenly' }));
    expect(screen.getByText('100.00 % — complete')).toBeInTheDocument();
  });

  it('keeps an inactive value it already has, labelled "(inactive)" (FR-027)', () => {
    renderWithProviders(<Example start={[{ optionId: 'SO-4', share: '100.00' }]} />);

    expect(screen.getByRole('textbox', { name: 'Share of Pennzoil (inactive)' })).toHaveValue(
      '100.00',
    );
    expect(screen.getByRole('group', { name: 'Brand split Needed to submit' })).toBeInTheDocument();
  });
});
