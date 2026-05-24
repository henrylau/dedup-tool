import { type MergeFileRow } from "../../../bindings/folder-similarity/app/dto";
import { existsSides } from "../../utils/row";

export function ExistsSquares({ row }: { row: MergeFileRow }) {
  const { left, right } = existsSides(row);
  const label = `Left: ${left ? "exists" : "missing"}. Right: ${right ? "exists" : "missing"}.`;
  return (
    <div className="exists-cell" role="img" aria-label={label}>
      <span
        className={`exists-square ${left ? "exists-yes" : "exists-no"}`}
        title={left ? "Left: present" : "Left: not present"}
        aria-hidden="true"
      />
      <span
        className={`exists-square ${right ? "exists-yes" : "exists-no"}`}
        title={right ? "Right: present" : "Right: not present"}
        aria-hidden="true"
      />
    </div>
  );
}
