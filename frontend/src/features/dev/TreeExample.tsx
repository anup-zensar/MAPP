import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { TreeTable } from '@/ui/TreeTable/TreeTable';
import { PLAN_TREE } from './gallery-data';

export function TreeExample() {
  const [expanded, setExpanded] = useState(new Set(['P-1001', 'C-1001']));
  function toggle(id: string) {
    const next = new Set(expanded);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setExpanded(next);
  }
  return (
    <Card title="TreeTable">
      <TreeTable
        label="Plan tree"
        columns={[
          { key: 'name', header: 'Name' },
          { key: 'owner', header: 'Owner' },
          { key: 'budget', header: 'Budget', align: 'end' },
        ]}
        nodes={PLAN_TREE}
        expandedIds={expanded}
        onToggle={toggle}
        footer={{ name: 'Total · 1 campaign', budget: '$500,000.00' }}
      />
    </Card>
  );
}
