import { z } from 'zod';
import type { components } from '@/api/generated/schema';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { isChoiceType, LEVEL_WORDS, LEVELS, linkOptionText } from './attribute-words';
import type { AttributeLevel, FormAttribute } from './attribute-words';

type FormOption = components['schemas']['FormOption'];

const levelSchema = z.enum(LEVELS);

const optionRowSchema = z.object({
  key: z.number(),
  label: z.string(),
  parentOptionIds: z.array(z.string()),
});

/** One first option being typed; `key` keeps the row's field while rows move. */
export type OptionRow = z.infer<typeof optionRowSchema>;

const attributeValuesSchema = z.object({
  label: z.string(),
  helpText: z.string(),
  level: levelSchema,
  type: z.string(),
  marking: z.string(),
  section: z.string(),
  parentAttributeId: z.string(),
  conditionOptionIds: z.array(z.string()),
  options: z.array(optionRowSchema),
  optionLinks: z.record(z.string(), z.array(z.string())),
});

/** The editor's values: "" for a type, marking or Depends on not chosen yet. */
export type AttributeValues = z.infer<typeof attributeValuesSchema>;

export interface AttributeRules {
  isNew: boolean;
  parent?: FormAttribute;
  /** An edited attribute that starts depending on another: its options, each needing a link. */
  optionsToLink: FormOption[] | null;
  /** Every attribute, for the API's "already has a Shell attribute called" check (038 FR-030). */
  others?: { id: string; label: string; level: AttributeLevel }[];
  /** The edited attribute, which keeps its own label. */
  ownId?: string;
}

const MAX_HELP_TEXT = 300;

/** The editor's checks, with the problems' names as the API gives them ("options[2].label"). */
export function attributeSchema(rules: AttributeRules) {
  return attributeValuesSchema.superRefine((values, context) => {
    for (const problem of attributeProblems(values, rules)) {
      context.addIssue({ code: 'custom', path: [problem.field], message: problem.message });
    }
  });
}

/** Instant feedback, in the form's order and the API's words (FR-018, FR-022); the API checks all. */
export function attributeProblems(values: AttributeValues, rules: AttributeRules): FieldProblem[] {
  const found: FieldProblem[] = [];
  const label = values.label.trim();
  const taken = rules.others?.find(
    (other) =>
      other.id !== rules.ownId &&
      other.level === values.level &&
      other.label.trim().toLowerCase() === label.toLowerCase(),
  );
  if (label === '') {
    found.push({ field: 'label', message: 'Enter a label.' });
  } else if (taken) {
    const message = `The ${LEVEL_WORDS[values.level]} form already has a Shell attribute called '${taken.label}'.`;
    found.push({ field: 'label', message });
  }
  if ([...values.helpText].length > MAX_HELP_TEXT) {
    found.push({ field: 'helpText', message: `Use ${MAX_HELP_TEXT} characters or fewer.` });
  }
  if (values.type === '') {
    found.push({ field: 'type', message: 'Choose a type.' });
  }
  if (values.marking === '') {
    found.push({ field: 'marking', message: 'Choose a marking.' });
  }
  if (rules.isNew && isChoiceType(values.type)) {
    found.push(...optionProblems(values.options, rules.parent));
  }
  if (rules.parent) {
    found.push(...linkProblems(values.optionLinks, rules.optionsToLink ?? [], rules.parent));
  }
  return found;
}

function optionProblems(rows: OptionRow[], parent: FormAttribute | undefined): FieldProblem[] {
  if (rows.length === 0) {
    return [{ field: 'options', message: 'Add at least one option.' }];
  }
  const found: FieldProblem[] = [];
  const seen = new Set<string>();
  const reported = new Set<string>();
  rows.forEach((row, index) => {
    const label = row.label.trim();
    const field = `options[${index}].label`;
    if (label === '') {
      found.push({ field, message: 'Enter a label for every option.' });
    } else {
      const key = label.toLowerCase();
      if (seen.has(key) && !reported.has(key)) {
        found.push({ field, message: `'${label}' appears twice.` });
        reported.add(key);
      }
      seen.add(key);
    }
    if (parent && row.parentOptionIds.length === 0) {
      found.push({
        field: `options[${index}].parentOptionIds`,
        message: linkOptionText(label, parent.label),
      });
    }
  });
  return found;
}

function linkProblems(
  links: Record<string, string[]>,
  options: FormOption[],
  parent: FormAttribute,
): FieldProblem[] {
  return options
    .filter((option) => (links[option.id] ?? []).length === 0)
    .map((option) => ({
      field: `optionLinks.${option.id}`,
      message: linkOptionText(option.label, parent.label),
    }));
}

export function optionSchema(parentName: string | null) {
  return z
    .object({ label: z.string(), parentOptionIds: z.array(z.string()) })
    .superRefine((values, context) => {
      if (values.label.trim() === '') {
        context.addIssue({ code: 'custom', path: ['label'], message: 'Enter a label.' });
      }
      if (parentName !== null && values.parentOptionIds.length === 0) {
        context.addIssue({
          code: 'custom',
          path: ['parentOptionIds'],
          message: linkOptionText(values.label, parentName),
        });
      }
    });
}

export type OptionValues = z.infer<ReturnType<typeof optionSchema>>;

const statusSchema = z.enum(['active', 'inactive', 'all']);
const helpSchema = z.enum(['all', 'missing']);

export type StatusFilter = z.infer<typeof statusSchema>;
export type HelpFilter = z.infer<typeof helpSchema>;

/** The level of the address (`?level=`); null when it names none. */
export function levelOf(params: URLSearchParams): AttributeLevel | null {
  const parsed = levelSchema.safeParse(params.get('level'));
  return parsed.success ? parsed.data : null;
}

export function attributeChoicesOf(params: URLSearchParams) {
  const level = levelOf(params);
  const status = statusSchema.safeParse(params.get('status'));
  const help = helpSchema.safeParse(params.get('help'));
  return {
    level: level ?? 'plan',
    /** The address names a level; without one, the help filter may choose it. */
    levelGiven: level !== null,
    status: status.success ? status.data : 'active',
    q: params.get('q') ?? '',
    /** Only the attributes without help (016 FR-048), as Settings › Field help links here. */
    help: help.success ? help.data : 'all',
  };
}
