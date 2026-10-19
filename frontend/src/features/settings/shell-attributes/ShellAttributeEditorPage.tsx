import { BuildingPage } from '@/features/placeholders/BuildingPage';

export interface ShellAttributeEditorPageProps {
  /** create: /settings/shell-attributes/new; edit: /settings/shell-attributes/:attributeId/edit. */
  mode: 'create' | 'edit';
}

/**
 * The Shell attribute editor (feature 002, US2, US3), a full page for create and edit — STUB of
 * the shared foundation. Feature 002 replaces this file with the real page; the routes stay.
 */
export function ShellAttributeEditorPage({ mode }: ShellAttributeEditorPageProps) {
  const title = mode === 'create' ? 'New Shell attribute' : 'Edit Shell attribute';
  return (
    <BuildingPage
      eyebrow="Settings"
      title={title}
      breadcrumb={[
        { label: 'Settings', to: '/settings' },
        { label: 'Shell attributes', to: '/settings/shell-attributes' },
        { label: title },
      ]}
    />
  );
}
