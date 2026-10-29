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
  /** The only teams the person may add or remove; chosen teams outside them are kept as they are. */
  offered?: string[];
}

const INACTIVE_TIP = "Inactive teams can't be chosen again. Keep it or remove it.";

function keptTeamsText(names: string[]): string {
  return `Kept, outside your teams: ${names.join(', ')}.`;
}

export function TeamPicker({ name, teams, value, onChange, offered, ...frame }: TeamPickerProps) {
  const byId = new Map(teams.map((team) => [team.id, team]));
  const mayChange = (id: string) => !offered || offered.includes(id);
  const kept = value.filter((id) => byId.has(id) && !mayChange(id));

  return (
    <Combobox
      {...frame}
      name={name}
      multiple
      options={teams
        .filter((team) => team.active && mayChange(team.id))
        .map((team) => ({ id: team.id, label: team.name, content: <TeamPill team={team} /> }))}
      value={value.filter((id) => byId.has(id) && mayChange(id))}
      onChange={(ids) => onChange([...kept, ...ids])}
      note={kept.length > 0 ? keptTeamsText(kept.map((id) => byId.get(id)?.name ?? id)) : undefined}
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
