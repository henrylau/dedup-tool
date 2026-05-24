import { type MergeFileRow } from "../../../bindings/folder-similarity/app/dto";

export function FileNameCell({ row }: { row: MergeFileRow }) {
  const L = (row.leftName || "").trim();
  const R = (row.rightName || "").trim();
  const bothDifferent = L.length > 0 && R.length > 0 && L !== R;

  if (!L && !R) {
    return <span className="file-name">—</span>;
  }
  if (bothDifferent) {
    return (
      <div className="file-name-stack">
        <span className="file-name file-name-primary" title={`Left: ${L}`}>
          {L}
        </span>
        <span className="file-name file-name-alt" title={`Right: ${R}`}>
          {R}
        </span>
      </div>
    );
  }
  return <span className="file-name">{L || R || "—"}</span>;
}
