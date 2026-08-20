/**
 * PPAP wordmark used as the sidebar brand mark.
 *
 * The SVG stays legible at the collapsed sidebar size and avoids shipping an
 * unrelated upstream bitmap. The exported name is retained for compatibility
 * with the existing layout import.
 */
const svgMarkup = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" role="img" aria-label="Playful Proxy API Panel">
  <defs>
    <linearGradient id="ppapBrandFill" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#5B8DEF"/>
      <stop offset="1" stop-color="#8E6DFF"/>
    </linearGradient>
  </defs>
  <rect x="6" y="6" width="116" height="116" rx="26" ry="26" fill="url(#ppapBrandFill)"/>
  <g fill="#ffffff" font-family="'Inter','Segoe UI',system-ui,-apple-system,sans-serif" font-weight="800" text-anchor="middle" letter-spacing="-2">
    <text x="64" y="78" font-size="42">PPAP</text>
  </g>
</svg>`;

export const INLINE_LOGO_JPEG =
  'data:image/svg+xml;utf8,' + encodeURIComponent(svgMarkup);
