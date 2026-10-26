import { SettingsCount } from '@/ui/SettingsCount/SettingsCount';
import { useRoles } from './api';

/** "5 roles" on the Roles and access card of the Settings page (014 plan, "Settings cards"). */
export function RolesCount() {
  const roles = useRoles();
  return (
    <SettingsCount
      loading={roles.isPending}
      count={roles.data?.items.length}
      text={(count) => (count === 1 ? '1 role' : `${count} roles`)}
    />
  );
}
