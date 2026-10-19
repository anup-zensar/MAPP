import { useCurrentUser, useSession } from '@/app/useSession';
import { PageHeader, SegmentedFilter } from '@/components';
import type { SegmentedFilterOption } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { AttentionCard } from './AttentionCard';
import { CurrencyPicker } from './CurrencyPicker';
import { GettingStartedCard } from './GettingStartedCard';
import { useHomeSummary } from './homeApi';
import type { HomeScope } from './homeApi';
import { HOME_DESCRIPTION, SCOPE_HINT } from './homeText';
import styles from './HomePage.module.css';
import { MyActivitiesCard } from './MyActivitiesCard';
import { PlansAtAGlanceCard } from './PlansAtAGlanceCard';
import { RecentChangesCard } from './RecentChangesCard';
import { SummaryCards } from './SummaryCards';
import { WaitingCard } from './WaitingCard';

const SCOPES: SegmentedFilterOption<HomeScope>[] = [
  { value: 'myTeams', label: 'My teams' },
  { value: 'everything', label: 'Everything' },
];

/**
 * Home (feature 009): the ZenReach-style dashboard. A greeting with the Currency picker, Getting
 * started for new people (FR-020), the My teams / Everything switch (FR-003), the four summary
 * cards (FR-005), Plans at a glance (FR-008), Needs your attention (FR-011), Waiting for my
 * approval (FR-013), My activities (FR-014) and Recent changes (FR-015). Every card asks its own
 * data, so one that fails never hides the others (FR-022).
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

  return (
    <>
      <PageHeader
        eyebrow="Home"
        title="Welcome back,"
        highlight={`${person.firstName}.`}
        description={HOME_DESCRIPTION}
        actions={<CurrencyPicker />}
      />
      <div className={styles.page}>
        {gettingStarted && <GettingStartedCard reason={gettingStarted} />}
        <div className={styles.scope}>
          <SegmentedFilter label="Scope" options={SCOPES} value={scope} onChange={changeScope} />
          <p className={styles.hint}>{SCOPE_HINT}</p>
        </div>
        <SummaryCards scope={scope} />
        <PlansAtAGlanceCard scope={scope} onShowEverything={() => changeScope('everything')} />
        <div className={styles.columns}>
          <AttentionCard />
          <WaitingCard />
          <MyActivitiesCard />
          <RecentChangesCard scope={scope} />
        </div>
      </div>
    </>
  );
}
