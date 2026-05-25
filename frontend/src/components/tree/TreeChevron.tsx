export function TreeChevron({ expanded }: { expanded: boolean }) {
  return (
    <svg className="tree-chevron-icon" width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
      {expanded ? (
        <path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      ) : (
        <path d="M6 4l4 4-4 4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      )}
    </svg>
  );
}
