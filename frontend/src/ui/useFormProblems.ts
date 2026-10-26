import { useState } from 'react';
import type { FieldValues, Path, UseFormReturn } from 'react-hook-form';
import type { FieldProblem } from './FormErrorSummary/FormErrorSummary';

/** Forms check on Save and keep their problems until the next Save (design system §6.4). */
export const CHECK_ON_SAVE = { mode: 'onSubmit', reValidateMode: 'onSubmit' } as const;

/** What useFormProblems gives a form. */
export interface FormProblems {
  /** Every problem of the last Save, for FormErrorSummary: a new list on each attempt. */
  problems: FieldProblem[];
  /** handleSubmit's second callback: every field the schema refused, in the form's order. */
  onInvalid: () => void;
  /** The API's field problems (a 400): at their fields and in the summary. */
  showServerProblems: (problems: FieldProblem[]) => void;
  clearProblems: () => void;
}

/** The error summary of a React Hook Form form whose fields, in order, are `fields`. */
export function useFormProblems<T extends FieldValues>(
  form: UseFormReturn<T>,
  fields: readonly Path<T>[],
): FormProblems {
  const [problems, setProblems] = useState<FieldProblem[]>([]);

  function onInvalid() {
    setProblems(
      fields.flatMap((field) => {
        const message = form.getFieldState(field).error?.message;
        return message ? [{ field, message }] : [];
      }),
    );
  }

  function showServerProblems(apiProblems: FieldProblem[]) {
    for (const problem of apiProblems) {
      const field = fields.find((name) => name === problem.field);
      if (field) {
        form.setError(field, { message: problem.message });
      }
    }
    setProblems(apiProblems);
  }

  return { problems, onInvalid, showServerProblems, clearProblems: () => setProblems([]) };
}
