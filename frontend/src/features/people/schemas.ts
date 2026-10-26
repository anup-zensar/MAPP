import { z } from 'zod';
import type { Person, PersonDetails, PersonUpdate, Team, TeamCreate, TeamUpdate } from './api';

// The API checks every field of these forms and says what is wrong; nothing is checked first.
export const personSchema = z.object({
  name: z.string(),
  email: z.string(),
  jobTitle: z.string(),
  teamIds: z.array(z.string()),
});

export type PersonValues = z.infer<typeof personSchema>;

export const teamSchema = z.object({
  name: z.string(),
  description: z.string(),
  colour: z.string(),
  memberIds: z.array(z.string()),
});

export type TeamValues = z.infer<typeof teamSchema>;

export const detailsSchema = z.object({ name: z.string(), jobTitle: z.string() });

export type DetailsValues = z.infer<typeof detailsSchema>;

export const preferencesSchema = z.object({ displayCurrency: z.string(), dateFormat: z.string() });

export type PreferencesValues = z.infer<typeof preferencesSchema>;

export function personValuesOf(person?: Person | PersonDetails): PersonValues {
  return {
    name: person?.name ?? '',
    email: person?.email ?? '',
    jobTitle: person?.jobTitle ?? '',
    teamIds: person?.teams.map((team) => team.id) ?? [],
  };
}

/** Only the changed fields and the version (D-210): after a conflict, Save replaces only these. */
export function personChanges(
  start: PersonValues,
  values: PersonValues,
  version: number,
): PersonUpdate {
  return {
    ...(values.name !== start.name ? { name: values.name } : {}),
    ...(values.email !== start.email ? { email: values.email } : {}),
    ...(values.jobTitle !== start.jobTitle ? { jobTitle: values.jobTitle } : {}),
    ...(values.teamIds.join() !== start.teamIds.join() ? { teamIds: values.teamIds } : {}),
    version,
  };
}

export function teamValuesOf(team?: Team): TeamValues {
  return {
    name: team?.name ?? '',
    description: team?.description ?? '',
    colour: team?.colour ?? '',
    memberIds: team?.members.map((member) => member.id) ?? [],
  };
}

/** Without a colour, the API takes the one the fewest teams use. */
export function newTeam({ name, description, colour, memberIds }: TeamValues): TeamCreate {
  return { name, description, memberIds, ...(colour ? { colour } : {}) };
}

export function teamChanges(start: TeamValues, values: TeamValues, version: number): TeamUpdate {
  return {
    ...(values.name !== start.name ? { name: values.name } : {}),
    ...(values.description !== start.description ? { description: values.description } : {}),
    ...(values.colour !== start.colour ? { colour: values.colour } : {}),
    ...(values.memberIds.join() !== start.memberIds.join() ? { memberIds: values.memberIds } : {}),
    version,
  };
}
