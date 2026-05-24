export function formatPct(n: number): string {
  if (Number.isNaN(n) || n < 0) return "—";
  return `${n.toFixed(1)}%`;
}

/** Elapsed scan time from milliseconds (avoids fake HH:MM:SS where the last field was total seconds). */
export function formatScanElapsed(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  const secTotal = ms / 1000;
  if (secTotal < 60) return `${secTotal.toFixed(2)}s`;
  const hrs = Math.floor(secTotal / 3600);
  const rem = secTotal - hrs * 3600;
  const mins = Math.floor(rem / 60);
  const sec = rem - mins * 60;
  if (hrs > 0) return `${hrs}h ${mins}m ${sec.toFixed(2)}s`;
  return `${mins}m ${sec.toFixed(2)}s`;
}

/** Binary byte size for folder stats (matches common OS reporting). */
export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) return "—";
  if (n < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB", "TB"] as const;
  let v = n / 1024;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  const digits = v < 10 && i > 0 ? 1 : 0;
  return `${v.toFixed(digits)} ${units[i]}`;
}

export function fileExtLower(name: string): string {
  const n = (name || "").trim();
  if (!n || n === "—") return "";
  const parts = n.split(".");
  if (parts.length < 2) return "";
  return (parts.pop() || "").toLowerCase();
}
