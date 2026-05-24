export function CircularProgress({ pct, label }: { pct: number; label: string }) {
  const r = 52;
  const circ = 2 * Math.PI * r;
  const offset = circ - (Math.min(100, Math.max(0, pct)) / 100) * circ;
  return (
    <div className="circ-progress-wrap">
      <svg className="circ-progress-svg" viewBox="0 0 120 120" width="120" height="120">
        <defs>
          <linearGradient id="circGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#7c4dff" />
            <stop offset="100%" stopColor="#5c6bc0" />
          </linearGradient>
        </defs>
        <circle cx="60" cy="60" r={r} className="circ-track" strokeWidth="10" />
        <circle
          cx="60" cy="60" r={r}
          stroke="url(#circGrad)"
          fill="none"
          strokeWidth="10"
          strokeLinecap="round"
          strokeDasharray={circ}
          strokeDashoffset={offset}
          transform="rotate(-90 60 60)"
          style={{ transition: "stroke-dashoffset 0.5s ease" }}
        />
      </svg>
      <div className="circ-progress-label">
        <span className="circ-pct">{pct.toFixed(0)}%</span>
        <span className="circ-sub">{label}</span>
      </div>
    </div>
  );
}
