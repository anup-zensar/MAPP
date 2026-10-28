import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router';
import { useCurrentUser } from '@/session/useSession';
import type { CurrentUser } from '@/session/useSession';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import { DATE_FORMATS } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { usePageTitle } from '@/ui/usePageTitle';
import { YourAccessCard } from './access/YourAccessCard';
import { useDisplayCurrencies } from './api';
import { EditDetailsPanel } from './EditDetailsPanel';
import { EditPreferencesPanel } from './EditPreferencesPanel';
import { currencyLabel, DISPLAY_CURRENCY_NOTE } from './texts';
import styles from './ProfilePage.module.css';

/** The side panel that is open. */
type OpenPanel = 'details' | 'preferences' | null;

/**
 * My profile (feature 003, US4, FR-021 – FR-025), from the account menu: the signed-in person's
 * details, their display preferences, their teams and their access (014 FR-024), each card that
 * can change with its own Edit side panel (D-068). Everything comes from the session, so the page
 * shows at once.
 */
export function ProfilePage() {
  usePageTitle('My profile');
  const user = useCurrentUser();
  const [open, setOpen] = useState<OpenPanel>(null);

  return (
    <>
      <PageHeader
        title="My profile"
        description="Your details, teams and access, and how SP&A One shows dates and money to you."
      />
      <div className={styles.cards}>
        <DetailsCard user={user} onEdit={() => setOpen('details')} />
        <PreferencesCard onEdit={() => setOpen('preferences')} />
        {/* Older answers have no teams (D-191). */}
        <TeamsCard teams={user.teams ?? []} />
        <YourAccessCard />
      </div>

      {open === 'details' && <EditDetailsPanel onClose={() => setOpen(null)} />}
      {open === 'preferences' && <EditPreferencesPanel onClose={() => setOpen(null)} />}
    </>
  );
}

/** Your details: avatar, name, job title and the e-mail, which is changed elsewhere (US4-1). */
function DetailsCard({ user, onEdit }: { user: CurrentUser; onEdit: () => void }) {
  return (
    <Card
      title="Your details"
      actions={
        <Button size="sm" icon={<Pencil />} onClick={onEdit}>
          Edit details
        </Button>
      }
    >
      <div className={styles.person}>
        <Avatar initials={user.initials} size="lg" />
        <dl className={styles.facts}>
          <div>
            <dt>Name</dt>
            <dd>{user.name}</dd>
          </div>
          <div>
            <dt>Job title</dt>
            <dd>{user.jobTitle || '—'}</dd>
          </div>
          <div>
            <dt>E-mail</dt>
            <dd className={styles.email}>
              {user.email}
              <LockedNote>
                Your e-mail is your sign-in name; it is changed in Settings › People.
              </LockedNote>
            </dd>
          </div>
        </dl>
      </div>
    </Card>
  );
}

/**
 * Display preferences: the display currency ("EUR — Euro", " (inactive)" when it was set inactive
 * and the person keeps it) and the date format with its example (FR-023, FR-024). Until the
 * currencies' names arrive, or when they can't be loaded, the code alone is shown.
 */
function PreferencesCard({ onEdit }: { onEdit: () => void }) {
  const { displayCurrency, dateFormat } = useFormatters();
  const currencies = useDisplayCurrencies();
  const currency = currencies.data?.items.find((option) => option.code === displayCurrency);
  const format = DATE_FORMATS.find((option) => option.value === dateFormat);

  return (
    <Card
      title="Display preferences"
      actions={
        <Button size="sm" icon={<Pencil />} onClick={onEdit}>
          Edit preferences
        </Button>
      }
    >
      <dl className={styles.facts}>
        <div>
          <dt>Display currency</dt>
          <dd>
            {currency
              ? `${currencyLabel(currency)}${currency.active ? '' : ' (inactive)'}`
              : displayCurrency}
          </dd>
        </div>
        <div>
          <dt>Date format</dt>
          <dd>{format?.label ?? dateFormat}</dd>
        </div>
      </dl>
      <p className={styles.note}>{DISPLAY_CURRENCY_NOTE}</p>
    </Card>
  );
}

/** My teams, read-only, with the way to Settings › Teams (US4-1, US4-5). */
function TeamsCard({ teams }: { teams: TeamRef[] }) {
  return (
    <Card title="My teams">
      {teams.length === 0 ? (
        <p className={styles.note}>
          You're not in any team yet, so you can't own a plan. Teams are managed in{' '}
          <Link to="/settings/teams" className={styles.link}>
            Settings › Teams
          </Link>
          .
        </p>
      ) : (
        <div className={styles.teams}>
          <TeamPills teams={teams} linked />
          <Link to="/settings/teams" className={styles.ownLink}>
            Manage teams in Settings › Teams
          </Link>
        </div>
      )}
    </Card>
  );
}
