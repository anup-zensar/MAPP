import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { DateRangeField } from '@/ui/DateRangeField/DateRangeField';

export function DatesExample() {
  const [range, setRange] = useState({ start: '2026-03-01', end: '' });
  return (
    <Card title="DateRangeField">
      <DateRangeField
        label="Dates"
        idPrefix="gallery"
        marking="Needed to publish"
        start={range.start}
        end={range.end}
        onChange={setRange}
        allowed={{ start: '2026-02-01', end: '2026-11-30' }}
        allowedHint="Must be inside the campaign's dates (01 Feb – 30 Nov 2026)"
      />
    </Card>
  );
}
