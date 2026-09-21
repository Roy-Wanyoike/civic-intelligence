import { ImageResponse } from 'next/og';

// opengraph-image.tsx — emitted at /opengraph-image by Next.js. This is the
// 1200×630 image that shows up when someone shares a link to the site on
// Twitter, Facebook, LinkedIn, WhatsApp, etc. The image is generated at
// build time so it's always in sync with the platform's branding.
//
// The design is intentionally minimal: forest-green background, the
// platform name, and the tagline. The font is the system serif so we
// don't need to ship a custom font for the OG image.
export const alt = 'Civic Intelligence — Understand what your government is doing';
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default function OpenGraphImage() {
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
        <div style={{ fontSize: 28, opacity: 0.85, marginBottom: 16, letterSpacing: 2, textTransform: 'uppercase' }}>
          Civic Intelligence
        </div>
        <div style={{ fontSize: 72, fontWeight: 700, lineHeight: 1.1, marginBottom: 24 }}>
          Understand what your
          <br />
          government is doing.
        </div>
        <div style={{ fontSize: 28, opacity: 0.75, maxWidth: 800 }}>
          Evidence-grounded explanations of legislation, Hansard, and civic activity
          across 14 African countries.
        </div>
        <div style={{ display: 'flex', marginTop: 48, gap: 32, fontSize: 22, opacity: 0.6 }}>
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
