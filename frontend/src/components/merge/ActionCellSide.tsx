import { type MergeFileRow } from "../../../bindings/folder-similarity/app/dto";
import { actionShowsOnSide } from "../../utils/action";
import { ActionCell } from "./ActionCell";

export function ActionCellSide({ row, side }: { row: MergeFileRow; side: "left" | "right" }) {
  const a = row.action || "none";
  if (!actionShowsOnSide(a, side)) {
    return <span className="action-cell-blank" />;
  }
  return <ActionCell row={row} />;
}
