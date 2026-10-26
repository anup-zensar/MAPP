import { useCurrentUser, useSession } from '@/app/useSession';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import type { SegmentedFilterOption } from '@/ui/SegmentedFilter/SegmentedFilter';
import { usePageTitle } from '@/lib/usePageTitle';
import { AttentionCard } from './AttentionCard';
import { CurrencyPicker } from './CurrencyPicker';
import { FinancialsCard } from './FinancialsCard';
import { GettingStartedCard } from './GettingStartedCard';
import { useHomeSummary } from './homeApi';
import type { HomeScope } from './homeApi';
import { SCOPE_HINT, greetingText } from './homeText';
import styles from './HomePage.module.css';
import { MyActivitiesCard } from './MyActivitiesCard';
import { NoRoleCard } from './NoRoleCard';
import { PlansAtAGlanceCard } from './PlansAtAGlanceCard';
import { RecentChangesCard } from './RecentChangesCard';
import { SummaryCards } from './SummaryCards';
import { WaitingCard } from './WaitingCard';

const SCOPES: SegmentedFilterOption<HomeScope>[] = [
  { value: 'myTeams', label: 'My teams' },
  { value: 'everything', label: 'Everything' },
];

/**
 * Home (feature 009) on the dashboard template of design system §2.4 E: the greeting with the My
 * teams / Everything switch (FR-003) and the Currency picker in the header's actions, Getting
 * started for new people (FR-020), the four summary cards (FR-005), the Financials card right under
 * them (015 FR-025), Needs your attention (FR-011) and Waiting for my approval (FR-013) side by side
 * at one height, then Plans at a glance (FR-008), My activities (FR-014) and Recent changes
 * (FR-015) across the page. Every card asks its own
 * data, so one that fails never hides the others (FR-022). A person without any role sees why and
 * whom to ask instead of the cards (014 edge case "No role").
 */
export function HomePage() {
  usePageTitle('Home');
  const person = useCurrentUser();
  const { updatePreferences } = useSession();
  // The switch shows the choice at once and saves it in the background (001's preference pattern).
  const scope: HomeScope = person.preferences.homeScope ?? 'myTeams';
  const changeScope = (homeScope: HomeScope) => updatePreferences({ homeScope });
  const summary = useHomeSummary(scope);
  const gettingStarted = summary.data?.gettingStarted ?? null;
  // Before 014 is served the person has no access object: Home shows its cards as before.
  const noRole = person.access !== undefined && person.access.roles.length === 0;

  return (
    <>
      <PageHeader
        title={greetingText(person.firstName)}
        description={noRole ? undefined : SCOPE_HINT}
        actions={
          !noRole && (
            <div className={styles.actions}>
              <SegmentedFilter
                label="Scope"
                options={SCOPES}
                value={scope}
                onChange={changeScope}
              />
              <CurrencyPicker />
            </div>
          )
        }
      />
      {noRole ? (
        <div className={styles.page}>
          <NoRoleCard
            administrators={person.access?.administrators.map((admin) => admin.name) ?? []}
          />
        </div>
      ) : (
        <div className={styles.page}>
          {gettingStarted && <GettingStartedCard reason={gettingStarted} />}
          <SummaryCards scope={scope} />
          <FinancialsCard scope={scope} />
          <div className={styles.pair}>
            <AttentionCard />
            <WaitingCard />
          </div>
          <PlansAtAGlanceCard scope={scope} onShowEverything={() => changeScope('everything')} />
          <MyActivitiesCard />
          <RecentChangesCard scope={scope} />
        </div>
      )}
    </>
  );
}
