'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { BookOpen, RefreshCw, ExternalLink, Scale } from 'lucide-react';
import { constitutionArticles, type ConstitutionArticle } from '@/data/constitution-articles';

function getRandomArticle(exclude?: string): ConstitutionArticle {
  const available = exclude
    ? constitutionArticles.filter(a => a.number !== exclude)
    : constitutionArticles;
  return available[Math.floor(Math.random() * available.length)];
}

export function ConstitutionSpotlight() {
  const [article, setArticle] = useState<ConstitutionArticle | null>(null);

  useEffect(() => {
    setArticle(getRandomArticle());
  }, []);

  const refresh = () => {
    setArticle(getRandomArticle(article?.number));
  };

  if (!article) {
    return (
      <div className="mt-8 max-w-md animate-pulse rounded-xl border border-white/20 bg-white/5 p-6">
        <div className="h-4 w-24 rounded bg-white/20"></div>
        <div className="mt-3 h-6 w-48 rounded bg-white/20"></div>
        <div className="mt-3 h-16 w-full rounded bg-white/10"></div>
      </div>
    );
  }

  const heading = article.type === 'fact' ? 'Did you know?' : 'Do you know?';

  return (
    <div className="mt-8 max-w-md rounded-xl border border-white/20 bg-white/5 p-6 backdrop-blur-sm">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Scale className="h-4 w-4 text-civic-acacia" aria-hidden="true" />
          <span className="text-xs font-semibold uppercase tracking-wider text-civic-acacia">
            {heading}
          </span>
        </div>
        <button
          type="button"
          onClick={refresh}
          className="rounded-full p-1 text-white/60 transition hover:bg-white/10 hover:text-white"
          aria-label="Show another constitutional clause"
        >
          <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      </div>

      {/* Article number + title */}
      <h3 className="mt-3 font-serif text-lg font-semibold text-white">
        {article.number}: {article.title}
      </h3>
      <p className="mt-1 text-xs text-white/60">{article.chapter}</p>

      {/* Article text */}
      <p className="mt-3 text-sm leading-relaxed text-white/90">
        {article.text}
      </p>

      {/* Actions */}
      <div className="mt-4 flex items-center gap-3 text-xs">
        <Link
          href="/about"
          className="inline-flex items-center gap-1 text-civic-acacia hover:underline"
        >
          <BookOpen className="h-3 w-3" aria-hidden="true" />
          Read the Constitution
        </Link>
        <a
          href={article.source}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-white/60 hover:text-white"
        >
          Source
          <ExternalLink className="h-3 w-3" aria-hidden="true" />
        </a>
      </div>
    </div>
  );
}
