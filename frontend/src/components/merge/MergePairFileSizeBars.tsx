import { type MergeFileRow } from "../../../bindings/folder-similarity/app/dto";
import { FILE_SIZE_HIST_CHART_PX } from "../../constants";

/** One bar per merge file row (match pair); missing side uses size 0 (short bar). */
export function MergePairFileSizeBars({
  rows,
  side,
  barColor,
}: {
  rows: MergeFileRow[] | undefined;
  side: "left" | "right";
  barColor: string;
}) {
  const fileRows = (rows || []).filter((r) => !r.isFolder);
  const sizes = fileRows.map((r) => {
    const raw = side === "left" ? r.leftSizeBytes : r.rightSizeBytes;
    const n = typeof raw === "number" ? raw : Number(raw);
    return Number.isFinite(n) && n > 0 ? n : 0;
  });
  const max = Math.max(1, ...sizes);
  const minInnerWidthPx = Math.max(52, fileRows.length * 4);

  return (
    <div className="folder-size-hist">
      <div className="folder-size-hist-label">File sizes (paired rows)</div>
      <div className="folder-size-hist-scroll">
        <div
          className="folder-size-hist-bars"
          style={{ minWidth: `${minInnerWidthPx}px` }}
          role="img"
          aria-label={
            side === "left"
              ? "Left folder: one bar per matched file row; height ∝ file size; missing left file shows as minimal bar"
              : "Right folder: one bar per matched file row; height ∝ file size; missing right file shows as minimal bar"
          }
        >
          {fileRows.length === 0 ? (
            <div className="folder-size-hist-empty" aria-hidden>No file rows</div>
          ) : (
            sizes.map((bytes, i) => {
              const hPx = bytes === 0 ? 3 : Math.max(4, Math.round((bytes / max) * FILE_SIZE_HIST_CHART_PX));
              return (
                <div key={i} className="folder-size-hist-col">
                  <div
                    className="folder-size-hist-bar"
                    style={{
                      height: `${hPx}px`,
                      backgroundColor: barColor,
                      opacity: bytes === 0 ? 0.22 : 0.95,
                    }}
                  />
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
