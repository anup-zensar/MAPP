import type { TreeNode } from './TreeTable';

/**
 * The rows of a tree that match a search (feature 004, plan tree search): every node whose name or
 * ID contains the text (ignoring case and surrounding spaces) with all its ancestors, and the ids
 * of those ancestors, to open so the matches show. An empty search keeps everything.
 */
export function filterTree(
  nodes: TreeNode[],
  query: string,
): { nodes: TreeNode[]; openIds: Set<string> } {
  const text = query.trim().toLowerCase();
  const openIds = new Set<string>();
  if (!text) {
    return { nodes, openIds };
  }

  function keep(node: TreeNode): TreeNode | null {
    const children = node.children.map(keep).filter((child): child is TreeNode => child !== null);
    const matches = node.name.toLowerCase().includes(text) || node.id.toLowerCase().includes(text);
    if (children.length > 0) {
      openIds.add(node.id);
    }
    return matches || children.length > 0 ? { ...node, children } : null;
  }

  return {
    nodes: nodes.map(keep).filter((node): node is TreeNode => node !== null),
    openIds,
  };
}
