import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { CountBadge } from '@/ui/CountBadge/CountBadge';
import { Tabs } from '@/ui/Tabs/Tabs';

export function TabsExample() {
  const [tab, setTab] = useState<'waiting' | 'open' | 'decided'>('waiting');
  return (
    <Card title="Tabs and CountBadge">
      <Tabs
        label="Approvals"
        value={tab}
        onChange={setTab}
        tabs={[
          {
            value: 'waiting',
            label: 'Waiting for me',
            badge: <CountBadge count={3} label="waiting" />,
          },
          { value: 'open', label: 'All open', badge: <CountBadge count={120} label="open" /> },
          { value: 'decided', label: 'Decided by me' },
        ]}
      >
        <p>The {tab} tab.</p>
      </Tabs>
    </Card>
  );
}
