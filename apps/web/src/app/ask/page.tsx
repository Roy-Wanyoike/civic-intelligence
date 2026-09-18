import type { Metadata } from 'next';
import { AskForm } from './ask-form';

export const metadata: Metadata = {
  title: 'Ask Kenya',
  description: 'Ask any question about Kenyan civic activity. Every answer is evidence-grounded and labelled AI-GENERATED.',
};

export const dynamic = 'force-dynamic';

const EXAMPLE_QUESTIONS = [
  'What is happening with the Housing Bill?',
  'What did the President sign today?',
  'What laws affect small businesses?',
  'What changed in data protection?',
  'What is Parliament doing about AI?',
  'What regulations affect tenants?',
  'What government loans were taken in 2024?',
];

export default function AskPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Ask Kenya</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Ask any question about Kenyan legislation, regulations, or government
          activity. Answers are AI-GENERATED, evidence-grounded, and surfaced
          with their citations so you can verify against primary sources.
        </p>
      </header>

      <AskForm examples={EXAMPLE_QUESTIONS} />
    </div>
  );
}
