import { useAttributeForm } from '@/features/settings';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { recordHref } from '@/ui/record-links';
import { Detail, DetailsList } from './DetailsList';
import { planIdOf } from './line-item-links';
import { moneyText } from './line-item-math';
import type { LineItem, LineItemActivity } from './api';
import { inactiveItemText, LABEL } from './line-item-text';
import styles from './LineItemPage.module.css';

const FINANCE_CODING = 'Finance coding';

/** Finance coding, titled with the attributes' one group's name: no heading repeats (LI-03). */
export function LineItemFinanceCard({
  lineItem,
  activity,
}: {
  lineItem: LineItem;
  activity: LineItemActivity;
}) {
  const attributeForm = useAttributeForm('lineItem');
  const planId = planIdOf(activity);
  const sections = attributeForm.data?.sections ?? [];
  const onlyGroup = sections.length === 1 ? sections[0] : undefined;
  return (
    <Card title={onlyGroup ? onlyGroup.name : FINANCE_CODING}>
      <div className={styles.stack}>
        {attributeForm.data ? (
          <ShellAttributeValues
            form={attributeForm.data}
            values={lineItem.attributes}
            businessUnit={activity.businessUnitOptionIds}
            planLink={planId ? recordHref('plan', planId) : undefined}
            sectionHeading={onlyGroup ? 'none' : 'h3'}
          />
        ) : attributeForm.isError ? (
          <ErrorState error={attributeForm.error} onRetry={() => void attributeForm.refetch()} />
        ) : (
          <Skeleton lines={2} />
        )}
        <FinanceDetails lineItem={lineItem} />
      </div>
    </Card>
  );
}

/** The contracted amount, the finance budget and the GSAP request, under the attributes. */
function FinanceDetails({ lineItem }: { lineItem: LineItem }) {
  const item = lineItem.financeBudgetItem;
  return (
    <>
      <DetailsList>
        <Detail label={LABEL.contractedAmount} helpKey="lineItem.contractedAmount" numeric>
          {lineItem.contractedAmount ? moneyText(lineItem.contractedAmount) : null}
        </Detail>
        <Detail label={LABEL.financeBudgetYear} helpKey="lineItem.financeBudgetYear">
          {lineItem.financeBudgetYear}
        </Detail>
        <Detail label={LABEL.financeBudgetItem} helpKey="lineItem.financeBudgetItem">
          {item
            ? `${[item.name, item.budgetName, item.code].filter(Boolean).join(' · ')}${item.active ? '' : ' (inactive)'}`
            : null}
        </Detail>
        <Detail label={LABEL.requestSwbs} helpKey="lineItem.requestSwbs">
          {lineItem.requestSwbs ? 'Yes' : 'No'}
        </Detail>
      </DetailsList>
      {item && !item.active && <p className={styles.problem}>{inactiveItemText(item.name)}</p>}
    </>
  );
}
