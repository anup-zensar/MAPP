import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestPerson, useCheckPersonImpact, useCreatePerson, useUpdatePerson } from './api';
import type { ImpactWarning, Person, PersonDetails, PersonUpdate } from './api';
import { formFailure } from './feedback';
import { OwnerImpactDialog } from './OwnerImpactDialog';
import { removedIds } from './removed-ids';
import { personChanges, personSchema, personValuesOf } from './schemas';
import type { PersonValues } from './schemas';
import { TeamField } from './TeamField';

const FIELDS = ['name', 'email', 'jobTitle', 'teamIds'] as const;

export interface PersonFormPanelProps {
  /** The person to edit; without one, the panel adds a new person. */
  person?: Person | PersonDetails;
  onClose: () => void;
}

/**
 * "New person" or "Edit person" (US1-1, US1-2, US1-4): name, e-mail, job title and teams. When
 * teams were removed, the API's impact check comes first and its warnings are confirmed in
 * OwnerImpactDialog (plan "Save flow").
 */
export function PersonFormPanel({ person, onClose }: PersonFormPanelProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createPerson = useCreatePerson();
  const updatePerson = useUpdatePerson();
  const checkImpact = useCheckPersonImpact();
  // The edit-form rule of feature 001: the person is copied into the form once, when it opens.
  const start = personValuesOf(person);
  const form = useForm<PersonValues>({
    resolver: zodResolver(personSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(person?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [warnings, setWarnings] = useState<ImpactWarning[]>([]);

  function done(saved: PersonDetails, verb: 'created' | 'saved') {
    show({
      message: `Person '${saved.name}' ${verb}`,
      action: {
        label: 'View',
        altText: 'Open them from the People list',
        onClick: () => void navigate(`/settings/people/${encodeURIComponent(saved.id)}`),
      },
    });
    onClose();
  }

  async function fail(error: unknown) {
    const result = formFailure(error, 'person', dateFormat);
    showServerProblems(result.problems);
    setFailure(result.failure);
    if (result.conflict && person) {
      // Typed values stay; the form takes the version with the other person's change, so Save
      // again replaces it on purpose.
      const latest = await fetchLatestPerson(queryClient, person.id);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  async function update(id: string, body: PersonUpdate, values: PersonValues) {
    if (removedIds(start.teamIds, values.teamIds).length > 0) {
      const impact = await checkImpact.mutateAsync({ id, body });
      if (impact.warnings.length > 0) {
        showServerProblems([]);
        setFailure(null);
        setWarnings(impact.warnings);
        return;
      }
    }
    done(await updatePerson.mutateAsync({ id, body }), 'saved');
  }

  async function save(values: PersonValues) {
    setSaving(true);
    try {
      if (person) {
        await update(person.id, personChanges(start, values, version), values);
      } else {
        done(await createPerson.mutateAsync(values), 'created');
      }
    } catch (error) {
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveAfterWarnings() {
    if (!person) {
      return;
    }
    setSaving(true);
    try {
      const body = personChanges(start, form.getValues(), version);
      const saved = await updatePerson.mutateAsync({ id: person.id, body });
      setWarnings([]);
      done(saved, 'saved');
    } catch (error) {
      setWarnings([]);
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <FormPanel
        title={person ? 'Edit person' : 'New person'}
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
            <TextField label="Name" marking="Required" autoComplete="off" {...field} />
          )}
        />
        <FormField
          control={form.control}
          name="email"
          render={(field) => (
            <TextField
              label="E-mail"
              type="email"
              marking="Required"
              autoComplete="off"
              hint={
                person
                  ? 'Their sign-in name — the next sign-in uses the new e-mail.'
                  : 'Their sign-in name. They can sign in straight away with the demo password.'
              }
              {...field}
            />
          )}
        />
        <FormField
          control={form.control}
          name="jobTitle"
          render={(field) => <TextField label="Job title" autoComplete="off" {...field} />}
        />
        <FormField
          control={form.control}
          name="teamIds"
          render={(field) => (
            <TeamField label="Teams" hint="Only active teams can be added." {...field} />
          )}
        />
      </FormPanel>
      <OwnerImpactDialog
        warnings={warnings}
        busy={saving}
        onKeepEditing={() => setWarnings([])}
        onSave={saveAfterWarnings}
      />
    </>
  );
}
