import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import styles from './ComponentGalleryPage.module.css';

export function NumberFieldsExample() {
  const [budget, setBudget] = useState('12.5k');
  const [rate, setRate] = useState('2');
  const [inq, setInq] = useState('250000');
  return (
    <Card title="MoneyField and DecimalField">
      <div className={styles.fields}>
        <MoneyField
          label="Plan budget"
          name="galleryBudget"
          currency="USD"
          marking="Needed to publish"
          value={budget}
          onChange={setBudget}
        />
        <DecimalField
          label="INQ→MQL rate"
          name="galleryRate"
          decimals={2}
          suffix="%"
          value={rate}
          onChange={setRate}
        />
        <DecimalField
          label="Goal amount (INQ)"
          name="galleryInq"
          decimals={0}
          value={inq}
          onChange={setInq}
        />
      </div>
    </Card>
  );
}
