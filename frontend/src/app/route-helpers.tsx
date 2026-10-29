import type { ReactNode } from 'react';
import type { Permission } from '@/session/access';
import type { PageHandle } from '@/session/page-handle';
import { RequirePermission } from '@/session/RequirePermission';

export function titled(title: string): { handle: PageHandle } {
  return { handle: { title } };
}

// A page only some roles open; anyone else sees the no-access page at its address.
export function gated(permission: Permission, page: ReactNode): ReactNode {
  return <RequirePermission permission={permission}>{page}</RequirePermission>;
}
