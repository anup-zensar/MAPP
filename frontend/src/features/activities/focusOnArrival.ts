/**
 * A ref callback for a dialog's fields that arrive after it opened (the move and copy dialogs load
 * the initiatives first): the first field takes the focus, which FormDialog had put on the submit
 * button meanwhile — unless the person already moved it into a field. In a group of radios that
 * is the chosen one, where Tab would land too.
 */
export function focusOnArrival(node: HTMLElement | null): void {
  if (!node || document.activeElement instanceof HTMLInputElement) {
    return;
  }
  const inputs = Array.from(node.querySelectorAll<HTMLInputElement>('input:not([disabled])'));
  const first = inputs[0];
  if (first?.type === 'radio') {
    const chosen = inputs.find((input) => input.name === first.name && input.checked);
    (chosen ?? first).focus();
    return;
  }
  first?.focus();
}
