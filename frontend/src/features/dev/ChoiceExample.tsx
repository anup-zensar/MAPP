import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { COST_CENTRES } from './gallery-data';
import styles from './ComponentGalleryPage.module.css';

export function ChoiceExample() {
  const [type, setType] = useState('');
  const [channels, setChannels] = useState<string[]>([]);
  const [centre, setCentre] = useState('');
  return (
    <Card title="ChoiceField">
      <div className={styles.fields}>
        <ChoiceField
          label="Activity type"
          name="galleryType"
          marking="Required"
          options={[
            { id: 'AT-1', label: 'Event', active: true },
            { id: 'AT-2', label: 'Paid media', active: true },
            { id: 'AT-3', label: 'Print', active: false },
          ]}
          value={type}
          onChange={setType}
        />
        <ChoiceField
          label="Channels"
          name="galleryChannels"
          multiple
          options={[
            { id: 'SO-TV', label: 'TV', active: true },
            { id: 'SO-RADIO', label: 'Radio', active: true },
            { id: 'SO-WEB', label: 'Web', active: true },
          ]}
          value={channels}
          onChange={setChannels}
        />
        <ChoiceField
          label="Cost centre"
          name="galleryCentre"
          hint="1,217 options: type to search."
          options={COST_CENTRES}
          value={centre}
          onChange={setCentre}
        />
        <ChoiceField
          label="Region"
          name="galleryRegion"
          options={[]}
          value=""
          onChange={() => {}}
          disabledReason="Choose a cluster first."
        />
      </div>
    </Card>
  );
}
