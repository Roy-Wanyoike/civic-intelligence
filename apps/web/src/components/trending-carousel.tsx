'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { ChevronLeft, ChevronRight, TrendingUp, FileSignature } from 'lucide-react';

interface CarouselBill {
  id: string;
  title: string;
  house: string;
  year: number;
  source_url: string;
  publication_date?: string;
  current_stage?: string;
}

interface TrendingCarouselProps {
  bills: CarouselBill[];
  autoPlay?: boolean;
  interval?: number;
}

export function TrendingCarousel({ bills, autoPlay = true, interval = 5000 }: TrendingCarouselProps) {
  const [current, setCurrent] = useState(0);
  const [isPaused, setIsPaused] = useState(false);

  const next = useCallback(() => {
    setCurrent((prev) => (prev + 1) % Math.max(bills.length, 1));
  }, [bills.length]);

  const prev = useCallback(() => {
    setCurrent((prev) => (prev - 1 + bills.length) % Math.max(bills.length, 1));
  }, [bills.length]);

  useEffect(() => {
    if (!autoPlay || isPaused || bills.length <= 1) return;
    const timer = setInterval(next, interval);
    return () => clearInterval(timer);
  }, [autoPlay, isPaused, next, interval, bills.length]);

  if (!bills || bills.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        No trending bills available this month.
      </div>
    );
  }

  const bill = bills[current];

  return (
    <div
      className="relative overflow-hidden rounded-xl border border-civic-border bg-civic-paper"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
      role="region"
      aria-label="Trending Bills this month"
    >
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-civic-border bg-civic-mist px-4 py-3">
        <TrendingUp className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
        <h2 className="font-serif text-sm font-semibold text-civic-ink">
          Trending Bills This Month
        </h2>
        <span className="ml-auto text-xs text-civic-stone">
          {current + 1} / {bills.length}
        </span>
      </div>

      {/* Slide */}
      <div className="p-6">
        <Link href={`/bills/${bill.id}`} className="block">
          <div className="flex items-center gap-2 text-xs text-civic-stone">
            <FileSignature className="h-4 w-4 text-civic-clay" aria-hidden="true" />
            <span>{bill.house} · {bill.year}</span>
            {bill.current_stage && (
              <span className="rounded-full bg-civic-acacia/20 px-2 py-0.5 font-medium text-civic-ink">
                {bill.current_stage}
              </span>
            )}
          </div>
          <h3 className="mt-3 font-serif text-xl font-semibold text-civic-ink hover:text-civic-leaf">
            {bill.title}
          </h3>
          {bill.publication_date && (
            <p className="mt-2 text-xs text-civic-stone">
              Published: {bill.publication_date}
            </p>
          )}
          <p className="mt-2 text-xs text-civic-leaf truncate">
            {bill.source_url}
          </p>
        </Link>
      </div>

      {/* Navigation arrows */}
      {bills.length > 1 && (
        <>
          <button
            type="button"
            onClick={prev}
            className="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-civic-paper/80 p-2 text-civic-stone shadow-md hover:bg-civic-paper hover:text-civic-leaf"
            aria-label="Previous bill"
          >
            <ChevronLeft className="h-5 w-5" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={next}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-civic-paper/80 p-2 text-civic-stone shadow-md hover:bg-civic-paper hover:text-civic-leaf"
            aria-label="Next bill"
          >
            <ChevronRight className="h-5 w-5" aria-hidden="true" />
          </button>
        </>
      )}

      {/* Dots */}
      {bills.length > 1 && (
        <div className="flex justify-center gap-1.5 pb-3">
          {bills.map((_, i) => (
            <button
              key={i}
              type="button"
              onClick={() => setCurrent(i)}
              className={`h-2 rounded-full transition-all ${
                i === current ? 'w-6 bg-civic-leaf' : 'w-2 bg-civic-border'
              }`}
              aria-label={`Go to slide ${i + 1}`}
            />
          ))}
        </div>
      )}
    </div>
  );
}
