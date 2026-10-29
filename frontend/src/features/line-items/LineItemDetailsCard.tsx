import { Card } from '@/ui/Card/Card';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { Detail, DetailsList } from './DetailsList';
import { moneyText } from './line-item-math';
import type { LineItem } from './api';
import { bookkeepingText, currencyLabel, LABEL } from './line-item-text';
import styles from './LineItemPage.module.css';

export function LineItemDetailsCard({ lineItem }: { lineItem: LineItem }) {
  const { date } = useFormatters();
  return (
    <Card title="Details">
      <div className={styles.stack}>
        <DetailsList>
          <Detail label={LABEL.name} helpKey="lineItem.name">
            {lineItem.name}
          </Detail>
          <Detail label={LABEL.owner} helpKey="lineItem.owner">
            {lineItem.owner
              ? `${lineItem.owner.name}${lineItem.owner.active ? '' : ' (inactive)'}`
              : null}
          </Detail>
          <Detail label={LABEL.currency} helpKey="lineItem.currency">
            {currencyLabel(
              lineItem.currency.code,
              lineItem.currency.name,
              lineItem.currency.active,
            )}
          </Detail>
          <Detail label={LABEL.plannedCost} helpKey="lineItem.plannedCost" numeric>
            {moneyText(lineItem.plannedCost)}
          </Detail>
          <Detail label={LABEL.description} helpKey="lineItem.description">
            {lineItem.description ? (
              <span className={styles.description}>{lineItem.description}</span>
            ) : null}
          </Detail>
        </DetailsList>
        <p className={styles.bookkeeping}>
          {bookkeepingText(
            lineItem.createdBy.name,
            date(localDate(lineItem.createdAt)),
            lineItem.updatedBy.name,
            date(localDate(lineItem.updatedAt)),
          )}
        </p>
      </div>
    </Card>
  );
}
