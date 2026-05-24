/** Which merge row paths appear in Open / Reveal context menu */
export type MergeCtxMenuPaths = "all" | "left" | "right";

export type ActionImpact = { deleteCount: number; deleteSize: number; addCount: number; addSize: number };

export type ActiveTab = "results" | "browseLeft" | "browseRight" | "browseSplit";

export type TreeSort = "name-asc" | "name-desc" | "size-desc";

export type GalleryCategory = "image" | "pdf" | "archive" | "video" | "audio" | "sheet" | "code" | "generic";
