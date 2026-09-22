import { ImageResponse } from 'next/og';

// twitter-image.tsx — emitted at /twitter-image by Next.js. Twitter's card
// image (1200×630). Same as the Open Graph image — kept as a separate file
// so the design can diverge in the future if Twitter/X adds format-specific
// constraints.
//
// Satori (the renderer used by next/og) requires every <div> with multiple
// children to have explicit `display: flex`. Single-text divs use
// `display: flex` with `flexDirection: column` so multi-line text wraps
// correctly.
export const alt = 'Civic Intelligence — Understand what your government is doing';
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default function TwitterImage() {
  return new ImageResponse(
    (
      <div
        style={{
          background: 'linear-gradient(135deg, #0c3b2e 0%, #1a5e44 100%)',
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-start',
          justifyContent: 'center',
          padding: '80px',
          color: '#f5f7f6',
          fontFamily: 'serif',
        }}
      >
        <div
          style={{
            display: 'flex',
            fontSize: 28,
            opacity: 0.85,
            marginBottom: 16,
            letterSpacing: 2,
            textTransform: 'uppercase',
          }}
        >
          Civic Intelligence
        </div>
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            fontSize: 72,
            fontWeight: 700,
            lineHeight: 1.1,
            marginBottom: 24,
          }}
        >
          <div>Understand what your</div>
          <div>government is doing.</div>
        </div>
        <div
          style={{
            display: 'flex',
            fontSize: 28,
            opacity: 0.75,
            maxWidth: 800,
          }}
        >
          Evidence-grounded explanations of legislation, Hansard, and civic activity across 14 African countries.
        </div>
        <div
          style={{
            display: 'flex',
            marginTop: 48,
            gap: 32,
            fontSize: 22,
            opacity: 0.6,
          }}
        >
          <span>🇰🇪 🇺🇬 🇹🇿 🇬🇭 🇳🇬 🇿🇦</span>
          <span>🇷🇼 🇿🇲 🇸🇳 🇪🇬 🇲🇦 🇨🇩 🇪🇹 🇲🇼</span>
        </div>
      </div>
    ),
    {
      ...size,
    }
  );
}
