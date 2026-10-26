import { useParams, useSearchParams } from 'react-router';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePageTitle } from '@/ui/usePageTitle';
import { usePermissionCatalogue, useRole } from './api';
import { rolePath, rolesTabPath } from './access-paths';
import { RoleForm } from './RoleForm';
import { copyOfParam, EMPTY_ROLE } from './schemas';
import type { RoleValues } from './schemas';

/**
 * New role (empty, or `?copyOf=RL-…` filled in from another role as "Copy of …") at
 * /settings/access/roles/new, and Edit role at /settings/access/roles/:roleId/edit (US4, FR-018):
 * name, description, default scope and the permissions by group. Unsaved changes are protected.
 */
export function RoleFormPage() {
  const { roleId } = useParams();
  const [params] = useSearchParams();
  const copyOf = copyOfParam(params);
  const sourceId = roleId ?? copyOf;
  const source = useRole(sourceId);
  const catalogue = usePermissionCatalogue();
  const editing = Boolean(roleId);
  const title = editing ? `Edit role${source.data ? ` '${source.data.name}'` : ''}` : 'New role';
  usePageTitle(editing ? 'Edit role' : 'New role');

  const breadcrumb = [
    { label: 'Settings', to: '/settings' },
    { label: 'Roles and access', to: rolesTabPath() },
    ...(editing && source.data ? [{ label: source.data.name, to: rolePath(source.data.id) }] : []),
    { label: editing ? 'Edit' : 'New role' },
  ];

  const waitingForSource = Boolean(sourceId) && !source.data;
  if (waitingForSource || !catalogue.data) {
    const error = (sourceId ? source.error : null) ?? catalogue.error;
    return (
      <>
        <PageHeader breadcrumb={breadcrumb} title={title} />
        <Card>
          {error ? (
            <ErrorState
              error={error}
              onRetry={() => {
                void source.refetch();
                void catalogue.refetch();
              }}
            />
          ) : (
            <Skeleton lines={8} height="20px" />
          )}
        </Card>
      </>
    );
  }

  const start: RoleValues = source.data
    ? {
        name: editing ? source.data.name : `Copy of ${source.data.name}`,
        description: source.data.description,
        defaultScope: source.data.defaultScope,
        permissions: source.data.permissions,
      }
    : EMPTY_ROLE;

  return (
    <>
      <PageHeader breadcrumb={breadcrumb} title={title} />
      <RoleForm
        key={sourceId}
        role={editing ? source.data : undefined}
        start={start}
        catalogue={catalogue.data}
      />
    </>
  );
}
