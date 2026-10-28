import type { FieldError, FieldErrors } from 'react-hook-form';
import type { ApiError } from '@/api/errors';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { PurchaseRequestFormValues } from './form-schema';

const HEADER_FIELDS = [
  'name',
  'description',
  'ownerId',
  'approvalWorkflowId',
  'modificationWorkflowId',
] as const;
const LINE_FIELDS = ['currency', 'amount', 'deliveryDate', 'overallLimit', 'description'] as const;

function problem(field: string, error: FieldError | undefined): FieldProblem[] {
  return error?.message ? [{ field, message: error.message }] : [];
}

/** The schema's problems in form order, named like the API's fields ("lines[0].amount"). */
export function clientProblems(errors: FieldErrors<PurchaseRequestFormValues>): FieldProblem[] {
  const header = HEADER_FIELDS.flatMap((field) => problem(field, errors[field]));
  const lines = errors.lines;
  const linesMessage = lines?.message ?? lines?.root?.message;
  const perLine = Array.isArray(lines)
    ? lines.flatMap(
        (line: FieldErrors<PurchaseRequestFormValues['lines'][number]> | undefined, index) =>
          LINE_FIELDS.flatMap((field) => problem(`lines[${index}].${field}`, line?.[field])),
      )
    : [];
  return [
    ...header,
    ...(linesMessage ? [{ field: 'lines', message: linesMessage }] : []),
    ...perLine,
  ];
}

export function problemAt(problems: FieldProblem[], field: string): string | undefined {
  return problems.find((each) => each.field === field)?.message;
}

export function problemMap(problems: FieldProblem[]): Record<string, string> {
  return Object.fromEntries(problems.map((each) => [each.field, each.message]));
}

/** 409 not_ready: every gate problem of the submission, grouped by record (032 FR-021). */
export function notReadyLines(error: ApiError): string[] {
  return error.missing.flatMap((group) =>
    group.fields.map((field) => `${group.record.name}: ${field.reason || field.label}`),
  );
}
