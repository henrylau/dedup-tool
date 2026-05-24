/** Absolute filesystem path for merge preview folder keys under the scan root (matches Go filepath.Join). */
export function mergeFolderAbsPath(rootAbs: string | undefined, relKey: string | undefined): string | undefined {
  const root = (rootAbs ?? "").trim();
  const rel = (relKey ?? "").trim();
  if (!root) return undefined;
  const sep = root.includes("\\") ? "\\" : "/";
  const base = root.replace(/[/\\]+$/, "");
  if (!rel || rel === ".") return base || undefined;
  const tail = rel.replace(/[/\\]/g, sep);
  return `${base}${sep}${tail}`;
}

export function fileManagerRevealLabel(): string {
  if (typeof navigator === "undefined") return "Show in file manager";
  const u = navigator.userAgent || "";
  if (/Windows/i.test(u)) return "Show in File Explorer";
  if (/Macintosh|Mac OS X/i.test(u)) return "Reveal in Finder";
  return "Show in file manager";
}

export function clampResultsCtxPosition(clientX: number, clientY: number): { left: number; top: number } {
  const pad = 12;
  const menuW = 240;
  const w = typeof window !== "undefined" ? window.innerWidth : 960;
  const h = typeof window !== "undefined" ? window.innerHeight : 720;
  return {
    left: Math.max(pad, Math.min(clientX, w - menuW - pad)),
    top: Math.max(pad, Math.min(clientY, h - pad)),
  };
}
