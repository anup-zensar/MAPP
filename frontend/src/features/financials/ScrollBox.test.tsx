import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ScrollBox } from './ScrollBox';

/** A ResizeObserver that reports once, as soon as it observes, with the widths below. */
function measure(scrollWidth: number, clientWidth: number) {
  vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockReturnValue(scrollWidth);
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(clientWidth);
  vi.stubGlobal(
    'ResizeObserver',
    class {
      callback: () => void;
      constructor(callback: () => void) {
        this.callback = callback;
      }
      observe() {
        this.callback();
      }
      unobserve() {}
      disconnect() {}
    },
  );
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('ScrollBox (design system §5.4)', () => {
  it('is a named region in the tab order while its table is wider than the box', () => {
    measure(900, 390);
    render(
      <ScrollBox label="Allocations">
        <table />
      </ScrollBox>,
    );

    const box = screen.getByRole('region', { name: 'Allocations, scrolls sideways' });
    expect(box).toHaveAttribute('tabindex', '0');
  });

  it('stays out of the tab order when everything fits', () => {
    measure(800, 1100);
    const { container } = render(
      <ScrollBox label="Allocations">
        <table />
      </ScrollBox>,
    );

    expect(screen.queryByRole('region')).not.toBeInTheDocument();
    expect(container.firstElementChild).not.toHaveAttribute('tabindex');
  });
});
