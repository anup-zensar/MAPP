/** The element, typed as an HTML element; fails the test with `what` when it is missing. */
export function htmlElement(element: Element | null | undefined, what = 'element'): HTMLElement {
  if (!(element instanceof HTMLElement)) {
    throw new Error(`Expected an HTML element: ${what}.`);
  }
  return element;
}

/** The nearest ancestor of `element` (or itself) that matches `selector`. */
export function closestTo(element: Element, selector: string): HTMLElement {
  return htmlElement(element.closest(selector), `an ancestor matching "${selector}"`);
}

export function parentOf(element: Element): HTMLElement {
  return htmlElement(element.parentElement, 'a parent element');
}

export function nextOf(element: Element): HTMLElement {
  return htmlElement(element.nextElementSibling, 'a next sibling');
}

/** The item at `index`; fails the test when the list is shorter. */
export function nth<T>(items: readonly T[], index: number): T {
  const item = items.at(index);
  if (item === undefined) {
    throw new Error(`Expected an item at ${index} of ${items.length}.`);
  }
  return item;
}
