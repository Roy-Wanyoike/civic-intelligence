'use client';

import { Printer } from 'lucide-react';

export function PrintButton() {
  return (
    <button
      type="button"
      onClick={() => window.print()}
      className="inline-flex items-center gap-1 rounded-md border border-civic-border px-3 py-1.5 text-sm hover:bg-civic-mist"
    >
      <Printer className="h-4 w-4" aria-hidden="true" />
      Print
    </button>
  );
}
