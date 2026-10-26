import type { components } from '@/api/generated/schema';
import { Combobox } from '../Combobox/Combobox';
import type { ComboboxOption } from '../Combobox/Combobox';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import { PersonLabel } from '../PersonLabel/PersonLabel';

type Person = components['schemas']['Person'];

/** What the picker needs of a person (GET /api/v1/people answers these and more). */
export type PickerPerson = Pick<
  Person,
  'id' | 'name' | 'email' | 'jobTitle' | 'initials' | 'active' | 'teams'
>;

interface PersonPickerBase extends FieldFrameProps {
  /** The field's name and the input's id, so FormErrorSummary can link to it. */
  name: string;
  /** Everyone, active or not (inactive people are never offered, only kept). */
  people: PickerPerson[];
  /** The signed-in person: listed first with " (you)" when they qualify. */
  currentUserId: string;
  /**
   * Only active members of these teams are offered, each with the teams that make them qualify
   * ("Only members of the selected teams"). An empty list locks the field with `disabledReason`.
   */
  teamIds?: string[];
  /** Why the field is locked; by default the plan owner's reason when `teamIds` is empty. */
  disabledReason?: string;
  /** Said under the field when a chosen person is in none of `teamIds`. */
  notMemberText?: string;
}

export interface SinglePersonPickerProps extends PersonPickerBase {
  multiple?: false;
  /** The chosen person's id; "" for nobody. */
  value: string;
  onChange: (value: string) => void;
}

export interface MultiplePersonPickerProps extends PersonPickerBase {
  multiple: true;
  value: string[];
  onChange: (value: string[]) => void;
}

export type PersonPickerProps = SinglePersonPickerProps | MultiplePersonPickerProps;

const NO_TEAMS_REASON = "Choose teams first — the owner must belong to one of the plan's teams.";
const NOT_MEMBER = "Not a member of the plan's teams";
const INACTIVE_TIP = "Inactive people can't be chosen again. Keep them or choose someone else.";

/**
 * Every person field of MAPP (feature 003, FR-016 – FR-020): searchable by name, e-mail or
 * job title; only active people are offered, the signed-in person first; with `teamIds` only
 * members of those teams. A chosen person who became inactive is kept, labelled and explained.
 * Load the people with features/people's PersonField, which wraps this picker.
 */
export function PersonPicker(props: PersonPickerProps) {
  const {
    name,
    people,
    currentUserId,
    teamIds,
    disabledReason,
    notMemberText = NOT_MEMBER,
    label,
    marking,
    hint,
    tip,
    error,
    labelHidden,
  } = props;
  const chosen = props.multiple ? props.value : props.value ? [props.value] : [];
  const byId = new Map(people.map((person) => [person.id, person]));
  const qualifyingTeams = (person: PickerPerson) =>
    teamIds ? person.teams.filter((team) => teamIds.includes(team.id)) : [];

  const offered = people
    .filter((person) => person.active && (!teamIds || qualifyingTeams(person).length > 0))
    .sort(
      (a, b) =>
        Number(b.id === currentUserId) - Number(a.id === currentUserId) ||
        a.name.localeCompare(b.name, 'en', { sensitivity: 'base' }) ||
        a.email.localeCompare(b.email),
    );
  const sharedNames = new Set(
    offered.map((person) => person.name).filter((n, i, all) => all.indexOf(n) !== i),
  );

  const options: ComboboxOption[] = offered.map((person) => {
    const second = sharedNames.has(person.name)
      ? [person.jobTitle, person.email].filter(Boolean).join(' · ')
      : person.jobTitle || person.email;
    const teams = qualifyingTeams(person).map((team) => team.name);
    return {
      id: person.id,
      label: person.name,
      searchText: `${person.name} ${person.email} ${person.jobTitle}`,
      content: (
        <PersonLabel
          person={person}
          you={person.id === currentUserId}
          detail={[second, ...teams].filter(Boolean).join(' · ')}
        />
      ),
    };
  });

  const known = chosen.filter((id) => byId.has(id));
  const outsideTeams =
    teamIds !== undefined &&
    teamIds.length > 0 &&
    known.some((id) => qualifyingTeams(byId.get(id) as PickerPerson).length === 0);
  const locked = disabledReason ?? (teamIds?.length === 0 ? NO_TEAMS_REASON : undefined);

  return (
    <Combobox
      label={label}
      marking={marking}
      hint={hint}
      tip={tip}
      error={error}
      labelHidden={labelHidden}
      name={name}
      options={options}
      value={known}
      onChange={(ids) =>
        props.multiple ? props.onChange(ids) : props.onChange(ids[ids.length - 1] ?? '')
      }
      multiple={props.multiple}
      renderChip={(id) => {
        const person = byId.get(id) as PickerPerson;
        return {
          label: person.name,
          content: <PersonLabel person={person} you={id === currentUserId} />,
          tip: person.active ? undefined : INACTIVE_TIP,
        };
      }}
      placeholder="Search name, e-mail or job title"
      countText={(count) => (count === 1 ? '1 person' : `${count} people`)}
      noMatchText={(query) =>
        teamIds
          ? 'No members of the selected teams match. Add people to these teams in Settings › Teams.'
          : `No one matches '${query}'. Check the spelling, or add the person in Settings › People.`
      }
      disabledReason={locked}
      note={outsideTeams ? notMemberText : undefined}
    />
  );
}
