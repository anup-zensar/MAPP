import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useCurrentUser } from '@/session/useSession';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { DATE_FORMATS } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { formFailure } from './feedback';
import { FieldLoading } from './FieldLoadStates';
import {
  fetchLatestPerson,
  useDisplayCurrencies,
  useUpdateDisplayPreferences,
  useUpdateProfile,
} from './peopleApi';
import type { DisplayCurrencyOption } from './peopleApi';
import { currencyLabel, DISPLAY_CURRENCY_NOTE } from './texts';

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
 * "Edit your details" (US4-2, FR-022): the signed-in person's name and job title, saved with
 * PATCH /api/v1/me. The e-mail is not here: it is the sign-in name, changed in Settings › People.
 * The account button shows the new name as soon as it is saved.
 */
export function EditDetailsPanel({ onClose }: { onClose: () => void }) {
  const user = useCurrentUser();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const updateProfile = useUpdateProfile();
  // The edit-form rule of feature 001: the person is copied into the form once, when it opens.
  const [start] = useState(() => ({ name: user.name, jobTitle: user.jobTitle }));
  const [name, setName] = useState(start.name);
  const [jobTitle, setJobTitle] = useState(start.jobTitle);
  // Always sent by feature 003's API; 0 only for an older answer without it (D-191).
  const [version, setVersion] = useState(user.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save() {
    setSaving(true);
    try {
      await updateProfile.mutateAsync({ name, jobTitle, version });
      show({ message: 'Your details saved' });
      onClose();
    } catch (error) {
      const result = formFailure(error, 'profile', dateFormat);
      setProblems(result.problems);
      setFailure(result.failure);
      if (result.conflict) {
        // Typed values stay; the form takes the version with the other person's change, so Save
        // again replaces it on purpose.
        const latest = await fetchLatestPerson(queryClient, user.id);
        if (latest) {
          setVersion(latest.version);
        }
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <FormPanel
      title="Edit your details"
      isDirty={name !== start.name || jobTitle !== start.jobTitle}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={save}
      onClose={onClose}
    >
      <TextField
        label="Name"
        name="name"
        marking="Required"
        autoComplete="name"
        value={name}
        onChange={setName}
        error={problemOf(problems, 'name')}
      />
      <TextField
        label="Job title"
        name="jobTitle"
        autoComplete="organization-title"
        value={jobTitle}
        onChange={setJobTitle}
        error={problemOf(problems, 'jobTitle')}
      />
    </FormPanel>
  );
}

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
  const [start] = useState<{ displayCurrency: string; dateFormat: string }>(() => ({
    displayCurrency: format.displayCurrency,
    dateFormat: format.dateFormat,
  }));
  const [displayCurrency, setDisplayCurrency] = useState(start.displayCurrency);
  const [dateFormat, setDateFormat] = useState(start.dateFormat);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save() {
    setSaving(true);
    try {
      // Both preferences in one request: one save, one meaning (D-068).
      await updatePreferences.mutateAsync({ displayCurrency, dateFormat });
      show({ message: 'Your preferences saved' });
      onClose();
    } catch (error) {
      const result = formFailure(error, 'profile', format.dateFormat);
      setProblems(result.problems);
      setFailure(result.failure);
    } finally {
      setSaving(false);
    }
  }

  return (
    <FormPanel
      title="Edit display preferences"
      isDirty={displayCurrency !== start.displayCurrency || dateFormat !== start.dateFormat}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={save}
      onClose={onClose}
    >
      {currencies.isPending && <FieldLoading label="Display currency" />}
      {currencies.isError && (
        <ErrorState error={currencies.error} onRetry={() => void currencies.refetch()} />
      )}
      {currencies.data && (
        <ChoiceField
          label="Display currency"
          name="displayCurrency"
          hint={DISPLAY_CURRENCY_NOTE}
          options={currencies.data.items.map(currencyChoice)}
          value={displayCurrency}
          onChange={setDisplayCurrency}
          error={problemOf(problems, 'displayCurrency')}
        />
      )}
      <ChoiceField
        label="Date format"
        name="dateFormat"
        options={DATE_FORMAT_CHOICES}
        value={dateFormat}
        onChange={setDateFormat}
        error={problemOf(problems, 'dateFormat')}
      />
    </FormPanel>
  );
}
