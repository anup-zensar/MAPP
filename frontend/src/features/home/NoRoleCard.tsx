import { KeyRound } from 'lucide-react';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { NO_ROLE_TITLE, noRoleText } from './homeText';

export interface NoRoleCardProps {
  /** The active administrators' names, from the person's access ("Ask … to give you a role"). */
  administrators: string[];
}

/**
 * Home for a person without any role (014 edge case "No role", text 36): they may sign in, but
 * see no plans, so Home says why and whom to ask instead of showing empty cards.
 */
export function NoRoleCard({ administrators }: NoRoleCardProps) {
  return (
    <Card>
      <EmptyState
        icon={<KeyRound />}
        title={NO_ROLE_TITLE}
        text={noRoleText(administrators)}
        action={<LinkButton to="/profile">Open my profile</LinkButton>}
      />
    </Card>
  );
}
