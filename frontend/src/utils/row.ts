import { type MergeFileRow } from "../../bindings/folder-similarity/app/dto";

/** File rows in merge table order (rowIndex list), for shift-range selection on the Results tab. */
export function mergeFileRowOrdering(rows: MergeFileRow[]): number[] {
  return rows.filter((r) => !r.isFolder).map((r) => r.rowIndex);
}

/** Rows visible in a folder explorer pane (same filtering as ExplorerSidePanel). */
export function explorerVisibleRowIndexes(rows: MergeFileRow[], side: "left" | "right"): number[] {
  return rows
    .filter((r) => {
      const nm = side === "left" ? (r.leftName || "").trim() : (r.rightName || "").trim();
      return nm.length > 0;
    })
    .map((r) => r.rowIndex);
}

export function rowIndexesInclusiveRange(orderedRowIndexes: number[], anchorIndex: number, endIndex: number): number[] {
  const ia = orderedRowIndexes.indexOf(anchorIndex);
  const ib = orderedRowIndexes.indexOf(endIndex);
  if (ia === -1 || ib === -1) return [endIndex];
  const lo = Math.min(ia, ib);
  const hi = Math.max(ia, ib);
  return orderedRowIndexes.slice(lo, hi + 1);
}

/** One size column: same value when matched; both sides when they differ. Folder rows use meta (counts / dup), not bytes. */
export function unifiedSizeDisplay(r: MergeFileRow): { text: string; title: string; variant: "empty" | "match" | "diff" | "side" | "folder" } {
  const L = (r.leftMeta || "").trim();
  const R = (r.rightMeta || "").trim();

  if (r.isFolder) {
    if (!L && !R) return { text: "—", title: "", variant: "empty" };
    if (L && L === R) return { text: L, title: "Both sides", variant: "folder" };
    const both = [L, R].filter(Boolean);
    return {
      text: both.join(" · "),
      title: [L && `Left: ${L}`, R && `Right: ${R}`].filter(Boolean).join("\n"),
      variant: "folder",
    };
  }

  if (!L && !R) return { text: "—", title: "", variant: "empty" };
  if (L && R && L === R) {
    return { text: L, title: "Same size on left and right", variant: "match" };
  }
  if (L && R && L !== R) {
    return { text: `${L} · ${R}`, title: `Left: ${L}\nRight: ${R}`, variant: "diff" };
  }
  if (L) return { text: L, title: "Left file only", variant: "side" };
  return { text: R, title: "Right file only", variant: "side" };
}

export function getRowStatus(r: MergeFileRow): "exact" | "similar" | "unique" {
  if (r.kind === "pair") {
    return r.similarityPct === 100 ? "exact" : "similar";
  }
  return "unique";
}

/** File row paired on both sides (exact or similar match in compare view). */
export function mergeRowIsMatchedFile(r: MergeFileRow): boolean {
  return !r.isFolder && r.kind === "pair";
}

/** Whether a file (or folder entry) exists on each side of the row. */
export function existsSides(r: MergeFileRow): { left: boolean; right: boolean } {
  if (r.isFolder) {
    return {
      left: !!(r.leftName?.trim() || r.leftPath?.trim()),
      right: !!(r.rightName?.trim() || r.rightPath?.trim()),
    };
  }
  switch (r.kind) {
    case "pair":
      return { left: true, right: true };
    case "left":
      return { left: true, right: false };
    case "right":
      return { left: false, right: true };
    default:
      return {
        left: !!(r.leftName?.trim() || r.leftPath?.trim()),
        right: !!(r.rightName?.trim() || r.rightPath?.trim()),
      };
  }
}
