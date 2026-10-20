import { useEffect } from 'react';
import type { RefObject } from 'react';

/**
 * Keeps the chosen tab visible inside its tab list, which scrolls sideways on a phone: a view
 * opened from the address (?tab=monitoring) then shows its own tab. Only the tab list scrolls,
 * never the page.
 */
export function useChosenTabInView(container: RefObject<HTMLElement | null>, tab: string) {
  useEffect(() => {
    const list = container.current?.querySelector<HTMLElement>('[role="tablist"]');
    const chosen = list?.querySelector<HTMLElement>('[role="tab"][aria-selected="true"]');
    if (!list || !chosen) {
      return;
    }
    const listBox = list.getBoundingClientRect();
    const tabBox = chosen.getBoundingClientRect();
    if (tabBox.right > listBox.right) {
      list.scrollLeft += tabBox.right - listBox.right;
    } else if (tabBox.left < listBox.left) {
      list.scrollLeft -= listBox.left - tabBox.left;
    }
  }, [container, tab]);
}
