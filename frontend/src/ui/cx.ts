/**
 * Joins the class names that apply, skipping false, null and undefined:
 * cx('item', isActive && 'active').
 */
export function cx(...classNames: (string | false | null | undefined)[]): string {
  return classNames.filter(Boolean).join(' ');
}
