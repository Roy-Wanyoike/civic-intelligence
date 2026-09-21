import type { Metadata } from 'next';
import { ReportForm } from './report-form';

// report/page.tsx — server component wrapper for the /report page. The
// interactive form (with state, validation, and API submission) lives in
// the ReportForm client component. This server wrapper exists so the page
// can export `metadata` (Next.js 14 forbids exporting metadata from a
// client component).
//
// The /report page is NOT indexed by search engines (see robots: noindex
// below) — it's a submit-only form, not a content destination.
export const metadata: Metadata = {
  title: 'Report an Error',
  description:
    'Spotted an incorrect fact, missing evidence, or broken link on Civic Intelligence? Report it here — every report goes into the corrections queue with an audit trail.',
  alternates: { canonical: '/report' },
  robots: { index: false, follow: true },
};

export default function ReportPage() {
  return <ReportForm />;
}
