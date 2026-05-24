import { type MergeFileRow } from "../../../bindings/folder-similarity/app/dto";
import { actionAbbr, actionTitle } from "../../utils/action";

export function ActionCell({ row }: { row: MergeFileRow }) {
  const a = row.action || "none";
  const pillClass =
    a === "deleteRight"
      ? "action-pill action-del-r"
      : a === "deleteLeft"
        ? "action-pill action-del-l"
        : a === "moveToRight"
          ? "action-pill action-mv-r"
          : a === "moveToLeft"
            ? "action-pill action-mv-l"
            : "action-pill action-none";
  return (
    <span className={pillClass} title={actionTitle(a)}>
      {actionAbbr(a)}
    </span>
  );
}
