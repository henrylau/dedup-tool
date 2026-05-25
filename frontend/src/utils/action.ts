import { type MergeFileRow } from "../../bindings/folder-similarity/app/dto";
import type { ActionImpact } from "../types";

export function actionAbbr(a: string): string {
  switch (a) {
    case "deleteRight": return "⌦";
    case "deleteLeft": return "⌫";
    case "moveToRight": return "→";
    case "moveToLeft": return "←";
    default: return "—";
  }
}

export function actionTitle(a: string): string {
  switch (a) {
    case "deleteRight":
      return "Delete file on the right";
    case "deleteLeft":
      return "Delete file on the left";
    case "moveToRight":
      return "Move to right folder";
    case "moveToLeft":
      return "Move to left folder";
    case "none":
      return "No action";
    default:
      return a || "No action";
  }
}

/** Which side of the compare UI should show the action flag (matches merge semantics). */
export function actionShowsOnSide(action: string, side: "left" | "right"): boolean {
  const a = (action || "none").trim();
  if (a === "none" || a === "") return false;
  if (side === "left") return a === "deleteLeft" || a === "moveToRight";
  return a === "deleteRight" || a === "moveToLeft";
}

/** Compute per-side counts/sizes for pending actions on file rows. */
export function computeActionImpact(rows: MergeFileRow[], side: "left" | "right"): ActionImpact {
  const out: ActionImpact = { deleteCount: 0, deleteSize: 0, addCount: 0, addSize: 0 };
  for (const r of rows) {
    if (r.isFolder) continue;
    const a = (r.action || "none").trim();
    if (a === "none" || a === "") continue;
    const leftSize = Number(r.leftSizeBytes ?? 0);
    const rightSize = Number(r.rightSizeBytes ?? 0);
    if (side === "left") {
      if (a === "deleteLeft") { out.deleteCount++; out.deleteSize += leftSize; }
      else if (a === "moveToRight") { out.deleteCount++; out.deleteSize += leftSize; }
      else if (a === "moveToLeft") { out.addCount++; out.addSize += rightSize; }
    } else {
      if (a === "deleteRight") { out.deleteCount++; out.deleteSize += rightSize; }
      else if (a === "moveToLeft") { out.deleteCount++; out.deleteSize += rightSize; }
      else if (a === "moveToRight") { out.addCount++; out.addSize += leftSize; }
    }
  }
  return out;
}
