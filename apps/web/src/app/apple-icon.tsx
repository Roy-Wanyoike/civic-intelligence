import { ImageResponse } from 'next/og';

// apple-icon.tsx — emitted at /apple-icon by Next.js. Renders the Apple Touch
// Icon (180×180) — used by iOS Safari when a user adds the site to their
// home screen. Same forest-green C-mark as the favicon.
export const size = {
  width: 180,
  height: 180,
};
export const contentType = 'image/png';

export default function AppleIcon() {
  return new ImageResponse(
    (
      <div
        style={{
          background: '#0c3b2e',
          width: '100%',
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#f5f7f6',
          fontSize: 120,
          fontWeight: 700,
          fontFamily: 'serif',
        }}
      >
        C
      </div>
    ),
    {
      ...size,
    }
  );
}
