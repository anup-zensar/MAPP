/**
 * The forms with typed changes that are not saved yet, so a lost session can say whose changes
 * were lost (038 FR-031, D-357). Drafts are not kept; only the form's title is.
 */
const unsaved = new Map<symbol, () => string>();

/** Marks a form as having unsaved changes until the returned function is called. */
export function markUnsaved(title: () => string): () => void {
  const key = Symbol('unsaved form');
  unsaved.set(key, title);
  return () => {
    unsaved.delete(key);
  };
}

/** The title of the first form with unsaved changes; null when there is none. */
export function unsavedFormTitle(): string | null {
  const first = unsaved.values().next();
  return first.done ? null : first.value();
}
