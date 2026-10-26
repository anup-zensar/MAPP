import { z } from 'zod';
import { HELP_MAX_LENGTH } from './api';
import { HELP_LEVELS, TOO_LONG } from './field-help-words';

/** Line breaks are not kept: the tip is one paragraph (data-model §2.5). */
export function oneParagraph(text: string): string {
  return text.replace(/\s*\n\s*/g, ' ').trim();
}

export const fieldHelpSchema = z.object({
  text: z.string().refine((text) => [...oneParagraph(text)].length <= HELP_MAX_LENGTH, TOO_LONG),
});

export type FieldHelpValues = z.infer<typeof fieldHelpSchema>;

const levelSchema = z.enum(HELP_LEVELS);
const shownSchema = z.enum(['all', 'edited']);

export type Shown = z.infer<typeof shownSchema>;

export interface HelpChoices {
  level: z.infer<typeof levelSchema>;
  shown: Shown;
  q: string;
  openKey: string;
}

/** The page's choices in the address; an unknown level or Show choice falls back to its default. */
export function helpChoicesOf(params: URLSearchParams): HelpChoices {
  const level = levelSchema.safeParse(params.get('level'));
  const shown = shownSchema.safeParse(params.get('show'));
  return {
    level: level.success ? level.data : 'plan',
    shown: shown.success ? shown.data : 'all',
    q: params.get('q') ?? '',
    openKey: params.get('key') ?? '',
  };
}
