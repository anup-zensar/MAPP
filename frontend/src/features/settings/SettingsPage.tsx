import { ChevronRight, CircleAlert, RotateCw } from 'lucide-react';
import { Link } from 'react-router';
import { useSettingsOverview } from '@/api/settingsQueries';
import type { components } from '@/api/schema';
import { Button, PageHeader, StatusChip } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { SETTINGS_SECTIONS } from './sections';
import type { SettingsSection } from './sections';
import styles from './SettingsPage.module.css';
import { SettingsCount } from './shared/SettingsCount';

type SettingsOverview = components['schemas']['SettingsOverview'];

/** What the counts of feature 002's cards are doing. */
type Counts =
  { state: 'loading' } | { state: 'failed' } | { state: 'loaded'; overview: SettingsOverview };

/**
 * Settings (feature 002, FR-001): one card per section — its icon, name, one-line explanation and
 * the number of active records — each opening the section's own page. Sections still to come show
 * "Coming soon" and are not links. When the counts can't be loaded the cards stay usable.
 */
export function SettingsPage() {
  usePageTitle('Settings');
  const overview = useSettingsOverview();
  const counts: Counts = overview.data
    ? { state: 'loaded', overview: overview.data }
    : overview.isError
      ? { state: 'failed' }
      : { state: 'loading' };

  return (
    <>
      <PageHeader
        title="Settings"
        description="Set up the lists and rules every plan uses. Changes here apply to everyone straight away."
      />
      {counts.state === 'failed' && (
        <div role="alert" className={styles.countsFailed}>
          <CircleAlert aria-hidden="true" />
          <p>Counts couldn&apos;t be loaded.</p>
          <Button size="sm" icon={<RotateCw />} onClick={() => void overview.refetch()}>
            Try again
          </Button>
        </div>
      )}
      <ul role="list" aria-label="Sections" className={styles.grid}>
        {SETTINGS_SECTIONS.map((section) => (
          <SectionCard key={section.path} section={section} counts={counts} />
        ))}
      </ul>
    </>
  );
}

interface SectionCardProps {
  section: SettingsSection;
  counts: Counts;
}

/** One section: the whole card opens it; the link itself is named by the section's title. */
function SectionCard({ section, counts }: SectionCardProps) {
  const Icon = section.icon;
  const comingSoon = section.comingSoon === true;

  return (
    <li className={comingSoon ? styles.card : `${styles.card} ${styles.linkCard}`}>
      <div className={styles.head}>
        <span className={styles.icon} aria-hidden="true">
          <Icon />
        </span>
        <h2 className={styles.title}>
          {comingSoon ? section.title : <Link to={section.path}>{section.title}</Link>}
        </h2>
        {!comingSoon && (
          <span className={styles.chevron} aria-hidden="true">
            <ChevronRight />
          </span>
        )}
      </div>
      <p className={styles.description}>{section.description}</p>
      <div className={styles.footer}>
        {comingSoon ? (
          <StatusChip tone="neutral">Coming soon</StatusChip>
        ) : (
          <ActiveCount section={section} counts={counts} />
        )}
      </div>
    </li>
  );
}

/**
 * "<n> active": from the overview for 002's sections, from the section's own component otherwise
 * (which shows it with the same SettingsCount).
 */
function ActiveCount({ section, counts }: SectionCardProps) {
  if (!section.overviewKey) {
    return section.ActiveCount ? <section.ActiveCount /> : null;
  }
  return (
    <SettingsCount
      loading={counts.state === 'loading'}
      count={counts.state === 'loaded' ? counts.overview[section.overviewKey] : undefined}
    />
  );
}
