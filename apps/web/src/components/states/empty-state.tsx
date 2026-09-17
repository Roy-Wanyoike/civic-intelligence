import Link from 'next/link';

interface EmptyStateProps {
  title: string;
  description?: string;
  actionLabel?: string;
  actionHref?: string;
}

export function EmptyState({ title, description, actionLabel, actionHref }: EmptyStateProps) {
  return (
    <div className="rounded-lg border border-dashed border-civic-border p-12 text-center">
      <h3 className="font-serif text-lg font-semibold text-civic-ink">{title}</h3>
      {description && (
        <p className="mt-2 text-sm text-civic-stone">{description}</p>
      )}
      {actionLabel && actionHref && (
        <Link
          href={actionHref}
          className="mt-4 inline-flex items-center gap-1 rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white hover:bg-civic-leaf/90"
        >
          {actionLabel}
        </Link>
      )}
    </div>
  );
}
