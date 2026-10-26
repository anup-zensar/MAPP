import type { FieldProblem } from '../FormErrorSummary/FormErrorSummary';

/** A failed save that is not about one field, shown above the form (FR-020 of feature 001). */
export interface SaveFailure {
  message: string;
  /** What the person can do now, after a conflict. */
  nextStep?: string;
}

/** The problem with one field, shown under it: the message of the first problem with that field. */
export function problemOf(problems: FieldProblem[], field: string): string | undefined {
  return problems.find((problem) => problem.field === field)?.message;
}
