import { type MouseEvent, type Ref } from "react";
import { type MergeFileRow, type MergePreview } from "../../../bindings/folder-similarity/app/dto";
import { actionShowsOnSide, actionTitle } from "../../utils/action";
import { mergeFolderAbsPath } from "../../utils/path";
import { mergeRowIsMatchedFile } from "../../utils/row";
import { ActionCell } from "../merge/ActionCell";
import { GalleryThumb } from "./GalleryThumb";

export function ExplorerSidePanel(props: {
  merge: MergePreview;
  side: "left" | "right";
  selectedRowIndexes: ReadonlySet<number>;
  onMergeRowPick: (rowIndex: number, ev: Pick<MouseEvent<Element>, "shiftKey">) => void;
  rootAbs?: string;
  showMergeActions?: boolean;
  galleryScrollRef?: Ref<HTMLDivElement>;
  onRowContextMenu?: (e: MouseEvent<HTMLButtonElement>, row: MergeFileRow) => void;
  onMergeRowDoubleClick?: (row: MergeFileRow) => void;
}) {
  const { merge, side, selectedRowIndexes, onMergeRowPick, rootAbs, showMergeActions, galleryScrollRef, onRowContextMenu, onMergeRowDoubleClick } = props;
  const rowsAll = merge.rows || [];
  const rows = rowsAll.filter((r) => {
    const nm = side === "left" ? (r.leftName || "").trim() : (r.rightName || "").trim();
    return nm.length > 0;
  });
  const pathLabel = side === "left" ? merge.leftPath : merge.rightPath;
  const absPath = mergeFolderAbsPath(rootAbs, pathLabel);
  const displayPath =
    absPath ?? ((pathLabel ?? "").trim() || "—");

  return (
    <div className="explorer-panel">
      <div className={"explorer-banner explorer-banner-" + side}>
        <div className="explorer-banner-icon-wrap" aria-hidden>
          <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
            <path d="M4 6h5l1.5 2H18a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2z" stroke="currentColor" strokeWidth="1.4" />
          </svg>
        </div>
        <div className="explorer-banner-abs" title={displayPath !== "—" ? displayPath : undefined}>
          {displayPath}
        </div>
      </div>
      <div
        className="explorer-gallery-wrap"
        ref={galleryScrollRef}
      >
        {rows.length === 0 ? (
          <div className="explorer-gallery-empty">No items on this side.</div>
        ) : (
          <div className="explorer-gallery">
            {rows.map((r) => {
              const name = side === "left" ? (r.leftName || "").trim() : (r.rightName || "").trim();
              const meta = side === "left" ? (r.leftMeta || "").trim() : (r.rightMeta || "").trim();
              const sel = selectedRowIndexes.has(r.rowIndex);
              return (
                <button
                  key={r.rowIndex}
                  type="button"
                  className={
                    "gallery-card" +
                    (sel ? " gallery-card-sel" : "") +
                    (r.isFolder ? " gallery-card-folder" : "") +
                    (mergeRowIsMatchedFile(r) ? " gallery-card-matched" : "")
                  }
                  onClick={(e) => onMergeRowPick(r.rowIndex, e)}
                  onDoubleClick={(e) => {
                    e.preventDefault();
                    onMergeRowDoubleClick?.(r);
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onMergeRowPick(r.rowIndex, { shiftKey: false });
                    onRowContextMenu?.(e, r);
                  }}
                >
                  <div className="gallery-card-thumb">
                    {showMergeActions && actionShowsOnSide(r.action || "none", side) ? (
                      <div className="gallery-card-action-overlay" title={actionTitle(r.action || "none")}>
                        <ActionCell row={r} />
                      </div>
                    ) : null}
                    <GalleryThumb row={r} side={side} rootAbs={rootAbs} />
                  </div>
                  <div className="gallery-card-caption">
                    <span className="gallery-card-name" title={name}>{name}</span>
                    {meta ? <span className="gallery-card-meta">{meta}</span> : null}
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
