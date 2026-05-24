import type { GalleryCategory } from "../../types";

export function GalleryKindVisual({ category, ext }: { category: GalleryCategory; ext: string }) {
  const label = ext ? ext.toUpperCase() : "";
  const cls = `gallery-kind-label gallery-kind-${category}`;
  switch (category) {
    case "pdf":
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <rect x="8" y="4" width="28" height="36" rx="3" fill="rgba(239,68,68,0.2)" stroke="#f87171" strokeWidth="1.6" />
            <path d="M14 14h16M14 20h16M14 26h10" stroke="#fca5a5" strokeWidth="1.6" strokeLinecap="round" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
    case "archive":
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <rect x="12" y="10" width="20" height="26" rx="2" fill="rgba(168,85,247,0.18)" stroke="#c084fc" strokeWidth="1.6" />
            <path d="M16 16h12v6H16z" fill="rgba(168,85,247,0.35)" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
    case "video":
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <rect x="8" y="11" width="28" height="22" rx="3" fill="rgba(59,130,246,0.18)" stroke="#60a5fa" strokeWidth="1.6" />
            <path d="M19 17v10l8-5-8-5z" fill="#93c5fd" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
    case "audio":
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <path d="M14 28V14l14-4v22" stroke="#34d399" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M14 28a4 4 0 1 0 0 .01zM28 24a4 4 0 1 0 0 .01z" fill="#34d399" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
    case "sheet":
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <rect x="10" y="8" width="24" height="28" rx="2" fill="rgba(34,197,94,0.14)" stroke="#4ade80" strokeWidth="1.6" />
            <path d="M14 16h16M14 22h16M14 28h10" stroke="#86efac" strokeWidth="1.4" strokeLinecap="round" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
    case "code":
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <path d="M14 16l-6 6 6 6M30 16l6 6-6 6M24 12l-4 20" stroke="#a5b4fc" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
    default:
      return (
        <span className="gallery-thumb-fallback">
          <svg width="44" height="44" viewBox="0 0 44 44" fill="none" aria-hidden>
            <path d="M12 8h14l6 6v20a3 3 0 0 1-3 3H12a3 3 0 0 1-3-3V11a3 3 0 0 1 3-3z" fill="rgba(148,163,255,0.12)" stroke="#94a3f8" strokeWidth="1.6" />
            <path d="M26 8v8h6" stroke="#94a3f8" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          {label ? <span className={cls}>{label}</span> : null}
        </span>
      );
  }
}
