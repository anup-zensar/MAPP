/**
 * The rules of a saved view's name (data-model §1.1, FR-029), checked in the browser first so all
 * problems show at once; the API checks them again.
 */
import { ApiError } from '@/api/errors';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SavedView } from './allWorkTypes';

export const NAME_MAX = 60;

/** The problem with a name typed for a view, or null: required and at most 60 characters. */
export function nameProblem(name: string): string | null {
  const trimmed = name.trim();
  if (trimmed === '') {
    return 'Enter a name for the view.';
  }
  // Characters, not UTF-16 units: "é" or an emoji counts once.
  if ([...trimmed].length > NAME_MAX) {
    return 'Use 60 characters or fewer.';
  }
  return null;
}

/** The person's other view with this name, ignoring letter case (names are unique per person). */
export function viewNamed(
  views: SavedView[],
  name: string,
  except?: string,
): SavedView | undefined {
  const wanted = name.trim().toLocaleLowerCase();
  return views.find((view) => view.id !== except && view.name.toLocaleLowerCase() === wanted);
}

/** "You already have a view called '<name>'." */
export function nameTaken(name: string): string {
  return `You already have a view called '${name.trim()}'.`;
}

/**
 * The field problems of a save the API refused for its content (the name's at the name field), or
 * null when it was refused for another reason.
 */
export function problemsOf(error: unknown, nameField: string): FieldProblem[] | null {
  if (!(error instanceof ApiError) || error.kind !== 'validation' || error.fields.length === 0) {
    return null;
  }
  return error.fields.map((field) => ({
    field: field.field === 'name' ? nameField : field.field,
    message: field.message,
  }));
}
