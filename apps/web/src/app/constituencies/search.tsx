'use client';

import { Search, X } from 'lucide-react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState, useTransition } from 'react';

interface Props {
  initialQuery: string;
}

/**
 * ConstituencySearch is a client-side search box that updates the URL `q`
 * query parameter. Using the URL (instead of local state) keeps the search
 * shareable + bookmarkable: a citizen can copy the URL to send a filtered
 * list to a friend.
 *
 * The input debounces naturally — `useTransition` keeps the input responsive
 * while the server component re-runs in the background. We do NOT submit on
 * every keystroke; instead, the user presses Enter (form submit) or the
 * "Clear" button to reset.
 */
export function ConstituencySearch({ initialQuery }: Props) {
  const router = useRouter();
  const params = useSearchParams();
  const [isPending, startTransition] = useTransition();
  const [value, setValue] = useState(initialQuery);

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const q = value.trim();
    const next = new URLSearchParams(params?.toString() ?? '');
    if (q) {
      next.set('q', q);
    } else {
      next.delete('q');
    }
    startTransition(() => {
      router.push(`/constituencies?${next.toString()}`);
    });
  }

  function handleClear() {
    setValue('');
    const next = new URLSearchParams(params?.toString() ?? '');
    next.delete('q');
    startTransition(() => {
      router.push(`/constituencies?${next.toString()}`);
    });
  }

  return (
    <form
      role="search"
      onSubmit={handleSubmit}
      className="flex w-full items-center gap-2 sm:w-80"
      aria-label="Search constituencies"
    >
      <div className="relative flex-1">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-civic-stone"
          aria-hidden="true"
        />
        <label htmlFor="constituency-q" className="sr-only">
          Search by name, county, region or MP
        </label>
        <input
          id="constituency-q"
          type="search"
          name="q"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="Name, county, region, MP…"
          autoComplete="off"
          className="w-full rounded-md border border-civic-border bg-civic-paper py-2 pl-9 pr-9 text-sm placeholder:text-civic-stone focus:border-civic-leaf focus:outline-none"
          aria-busy={isPending}
        />
        {value && (
          <button
            type="button"
            onClick={handleClear}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full p-1 text-civic-stone hover:bg-civic-mist"
            aria-label="Clear search"
          >
            <X className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        )}
      </div>
    </form>
  );
}
