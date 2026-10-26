import { z } from 'zod';

const recordParamSchema = z.string();

/** The record ID of `?record=`, from a link in the activity log; '' for none. */
export function highlightedRecordOf(params: URLSearchParams): string {
  const parsed = recordParamSchema.safeParse(params.get('record'));
  return parsed.success ? parsed.data : '';
}
