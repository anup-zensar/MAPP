import { useFormContext, useWatch } from 'react-hook-form';
import { FormField } from '@/ui/FormField/FormField';
import { FormSection } from '@/ui/FormSection/FormSection';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { TextField } from '@/ui/TextField/TextField';
import { lockReason } from './attribute-words';
import type { ShellAttribute } from './api';
import { FixedField } from './FixedField';
import type { AttributeValues } from './schemas';

export interface PlaceSectionProps {
  attribute?: ShellAttribute;
  /** Every attribute of every level, for the sections the level already has. */
  allAttributes: ShellAttribute[];
}

/** Place on the form: the section, with the level's sections offered while typing (FR-019). */
export function PlaceSection({ attribute, allAttributes }: PlaceSectionProps) {
  const form = useFormContext<AttributeValues>();
  const [level, section] = useWatch({ control: form.control, name: ['level', 'section'] });
  const sectionLock = attribute && lockReason(attribute, 'section');
  const sections = [
    ...new Set(allAttributes.filter((each) => each.level === level).map((each) => each.section)),
  ];

  return (
    <FormSection title="Place on the form">
      {sectionLock ? (
        <FixedField label="Section" value={section}>
          <LockedNote>{sectionLock}</LockedNote>
        </FixedField>
      ) : (
        <FormField
          control={form.control}
          name="section"
          render={(field) => (
            <TextField
              label="Section"
              hint="Attributes of the same section are shown together; sections appear in the order of their first attribute."
              maxLength={60}
              suggestions={sections}
              {...field}
            />
          )}
        />
      )}
    </FormSection>
  );
}
