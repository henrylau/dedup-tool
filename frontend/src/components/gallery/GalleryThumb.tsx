import { type MergeFileRow } from "../../../bindings/folder-similarity/app/dto";
import { fileExtLower } from "../../utils/format";
import { galleryCategory } from "../../utils/gallery";
import { GalleryFolderVisual } from "./GalleryFolderVisual";
import { GalleryImageThumb } from "./GalleryImageThumb";
import { GalleryKindVisual } from "./GalleryKindVisual";

export function GalleryThumb(props: {
  row: MergeFileRow;
  side: "left" | "right";
  rootAbs?: string;
}) {
  const { row, side, rootAbs } = props;
  if (row.isFolder) {
    return <GalleryFolderVisual />;
  }
  const nm = (side === "left" ? row.leftName : row.rightName) || "";
  const ext = fileExtLower(nm.trim());
  const cat = galleryCategory(ext);
  const relPath = side === "left" ? row.leftPath : row.rightPath;
  const rp = (relPath || "").trim();
  const root = (rootAbs || "").trim();
  if ((cat === "image" || cat === "video") && root && rp) {
    return <GalleryImageThumb relPath={rp} rootAbs={root} />;
  }
  return <GalleryKindVisual category={cat === "image" ? "generic" : cat} ext={ext} />;
}
