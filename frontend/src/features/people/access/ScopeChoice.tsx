import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import { TeamPicker } from '@/ui/TeamPicker/TeamPicker';
import type { AccessScopeType, Role } from './api';
import { ADMINISTRATOR_SCOPE, SCOPE_WORDS } from './access-words';
import styles from './access.module.css';
import type { RoleDraft } from './role-drafts';

const CHOICES: AccessScopeType[] = ['all', 'myTeams', 'teams'];

export interface ScopeChoiceProps {
  role: Role;
  draft: RoleDraft;
  index: number;
  allTeams: TeamRef[];
  problems: FieldProblem[];
  onChange: (changes: Partial<RoleDraft>) => void;
}

/** Where a ticked role applies: All markets, My teams, or Chosen teams with the team picker. */
export function ScopeChoice({
  role,
  draft,
  index,
  allTeams,
  problems,
  onChange,
}: ScopeChoiceProps) {
  if (role.builtIn === 'administrator') {
    return (
      <div className={styles.scope}>
        <LockedNote>{ADMINISTRATOR_SCOPE}</LockedNote>
      </div>
    );
  }
  const scopeError = problemOf(problems, `roles.${index}.scope`);
  return (
    <fieldset className={styles.scope}>
      <legend className={styles.legend}>{`Where ${role.name} applies`}</legend>
      {CHOICES.map((type) => (
        <label key={type} className={styles.radio}>
          <input
            type="radio"
            name={`scope-${role.id}`}
            value={type}
            checked={draft.scope === type}
            onChange={() => onChange({ scope: type })}
          />
          {SCOPE_WORDS[type]}
        </label>
      ))}
      {draft.scope === 'teams' && (
        <TeamPicker
          label="Teams"
          name={`roles.${index}.teamIds`}
          teams={allTeams}
          value={draft.teamIds}
          onChange={(teamIds) => onChange({ teamIds })}
          error={problemOf(problems, `roles.${index}.teamIds`)}
        />
      )}
      {scopeError && <p className={styles.problem}>{scopeError}</p>}
    </fieldset>
  );
}
