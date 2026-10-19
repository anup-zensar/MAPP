import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Logo } from './Logo';

describe('Logo', () => {
  it('shows the NorthStar name next to the mark when asked (FR-019)', () => {
    render(<Logo withWordmark />);

    expect(screen.getByText('NorthStar')).toBeInTheDocument();
  });

  it('is only decoration without the name, so the link around it names itself (FR-019)', () => {
    render(
      <a href="/" aria-label="NorthStar home">
        <Logo />
      </a>,
    );

    expect(screen.getByRole('link', { name: 'NorthStar home' })).toBeInTheDocument();
    expect(screen.queryByText('NorthStar')).not.toBeInTheDocument();
  });

  it('gives a link around the logo and the name a readable name (FR-019)', () => {
    render(
      <a href="/">
        <Logo size="lg" withWordmark />
      </a>,
    );

    expect(screen.getByRole('link', { name: 'NorthStar' })).toBeInTheDocument();
  });
});
