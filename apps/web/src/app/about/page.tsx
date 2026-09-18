import Link from 'next/link';
import { Scale, ShieldCheck, BookOpen, Sparkles, FileText, Globe } from 'lucide-react';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'About',
  description: 'Civic Intelligence is an evidence-grounded civic-information platform for Kenya. We translate government information into plain language.',
};

export default function AboutPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-4xl font-semibold text-civic-forest">About Civic Intelligence</h1>
        <p className="mt-3 text-lg text-civic-stone">
          Government information is public. But public information is often
          difficult for ordinary citizens to understand.
        </p>
        <p className="mt-3 text-civic-stone">
          We build the infrastructure that transforms authoritative government
          information into understandable explanations, searchable knowledge,
          timelines, evidence-backed answers, and citizen alerts.
        </p>
      </header>

      <section className="mt-10">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Our principle</h2>
        <blockquote className="mt-4 border-l-4 border-civic-acacia pl-4 italic text-civic-ink">
          Show people what happened. Explain what it means. Show them the
          evidence. Let them decide what they think.
        </blockquote>
      </section>

      <section className="mt-10">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Non-negotiable principles</h2>
        <ul className="mt-4 space-y-3">
          {[
            { icon: ShieldCheck, title: 'Evidence before AI', text: 'Official sources are the foundation. AI explains evidence. AI does not replace evidence.' },
            { icon: FileText, title: 'Never fabricate', text: 'Never invent legislation, votes, dates, MPs, senators, committees, quotes, or stages. If evidence is unavailable, we say so.' },
            { icon: BookOpen, title: 'Version everything', text: 'Never overwrite historical legislative information. Every Bill version is preserved immutably.' },
            { icon: Sparkles, title: 'Separate fact from interpretation', text: 'The UI distinguishes FACT, EXPLANATION, INFERENCE, and UNKNOWN.' },
            { icon: Scale, title: 'Political neutrality', text: 'Never rank politicians as good or bad. Never encourage voting for or against a political party.' },
            { icon: Globe, title: 'Country independence', text: 'Kenya is a country adapter — never hard-coded into the global domain model.' },
          ].map(({ icon: Icon, title, text }) => (
            <li key={title} className="flex gap-3">
              <Icon className="h-5 w-5 flex-shrink-0 text-civic-leaf" aria-hidden="true" />
              <div>
                <h3 className="text-base font-semibold text-civic-ink">{title}</h3>
                <p className="mt-1 text-sm text-civic-stone">{text}</p>
              </div>
            </li>
          ))}
        </ul>
      </section>

      <section className="mt-10">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">What we are not</h2>
        <ul className="mt-4 list-disc space-y-2 pl-5 text-sm text-civic-stone">
          <li>We are not affiliated with the Government of Kenya.</li>
          <li>We do not provide legal advice.</li>
          <li>We do not tell you what political position to take.</li>
          <li>We do not rank politicians, parties, or ideologies.</li>
          <li>We do not invent information when sources are silent.</li>
        </ul>
      </section>

      <section className="mt-10">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Architecture</h2>
        <p className="mt-3 text-sm text-civic-stone">
          The platform is built on a modular-monolith-plus-workers
          architecture: a Go backend with strict domain boundaries, a Python
          AI gateway with evidence-grounded RAG, and a Next.js frontend. The
          first country is Kenya. The architecture is country-agnostic —
          adding a country means writing a new adapter, not rebuilding the
          platform.
        </p>
        <p className="mt-3 text-sm text-civic-stone">
          The single most important architectural rule:
        </p>
        <blockquote className="mt-3 border-l-4 border-civic-leaf pl-4 text-sm italic text-civic-ink">
          AI may propose candidate facts. It may never directly write to
          canonical legislative state. Every AI response passes citation
          validation; responses with unsupported claims fail validation.
        </blockquote>
      </section>

      <section className="mt-10">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Open source</h2>
        <p className="mt-3 text-sm text-civic-stone">
          Civic Intelligence is open source. The full codebase, architecture,
          and decision records are on{' '}
          <a
            href="https://github.com/Roy-Wanyoike/civic-intelligence"
            className="text-civic-leaf hover:underline"
            target="_blank"
            rel="noopener noreferrer"
          >
            GitHub ↗
          </a>
          . Contributions are welcome — see the contributing guide.
        </p>
        <div className="mt-6">
          <Link
            href="/"
            className="inline-flex items-center gap-2 rounded-md bg-civic-leaf px-5 py-2.5 font-semibold text-white hover:bg-civic-leaf/90"
          >
            Explore the platform
          </Link>
        </div>
      </section>
    </div>
  );
}
