import { useCurrentUser } from '@/session/useSession';
import { PersonPicker } from '@/ui/PersonPicker/PersonPicker';
import type { PersonPickerProps } from '@/ui/PersonPicker/PersonPicker';
import { FieldLoadFailed, FieldLoading } from './FieldLoadStates';
import { usePeople } from './peopleApi';

/** Omit for each member of a union, so `multiple` keeps choosing the value's type. */
type DistributiveOmit<T, K extends PropertyKey> = T extends unknown ? Omit<T, K> : never;

/** PersonPicker's props, without the people and the signed-in person, which it loads itself. */
export type PersonFieldProps = DistributiveOmit<PersonPickerProps, 'people' | 'currentUserId'>;

/**
 * A person field for any form (feature 003, US3): PersonPicker with everyone loaded from the API
 * and the signed-in person first. While loading, a field-sized grey shape; on failure "People
 * couldn't be loaded." with Try again.
 *
 *   <PersonField label="Owner" name="ownerId" teamIds={teamIds} value={ownerId} onChange={setOwnerId} />
 */
export function PersonField(props: PersonFieldProps) {
  const people = usePeople();
  const user = useCurrentUser();

  if (people.isPending) {
    return <FieldLoading label={props.label} />;
  }
  if (people.isError) {
    return (
      <FieldLoadFailed
        label={props.label}
        message="People couldn't be loaded."
        onRetry={() => void people.refetch()}
      />
    );
  }
  return <PersonPicker {...props} people={people.data.items} currentUserId={user.id} />;
}
