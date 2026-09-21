import { ImageResponse } from 'next/og';

// icon.tsx — emitted at /icon by Next.js. Renders the platform's favicon as a
// generated SVG→PNG so we don't have to ship a binary .ico/.png file. The
// icon is a forest-green rounded square with the Civic Intelligence "C"
// mark — same visual identity as the header logo.
//
// The size 32×32 is the standard favicon size. Next.js will also generate
// the apple-touch-icon (180×180) from this same source.
export const size = {
  width: 32,
  height: 32,
};
export const contentType = 'image/png';

export default function Icon() {
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
          fontSize: 24,
          fontWeight: 700,
          fontFamily: 'serif',
          borderRadius: 6,
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
