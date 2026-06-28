import { cn } from "@/lib/utils";

/**
 * PPAP brand mark — a stacked-node glyph evoking a routing proxy: three request
 * lanes converging into a single upstream. Uses `currentColor` + the brand
 * gradient so it adapts to light/dark surfaces.
 */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      role="img"
      aria-hidden="true"
      className={cn("size-8", className)}
    >
      <defs>
        <linearGradient id="ppap-mark" x1="4" y1="4" x2="28" y2="28" gradientUnits="userSpaceOnUse">
          <stop stopColor="var(--color-brand-400)" />
          <stop offset="1" stopColor="var(--color-brand-600)" />
        </linearGradient>
      </defs>
      <rect x="2" y="2" width="28" height="28" rx="8" fill="url(#ppap-mark)" />
      <path
        d="M9 10.5h5.5c3 0 4.8 1.8 4.8 4.4 0 2.6-1.8 4.4-4.8 4.4H11.6V23"
        stroke="white"
        strokeWidth="2.1"
        strokeLinecap="round"
        strokeLinejoin="round"
        opacity="0.95"
      />
      <circle cx="22.5" cy="11" r="2.1" fill="var(--color-accent-300)" />
    </svg>
  );
}

/** Full logo lockup: mark + wordmark. Used in the topbar and login screen. */
export function LogoLockup({
  className,
  showTagline = false,
}: {
  className?: string;
  showTagline?: boolean;
}) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <LogoMark />
      <div className="flex flex-col leading-tight">
        <span className="text-sm font-semibold tracking-tight text-foreground">
          Playful Proxy
        </span>
        {showTagline ? (
          <span className="text-[11px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
            API Panel
          </span>
        ) : null}
      </div>
    </div>
  );
}
