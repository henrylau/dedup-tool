import { useCallback, useEffect, useState } from "react";
import { Similarity } from "../../../bindings/folder-similarity/app/service";
import type { DirEntry } from "../../../bindings/folder-similarity/app/dto";

type Props = {
  open: boolean;
  title: string;
  filterExt?: string;
  onSelect: (path: string) => void;
  onClose: () => void;
};

function parentOf(path: string): string {
  if (!path) return path;
  const sep = path.includes("\\") && !path.includes("/") ? "\\" : "/";
  const trimmed = path.replace(/[/\\]+$/, "");
  const idx = trimmed.lastIndexOf(sep);
  if (idx <= 0) return sep;
  return trimmed.substring(0, idx);
}

export function ServerFolderBrowser({ open, title, filterExt, onSelect, onClose }: Props) {
  const [path, setPath] = useState<string>("");
  const [entries, setEntries] = useState<DirEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async (next: string) => {
    setLoading(true);
    setErr(null);
    try {
      const list = await Similarity.ListDir(next, filterExt ?? "");
      setPath(next);
      setEntries(list);
    } catch (e: any) {
      setErr(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  }, [filterExt]);

  useEffect(() => {
    if (!open) return;
    if (path) return;
    void (async () => {
      try {
        const home = await Similarity.HomeDir();
        await load(home);
      } catch (e: any) {
        setErr(e?.message || String(e));
      }
    })();
  }, [open, path, load]);

  if (!open) return null;

  const onEntryClick = (e: DirEntry) => {
    if (e.isDir) {
      void load(e.path);
    } else if (filterExt) {
      onSelect(e.path);
    }
  };

  const onUp = () => void load(parentOf(path));
  const onConfirmFolder = () => {
    if (path) onSelect(path);
  };

  return (
    <div
      className="modal-backdrop group-picker-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="server-folder-browser-title"
      onClick={onClose}
    >
      <div className="apply-modal" onClick={(e) => e.stopPropagation()} style={{ maxWidth: "min(92vw, 560px)" }}>
        <div className="apply-modal-header">
          <h3 id="server-folder-browser-title">{title}</h3>
          <button
            type="button"
            className="apply-modal-close-x"
            onClick={onClose}
            aria-label="Close"
          >
            ✕
          </button>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8 }}>
          <button type="button" onClick={onUp} disabled={loading}>↑ Up</button>
          <input
            type="text"
            value={path}
            onChange={(e) => setPath(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") void load(path); }}
            spellCheck={false}
            style={{ flex: 1, fontFamily: "ui-monospace, monospace", fontSize: 12 }}
            aria-label="Current path"
          />
        </div>

        {err ? <div className="err-banner">{err}</div> : null}

        <div
          style={{
            maxHeight: "45vh",
            overflowY: "auto",
            border: "1px solid var(--border-strong)",
            borderRadius: "var(--radius-sm)",
            background: "rgba(0,0,0,0.25)",
            fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
            fontSize: 12,
          }}
        >
          {loading ? (
            <div style={{ padding: 12, opacity: 0.7 }}>Loading…</div>
          ) : entries.length === 0 ? (
            <div style={{ padding: 12, opacity: 0.7 }}>Empty</div>
          ) : (
            entries.map((e) => (
              <div
                key={e.path}
                onClick={() => onEntryClick(e)}
                role="button"
                tabIndex={0}
                onKeyDown={(ev) => { if (ev.key === "Enter") onEntryClick(e); }}
                style={{
                  padding: "4px 10px",
                  cursor: e.isDir || filterExt ? "pointer" : "default",
                  opacity: e.isDir || filterExt ? 1 : 0.5,
                }}
              >
                {e.isDir ? "📁 " : "📄 "}{e.name}
              </div>
            ))
          )}
        </div>

        <div className="apply-modal-actions">
          <button type="button" onClick={onClose}>Cancel</button>
          {!filterExt ? (
            <button type="button" className="btn-primary" onClick={onConfirmFolder} disabled={!path}>
              Select this folder
            </button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
