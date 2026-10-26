import { useSearchParams } from 'react-router';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Tabs } from '@/ui/Tabs/Tabs';
import { usePageTitle } from '@/ui/usePageTitle';
import { PeopleAccessTab } from './PeopleAccessTab';
import { RecentAccessChanges } from './RecentAccessChanges';
import { RolesTab } from './RolesTab';
import { RulesTab } from './RulesTab';
import { accessTabOf, withAccessTab } from './schemas';
import type { AccessTab } from './schemas';
import styles from './access.module.css';

const TABS: { value: AccessTab; label: string }[] = [
  { value: 'people', label: 'People' },
  { value: 'roles', label: 'Roles' },
  { value: 'rules', label: 'Rules' },
];

/**
 * Settings › Roles and access (feature 014, US3, US4, US7; plan "Pages"): the People, Roles and
 * Rules tabs, kept in the address (`?tab=`, `?person=` opens a person's roles), and the latest
 * access changes under them.
 */
export function AccessPage() {
  usePageTitle('Roles and access');
  const [params, setParams] = useSearchParams();
  const tab = accessTabOf(params);

  function chooseTab(next: AccessTab) {
    setParams((previous) => withAccessTab(previous, next), { replace: true });
  }

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Roles and access' }]}
        title="Roles and access"
        description="Who can see and change what, by role and market. Changes apply on each person's next action."
      />
      <div className={styles.page}>
        <Tabs label="Roles and access" tabs={TABS} value={tab} onChange={chooseTab}>
          {tab === 'people' && <PeopleAccessTab />}
          {tab === 'roles' && <RolesTab />}
          {tab === 'rules' && <RulesTab />}
        </Tabs>
        <RecentAccessChanges />
      </div>
    </>
  );
}
