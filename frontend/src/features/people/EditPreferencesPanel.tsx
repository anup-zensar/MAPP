import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { DATE_FORMATS } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { useDisplayCurrencies, useUpdateDisplayPreferences } from './api';
import type { DisplayCurrencyOption } from './api';
import { formFailure } from './feedback';
import { FieldLoading } from './FieldLoadStates';
import { preferencesSchema } from './schemas';
import type { PreferencesValues } from './schemas';
import { currencyLabel, DISPLAY_CURRENCY_NOTE } from './texts';

const FIELDS = ['displayCurrency', 'dateFormat'] as const;

/** "EUR — Euro"; an inactive one the person keeps gets " (inactive)" from ChoiceField. */
function currencyChoice(currency: DisplayCurrencyOption): ChoiceOption {
  return { id: currency.code, label: currencyLabel(currency), active: currency.active };
}

/** The four date formats, each with 12 March 2026 written in it (FR-024). */
const DATE_FORMAT_CHOICES: ChoiceOption[] = DATE_FORMATS.map((format) => ({
  id: format.value,
  label: format.label,
  active: true,
}));

/**
 * "Edit display preferences" (US4-3, US4-4, FR-023, FR-024): the display currency and the date
 * format, saved together with PATCH /api/v1/me/preferences. Afterwards dates use the new format at
 * once and every list is asked again, so amounts come converted by the API (SC-006).
 */
export function EditPreferencesPanel({ onClose }: { onClose: () => void }) {
  const format = useFormatters();
  const { show } = useToast();
  const currencies = useDisplayCurrencies();
  const updatePreferences = useUpdateDisplayPreferences();
  // A field may hold "" ("Choose…"); the API then says what to choose.
  const form = useForm<PreferencesValues>({
    resolver: zodResolver(preferencesSchema),
    defaultValues: { displayCurrency: format.displayCurrency, dateFormat: format.dateFormat },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save({ displayCurrency, dateFormat }: PreferencesValues) {
    setSaving(true);
    try {
      // Both preferences in one request: one save, one meaning (D-068).
      await updatePreferences.mutateAsync({ displayCurrency, dateFormat });
      show({ message: 'Your preferences saved' });
      onClose();
    } catch (error) {
      const result = formFailure(error, 'profile', format.dateFormat);
      showServerProblems(result.problems);
      setFailure(result.failure);
    } finally {
      setSaving(false);
    }
  }

  return (
    <FormPanel
      title="Edit display preferences"
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(save, onInvalid)()}
      onClose={onClose}
    >
      {currencies.isPending && <FieldLoading label="Display currency" />}
      {currencies.isError && (
        <ErrorState error={currencies.error} onRetry={() => void currencies.refetch()} />
      )}
      {currencies.data && (
        <FormField
          control={form.control}
          name="displayCurrency"
          render={(field) => (
            <ChoiceField
              label="Display currency"
              hint={DISPLAY_CURRENCY_NOTE}
              options={currencies.data.items.map(currencyChoice)}
              {...field}
            />
          )}
        />
      )}
      <FormField
        control={form.control}
        name="dateFormat"
        render={(field) => (
          <ChoiceField label="Date format" options={DATE_FORMAT_CHOICES} {...field} />
        )}
      />
    </FormPanel>
  );
}
