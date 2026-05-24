import { useState, type MouseEvent } from "react";
import { type FolderNode } from "../../../bindings/folder-similarity/app/dto";
import { TREE_INITIAL_EXPAND_MAX_DEPTH } from "../../constants";
import { TreeChevron } from "./TreeChevron";

export function TreeList(props: {
  node: FolderNode;
  depth: number;
  selected: string;
  onSelect: (path: string) => void;
  searchQuery: string;
}) {
  const { node, depth, selected, onSelect, searchQuery } = props;
  const [expanded, setExpanded] = useState(() => depth < TREE_INITIAL_EXPAND_MAX_DEPTH);
  const pad = 8 + depth * 16;
  const hasChildren = (node.children?.length ?? 0) > 0;
  const name = node.name || node.path;

  if (!node.path && !node.name) return null;

  const matchesSearch = !searchQuery || name.toLowerCase().includes(searchQuery.toLowerCase());
  const childrenMatchSearch = !searchQuery || node.children?.some(c =>
    (c.name || c.path).toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (searchQuery && !matchesSearch && !childrenMatchSearch) return null;

  const toggleExpand = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    setExpanded((v) => !v);
  };

  const onActivateFolder = () => {
    if (hasChildren && !expanded) setExpanded(true);
    onSelect(node.path || ".");
  };

  return (
    <div className="tree-block">
      <div
        className={
          "tree-row" +
          (node.path === selected ? " tree-row-sel" : "") +
          (hasChildren ? " tree-row-branch" : "")
        }
        style={{ paddingLeft: pad }}
      >
        {hasChildren ? (
          <button
            type="button"
            className="tree-expand"
            aria-expanded={expanded}
            aria-label={expanded ? "Collapse folder" : "Expand folder"}
            title={expanded ? "Collapse" : "Expand"}
            onClick={toggleExpand}
          >
            <TreeChevron expanded={expanded} />
          </button>
        ) : (
          <span className="tree-expand tree-expand-leaf" title="No subfolders">
            <svg className="tree-leaf-dot" width="16" height="16" viewBox="0 0 16 16" aria-hidden>
              <circle cx="8" cy="8" r="2" fill="currentColor" />
            </svg>
          </span>
        )}
        <button type="button" className="tree-label tree-label-hit" onClick={onActivateFolder}>
          <span className="tree-folder-icon" aria-hidden>📁</span>
          <span className="tree-label-text">{name}</span>
        </button>
      </div>
      {expanded && node.children?.map((c, i) => (
        <TreeList
          key={c.path + i}
          node={c}
          depth={depth + 1}
          selected={selected}
          onSelect={onSelect}
          searchQuery={searchQuery}
        />
      ))}
    </div>
  );
}
