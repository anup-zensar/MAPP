import { Pencil } from 'lucide-react';
import { ActionsMenu, Button, LockedNote, SidePanel, StatusChip } from '@/components';
import { localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import { MetaLine } from '../shared/MetaLine';
import { SETTING_NOUNS } from '../shared/settingsNouns';
import styles from '../shared/SettingsTable.module.css';
import { UsageLine } from '../shared/UsageLine';
import type { Currency } from './currenciesApi';
import { inverseLine, lockOf, moreActions, rateLine } from './currencyActions';
import type { CurrencyActions } from './currencyActions';

export interface CurrencyDetailsProps {
  /** The currency; null when it no longer exists. */
  currency: Currency | null;
  actions: CurrencyActions;
  onClose: () => void;
}

/**
 * A currency's details in a side panel (US4 scenarios 2 and 4): its rate both ways, who last
 * changed the rate and when, where it is used and its locks; **Edit** and ⋯ (Set inactive or
 * Reactivate, Delete).
 */
export function CurrencyDetails({ currency, actions, onClose }: CurrencyDetailsProps) {
  const { date } = useFormatters();

  if (!currency) {
    return (
      <SidePanel
        open
        onOpenChange={(open) => !open && onClose()}
        title="Currency"
        footer={<Button onClick={onClose}>Back to Currencies</Button>}
      >
        <p role="alert" className={styles.gone}>
          {SETTING_NOUNS.currency.notFound}
        </p>
      </SidePanel>
    );
  }

  const rateLock = lockOf(currency, 'rate');
  const statusLock = lockOf(currency, 'active');

  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title={`${currency.name} (${currency.code})`}
      footer={
        <>
          <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(currency)}>
            Edit
          </Button>
          <ActionsMenu label={currency.code} items={moreActions(currency, actions)} />
        </>
      }
    >
      <div className={styles.details}>
        <div className={styles.statusRow}>
          <StatusChip tone={currency.active ? 'positive' : 'neutral'}>
            {currency.active ? 'Active' : 'Inactive'}
          </StatusChip>
        </div>
        {statusLock && <LockedNote>{statusLock}</LockedNote>}
        <dl className={styles.facts}>
          <div>
            <dt>Code</dt>
            <dd>{currency.code}</dd>
          </div>
          <div>
            <dt>Name</dt>
            <dd>{currency.name}</dd>
          </div>
          <div>
            <dt>Symbol</dt>
            <dd>{currency.symbol || '—'}</dd>
          </div>
          <div>
            <dt>Rate (units per 1 USD)</dt>
            <dd>
              {rateLine(currency.code, currency.rate)}
              <span className={styles.secondLine}>
                {inverseLine(currency.code, currency.inverseRate)}
              </span>
              <span className={styles.secondLine}>
                {`Rate last changed by ${currency.rateChangedBy.name} on ${date(localDate(currency.rateChangedAt))}`}
              </span>
            </dd>
            {rateLock && <LockedNote>{rateLock}</LockedNote>}
          </div>
          <div>
            <dt>Used by</dt>
            <dd>
              <UsageLine usage={currency.usage} />
            </dd>
          </div>
        </dl>
        <MetaLine record={currency} />
      </div>
    </SidePanel>
  );
}
