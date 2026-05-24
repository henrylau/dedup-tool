import { useEffect, useRef, useState, type ReactNode } from "react";
import { Similarity } from "../../../bindings/folder-similarity/app/service";
import { GalleryKindVisual } from "./GalleryKindVisual";

export function GalleryImageThumb({ relPath, rootAbs }: { relPath: string; rootAbs: string }) {
  const wrapRef = useRef<HTMLSpanElement>(null);
  const [shouldLoad, setShouldLoad] = useState(false);
  const [broken, setBroken] = useState(false);
  const [src, setSrc] = useState<string | null>(null);

  useEffect(() => {
    const el = wrapRef.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      (entries, observer) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            setShouldLoad(true);
            observer.unobserve(e.target);
          }
        }
      },
      { root: null, rootMargin: "180px 0px 240px 0px", threshold: 0 }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, []);

  useEffect(() => {
    if (!shouldLoad) return;
    let cancelled = false;
    setBroken(false);
    setSrc(null);
    Similarity.ReadGalleryImagePreview(relPath)
      .then((dataUrl) => {
        if (!cancelled) setSrc(dataUrl);
      })
      .catch(() => {
        if (!cancelled) setBroken(true);
      });
    return () => {
      cancelled = true;
    };
  }, [shouldLoad, relPath, rootAbs]);

  let inner: ReactNode;
  if (broken) {
    inner = <GalleryKindVisual category="generic" ext="" />;
  } else if (!src) {
    inner = <span className="gallery-thumb-pending" aria-hidden />;
  } else {
    inner = <img src={src} alt="" className="gallery-thumb-img" onError={() => setBroken(true)} loading="lazy" decoding="async" />;
  }

  return (
    <span ref={wrapRef} className="gallery-thumb-lazy-root">
      {inner}
    </span>
  );
}
