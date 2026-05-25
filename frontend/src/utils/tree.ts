import { type FolderNode } from "../../bindings/folder-similarity/app/dto";
import type { TreeSort } from "../types";

export function sortTree(node: FolderNode, mode: TreeSort): FolderNode {
  if (!node.children?.length) return node;
  const cmp =
    mode === "name-desc"
      ? (a: FolderNode, b: FolderNode) => (b.name || b.path).localeCompare(a.name || a.path)
      : mode === "size-desc"
      ? (a: FolderNode, b: FolderNode) => (b.totalSizeBytes ?? 0) - (a.totalSizeBytes ?? 0)
      : (a: FolderNode, b: FolderNode) => (a.name || a.path).localeCompare(b.name || b.path);
  const sortedChildren = node.children.map((c) => sortTree(c, mode)).sort(cmp);
  return { ...node, children: sortedChildren } as FolderNode;
}
