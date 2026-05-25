import { Fragment, useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from "react";
import { Events } from "@wailsio/runtime";
import { Similarity } from "../bindings/folder-similarity/app/service";
import {
  type AppState,
  type FolderNode,
  type MergeFileRow,
  type MergePreview,
  ProgressEvent,
} from "../bindings/folder-similarity/app/dto";

import type { ActiveTab, MergeCtxMenuPaths, TreeSort } from "./types";
import { MAX_LOG } from "./constants";
import { formatBytes, formatPct, formatScanElapsed } from "./utils/format";
import { clampResultsCtxPosition, fileManagerRevealLabel, mergeFolderAbsPath } from "./utils/path";
import {
  explorerVisibleRowIndexes,
  getRowStatus,
  mergeFileRowOrdering,
  rowIndexesInclusiveRange,
  unifiedSizeDisplay,
} from "./utils/row";
import { computeActionImpact } from "./utils/action";
import { sortTree } from "./utils/tree";
import { CircularProgress } from "./components/ui/CircularProgress";
import { TreeList } from "./components/tree/TreeList";
import { ActionCellSide } from "./components/merge/ActionCellSide";
import { ExistsSquares } from "./components/merge/ExistsSquares";
import { FileNameCell } from "./components/merge/FileNameCell";
import { MergePairFileSizeBars } from "./components/merge/MergePairFileSizeBars";
import { ExplorerSidePanel } from "./components/gallery/ExplorerSidePanel";
import { ServerFolderBrowser } from "./components/dialog/ServerFolderBrowser";

type ServerPicker = { kind: "folder" | "json"; resolve: (path: string) => void } | null;

function App() {
  const [serverMode, setServerMode] = useState(false);
  const [serverPicker, setServerPicker] = useState<ServerPicker>(null);
  const [appState, setAppState] = useState<AppState | null>(null);
  const [tree, setTree] = useState<FolderNode | null>(null);
  const [merge, setMerge] = useState<MergePreview | null>(null);
  const [logs, setLogs] = useState<string[]>([]);
  const [treeSimilarityOnly, setTreeSimilarityOnly] = useState(true);
  const [treeSort, setTreeSort] = useState<TreeSort>("name-asc");
  const [err, setErr] = useState<string | null>(null);
  const [selectedMergeRows, setSelectedMergeRows] = useState<Set<number>>(() => new Set());
  const shiftRangeAnchorRef = useRef<number | null>(null);
  const [mergeCtxMenu, setMergeCtxMenu] = useState<{
    clientX: number;
    clientY: number;
    row: MergeFileRow;
    paths: MergeCtxMenuPaths;
  } | null>(null);
  const [applyLines, setApplyLines] = useState<string[] | null>(null);
  const [applyProgress, setApplyProgress] = useState<ProgressEvent | null>(null);
  const [logOpen, setLogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeTab, setActiveTab] = useState<ActiveTab>("results");
  const [scanDuration, setScanDuration] = useState<string>("");
  const [scanFileCount, setScanFileCount] = useState<number>(0);
  const [similarityGroupPickIdx, setSimilarityGroupPickIdx] = useState(0);

  const logPreRef = useRef<HTMLPreElement>(null);
  const mergePaneRef = useRef<HTMLElement | null>(null);
  const resultsTableSelectAllCheckboxRef = useRef<HTMLInputElement | null>(null);
  const scanStartRef = useRef<number | null>(null);
  const explorerLeftGalleryRef = useRef<HTMLDivElement | null>(null);
  const explorerRightGalleryRef = useRef<HTMLDivElement | null>(null);

  const mergeRowPickWithOrdering = useCallback(
    (orderedRowIndexes: number[], rowIndex: number, shiftKey: boolean) => {
      if (!shiftKey || shiftRangeAnchorRef.current === null) {
        shiftRangeAnchorRef.current = rowIndex;
        setSelectedMergeRows(new Set([rowIndex]));
        return;
      }
      const anchor = shiftRangeAnchorRef.current;
      if (!orderedRowIndexes.includes(anchor)) {
        shiftRangeAnchorRef.current = rowIndex;
        setSelectedMergeRows(new Set([rowIndex]));
        return;
      }
      setSelectedMergeRows(new Set(rowIndexesInclusiveRange(orderedRowIndexes, anchor, rowIndex)));
    },
    [],
  );

  const onExplorerMergeRowPick = useCallback(
    (side: "left" | "right", rowIndex: number, ev: Pick<MouseEvent<Element>, "shiftKey">) => {
      const rowsAll = merge?.rows;
      if (!rowsAll?.length) return;
      mergeRowPickWithOrdering(explorerVisibleRowIndexes(rowsAll, side), rowIndex, ev.shiftKey);
    },
    [merge?.rows, mergeRowPickWithOrdering],
  );

  const onResultsTableMergeRowPick = useCallback(
    (rowIndex: number, modifiers: { shiftKey: boolean }) => {
      const rowsAll = merge?.rows;
      if (!rowsAll?.length) return;
      mergeRowPickWithOrdering(mergeFileRowOrdering(rowsAll), rowIndex, modifiers.shiftKey);
    },
    [merge?.rows, mergeRowPickWithOrdering],
  );

  const clearMergeSelection = useCallback(() => {
    setSelectedMergeRows(new Set());
    shiftRangeAnchorRef.current = null;
  }, []);

  /** Reserve space for macOS traffic lights (Wails MacTitleBarHiddenInset draws them over the webview). */
  useEffect(() => {
    const ua = typeof navigator !== "undefined" ? navigator.userAgent : "";
    const plat = typeof navigator !== "undefined" ? navigator.platform : "";
    const isMac =
      /Mac OS X|Mac OS\b/i.test(ua) ||
      /^Mac/.test(plat);
    const root = document.documentElement;
    if (!isMac) return;
    root.style.setProperty("--mac-titlebar-inset-left", "78px");
    return () => {
      root.style.removeProperty("--mac-titlebar-inset-left");
    };
  }, []);

  /** Split-pane gallery scroll sync: passive listeners + ≤1 DOM write/frame (smooth on WKWebView / Chromium). */
  useEffect(() => {
    const rowCount = merge?.rows?.length ?? 0;
    if (activeTab !== "browseSplit" || rowCount === 0) return;

    const left = explorerLeftGalleryRef.current;
    const right = explorerRightGalleryRef.current;
    if (!left || !right) return;

    let rafId = 0;
    let syncSource: HTMLDivElement | null = null;
    let syncingPeer = false;

    const flush = () => {
      rafId = 0;
      const src = syncSource;
      if (!src || syncingPeer) return;
      const peer = src === left ? right : src === right ? left : null;
      if (!peer) return;
      const st = src.scrollTop;
      const sl = src.scrollLeft;
      if (peer.scrollTop === st && peer.scrollLeft === sl) return;
      syncingPeer = true;
      peer.scrollTop = st;
      peer.scrollLeft = sl;
      syncingPeer = false;
    };

    const onScroll = (ev: Event) => {
      if (syncingPeer) return;
      syncSource = ev.currentTarget as HTMLDivElement;
      if (rafId !== 0) return;
      rafId = window.requestAnimationFrame(flush);
    };

    left.addEventListener("scroll", onScroll, { passive: true });
    right.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      left.removeEventListener("scroll", onScroll);
      right.removeEventListener("scroll", onScroll);
      if (rafId !== 0) window.cancelAnimationFrame(rafId);
    };
  }, [activeTab, merge?.rows?.length]);

  const pushLog = useCallback((line: string) => {
    setLogs((prev) => {
      const n = prev.concat(line);
      return n.length > MAX_LOG ? n.slice(-MAX_LOG) : n;
    });
  }, []);

  useEffect(() => {
    if (!logOpen || !logPreRef.current) return;
    const el = logPreRef.current;
    el.scrollTop = el.scrollHeight;
  }, [logs, logOpen]);

  const mergeSelectionScrollKey = useMemo(
    () => [...selectedMergeRows].sort((a, b) => a - b).join(","),
    [selectedMergeRows],
  );

  const sortedTree = useMemo(
    () => (tree ? sortTree(tree, treeSort) : null),
    [tree, treeSort],
  );

  useEffect(() => {
    if (selectedMergeRows.size === 0) return;
    const pane = mergePaneRef.current;
    if (!pane) return;
    const id = requestAnimationFrame(() => {
      pane.querySelector("tr.row-sel, .gallery-card.gallery-card-sel")?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    });
    return () => cancelAnimationFrame(id);
  }, [mergeSelectionScrollKey, selectedMergeRows.size, activeTab]);

  const refresh = useCallback(async () => {
    setErr(null);
    try {
      const st = await Similarity.GetState();
      setAppState(st);
      if (!st.hasData) {
        setTree(null);
        setMerge(null);
        return;
      }
      const t = await Similarity.GetTree(treeSimilarityOnly);
      setTree(t);
      try {
        const m = await Similarity.GetMergePreview();
        setMerge(m);
      } catch {
        setMerge(null);
      }
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  }, [treeSimilarityOnly]);

  useEffect(() => {
    refresh();
  }, [refresh, treeSimilarityOnly]);

  useEffect(() => {
    void Similarity.IsServerMode()
      .then((v) => setServerMode(Boolean(v)))
      .catch(() => setServerMode(false));
  }, []);

  const pickServerPath = useCallback(
    (kind: "folder" | "json") =>
      new Promise<string>((resolve) => {
        setServerPicker({ kind, resolve });
      }),
    [],
  );

  useEffect(() => {
    const offLog = Events.On("log", (ev: any) => {
      const d = ev?.data;
      if (typeof d === "string") { pushLog(d); return; }
      if (d != null) pushLog(String(d));
    });
    const offPhase = Events.On("phase", (ev: any) => {
      const d = ev?.data;
      if (d?.phase === "scanning") {
        scanStartRef.current = Date.now();
      } else if (d?.phase === "ready" && scanStartRef.current) {
        const ms = Date.now() - scanStartRef.current;
        setScanDuration(formatScanElapsed(ms));
        scanStartRef.current = null;
      } else if (d?.phase === "idle" && String(d?.message || "").toLowerCase() === "cancelled") {
        scanStartRef.current = null;
        shiftRangeAnchorRef.current = null;
        setSelectedMergeRows(new Set());
        setMergeCtxMenu(null);
        setScanDuration("");
        setScanFileCount(0);
      }
      if (d?.phase === "ready") {
        Similarity.SelectFolder(".")
          .then(() => refresh())
          .catch((e) => setErr(String(e)));
      } else {
        void refresh();
      }
    });
    const offProgress = Events.On("progress", (ev: any) => {
      const d = ev?.data;
      if (d && typeof d === "object") setApplyProgress(d);
      const ph = d?.phase;
      if (ph === "done" || ph === "error" || ph === "cancelled") void refresh();
    });
    const offScanProgress = Events.On("scanProgress", (ev: any) => {
      const d = ev?.data;
      const n =
        typeof d?.filesScanned === "number"
          ? d.filesScanned
          : typeof d?.FilesScanned === "number"
            ? d.FilesScanned
            : NaN;
      if (Number.isFinite(n) && n >= 0) setScanFileCount(Math.floor(n));
    });
    return () => {
      offLog();
      offPhase();
      offProgress();
      offScanProgress();
    };
  }, [pushLog, refresh]);

  const pickAndScan = async () => {
    setErr(null);
    try {
      const p = serverMode ? await pickServerPath("folder") : await Similarity.PickRootFolder();
      if (!p) return;
      setScanDuration("");
      setScanFileCount(0);
      await Similarity.StartScan(p);
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  };

  const loadJson = async () => {
    setErr(null);
    try {
      const p = serverMode ? await pickServerPath("json") : await Similarity.PickJSONFile();
      if (!p) return;
      await Similarity.LoadFromJSONFile(p);
      await Similarity.SelectFolder(".");
      await refresh();
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  };

  const exportJson = async () => {
    setErr(null);
    try {
      if (serverMode) {
        const data = await Similarity.GetExportedJSON();
        const blob = new Blob([data], { type: "application/json" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = "db.json";
        document.body.appendChild(a);
        a.click();
        a.remove();
        URL.revokeObjectURL(url);
      } else {
        await Similarity.ExportDataToJSON();
      }
      await refresh();
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  };

  const onRevealLeft = useCallback(async () => {
    try { await Similarity.RevealMergeFolder("left"); } catch (e: any) { setErr(e?.message || String(e)); }
  }, []);

  const onRevealRight = useCallback(async () => {
    try { await Similarity.RevealMergeFolder("right"); } catch (e: any) { setErr(e?.message || String(e)); }
  }, []);

  /** Same as context menu “Open … file/folder”: default app for that side’s path. */
  const onExplorerMergeRowOpen = useCallback((side: "left" | "right", row: MergeFileRow) => {
    const rel = (side === "left" ? row.leftPath : row.rightPath)?.trim() ?? "";
    if (!rel) return;
    void Similarity.OpenScannedPath(rel).catch((e: unknown) => {
      setErr(String((e as { message?: string })?.message || e));
    });
  }, []);

  const closeMergeCtxMenu = useCallback(() => setMergeCtxMenu(null), []);

  const onSelectFolder = async (path: string) => {
    setErr(null);
    clearMergeSelection();
    try {
      await Similarity.SelectFolder(path);
      const m = await Similarity.GetMergePreview();
      setMerge(m);
      setAppState(await Similarity.GetState());
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  };

  const onPickGroup = async (idx: number) => {
    setErr(null);
    try {
      await Similarity.SelectSimilarityGroupIndex(idx);
      clearMergeSelection();
      const m = await Similarity.GetMergePreview();
      setMerge(m);
      setAppState(await Similarity.GetState());
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  };

  const onApplyPreview = useCallback(async () => {
    setErr(null);
    setApplyProgress(null);
    try {
      const lines = await Similarity.GetApplyPreview();
      setApplyLines(lines);
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  }, []);

  const runApplyOnDisk = useCallback(async () => {
    setErr(null);
    try {
      setApplyProgress(new ProgressEvent({ phase: "running", message: "starting…", current: 0, total: 0 }));
      await Similarity.ApplyExecute();
    } catch (e: any) {
      setErr(e?.message || String(e));
      setApplyProgress(null);
    }
  }, []);

  const clearApplyModal = useCallback(() => {
    setApplyProgress(null);
    setApplyLines(null);
  }, []);

  const onBulk = useCallback(async (action: string) => {
    setErr(null);
    try {
      if (selectedMergeRows.size === 0) {
        await Similarity.SetAllCompareRowActions(action);
      } else {
        await Promise.all(
          [...selectedMergeRows].map((rowIdx) => Similarity.SetCompareRowAction(rowIdx, action)),
        );
      }
      await refresh();
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  }, [refresh, selectedMergeRows]);

  const onClearAll = useCallback(async () => {
    setErr(null);
    try {
      await Similarity.ClearAllCompareRowActions();
      await refresh();
    } catch (e: any) {
      setErr(e?.message || String(e));
    }
  }, [refresh]);

  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => {
      if (appState?.applyRunning) return;
      if (applyLines) return;
      if (appState?.needGroupPick && (appState?.groupLabels?.length || 0) > 0) return;
      const target = ev.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;

      if (ev.key === "Escape") {
        if (mergeCtxMenu) {
          ev.preventDefault();
          closeMergeCtxMenu();
          return;
        }
        if (selectedMergeRows.size > 0) {
          ev.preventDefault();
          clearMergeSelection();
          return;
        }
      }

      const rowsNav = merge?.rows;
      if (rowsNav?.length && (ev.key === "ArrowUp" || ev.key === "ArrowDown")) {
        const blockingBtn = target.closest("button");
        if (blockingBtn && !blockingBtn.classList.contains("gallery-card")) return;
        ev.preventDefault();
        const down = ev.key === "ArrowDown";

        let pos: number | null = null;
        const anch = shiftRangeAnchorRef.current;
        if (anch !== null && selectedMergeRows.has(anch)) {
          const p = rowsNav.findIndex((r) => r.rowIndex === anch);
          if (p >= 0) pos = p;
        }
        if (pos === null && selectedMergeRows.size > 0) {
          const positions = [...selectedMergeRows]
            .map((idx) => rowsNav.findIndex((r) => r.rowIndex === idx))
            .filter((p) => p >= 0)
            .sort((a, b) => a - b);
          pos = positions.length === 0 ? null : down ? positions[positions.length - 1] : positions[0];
        }

        let nextIdx: number;
        if (pos === null) {
          nextIdx = down ? 0 : rowsNav.length - 1;
        } else {
          const cand = pos + (down ? 1 : -1);
          if (cand < 0 || cand >= rowsNav.length) return;
          nextIdx = cand;
        }

        const nextRow = rowsNav[nextIdx];
        shiftRangeAnchorRef.current = nextRow.rowIndex;
        setSelectedMergeRows(new Set([nextRow.rowIndex]));
        return;
      }

      if (!merge?.interactive) return;

      const nFile = (merge?.rows || []).filter((r) => !r.isFolder).length;
      const hasFiles = nFile > 0;

      if (ev.key === ">" && hasFiles) { ev.preventDefault(); void onBulk("deleteRight"); return; }
      if (ev.key === "<" && hasFiles) { ev.preventDefault(); void onBulk("deleteLeft"); return; }
      if (ev.key === "ArrowRight" && ev.shiftKey && hasFiles) { ev.preventDefault(); void onBulk("moveToRight"); return; }
      if (ev.key === "ArrowLeft" && ev.shiftKey && hasFiles) { ev.preventDefault(); void onBulk("moveToLeft"); return; }
      if (ev.key === "A" && !ev.metaKey && !ev.ctrlKey) { ev.preventDefault(); void onApplyPreview(); return; }
      if (ev.key === "C" && ev.shiftKey) { ev.preventDefault(); void onClearAll(); return; }

      const targetRows = [...selectedMergeRows];
      const hasExplicitSelection = targetRows.length > 0;

      const map: Record<string, string> = { ",": "deleteLeft", ".": "deleteRight", ArrowLeft: "moveToLeft", ArrowRight: "moveToRight" };
      const needsSelection = ["c", "Backspace"].includes(ev.key) || !!map[ev.key];
      if (needsSelection && !hasExplicitSelection) return;

      if (ev.key === "c" && !ev.shiftKey && !ev.metaKey && !ev.ctrlKey) {
        ev.preventDefault();
        void Promise.all(targetRows.map((rowIdx) => Similarity.ClearCompareRowAction(rowIdx))).then(() => void refresh());
        return;
      }
      if (ev.key === "Backspace") {
        ev.preventDefault();
        void Promise.all(targetRows.map((rowIdx) => Similarity.ClearCompareRowAction(rowIdx))).then(() => void refresh());
        return;
      }
      const a = map[ev.key];
      if (a && !ev.ctrlKey && !ev.metaKey) {
        ev.preventDefault();
        void Promise.all(targetRows.map((rowIdx) => Similarity.SetCompareRowAction(rowIdx, a))).then(() => void refresh());
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [
    merge,
    selectedMergeRows,
    applyLines,
    refresh,
    onBulk,
    onClearAll,
    onApplyPreview,
    appState?.applyRunning,
    appState?.needGroupPick,
    appState?.groupLabels?.length,
    mergeCtxMenu,
    closeMergeCtxMenu,
    clearMergeSelection,
  ]);

  const st = appState;
  const scanning = st?.scanning;
  const needGroup = st?.needGroupPick && (st?.groupLabels?.length || 0) > 0;

  useEffect(() => {
    if (needGroup) setSimilarityGroupPickIdx(0);
  }, [needGroup]);

  // ApplyRunning can stay true briefly after a terminal progress event while refreshAfterApply runs (Go defer finishApply).
  const applyTerminalProgress =
    applyProgress?.phase === "done" ||
    applyProgress?.phase === "cancelled" ||
    applyProgress?.phase === "error";
  const applyInProgress = Boolean(
    !applyTerminalProgress && (st?.applyRunning || applyProgress?.phase === "running"),
  );
  const applyCompletedSuccessfully = applyProgress?.phase === "done";
  const canOpenInOS = Boolean(st?.rootPath && merge?.interactive);
  const hasRevealRoot = Boolean(st?.rootPath && String(st.rootPath).trim());
  const scanDone = st?.phase === "ready" && st?.hasData;

  const fileRows = (merge?.rows || []).filter(r => !r.isFolder);
  const exactCount = fileRows.filter(r => getRowStatus(r) === "exact").length;
  const similarCount = fileRows.filter(r => getRowStatus(r) === "similar").length;

  const leftImpact = computeActionImpact(fileRows, "left");
  const rightImpact = computeActionImpact(fileRows, "right");

  const overallSim = merge
    ? ((merge.leftDupPct + merge.rightDupPct) / 2)
    : 0;

  const tabRows = (() => {
    const all = merge?.rows || [];
    switch (activeTab) {
      case "browseLeft":
      case "browseRight":
      case "browseSplit":
        return [];
      default:
        return all.filter(r => !r.isFolder);
    }
  })();

  const tabRowsKey = tabRows.map((r) => r.rowIndex).join(",");

  useEffect(() => {
    const el = resultsTableSelectAllCheckboxRef.current;
    if (!el || tabRows.length === 0) return;
    el.indeterminate =
      tabRows.some((r) => selectedMergeRows.has(r.rowIndex)) &&
      !tabRows.every((r) => selectedMergeRows.has(r.rowIndex));
  }, [tabRowsKey, mergeSelectionScrollKey, tabRows.length, selectedMergeRows]);

  const toggleSelectAllResultsRows = () => {
    if (tabRows.length === 0) return;
    if (tabRows.every((r) => selectedMergeRows.has(r.rowIndex))) clearMergeSelection();
    else {
      const first = tabRows[0];
      shiftRangeAnchorRef.current = first ? first.rowIndex : null;
      setSelectedMergeRows(new Set(tabRows.map((r) => r.rowIndex)));
    }
  };

  const browseSide = activeTab === "browseLeft" ? "left" : activeTab === "browseRight" ? "right" : null;
  const showFolderExplorer = browseSide !== null || activeTab === "browseSplit";

  const selectedRowsWithAction = fileRows.filter(r => r.action !== "none" && r.action !== "");
  const hasFileRows = fileRows.length > 0;

  return (
    <div className="app-shell">
      {/* Top Bar */}
      <header className="topbar">
        <div className="topbar-brand">
          <div className="brand-icon">
            <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
              <rect width="28" height="28" rx="8" fill="url(#brandGrad)" />
              <path d="M7 9h14M7 14h9M7 19h11" stroke="white" strokeWidth="2" strokeLinecap="round" />
              <defs>
                <linearGradient id="brandGrad" x1="0" y1="0" x2="28" y2="28">
                  <stop stopColor="#7c4dff" /><stop offset="1" stopColor="#5c6bc0" />
                </linearGradient>
              </defs>
            </svg>
          </div>
          <span className="brand-title">Folder Similarity</span>
        </div>
        <div className="topbar-actions">
          <button type="button" className="btn-primary topbar-btn" onClick={pickAndScan} disabled={!!scanning || applyInProgress}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><circle cx="6" cy="6" r="4.5" stroke="currentColor" strokeWidth="1.5" /><path d="M9.5 9.5L12 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" /></svg>
            Pick &amp; Scan
          </button>
          <button type="button" className="topbar-btn" onClick={loadJson} disabled={!!scanning || applyInProgress}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2 3h10M2 7h10M2 11h6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" /></svg>
            Load JSON
          </button>
          <button type="button" className="topbar-btn" onClick={() => void exportJson()} disabled={!!scanning || applyInProgress || !st?.hasData}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2v8M4 7l3 3 3-3M2 11h10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" /></svg>
            Export JSON
          </button>
          <button type="button" className="topbar-btn" onClick={() => void refresh()} disabled={!!scanning}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2 7a5 5 0 1 1 1 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" /><path d="M2 10V7h3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" /></svg>
            Refresh
          </button>
          {scanning ? (
            <button type="button" className="topbar-btn btn-danger" onClick={() => Similarity.CancelScan()}>
              Cancel
            </button>
          ) : null}
          <button type="button" className="topbar-btn topbar-settings" title="Settings">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 10a2 2 0 1 0 0-4 2 2 0 0 0 0 4z" stroke="currentColor" strokeWidth="1.4" /><path d="M13.3 6.7l-.8-.5a4.7 4.7 0 0 0 0-1.4l.8-.5a1 1 0 0 0 .4-1.3L13 1.9a1 1 0 0 0-1.3-.4l-.8.5a5 5 0 0 0-1.2-.7V.5A1 1 0 0 0 8.7 0h-.9a1 1 0 0 0-1 1v.8a5 5 0 0 0-1.2.7l-.8-.5a1 1 0 0 0-1.3.4l-.5 1a1 1 0 0 0 .4 1.3l.8.5a4.7 4.7 0 0 0 0 1.4l-.8.5a1 1 0 0 0-.4 1.3l.5 1a1 1 0 0 0 1.3.4l.8-.5a5 5 0 0 0 1.2.7v.8a1 1 0 0 0 1 1h.9a1 1 0 0 0 1-1v-.8a5 5 0 0 0 1.2-.7l.8.5a1 1 0 0 0 1.3-.4l.5-1a1 1 0 0 0-.4-1.3z" stroke="currentColor" strokeWidth="1.2" /></svg>
          </button>
        </div>
      </header>

      {err ? <div className="err-banner">{err}<button type="button" onClick={() => setErr(null)}>✕</button></div> : null}

      {/* Server-mode folder / JSON picker (modal) */}
      <ServerFolderBrowser
        open={serverPicker !== null}
        title={serverPicker?.kind === "json" ? "Select JSON file" : "Select scan root folder"}
        filterExt={serverPicker?.kind === "json" ? ".json" : undefined}
        onSelect={(p) => { const r = serverPicker?.resolve; setServerPicker(null); r?.(p); }}
        onClose={() => { const r = serverPicker?.resolve; setServerPicker(null); r?.(""); }}
      />

      {/* Similarity group picker (modal) */}
      {needGroup && st?.groupLabels ? (
        <div
          className="modal-backdrop group-picker-backdrop"
          role="dialog"
          aria-modal="true"
          aria-labelledby="group-picker-title"
        >
          <div className="apply-modal group-picker-modal" onClick={(e) => e.stopPropagation()}>
            <div className="apply-modal-header">
              <h3 id="group-picker-title">Select similarity group</h3>
            </div>
            <p className="apply-modal-note">
              Multiple similarity groups match this folder. Choose one to open the comparison view.
            </p>
            <select
              className="group-picker-select"
              size={Math.min(Math.max(st.groupLabels.length, 2), 12)}
              value={String(similarityGroupPickIdx)}
              onChange={(e) => setSimilarityGroupPickIdx(Number(e.target.value))}
              aria-label="Similarity groups"
            >
              {st.groupLabels.map((label, i) => (
                <option key={i} value={i}>{label}</option>
              ))}
            </select>
            <div className="apply-modal-actions group-picker-actions">
              <button
                type="button"
                className="btn-primary"
                onClick={() => void onPickGroup(similarityGroupPickIdx)}
              >
                Continue
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {/* Apply Modal */}
      {applyLines ? (
        <div className="modal-backdrop" onClick={() => { if (!applyInProgress && !applyCompletedSuccessfully) void clearApplyModal(); }}>
          <div className="apply-modal" onClick={(e) => e.stopPropagation()}>
            <div className="apply-modal-header">
              <h3>Apply — Preview &amp; On-disk</h3>
              <button
                type="button"
                className="apply-modal-close-x"
                aria-label="Close"
                title="Close"
                disabled={applyInProgress}
                onClick={() => clearApplyModal()}
              >
                ×
              </button>
            </div>
            <p className="apply-modal-note">
              Review the task list, then run once to execute on disk. The app validates folders and rejects unsafe paths before changing files.
            </p>
            <pre className="apply-pre">{applyLines.join("\n") || "(no actions)"}</pre>
            {applyProgress ? (
              <div className="apply-progress-wrap" aria-live="polite">
                {applyProgress.total != null && applyProgress.total > 0 && applyProgress.current != null ? (
                  <div className="apply-progress-bar" role="progressbar">
                    <div className="apply-progress-fill" style={{ width: `${Math.min(100, (100 * applyProgress.current) / applyProgress.total)}%` }} />
                  </div>
                ) : null}
                <div className="apply-progress-text">
                  {applyProgress.phase === "running" ? "Running…" : null}
                  {applyProgress.phase === "done" ? "Complete." : null}
                  {applyProgress.phase === "cancelled" ? "Cancelled." : null}
                  {applyProgress.phase === "error" ? (applyProgress.error || applyProgress.message || "Error") : null}
                </div>
                {applyProgress.message && applyProgress.phase === "running" ? (
                  <div className="apply-progress-current">{applyProgress.message}</div>
                ) : null}
              </div>
            ) : null}
            <div className="apply-modal-actions">
              {applyInProgress ? (
                <button type="button" className="btn-cancel-apply" onClick={() => Similarity.CancelApply()}>Cancel apply</button>
              ) : (
                <button type="button" className="btn-primary" disabled={applyCompletedSuccessfully} onClick={() => void runApplyOnDisk()}>Run on disk</button>
              )}
              <button type="button" onClick={() => clearApplyModal()} disabled={applyInProgress}>Close</button>
            </div>
          </div>
        </div>
      ) : null}

      {/* Main Layout */}
      <div className="main-layout">
        {/* Sidebar */}
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-header-row">
              <h2 className="sidebar-title">Folders</h2>
              <div className="sidebar-sort-controls">
                <button
                  type="button"
                  className={"sort-btn" + (treeSort.startsWith("name") ? " sort-btn-active" : "")}
                  title={treeSort === "name-desc" ? "Sort by name (Z→A) — click for A→Z" : "Sort by name (A→Z) — click for Z→A"}
                  aria-label="Sort by folder name"
                  onClick={() => setTreeSort((s) => (s === "name-asc" ? "name-desc" : "name-asc"))}
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
                    <text x="2" y="7" fontSize="6" fontWeight="700" fill="currentColor">A</text>
                    <text x="9" y="13" fontSize="6" fontWeight="700" fill="currentColor">Z</text>
                    {treeSort === "name-desc" ? (
                      <path d="M7 4l-2 2.5h4L7 4z" fill="currentColor" />
                    ) : (
                      <path d="M7 12l-2-2.5h4L7 12z" fill="currentColor" />
                    )}
                  </svg>
                </button>
                <button
                  type="button"
                  className={"sort-btn" + (treeSort === "size-desc" ? " sort-btn-active" : "")}
                  title="Sort by total size (largest first)"
                  aria-label="Sort by total size descending"
                  onClick={() => setTreeSort("size-desc")}
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
                    <rect x="2" y="3" width="11" height="2" fill="currentColor" />
                    <rect x="2" y="7" width="8" height="2" fill="currentColor" />
                    <rect x="2" y="11" width="5" height="2" fill="currentColor" />
                  </svg>
                </button>
              </div>
              <label className="sidebar-tree-filter">
                <input
                  type="checkbox"
                  checked={treeSimilarityOnly}
                  onChange={(e) => setTreeSimilarityOnly(e.target.checked)}
                  aria-label="Show only folders that participate in similarity"
                  title="When checked, only folders with similarity groups are shown. Uncheck to show the full tree."
                />
              </label>
            </div>
          </div>
          <div className="sidebar-search">
            <svg className="search-icon" width="14" height="14" viewBox="0 0 14 14" fill="none">
              <circle cx="6" cy="6" r="4.5" stroke="currentColor" strokeWidth="1.5" />
              <path d="M9.5 9.5L12 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
            <input
              type="text"
              className="search-input"
              placeholder="Search folders..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
            />
          </div>

          <div className="sidebar-tree">
            {sortedTree && (sortedTree.name || sortedTree.path) ? (
              <TreeList
                node={sortedTree}
                depth={0}
                selected={st?.selected || "."}
                onSelect={onSelectFolder}
                searchQuery={searchQuery}
              />
            ) : (
              <div className="tree-empty">
                {scanning ? "Scanning…" : "No tree yet — run a scan or load JSON."}
              </div>
            )}
          </div>

          {/* Scan Summary */}
          {st?.hasData && !scanning ? (
            <div className="scan-summary">
              <h3 className="scan-summary-title">Scan Summary</h3>
              <div className="summary-rows">
                <div className="summary-row">
                  <span className="summary-label">Folders Scanned</span>
                  <span className="summary-value">{st.groupCount > 0 ? st.groupCount : "—"}</span>
                </div>
                <div className="summary-row">
                  <span className="summary-label">Files Scanned</span>
                  <span className="summary-value highlight">{scanFileCount > 0 ? scanFileCount.toLocaleString() : (fileRows.length > 0 ? fileRows.length.toLocaleString() : "—")}</span>
                </div>
                <div className="summary-row">
                  <span className="summary-label">Duplicates Found</span>
                  <span className="summary-value highlight">{exactCount + similarCount > 0 ? (exactCount + similarCount).toLocaleString() : "—"}</span>
                </div>
                <div className="summary-row">
                  <span className="summary-label">Similarity</span>
                  <span className="summary-value summary-pct">{merge ? formatPct(overallSim) : "—"}</span>
                </div>
              </div>
              {merge ? (
                <div className="summary-progress-bar">
                  <div className="summary-progress-fill" style={{ width: `${Math.min(100, overallSim)}%` }} />
                </div>
              ) : null}
            </div>
          ) : null}

          {/* Scan Status */}
          <div className="sidebar-footer">
            {scanDone ? (
              <div className="scan-status scan-status-done">
                <div className="scan-done-icon" aria-hidden>
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                    <circle cx="12" cy="12" r="10.25" stroke="#22c55e" strokeWidth="1.35" />
                    <path d="M7.25 12l2.75 2.75 6.75-6.75" stroke="#22c55e" strokeWidth="1.65" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </div>
                <div className="scan-done-text">Scan completed</div>
                {scanFileCount > 0 && scanDuration ? (
                  <div className="scan-done-detail">{scanFileCount.toLocaleString()} files scanned in {scanDuration}</div>
                ) : null}
              </div>
            ) : scanning ? (
              <div className="scan-status">
                <div className="scan-spinner" />
                <div className="scan-done-text">Scanning…</div>
                {scanFileCount > 0 ? (
                  <div className="scan-done-detail">{scanFileCount.toLocaleString()} files scanned</div>
                ) : null}
              </div>
            ) : null}
            {(logs.length > 0 || scanning) ? (
              <button type="button" className="btn-view-log" onClick={() => setLogOpen(v => !v)}>
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <rect x="1" y="2" width="12" height="10" rx="2" stroke="currentColor" strokeWidth="1.4" />
                  <path d="M4 5h6M4 7.5h4" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
                </svg>
                View Scan Log
              </button>
            ) : null}
          </div>
        </aside>

        {/* Main Content */}
        <main className="content-area" ref={mergePaneRef} tabIndex={0}>
          {/* Similarity Overview */}
          {merge?.leftPath ? (
            <section className="overview-section">
              <h2 className="section-title">Similarity Overview</h2>
              <div className="overview-cards">
                {/* Circular progress */}
                <div className="ov-card ov-card-circ">
                  <CircularProgress pct={overallSim} label="Overall Similarity" />
                </div>

                {/* Left folder */}
                <div className="ov-card ov-card-folder ov-card-left">
                  <div className="folder-card-header">
                    <svg className="folder-icon folder-icon-left" width="20" height="20" viewBox="0 0 20 20" fill="none">
                      <path d="M2 5a2 2 0 0 1 2-2h3l2 2h7a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5z" fill="currentColor" fillOpacity=".9" />
                    </svg>
                    <span
                      className="folder-card-name"
                      title={
                        mergeFolderAbsPath(st?.rootPath, merge.leftPath) ??
                        ((merge.leftPath ?? "").trim() || undefined)
                      }
                    >{merge.leftLabel || "—"}</span>
                    {canOpenInOS ? (
                      <button type="button" className="btn-open-folder" onClick={onRevealLeft} disabled={applyInProgress}>Open</button>
                    ) : null}
                  </div>
                  <div className="folder-card-main">
                    <div className="folder-card-statblock">
                      <div className="folder-card-dup">
                        <span className="folder-dup-pct">{formatPct(merge.leftDupPct)}</span>
                      </div>
                      <div className="folder-card-files">Duplicates</div>
                      <div className="folder-card-totals">
                        <div className="folder-card-total">
                          Files: <strong>{(merge.leftTotalFiles ?? 0).toLocaleString()}</strong>
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(Number(merge.leftChildFilesSizeBytes ?? 0))}</span>
                        </div>
                        <div className="folder-card-total folder-card-total-recursive">
                          All files: <strong>{(merge.leftTotalFilesRecursive ?? 0).toLocaleString()}</strong>
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(Number(merge.leftTotalFilesSizeBytes ?? 0))}</span>
                        </div>
                      </div>
                      <div className="folder-card-impact">
                        <div
                          className={"folder-card-impact-row folder-card-impact-del" + (leftImpact.deleteCount === 0 ? " folder-card-impact-row-empty" : "")}
                          aria-hidden={leftImpact.deleteCount === 0}
                        >
                          Delete: <strong>{leftImpact.deleteCount.toLocaleString()}</strong> file{leftImpact.deleteCount === 1 ? "" : "s"}
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(leftImpact.deleteSize)}</span>
                        </div>
                        <div
                          className={"folder-card-impact-row folder-card-impact-add" + (leftImpact.addCount === 0 ? " folder-card-impact-row-empty" : "")}
                          aria-hidden={leftImpact.addCount === 0}
                        >
                          Add: <strong>{leftImpact.addCount.toLocaleString()}</strong> file{leftImpact.addCount === 1 ? "" : "s"}
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(leftImpact.addSize)}</span>
                        </div>
                      </div>
                    </div>
                    <MergePairFileSizeBars rows={merge.rows} side="left" barColor="#ef4444" />
                  </div>
                </div>

                {/* Right folder */}
                <div className="ov-card ov-card-folder ov-card-right">
                  <div className="folder-card-header">
                    <svg className="folder-icon folder-icon-right" width="20" height="20" viewBox="0 0 20 20" fill="none">
                      <path d="M2 5a2 2 0 0 1 2-2h3l2 2h7a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5z" fill="currentColor" fillOpacity=".9" />
                    </svg>
                    <span
                      className="folder-card-name"
                      title={
                        mergeFolderAbsPath(st?.rootPath, merge.rightPath) ??
                        ((merge.rightPath ?? "").trim() || undefined)
                      }
                    >{merge.rightLabel || "—"}</span>
                    {canOpenInOS ? (
                      <button type="button" className="btn-open-folder" onClick={onRevealRight} disabled={applyInProgress}>Open</button>
                    ) : null}
                  </div>
                  <div className="folder-card-main">
                    <div className="folder-card-statblock">
                      <div className="folder-card-dup">
                        <span className="folder-dup-pct folder-dup-pct-right">{formatPct(merge.rightDupPct)}</span>
                      </div>
                      <div className="folder-card-files">Duplicates</div>
                      <div className="folder-card-totals">
                        <div className="folder-card-total">
                          Files: <strong>{(merge.rightTotalFiles ?? 0).toLocaleString()}</strong>
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(Number(merge.rightChildFilesSizeBytes ?? 0))}</span>
                        </div>
                        <div className="folder-card-total folder-card-total-recursive">
                          All files: <strong>{(merge.rightTotalFilesRecursive ?? 0).toLocaleString()}</strong>
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(Number(merge.rightTotalFilesSizeBytes ?? 0))}</span>
                        </div>
                      </div>
                      <div className="folder-card-impact">
                        <div
                          className={"folder-card-impact-row folder-card-impact-del" + (rightImpact.deleteCount === 0 ? " folder-card-impact-row-empty" : "")}
                          aria-hidden={rightImpact.deleteCount === 0}
                        >
                          Delete: <strong>{rightImpact.deleteCount.toLocaleString()}</strong> file{rightImpact.deleteCount === 1 ? "" : "s"}
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(rightImpact.deleteSize)}</span>
                        </div>
                        <div
                          className={"folder-card-impact-row folder-card-impact-add" + (rightImpact.addCount === 0 ? " folder-card-impact-row-empty" : "")}
                          aria-hidden={rightImpact.addCount === 0}
                        >
                          Add: <strong>{rightImpact.addCount.toLocaleString()}</strong> file{rightImpact.addCount === 1 ? "" : "s"}
                          <span className="folder-card-stat-sep" aria-hidden="true"> · </span>
                          <span className="folder-card-size">{formatBytes(rightImpact.addSize)}</span>
                        </div>
                      </div>
                    </div>
                    <MergePairFileSizeBars rows={merge.rows} side="right" barColor="#fb923c" />
                  </div>
                </div>
              </div>
            </section>
          ) : null}

          {/* Results Section */}
          <section className="results-section">
            {/* Tabs + toolbar */}
            <div className="results-header">
              <div className="results-tabs">
                <button type="button" className={`tab ${activeTab === "results" ? "tab-active" : ""}`} onClick={() => { setActiveTab("results"); }}>Results</button>
                <button type="button" className={`tab ${activeTab === "browseLeft" ? "tab-active" : ""}`} onClick={() => { setActiveTab("browseLeft"); }}>Left folder</button>
                <button type="button" className={`tab ${activeTab === "browseRight" ? "tab-active" : ""}`} onClick={() => { setActiveTab("browseRight"); }}>Right folder</button>
                <button type="button" className={`tab ${activeTab === "browseSplit" ? "tab-active" : ""}`} onClick={() => { setActiveTab("browseSplit"); }}>Both folders</button>
              </div>
              {merge?.interactive && hasFileRows ? (
                <div className="results-toolbar">
                  {selectedMergeRows.size > 0 ? (
                    <span className="selection-badge">{selectedMergeRows.size} selected</span>
                  ) : null}
                  {selectedRowsWithAction.length > 0 ? (
                    <span className="selection-badge">{selectedRowsWithAction.length} with action</span>
                  ) : null}
                  <div className="toolbar-sep" />
                  <button type="button" className="toolbar-icon-btn" title={selectedMergeRows.size > 0 ? `Remove right-side duplicates for ${selectedMergeRows.size} selected row(s)` : "Remove right-side duplicates (all file rows)"} onClick={() => void onBulk("deleteRight")} disabled={!hasFileRows || applyInProgress}>⌦</button>
                  <button type="button" className="toolbar-icon-btn" title={selectedMergeRows.size > 0 ? `Remove left-side duplicates for ${selectedMergeRows.size} selected row(s)` : "Remove left-side duplicates (all file rows)"} onClick={() => void onBulk("deleteLeft")} disabled={!hasFileRows || applyInProgress}>⌫</button>
                  <button type="button" className="toolbar-icon-btn" title={selectedMergeRows.size > 0 ? `Move duplicates to right for ${selectedMergeRows.size} selected row(s)` : "Move duplicates to right (all file rows)"} onClick={() => void onBulk("moveToRight")} disabled={!hasFileRows || applyInProgress}>→</button>
                  <button type="button" className="toolbar-icon-btn" title={selectedMergeRows.size > 0 ? `Move duplicates to left for ${selectedMergeRows.size} selected row(s)` : "Move duplicates to left (all file rows)"} onClick={() => void onBulk("moveToLeft")} disabled={!hasFileRows || applyInProgress}>←</button>
                  <div className="toolbar-sep" />
                  <button type="button" className="toolbar-btn" onClick={onClearAll} disabled={applyInProgress}>Clear</button>
                  <button type="button" className="toolbar-btn btn-primary" onClick={() => void onApplyPreview()} disabled={applyInProgress}>Apply (preview)</button>
                </div>
              ) : null}
            </div>

            {/* Table */}
            {!merge ? (
              <div className="results-empty">Select a folder in the tree to load the comparison.</div>
            ) : merge.note && (!merge.rows || merge.rows.length === 0) ? (
              <div className="results-empty results-note">{merge.note}</div>
            ) : showFolderExplorer && activeTab === "browseSplit" ? (
              (merge.rows && merge.rows.length > 0) ? (
                <div className="explorer-wrap explorer-wrap-split">
                  <ExplorerSidePanel
                    merge={merge}
                    side="left"
                    selectedRowIndexes={selectedMergeRows}
                    onMergeRowPick={(idx, ev) => onExplorerMergeRowPick("left", idx, ev)}
                    onMergeRowDoubleClick={(row) => onExplorerMergeRowOpen("left", row)}
                    rootAbs={st?.rootPath}
                    showMergeActions={Boolean(merge.interactive)}
                    galleryScrollRef={explorerLeftGalleryRef}
                    onRowContextMenu={(e, row) =>
                      setMergeCtxMenu({
                        clientX: e.clientX,
                        clientY: e.clientY,
                        row,
                        paths: "left",
                      })
                    }
                  />
                  <ExplorerSidePanel
                    merge={merge}
                    side="right"
                    selectedRowIndexes={selectedMergeRows}
                    onMergeRowPick={(idx, ev) => onExplorerMergeRowPick("right", idx, ev)}
                    onMergeRowDoubleClick={(row) => onExplorerMergeRowOpen("right", row)}
                    rootAbs={st?.rootPath}
                    showMergeActions={Boolean(merge.interactive)}
                    galleryScrollRef={explorerRightGalleryRef}
                    onRowContextMenu={(e, row) =>
                      setMergeCtxMenu({
                        clientX: e.clientX,
                        clientY: e.clientY,
                        row,
                        paths: "right",
                      })
                    }
                  />
                </div>
              ) : (
                <div className="results-empty">No entries to list for this comparison.</div>
              )
            ) : showFolderExplorer && browseSide ? (
              (merge.rows && merge.rows.length > 0) ? (
                <div className="explorer-wrap">
                  <ExplorerSidePanel
                    merge={merge}
                    side={browseSide}
                    selectedRowIndexes={selectedMergeRows}
                    onMergeRowPick={(idx, ev) => onExplorerMergeRowPick(browseSide, idx, ev)}
                    onMergeRowDoubleClick={(row) => onExplorerMergeRowOpen(browseSide, row)}
                    rootAbs={st?.rootPath}
                    showMergeActions={Boolean(merge.interactive)}
                    onRowContextMenu={(e, row) =>
                      setMergeCtxMenu({
                        clientX: e.clientX,
                        clientY: e.clientY,
                        row,
                        paths: browseSide === "left" ? "left" : "right",
                      })
                    }
                  />
                </div>
              ) : (
                <div className="results-empty">No entries to list for this comparison.</div>
              )
            ) : tabRows.length === 0 ? (
              <div className="results-empty">No files to list for this comparison.</div>
            ) : (
              <>
                <div className="table-wrap">
                  <table className="results-table">
                    <thead>
                      <tr>
                        <th className="col-check">
                          <input
                            ref={resultsTableSelectAllCheckboxRef}
                            type="checkbox"
                            className="table-checkbox"
                            aria-label={tabRows.every((r) => selectedMergeRows.has(r.rowIndex)) ? "Clear row selection for visible files" : "Select all visible file rows"}
                            checked={tabRows.length > 0 && tabRows.every((r) => selectedMergeRows.has(r.rowIndex))}
                            disabled={tabRows.length === 0}
                            onClick={(ev) => ev.stopPropagation()}
                            onChange={() => toggleSelectAllResultsRows()}
                          />
                        </th>
                        <th className="col-no">#</th>
                        <th className="col-preview">Preview</th>
                        <th className="col-name">File Name</th>
                        <th className="col-size">Size</th>
                        <th className="col-exists">Exists</th>
                        <th className="col-action-side" title="Actions affecting the left folder">Left</th>
                        <th className="col-action-side" title="Actions affecting the right folder">Right</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tabRows.map((r: MergeFileRow) => {
                        const sel = selectedMergeRows.has(r.rowIndex);
                        const L = (r.leftName || "").trim();
                        const R = (r.rightName || "").trim();
                        const primaryForExt = L || R || "—";
                        const ext = primaryForExt.split(".").pop()?.toLowerCase() || "";
                        const isImage = ["jpg","jpeg","png","gif","webp","bmp","svg","heic","heif"].includes(ext);
                        const sizeCell = unifiedSizeDisplay(r);
                        const rowsAll = merge!.rows!;
                        return (
                          <tr
                            key={r.rowIndex}
                            className={`result-row${sel ? " row-sel" : ""}`}
                            onClick={(ev) => onResultsTableMergeRowPick(r.rowIndex, { shiftKey: ev.shiftKey })}
                            onContextMenu={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              if (rowsAll.length) {
                                mergeRowPickWithOrdering(mergeFileRowOrdering(rowsAll), r.rowIndex, false);
                              }
                              setMergeCtxMenu({ clientX: e.clientX, clientY: e.clientY, row: r, paths: "all" });
                            }}
                          >
                            <td className="col-check">
                              <input
                                type="checkbox"
                                className="table-checkbox"
                                checked={sel}
                                onChange={(ev) => {
                                  const nk = ev.nativeEvent as { shiftKey?: boolean };
                                  onResultsTableMergeRowPick(r.rowIndex, { shiftKey: Boolean(nk.shiftKey) });
                                }}
                                onClick={(e) => e.stopPropagation()}
                              />
                            </td>
                            <td className="col-no">{r.no || ""}</td>
                            <td className="col-preview">
                              <div className="preview-thumb">
                                {isImage ? (
                                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none"><rect width="24" height="24" rx="4" fill="rgba(124,140,255,0.15)" /><path d="M4 17l4-4 3 3 4-5 5 6H4z" fill="rgba(124,140,255,0.6)" /></svg>
                                ) : (
                                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none"><rect width="24" height="24" rx="4" fill="rgba(124,140,255,0.1)" /><path d="M7 8h10M7 12h8M7 16h5" stroke="rgba(124,140,255,0.6)" strokeWidth="1.5" strokeLinecap="round" /></svg>
                                )}
                              </div>
                            </td>
                            <td className="col-name">
                              <FileNameCell row={r} />
                            </td>
                            <td
                              className={`col-size col-size-unified size-${sizeCell.variant}`}
                              title={sizeCell.title || undefined}
                            >
                              {sizeCell.text}
                            </td>
                            <td className="col-exists"><ExistsSquares row={r} /></td>
                            <td className="col-action-side"><ActionCellSide row={r} side="left" /></td>
                            <td className="col-action-side"><ActionCellSide row={r} side="right" /></td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </section>
        </main>
      </div>

      {mergeCtxMenu ? (() => {
        const { left, top } = clampResultsCtxPosition(mergeCtxMenu.clientX, mergeCtxMenu.clientY);
        const row = mergeCtxMenu.row;
        const pathScope = mergeCtxMenu.paths;
        const fmLabel = fileManagerRevealLabel();
        const noun = row.isFolder ? "folder" : "file";
        const pairs: Array<{ tag: string; rel: string }> = [];
        const lp = (row.leftPath || "").trim();
        const rp = (row.rightPath || "").trim();
        if ((pathScope === "all" || pathScope === "left") && lp.length > 0) pairs.push({ tag: "Left", rel: lp });
        if ((pathScope === "all" || pathScope === "right") && rp.length > 0) pairs.push({ tag: "Right", rel: rp });
        const runOpen = (rel: string) => {
          void Similarity.OpenScannedPath(rel)
            .catch((e: unknown) => { setErr(String((e as { message?: string })?.message || e)); })
            .finally(() => closeMergeCtxMenu());
        };
        const runReveal = (rel: string) => {
          void Similarity.RevealScannedPath(rel)
            .catch((e: unknown) => { setErr(String((e as { message?: string })?.message || e)); })
            .finally(() => closeMergeCtxMenu());
        };
        return (
          <>
            <div className="results-ctx-scrim" aria-hidden="true" onMouseDown={() => closeMergeCtxMenu()} />
            <div
              role="menu"
              className="results-ctx-menu"
              style={{ position: "fixed", left, top }}
              onMouseDown={(e) => e.stopPropagation()}
            >
              {!hasRevealRoot ? (
                <div className="results-ctx-hint">Needs a scanned folder with an on-disk root (JSON-only loads can’t reveal paths).</div>
              ) : pairs.length === 0 ? (
                <div className="results-ctx-hint">No path available for this row.</div>
              ) : (
                pairs.map(({ tag, rel }, i) => (
                  <Fragment key={tag}>
                    {i > 0 ? <div className="results-ctx-sep" role="separator" /> : null}
                    <button
                      type="button"
                      role="menuitem"
                      className="results-ctx-item"
                      title={`Opens the ${tag.toLowerCase()} ${noun} with its default application.`}
                      onClick={() => runOpen(rel)}
                    >
                      Open {tag.toLowerCase()} {noun}
                    </button>
                    <button
                      type="button"
                      role="menuitem"
                      className="results-ctx-item"
                      title={row.isFolder ? `Opens this ${tag.toLowerCase()} folder in ${fmLabel}.` : `${fmLabel}: opens the enclosing folder with this ${noun} selected (where supported).`}
                      onClick={() => runReveal(rel)}
                    >
                      {fmLabel}
                      {": "}
                      {tag}
                    </button>
                  </Fragment>
                ))
              )}
              <button type="button" className="results-ctx-item results-ctx-cancel" role="menuitem" onClick={closeMergeCtxMenu}>
                Cancel
              </button>
            </div>
          </>
        );
      })() : null}

      {/* Log Dock */}
      <div className={"log-dock" + (logOpen ? " log-dock-open" : "")}>
        <button
          type="button"
          className="log-dock-toggle"
          onClick={() => setLogOpen(v => !v)}
          aria-expanded={logOpen}
        >
          <span className="log-dock-chevron">{logOpen ? "▼" : "▲"}</span>
          <span className="log-dock-label">Console</span>
          {logs.length > 0 ? <span className="log-dock-badge">{logs.length}</span> : null}
        </button>
        <div className="log-dock-panel" role="region" aria-label="Application log">
          <pre ref={logPreRef} className="log-pre">
            {logs.length ? logs.join("\n") : "No log lines yet."}
          </pre>
        </div>
      </div>
    </div>
  );
}

export default App;
