import { useParams, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePageTitle } from '@/ui/usePageTitle';
import { LEVEL_WORDS } from './attribute-words';
import { useShellAttribute } from './api';
import { AttributeForm } from './AttributeForm';
import { levelOf } from './schemas';
import { AttributeNotFound } from './ShellAttributePage';

export interface ShellAttributeEditorPageProps {
  /** create: /settings/shell-attributes/new; edit: /settings/shell-attributes/:attributeId/edit. */
  mode: 'create' | 'edit';
}

/**
 * The Shell attribute editor (feature 002, US2, US3, FR-016 – FR-023), a full page for create and
 * edit: Basics, Place on the form, Depends on and condition and — on create, or when an attribute
 * starts depending on another — its options with their links; locked fields with their reason;
 * every problem at once; the impact check before a change of marking or type; unsaved changes
 * protected.
 */
export function ShellAttributeEditorPage({ mode }: ShellAttributeEditorPageProps) {
  return mode === 'create' ? <CreateAttribute /> : <EditAttribute />;
}

/** /settings/shell-attributes/new?level=activity */
function CreateAttribute() {
  usePageTitle('New Shell attribute');
  const [params] = useSearchParams();
  const level = levelOf(params) ?? 'plan';

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: `/settings/shell-attributes?level=${level}` },
          { label: 'New Shell attribute' },
        ]}
        title="New Shell attribute"
      />
      <AttributeForm level={level} />
    </>
  );
}

/** /settings/shell-attributes/:attributeId/edit */
function EditAttribute() {
  const { attributeId = '' } = useParams();
  const attribute = useShellAttribute(attributeId);
  usePageTitle(attribute.data ? `Edit ${attribute.data.label}` : 'Edit Shell attribute');

  if (attribute.data) {
    const { id, label, level } = attribute.data;
    return (
      <>
        <PageHeader
          breadcrumb={[
            { label: 'Settings', to: '/settings' },
            { label: 'Shell attributes', to: '/settings/shell-attributes' },
            {
              label: `${LEVEL_WORDS[level]} form`,
              to: `/settings/shell-attributes?level=${level}`,
            },
            { label, to: `/settings/shell-attributes/${id}` },
            { label: 'Edit' },
          ]}
          title={`Edit Shell attribute '${label}'`}
        />
        {/* The edit-form rule: the attribute is copied into the form once, when it opens. */}
        <AttributeForm key={id} level={level} attribute={attribute.data} />
      </>
    );
  }

  const notFound = attribute.error instanceof ApiError && attribute.error.kind === 'notFound';
  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: '/settings/shell-attributes' },
          { label: 'Edit' },
        ]}
        title="Edit Shell attribute"
      />
      <Card>
        {notFound ? (
          <AttributeNotFound />
        ) : attribute.isError ? (
          <ErrorState error={attribute.error} onRetry={() => attribute.refetch()} />
        ) : (
          <Skeleton lines={6} height="18px" />
        )}
      </Card>
    </>
  );
}
