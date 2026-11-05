/** Three desks side by side; the cursor glides from the laptop to the other screens. */
export function HeroIllustration() {
  const path = "M 380 170 C 470 140, 560 140, 640 165 C 560 200, 470 200, 380 170 C 300 140, 200 140, 130 168 C 200 200, 300 200, 380 170"
  return (
    <svg viewBox="0 0 760 320" className="h-auto w-full" role="img" aria-label="Con trỏ chuột di chuyển giữa máy Linux, MacBook và máy Windows">
      <defs>
        <linearGradient id="scr" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="var(--primary)" stopOpacity="0.22" />
          <stop offset="1" stopColor="var(--primary)" stopOpacity="0.05" />
        </linearGradient>
      </defs>

      {/* Linux monitor */}
      <g>
        <rect x="20" y="70" width="220" height="140" rx="10" className="fill-card stroke-border" strokeWidth="2" />
        <rect x="32" y="82" width="196" height="116" rx="4" fill="url(#scr)" />
        <rect x="120" y="210" width="20" height="34" className="fill-muted" />
        <rect x="90" y="244" width="80" height="8" rx="4" className="fill-muted" />
        <text x="130" y="146" textAnchor="middle" className="fill-muted-foreground" fontSize="15" fontWeight="600">Linux</text>
      </g>

      {/* MacBook */}
      <g>
        <rect x="285" y="80" width="190" height="124" rx="10" className="fill-card stroke-border" strokeWidth="2" />
        <rect x="296" y="91" width="168" height="102" rx="4" fill="url(#scr)" />
        <path d="M260 206 h240 l-14 16 h-212 z" className="fill-muted stroke-border" strokeWidth="1.5" />
        <rect x="352" y="206" width="56" height="5" rx="2.5" className="fill-border" />
        <text x="380" y="146" textAnchor="middle" className="fill-muted-foreground" fontSize="15" fontWeight="600">MacBook</text>
        {/* keyboard + trackpad glyph */}
        <g transform="translate(330 252)" className="fill-primary" opacity="0.9">
          <rect width="100" height="34" rx="6" className="fill-card stroke-border" strokeWidth="1.5" />
          {Array.from({ length: 3 }).map((_, r) =>
            Array.from({ length: 8 }).map((_, c) => (
              <rect key={`${r}-${c}`} x={8 + c * 11} y={6 + r * 8} width="8" height="5" rx="1.2" className="fill-muted-foreground" opacity="0.35" />
            ))
          )}
        </g>
      </g>

      {/* Windows monitor */}
      <g>
        <rect x="520" y="62" width="230" height="150" rx="10" className="fill-card stroke-border" strokeWidth="2" />
        <rect x="532" y="74" width="206" height="126" rx="4" fill="url(#scr)" />
        <rect x="625" y="212" width="20" height="32" className="fill-muted" />
        <rect x="595" y="244" width="80" height="8" rx="4" className="fill-muted" />
        <text x="635" y="146" textAnchor="middle" className="fill-muted-foreground" fontSize="15" fontWeight="600">Windows</text>
      </g>

      {/* travel trail */}
      <path d={path} fill="none" className="stroke-primary" strokeWidth="1.5" strokeDasharray="4 6" opacity="0.35" />

      {/* cursor */}
      <g>
        <animateMotion dur="9s" repeatCount="indefinite" path={path} rotate="0"
          keyPoints="0;0;0.25;0.25;0.5;0.5;0.75;0.75;1" keyTimes="0;0.06;0.22;0.3;0.46;0.54;0.7;0.78;1" calcMode="linear" />
        <path d="M0 0 L0 22 L6 16 L10 26 L14 24 L10 14 L18 14 Z" className="fill-foreground stroke-background" strokeWidth="1.5" strokeLinejoin="round" />
      </g>
    </svg>
  )
}
