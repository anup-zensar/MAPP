import { useCallback, useEffect, useRef, useState } from 'react';

/** Which edges of a sideways-scrolling box have more content beyond them. */
export interface SideScroll {
  /** Content is hidden on the left: the box has been scrolled. */
  start: boolean;
  /** Content is hidden on the right: there are more columns. */
  end: boolean;
}

const NONE: SideScroll = { start: false, end: false };

/**
 * The scroll cue of the data table standard (design system §5.4, UI audit AW-05): watches a box
 * that scrolls sideways and says which edges hide content, so the view can shade them and make the
 * box focusable ("<table>, scrolls sideways") only while it really overflows.
 */
export function useSideScroll<T extends HTMLElement>(): {
  ref: (element: T | null) => void;
  scroll: SideScroll;
} {
  const [element, setElement] = useState<T | null>(null);
  const [scroll, setScroll] = useState<SideScroll>(NONE);
  const frame = useRef(0);

  const measure = useCallback(() => {
    if (!element) {
      return;
    }
    const { scrollLeft, scrollWidth, clientWidth } = element;
    const next = {
      start: scrollLeft > 1,
      end: scrollLeft + clientWidth < scrollWidth - 1,
    };
    setScroll((current) =>
      current.start === next.start && current.end === next.end ? current : next,
    );
  }, [element]);

  useEffect(() => {
    if (!element) {
      return undefined;
    }
    const onScroll = () => {
      window.cancelAnimationFrame(frame.current);
      frame.current = window.requestAnimationFrame(measure);
    };
    // Measured in the next frame, once the table is laid out.
    onScroll();
    element.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onScroll);
    // The table grows when rows open or columns change; jsdom has no ResizeObserver.
    const observer =
      typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(onScroll);
    observer?.observe(element);
    const table = element.firstElementChild;
    if (table) {
      observer?.observe(table);
    }
    return () => {
      window.cancelAnimationFrame(frame.current);
      element.removeEventListener('scroll', onScroll);
      window.removeEventListener('resize', onScroll);
      observer?.disconnect();
    };
  }, [element, measure]);

  return { ref: setElement, scroll };
}
