import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { Combobox } from './Combobox';
import type { ComboboxOption } from './Combobox';

const FRUIT: ComboboxOption[] = [
  { id: 'apple', label: 'Apple', searchText: 'Apple green' },
  { id: 'banana', label: 'Banana', searchText: 'Banana yellow' },
  { id: 'cherry', label: 'Cherry', searchText: 'Cherry red' },
];

interface ExampleProps {
  multiple?: boolean;
  start?: string[];
  options?: ComboboxOption[];
  disabledReason?: string;
  limit?: number;
}

function Example({ multiple, start = [], options = FRUIT, disabledReason, limit }: ExampleProps) {
  const [value, setValue] = useState(start);
  return (
    <>
      <Combobox
        label="Fruit"
        name="fruit"
        options={options}
        value={value}
        onChange={setValue}
        multiple={multiple}
        placeholder="Search fruit"
        countText={(n) => (n === 1 ? '1 fruit' : `${n} fruits`)}
        noMatchText={(query) => `No fruit matches '${query}'.`}
        limit={limit}
        limitText={(shown, total) => `Showing ${shown} of ${total} — keep typing to narrow down.`}
        disabledReason={disabledReason}
      />
      <p>chosen: {value.join(',')}</p>
    </>
  );
}

function combobox() {
  return screen.getByRole('combobox', { name: 'Fruit' });
}

describe('Combobox', () => {
  it('opens a list of every option, with a count told to screen readers (FR-020)', async () => {
    const { user } = renderWithProviders(<Example />);

    expect(combobox()).toHaveAttribute('aria-expanded', 'false');
    await user.click(combobox());

    expect(combobox()).toHaveAttribute('aria-expanded', 'true');
    const list = screen.getByRole('listbox', { name: 'Fruit' });
    expect(
      within(list)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual(['Apple', 'Banana', 'Cherry']);
    expect(combobox()).toHaveAttribute('aria-controls', list.id);
    expect(screen.getByRole('status')).toHaveTextContent('3 fruits');
  });

  it('filters by any part of the search text, ignoring case', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.type(combobox(), 'YELL');

    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual(['Banana']);
    expect(screen.getByRole('status')).toHaveTextContent('1 fruit');
  });

  it('says when nothing matches', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.type(combobox(), 'kiwi');

    expect(screen.getByText("No fruit matches 'kiwi'.")).toBeInTheDocument();
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
  });

  it('moves with the arrow keys and chooses with Enter; one value replaces the other', async () => {
    const { user } = renderWithProviders(<Example start={['apple']} />);

    await user.click(combobox());
    await user.keyboard('{ArrowDown}{ArrowDown}');
    const options = screen.getAllByRole('option');
    expect(combobox()).toHaveAttribute('aria-activedescendant', options[2].id);
    expect(options[0]).toHaveAttribute('aria-selected', 'true');

    await user.keyboard('{ArrowUp}{Enter}');

    expect(screen.getByText('chosen: banana')).toBeInTheDocument();
    expect(combobox()).toHaveAttribute('aria-expanded', 'false');
    expect(combobox()).toHaveFocus();
  });

  it('closes with Escape', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(combobox());
    await user.keyboard('{Escape}');

    expect(combobox()).toHaveAttribute('aria-expanded', 'false');
  });

  it('toggles several values, shows them as chips and removes them (multiple)', async () => {
    const { user } = renderWithProviders(<Example multiple />);

    await user.click(combobox());
    await user.click(screen.getByRole('option', { name: 'Apple' }));
    await user.click(screen.getByRole('option', { name: 'Cherry' }));

    expect(screen.getByText('chosen: apple,cherry')).toBeInTheDocument();
    expect(screen.getByRole('listbox')).toHaveAttribute('aria-multiselectable', 'true');
    expect(screen.getByRole('option', { name: 'Cherry' })).toHaveAttribute('aria-selected', 'true');

    await user.click(screen.getByRole('option', { name: 'Apple' }));
    expect(screen.getByText('chosen: cherry')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Remove Cherry' }));
    expect(screen.getByText('chosen:')).toBeInTheDocument();
  });

  it('removes the last chip with Backspace in an empty input (multiple)', async () => {
    const { user } = renderWithProviders(<Example multiple start={['apple', 'banana']} />);

    await user.click(combobox());
    await user.keyboard('{Backspace}');

    expect(screen.getByText('chosen: apple')).toBeInTheDocument();
  });

  it('lists at most `limit` matches and says how to narrow down', async () => {
    const many = Array.from({ length: 120 }, (_, index) => ({
      id: `o${index}`,
      label: `Option ${index}`,
    }));
    const { user } = renderWithProviders(<Example options={many} limit={50} />);

    await user.click(combobox());

    expect(screen.getAllByRole('option')).toHaveLength(50);
    expect(screen.getByText('Showing 50 of 120 — keep typing to narrow down.')).toBeInTheDocument();
  });

  it('reaches every one of 300 options without a limit', async () => {
    const many = Array.from({ length: 300 }, (_, index) => ({
      id: `o${index}`,
      label: `Option ${index}`,
    }));
    const { user } = renderWithProviders(<Example options={many} />);

    await user.click(combobox());

    expect(screen.getAllByRole('option')).toHaveLength(300);
    expect(screen.getByRole('status')).toHaveTextContent('300 fruits');
  });

  it('when locked, keeps its values, never opens and says why', async () => {
    const { user } = renderWithProviders(
      <Example start={['apple']} disabledReason="Choose a cluster first." />,
    );

    await user.click(combobox());

    expect(combobox()).toHaveAttribute('aria-disabled', 'true');
    expect(combobox()).toHaveAttribute('aria-expanded', 'false');
    expect(combobox()).toHaveAccessibleDescription('Choose a cluster first.');
    expect(screen.getByText('Apple')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Remove Apple' })).not.toBeInTheDocument();
  });
});
