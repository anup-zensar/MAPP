import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { Logo } from './Logo';
import { LogoTile } from './LogoTile';

describe('Logo', () => {
  it('is the name SP&A One, read as one name and with no symbol beside it (035 FR-004, FR-005)', () => {
    const { container } = render(<Logo />);

    expect(screen.getByRole('img', { name: 'SP&A One' })).toHaveTextContent('SP&A One');
    expect(container.querySelector('svg')).toBeNull();
  });

  it('names the link around it SP&A One (035 US1 scenario 3)', async () => {
    const { container } = render(
      <a href="/">
        <Logo size="lg" />
      </a>,
    );

    expect(screen.getByRole('link', { name: 'SP&A One' })).toBeInTheDocument();
    await expectNoAxeViolations(container);
  });
});

describe('LogoTile', () => {
  it('is decoration, so the link around it names itself (035 FR-005)', async () => {
    const { container } = render(
      <a href="/" aria-label="SP&A One home">
        <LogoTile />
      </a>,
    );

    expect(screen.getByRole('link', { name: 'SP&A One home' })).toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
    await expectNoAxeViolations(container);
  });
});
