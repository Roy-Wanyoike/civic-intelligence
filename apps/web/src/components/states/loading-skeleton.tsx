export function LoadingSkeleton() {
  return (
    <div className="animate-pulse space-y-4" aria-label="Loading">
      <div className="h-8 w-3/4 rounded-md bg-civic-mist"></div>
      <div className="h-4 w-1/2 rounded-md bg-civic-mist"></div>
      <div className="space-y-2">
        <div className="h-4 w-full rounded-md bg-civic-mist"></div>
        <div className="h-4 w-5/6 rounded-md bg-civic-mist"></div>
        <div className="h-4 w-4/6 rounded-md bg-civic-mist"></div>
      </div>
    </div>
  );
}

export function CardSkeleton() {
  return (
    <div className="animate-pulse rounded-lg border border-civic-border bg-civic-paper p-5" aria-label="Loading">
      <div className="h-3 w-1/3 rounded bg-civic-mist"></div>
      <div className="mt-3 h-5 w-2/3 rounded bg-civic-mist"></div>
      <div className="mt-2 h-4 w-full rounded bg-civic-mist"></div>
      <div className="mt-2 h-4 w-5/6 rounded bg-civic-mist"></div>
    </div>
  );
}

export function ListSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid gap-4 md:grid-cols-2" aria-label="Loading list">
      {Array.from({ length: count }).map((_, i) => (
        <CardSkeleton key={i} />
      ))}
    </div>
  );
}
