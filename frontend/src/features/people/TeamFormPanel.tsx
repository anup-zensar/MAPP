import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestTeam, useCheckTeamImpact, useCreateTeam, useUpdateTeam } from './api';
import type { ImpactWarning, Team, TeamUpdate } from './api';
import { formFailure } from './feedback';
import { OwnerImpactDialog } from './OwnerImpactDialog';
import styles from './panels.module.css';
import { PersonField } from './PersonField';
import { removedIds } from './removed-ids';
import { newTeam, teamChanges, teamSchema, teamValuesOf } from './schemas';
import type { TeamValues } from './schemas';
import { COLOURS } from './team-colours';

const FIELDS = ['name', 'description', 'colour', 'memberIds'] as const;

const COLOUR_CHOICES = COLOURS.map((option) => ({ ...option, active: true }));

export interface TeamFormPanelProps {
  /** The team to edit; without one, the panel creates a new team. */
  team?: Team;
  onClose: () => void;
}

/**
 * "New team" or "Edit team" (US2-1, US2-4, US2-5): name, description, colour and members. When
 * members were removed, the API's impact check comes first and its warnings are confirmed in
 * OwnerImpactDialog (plan "Save flow").
 */
export function TeamFormPanel({ team, onClose }: TeamFormPanelProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createTeam = useCreateTeam();
  const updateTeam = useUpdateTeam();
  const checkImpact = useCheckTeamImpact();
  // The edit-form rule of feature 001: the team is copied into the form once, when it opens.
  const start = teamValuesOf(team);
  const form = useForm<TeamValues>({
    resolver: zodResolver(teamSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(team?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [warnings, setWarnings] = useState<ImpactWarning[]>([]);

  function done(saved: Team, verb: 'created' | 'saved') {
    show({
      message: `Team '${saved.name}' ${verb}`,
      action: {
        label: 'View',
        altText: 'Open it from the Teams list',
        onClick: () => void navigate(`/settings/teams/${encodeURIComponent(saved.id)}`),
      },
    });
    onClose();
  }

  async function fail(error: unknown) {
    const result = formFailure(error, 'team', dateFormat);
    showServerProblems(result.problems);
    setFailure(result.failure);
    if (result.conflict && team) {
      // Typed values stay; the form takes the version with the other person's change, so Save
      // again replaces it on purpose.
      const latest = await fetchLatestTeam(queryClient, team.id);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  async function update(id: string, body: TeamUpdate, values: TeamValues) {
    if (removedIds(start.memberIds, values.memberIds).length > 0) {
      const impact = await checkImpact.mutateAsync({ id, body });
      if (impact.warnings.length > 0) {
        showServerProblems([]);
        setFailure(null);
        setWarnings(impact.warnings);
        return;
      }
    }
    done(await updateTeam.mutateAsync({ id, body }), 'saved');
  }

  async function save(values: TeamValues) {
    setSaving(true);
    try {
      if (team) {
        await update(team.id, teamChanges(start, values, version), values);
      } else {
        done(await createTeam.mutateAsync(newTeam(values)), 'created');
      }
    } catch (error) {
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveAfterWarnings() {
    if (!team) {
      return;
    }
    setSaving(true);
    try {
      const body = teamChanges(start, form.getValues(), version);
      const saved = await updateTeam.mutateAsync({ id: team.id, body });
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
        title={team ? 'Edit team' : 'New team'}
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
          render={(field) => <TextField label="Name" marking="Required" {...field} />}
        />
        <FormField
          control={form.control}
          name="description"
          render={(field) => (
            <TextAreaField label="Description" hint="Up to 300 characters." rows={3} {...field} />
          )}
        />
        <FormField
          control={form.control}
          name="colour"
          render={(field) => (
            <ChoiceField
              label="Colour"
              hint="Shown as a dot next to the team's name."
              options={COLOUR_CHOICES}
              note={colourNote(field.value)}
              {...field}
            />
          )}
        />
        <FormField
          control={form.control}
          name="memberIds"
          render={(field) => (
            <PersonField
              label="Members"
              hint="Only active people can be added."
              multiple
              {...field}
            />
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

/** The chosen colour as its dot and name; nothing before one is chosen. */
function colourNote(colour: string) {
  const chosen = COLOURS.find((option) => option.id === colour);
  return (
    chosen && (
      <span className={styles.colour} aria-hidden="true">
        <span className={styles.dot} style={{ background: `var(--team-${chosen.id})` }} />
        {chosen.label}
      </span>
    )
  );
}
