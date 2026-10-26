import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Avatar } from './Avatar';

describe('Avatar', () => {
  it('shows the person’s initials (FR-019)', () => {
    render(<Avatar initials="PR" />);

    expect(screen.getByText('PR')).toBeInTheDocument();
  });

  it('is left out of the accessible name, because the name is always next to it (FR-019)', () => {
    render(
      <button type="button">
        <Avatar initials="PR" size="sm" />
        Priya Raman
      </button>,
    );

    expect(screen.getByRole('button', { name: 'Priya Raman' })).toBeInTheDocument();
  });

  it.each(['sm', 'md', 'lg'] as const)('comes in the %s size (FR-019)', (size) => {
    render(<Avatar initials="DO" size={size} />);

    expect(screen.getByText('DO')).toBeInTheDocument();
  });
});
