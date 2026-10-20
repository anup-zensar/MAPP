import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ScrollBox } from './ScrollBox';

/** jsdom lays nothing out: the box's widths are set by hand. */
function widths(scrollWidth: number, clientWidth: number) {
  vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockReturnValue(scrollWidth);
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(clientWidth);
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('ScrollBox (design system §5.4, UI audit AW-05)', () => {
  it('is a named, focusable region with a cue on the right while the table is wider', async () => {
    widths(1400, 900);
    render(
      <ScrollBox label="All work grid">
        <table />
      </ScrollBox>,
    );

    const region = await screen.findByRole('region', { name: 'All work grid, scrolls sideways' });
    expect(region).toHaveAttribute('tabindex', '0');
    expect(region.parentElement?.className).toMatch(/cueEnd/);
    expect(region.parentElement?.className).not.toMatch(/cueStart/);
  });

  it('is a plain box, out of the tab order, while everything fits', async () => {
    widths(900, 900);
    render(
      <ScrollBox label="All work grid">
        <table />
      </ScrollBox>,
    );

    // The first measurement happens in the next frame.
    await new Promise((resolve) => window.requestAnimationFrame(resolve));
    await waitFor(() => expect(screen.queryByRole('region')).not.toBeInTheDocument());
  });
});
