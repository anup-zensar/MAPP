import type { ReactNode } from 'react';
import { ViewOnlyNote } from '@/ui/ViewOnlyNote/ViewOnlyNote';
import type { Permission } from '@/lib/access';
import { useCanChange } from './useCanChange';

export interface CanChangeProps {
  /** What a role of the person must include to change this list (feature 014, Table A). */
  permission?: Permission;
  /** The change actions: shown only to people who may change the list. */
  children: ReactNode;
  /** What shows instead, such as a Close button in a details panel's footer. */
  otherwise?: ReactNode;
}

/**
 * The change actions of a Settings page (New, Edit, the row menu's changes), shown only to people
 * whose roles may change the list (feature 014, FR-011). Everyone else reads the list and sees one
 * View only note instead (SettingsViewOnlyNote). The server refuses the changes anyway.
 */
export function CanChange({ permission = 'settings.manage', children, otherwise }: CanChangeProps) {
  return useCanChange(permission) ? children : (otherwise ?? null);
}

/**
 * The one View only note of a Settings page for people who may read but not change it (feature
 * 014, FR-011, D-230): under the page header, in place of a page of disabled buttons.
 */
export function SettingsViewOnlyNote({
  permission = 'settings.manage',
}: {
  permission?: Permission;
}) {
  return useCanChange(permission) ? null : <ViewOnlyNote area="settings" />;
}
