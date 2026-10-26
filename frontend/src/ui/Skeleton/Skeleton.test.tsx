import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Skeleton } from './Skeleton';

describe('Skeleton', () => {
  it('tells screen readers that the content is loading (FR-019)', () => {
    render(<Skeleton />);

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
  });

  it('hides its grey shapes from screen readers and draws one per line (FR-019)', () => {
    render(<Skeleton lines={3} width="240px" />);

    const shapes = screen.getByRole('status').querySelector('[aria-hidden="true"]');
    expect(shapes?.children).toHaveLength(3);
  });
});
