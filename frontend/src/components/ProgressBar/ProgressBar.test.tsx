import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ProgressBar } from './ProgressBar';

describe('ProgressBar', () => {
  it('is a named progress bar with its percentage (design system §4)', () => {
    render(<ProgressBar value={62.5} label="Budget used" />);

    const bar = screen.getByRole('progressbar', { name: 'Budget used' });
    expect(bar).toHaveAttribute('aria-valuenow', '62.5');
    expect(bar).toHaveAttribute('aria-valuetext', '62.5 %');
    expect(bar.firstElementChild).toHaveStyle({ width: '62.5%' });
  });

  it('stops the fill at 100 % but tells the real value, in red when asked', () => {
    render(<ProgressBar value={125} label="Budget used" tone="problem" />);

    const bar = screen.getByRole('progressbar', { name: 'Budget used' });
    expect(bar).toHaveAttribute('aria-valuenow', '100');
    expect(bar).toHaveAttribute('aria-valuetext', '125 %');
    expect(bar.firstElementChild).toHaveStyle({ width: '100%' });
  });

  it('has no value when there is nothing to measure', () => {
    render(<ProgressBar value={null} label="Budget used" />);

    expect(screen.getByRole('progressbar', { name: 'Budget used' })).not.toHaveAttribute(
      'aria-valuenow',
    );
  });
});
