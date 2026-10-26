import { TeamPicker } from '@/ui/TeamPicker/TeamPicker';
import type { ComponentProps } from 'react';
import { FieldLoadFailed, FieldLoading } from './FieldLoadStates';
import { useTeams } from './api';

/** TeamPicker's props, without the teams, which it loads itself. */
export type TeamFieldProps = Omit<ComponentProps<typeof TeamPicker>, 'teams'>;

/**
 * A teams field for any form (feature 003; a person's teams, a plan's teams): TeamPicker with the
 * teams loaded from the API. While loading, a field-sized grey shape; on failure "Teams couldn't
 * be loaded." with Try again.
 */
export function TeamField(props: TeamFieldProps) {
  const teams = useTeams();

  if (teams.isPending) {
    return <FieldLoading label={props.label} />;
  }
  if (teams.isError) {
    return (
      <FieldLoadFailed
        label={props.label}
        message="Teams couldn't be loaded."
        onRetry={() => void teams.refetch()}
      />
    );
  }
  return <TeamPicker {...props} teams={teams.data.items} />;
}
