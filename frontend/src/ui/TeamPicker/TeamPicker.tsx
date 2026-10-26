import { Combobox } from '../Combobox/Combobox';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import { TeamPill } from '../TeamPill/TeamPill';
import type { TeamRef } from '../TeamPill/TeamPill';

export interface TeamPickerProps extends FieldFrameProps {
  /** The field's name and the input's id, so FormErrorSummary can link to it. */
  name: string;
  /** Every team, active or not (inactive teams are never offered, only kept). */
  teams: TeamRef[];
  value: string[];
  onChange: (value: string[]) => void;
}

const INACTIVE_TIP = "Inactive teams can't be chosen again. Keep it or remove it.";

/**
 * Teams of a person or a plan (feature 003): searchable, several at once, shown as team pills.
 * Only active teams are offered; a chosen team that became inactive is kept and explained.
 */
export function TeamPicker({ name, teams, value, onChange, ...frame }: TeamPickerProps) {
  const byId = new Map(teams.map((team) => [team.id, team]));

  return (
    <Combobox
      {...frame}
      name={name}
      multiple
      options={teams
        .filter((team) => team.active)
        .map((team) => ({ id: team.id, label: team.name, content: <TeamPill team={team} /> }))}
      value={value.filter((id) => byId.has(id))}
      onChange={onChange}
      renderChip={(id) => {
        const team = byId.get(id);
        if (!team) {
          return { label: id };
        }
        return {
          label: team.name,
          content: <TeamPill team={team} />,
          tip: team.active ? undefined : INACTIVE_TIP,
        };
      }}
      placeholder="Search teams"
      countText={(count) => (count === 1 ? '1 team' : `${count} teams`)}
      noMatchText={(query) => `No team matches '${query}'.`}
    />
  );
}
