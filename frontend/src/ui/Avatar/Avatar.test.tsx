import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Avatar } from './Avatar';

describe('Avatar', () => {
  it('is left out of the accessible name, because the name is always next to it (FR-019)', () => {
    render(
      <button type="button">
        <Avatar initials="PR" size="sm" />
        Priya Raman
      </button>,
    );

    expect(screen.getByRole('button', { name: 'Priya Raman' })).toBeInTheDocument();
  });
});
