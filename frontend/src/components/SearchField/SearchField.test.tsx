import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { SearchField } from './SearchField';

function Example() {
  const [value, setValue] = useState('');
  return (
    <SearchField
      label="Search models"
      placeholder="Search name"
      value={value}
      onChange={setValue}
    />
  );
}

describe('SearchField', () => {
  it('is a named search box that says what it searches (design system §6.1)', () => {
    renderWithProviders(<Example />);

    const box = screen.getByRole('searchbox', { name: 'Search models' });
    expect(box).toHaveAttribute('placeholder', 'Search name');
    expect(screen.queryByRole('button', { name: 'Clear search' })).not.toBeInTheDocument();
  });

  it('offers Clear search once something is typed, and keeps the focus in the box', async () => {
    const { user } = renderWithProviders(<Example />);
    const box = screen.getByRole('searchbox', { name: 'Search models' });

    await user.type(box, 'Lubri');
    expect(box).toHaveValue('Lubri');

    await user.click(screen.getByRole('button', { name: 'Clear search' }));

    expect(box).toHaveValue('');
    expect(box).toHaveFocus();
    expect(screen.queryByRole('button', { name: 'Clear search' })).not.toBeInTheDocument();
  });
});
