import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return 'Date unknown';
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return 'Date unknown';
    return d.toLocaleDateString('en-KE', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  } catch {
    return 'Date unknown';
  }
}

export type ConfidenceLevel = 'high' | 'medium' | 'low' | 'unknown';

export function confidenceClass(level: ConfidenceLevel | string | undefined): string {
  switch (level) {
    case 'high': return 'pill-high';
    case 'medium': return 'pill-medium';
    case 'low': return 'pill-low';
    default: return 'pill-unknown';
  }
}
