import { describe, expect, it } from 'vitest';
import { filterTree } from './filterTree';
import type { TreeNode } from './TreeTable';

function node(id: string, name: string, children: TreeNode[] = []): TreeNode {
  return { id, level: 'campaign', name, cells: {}, children };
}

const TREE = [
  node('P-1001', 'FY2026 Lubricants Europe', [
    node('C-1001', 'Helix launch', [node('I-1001', 'Digital & social'), node('I-1002', 'Retail')]),
    node('C-1002', 'Fleet cards'),
  ]),
];

function ids(nodes: TreeNode[]): string[] {
  return nodes.flatMap((item) => [item.id, ...ids(item.children)]);
}

describe('filterTree (feature 004)', () => {
  it('keeps matches by name or ID with their ancestors, and opens the ancestors', () => {
    const result = filterTree(TREE, '  DIGITAL ');

    expect(ids(result.nodes)).toEqual(['P-1001', 'C-1001', 'I-1001']);
    expect([...result.openIds].sort()).toEqual(['C-1001', 'P-1001']);
  });

  it('finds records by ID', () => {
    expect(ids(filterTree(TREE, 'c-1002').nodes)).toEqual(['P-1001', 'C-1002']);
  });

  it('keeps everything for an empty search and nothing when nothing matches', () => {
    expect(filterTree(TREE, '  ').nodes).toBe(TREE);
    expect(filterTree(TREE, 'zzz').nodes).toEqual([]);
  });
});
