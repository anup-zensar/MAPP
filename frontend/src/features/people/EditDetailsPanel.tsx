import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useCurrentUser } from '@/session/useSession';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestPerson, useUpdateProfile } from './api';
import { formFailure } from './feedback';
import { detailsSchema } from './schemas';
import type { DetailsValues } from './schemas';

const FIELDS = ['name', 'jobTitle'] as const;

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
  const form = useForm<DetailsValues>({
    resolver: zodResolver(detailsSchema),
    defaultValues: { name: user.name, jobTitle: user.jobTitle },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  // Always sent by feature 003's API; 0 only for an older answer without it (D-191).
  const [version, setVersion] = useState(user.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save({ name, jobTitle }: DetailsValues) {
    setSaving(true);
    try {
      await updateProfile.mutateAsync({ name, jobTitle, version });
      show({ message: 'Your details saved' });
      onClose();
    } catch (error) {
      const result = formFailure(error, 'profile', dateFormat);
      showServerProblems(result.problems);
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
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(save, onInvalid)()}
      onClose={onClose}
    >
      <FormField
        control={form.control}
        name="name"
        render={(field) => (
          <TextField label="Name" marking="Required" autoComplete="name" {...field} />
        )}
      />
      <FormField
        control={form.control}
        name="jobTitle"
        render={(field) => (
          <TextField label="Job title" autoComplete="organization-title" {...field} />
        )}
      />
    </FormPanel>
  );
}
