import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import type { AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { CAMPAIGN_FORM } from './gallery-data';
import styles from './ComponentGalleryPage.module.css';

export function AttributesExample() {
  const stored: AttributeValues = { 'SA-CLUSTER': ['SO-B2B'], 'SA-REGION': ['SO-EU'] };
  const [values, setValues] = useState<AttributeValues>(stored);
  const [asking, setAsking] = useState(false);
  const cleared = valuesToClear(CAMPAIGN_FORM, values, stored);
  return (
    <Card
      title="ShellAttributeFields and ShellAttributeValues"
      actions={
        <Button size="sm" onClick={() => setAsking(true)}>
          Try saving
        </Button>
      }
    >
      <div className={styles.columns}>
        <ShellAttributeFields
          form={CAMPAIGN_FORM}
          businessUnit={['SO-LUB']}
          values={values}
          previousValues={stored}
          onChange={setValues}
        />
        <ShellAttributeValues form={CAMPAIGN_FORM} businessUnit={['SO-LUB']} values={values} />
      </div>
      <ClearValuesDialog
        open={asking && cleared.length > 0}
        onOpenChange={setAsking}
        cleared={cleared}
        onConfirm={() => setAsking(false)}
      />
    </Card>
  );
}
