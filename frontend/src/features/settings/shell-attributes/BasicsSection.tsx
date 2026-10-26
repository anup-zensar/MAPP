import { useFormContext, useWatch } from 'react-hook-form';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { FormSection } from '@/ui/FormSection/FormSection';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { chooseLevel, chooseType } from './attribute-choices';
import {
  LEVEL_WORDS,
  LEVELS,
  lockReason,
  MARKING_WORDS,
  markingsFor,
  TYPE_WORDS,
  typesFor,
} from './attribute-words';
import type { ShellAttribute } from './api';
import { FixedField } from './FixedField';
import type { AttributeValues } from './schemas';
import styles from './ShellAttributes.module.css';

const YES_NO_HINT = 'Always optional; shown as a tick box that defaults to No.';
const SPLIT_HINT =
  "People split the record's cost over the chosen values; the shares must add up to 100 %.";

/** The editor's Basics: label, help text, level, type and marking (FR-016, FR-017). */
export function BasicsSection({ attribute }: { attribute?: ShellAttribute }) {
  const form = useFormContext<AttributeValues>();
  const [level, type] = useWatch({ control: form.control, name: ['level', 'type'] });
  const levelLock = attribute && lockReason(attribute, 'level');
  const typeLock = attribute && lockReason(attribute, 'type');

  return (
    <FormSection title="Basics">
      <FormField
        control={form.control}
        name="label"
        render={(field) => <TextField label="Label" marking="Required" maxLength={80} {...field} />}
      />
      <FormField
        control={form.control}
        name="helpText"
        render={(field) => (
          <TextAreaField
            label="Help text"
            hint="Shown as an ⓘ tip next to the field on forms."
            maxLength={300}
            rows={3}
            {...field}
          />
        )}
      />
      {levelLock ? (
        <FixedField label="Level" value={LEVEL_WORDS[level]}>
          <LockedNote>{levelLock}</LockedNote>
        </FixedField>
      ) : (
        <FormField
          control={form.control}
          name="level"
          render={(field) => (
            <ChoiceField
              label="Level"
              marking="Required"
              options={LEVELS.map((each) => ({ id: each, label: LEVEL_WORDS[each], active: true }))}
              {...field}
              onChange={(next) => chooseLevel(form, next)}
            />
          )}
        />
      )}
      {typeLock && attribute ? (
        <FixedField label="Type" value={TYPE_WORDS[attribute.type]}>
          <LockedNote>{typeLock}</LockedNote>
        </FixedField>
      ) : (
        <FormField
          control={form.control}
          name="type"
          render={(field) => (
            <ChoiceField
              label="Type"
              marking="Required"
              hint={type === 'multipleChoiceSplit' ? SPLIT_HINT : undefined}
              options={typesFor(level).map((each) => ({
                id: each,
                label: TYPE_WORDS[each],
                active: true,
              }))}
              {...field}
              onChange={(next) => chooseType(form, next)}
            />
          )}
        />
      )}
      {type === 'yesNo' ? (
        <FixedField label="Marking" value={MARKING_WORDS.optional}>
          <p className={styles.hint}>{YES_NO_HINT}</p>
        </FixedField>
      ) : (
        <FormField
          control={form.control}
          name="marking"
          render={(field) => (
            <ChoiceField
              label="Marking"
              marking="Required"
              options={markingsFor(level).map((each) => ({
                id: each,
                label: MARKING_WORDS[each],
                active: true,
              }))}
              {...field}
            />
          )}
        />
      )}
    </FormSection>
  );
}
