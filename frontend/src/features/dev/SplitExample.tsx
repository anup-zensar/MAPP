import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { SplitEditor } from '@/ui/SplitEditor/SplitEditor';
import type { SplitShare } from '@/ui/SplitEditor/SplitEditor';

export function SplitExample() {
  const [split, setSplit] = useState<SplitShare[]>([
    { optionId: 'SO-HELIX', share: '60.00' },
    { optionId: 'SO-RIMULA', share: '30.00' },
  ]);
  return (
    <Card title="SplitEditor">
      <SplitEditor
        label="Brand split"
        name="gallerySplit"
        marking="Needed to submit"
        options={[
          { id: 'SO-HELIX', label: 'Shell Helix', active: true },
          { id: 'SO-RIMULA', label: 'Shell Rimula', active: true },
          { id: 'SO-TELLUS', label: 'Shell Tellus', active: true },
        ]}
        value={split}
        onChange={setSplit}
      />
    </Card>
  );
}
