import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { CountBadge } from './CountBadge';

describe('CountBadge', () => {
  it('shows the count and says what it counts to screen readers (feature 008)', () => {
    render(
      <a href="/approvals">
        Approvals <CountBadge count={3} label="waiting for you" />
      </a>,
    );

    expect(screen.getByRole('link', { name: 'Approvals 3 waiting for you' })).toBeInTheDocument();
  });

  it('caps the number at max+', () => {
    render(<CountBadge count={12} label="unread" max={9} />);

    expect(screen.getByText('9+')).toBeInTheDocument();
  });

  it('shows nothing for 0', () => {
    const { container } = render(<CountBadge count={0} label="unread" />);

    expect(container).toBeEmptyDOMElement();
  });
});
