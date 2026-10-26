import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import type { components } from '@/api/generated/schema';
import { linkedToHint } from './attributeWords';

type FormAttribute = components['schemas']['FormAttribute'];

export interface LinkedToFieldProps {
  /** The field's name, as the API names its problems: "parentOptionIds", "optionLinks.SO-1012". */
  name: string;
  /** The attribute the option's attribute depends on, with its options (from the level's form). */
  parent: FormAttribute;
  /** The parent options the option is linked to. */
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
}

/**
 * "Linked to" of an option of a dependent attribute (FR-022): the parent options with which it is
 * offered on forms. Only active parent options can be newly linked; inactive ones already linked
 * stay, marked "(inactive)". More than 10 parent options become a searchable list.
 */
export function LinkedToField({ name, parent, value, onChange, error }: LinkedToFieldProps) {
  return (
    <ChoiceField
      label="Linked to"
      name={name}
      marking="Required"
      hint={linkedToHint(parent.label)}
      multiple
      options={parent.options.map((option) => ({
        id: option.id,
        label: option.label,
        active: option.active,
      }))}
      value={value}
      onChange={onChange}
      error={error}
    />
  );
}
