import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useTeams } from '../api';
import { usePersonRoles, useRoles } from './api';
import { RolesForm } from './RolesForm';

export interface PersonRolesPanelProps {
  personId: string;
  /** The person is the one signed in (taking away your own Administrator role is checked). */
  isYou: boolean;
  onClose: () => void;
}

/**
 * A person's roles (US3 scenarios 2, 4, 5; FR-017, FR-021, FR-022): one check box per role with its
 * description, and for each ticked role where it applies — All markets, My teams or Chosen teams.
 * The summary of what the roles allow follows every change. Taking away approving, or your own
 * Administrator role, is checked first and its warnings confirmed (Keep editing · Save changes).
 */
export function PersonRolesPanel({ personId, isYou, onClose }: PersonRolesPanelProps) {
  const person = usePersonRoles(personId);
  const roles = useRoles();
  const teams = useTeams();

  if (person.data && roles.data) {
    return (
      <RolesForm
        person={person.data}
        roles={roles.data.items}
        allTeams={teams.data?.items ?? person.data.teams}
        isYou={isYou}
        onClose={onClose}
      />
    );
  }

  const error = person.error ?? roles.error;
  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title="Roles"
      footer={<Button onClick={onClose}>Back to Roles and access</Button>}
    >
      {error ? (
        <ErrorState
          error={error}
          onRetry={() => {
            void person.refetch();
            void roles.refetch();
          }}
        />
      ) : (
        <Skeleton lines={8} height="20px" />
      )}
    </SidePanel>
  );
}
