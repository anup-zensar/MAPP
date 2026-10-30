/** Titles of forms with unsaved changes, so a lost session can name them (D-363). */
const unsaved = new Map<symbol, () => string>();

export function markUnsaved(title: () => string): () => void {
  const key = Symbol('unsaved form');
  unsaved.set(key, title);
  return () => {
    unsaved.delete(key);
  };
}

export function unsavedFormTitle(): string | null {
  const first = unsaved.values().next();
  return first.done ? null : first.value();
}
