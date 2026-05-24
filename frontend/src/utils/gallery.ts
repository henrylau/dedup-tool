import type { GalleryCategory } from "../types";

export function galleryCategory(ext: string): GalleryCategory {
  if (!ext) return "generic";
  if (["jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "ico", "heic", "heif"].includes(ext)) return "image";
  if (ext === "pdf") return "pdf";
  if (["zip", "rar", "7z", "tar", "gz", "bz2", "xz"].includes(ext)) return "archive";
  if (["mp4", "mov", "mkv", "avi", "webm", "m4v"].includes(ext)) return "video";
  if (["mp3", "wav", "flac", "aac", "ogg", "m4a"].includes(ext)) return "audio";
  if (["csv", "xlsx", "xls", "ods"].includes(ext)) return "sheet";
  if (
    ["txt", "md", "json", "xml", "yaml", "yml", "toml", "ini", "env",
      "go", "ts", "tsx", "js", "jsx", "css", "scss", "html", "vue", "rs", "py", "rb", "java", "c", "cpp", "h", "cs"].includes(ext)
  ) {
    return "code";
  }
  return "generic";
}
