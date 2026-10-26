import { Plus } from 'lucide-react';
import { LinkButton } from '@/ui/Button/Button';
import { CanChange, SettingsViewOnlyNote } from '@/ui/CanChange/CanChange';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { usePageTitle } from '@/ui/usePageTitle';
import { AttributeList } from './AttributeList';
import { useAttributeFilters } from './useAttributeFilters';

/**
 * Shell attributes (feature 002, US2, FR-016 – FR-019, SC-009): one level's attributes in the
 * order of its form, grouped by section, with the move buttons that set that order.
 */
export function ShellAttributesPage() {
  usePageTitle('Shell attributes');
  const filters = useAttributeFilters();

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Shell attributes' }]}
        title="Shell attributes"
        description="Shell's own fields on each level's form. Changes apply to every form straight away."
        actions={
          <CanChange>
            <LinkButton
              variant="primary"
              icon={<Plus />}
              to={`/settings/shell-attributes/new?level=${filters.level}`}
            >
              New Shell attribute
            </LinkButton>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote />
      <AttributeList filters={filters} />
    </>
  );
}
