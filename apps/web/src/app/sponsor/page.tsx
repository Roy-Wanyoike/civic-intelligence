import type { Metadata } from 'next';
import { Heart, Smartphone, CreditCard } from 'lucide-react';
import { MpesaSponsorForm, CardSponsorForm } from '@/components/sponsor-form';

export const metadata: Metadata = {
  title: 'Sponsor Civic Intelligence',
  description: 'Support the Civic Intelligence Platform — transparent, evidence-grounded civic information for Kenya.',
};

export default function SponsorPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header className="text-center">
        <Heart className="mx-auto h-12 w-12 text-civic-clay" aria-hidden="true" />
        <h1 className="mt-4 font-serif text-3xl font-semibold text-civic-forest">
          Sponsor Civic Intelligence
        </h1>
        <p className="mt-3 text-sm text-civic-stone">
          Help us keep civic information free, transparent, and evidence-grounded for every Kenyan.
          Your support covers server costs, AI processing, and data sourcing.
        </p>
      </header>

      <div className="mt-8 grid gap-6 sm:grid-cols-2">
        {/* M-Pesa */}
        <div className="rounded-lg border border-civic-leaf/30 bg-civic-leaf/5 p-6">
          <div className="flex items-center gap-2">
            <Smartphone className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 className="font-serif text-lg font-semibold text-civic-ink">M-Pesa</h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">
            Pay via Safaricom M-Pesa STK Push. You'll receive a prompt on your phone to confirm.
          </p>
          <MpesaSponsorForm />
        </div>

        {/* Card */}
        <div className="rounded-lg border border-civic-acacia/30 bg-civic-acacia/5 p-6">
          <div className="flex items-center gap-2">
            <CreditCard className="h-5 w-5 text-civic-acacia" aria-hidden="true" />
            <h2 className="font-serif text-lg font-semibold text-civic-ink">Card</h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">
            Pay securely with Visa, Mastercard, or Amex via Stripe.
          </p>
          <CardSponsorForm />
        </div>
      </div>

      <div className="mt-8 rounded-lg border border-civic-border bg-civic-mist p-4 text-xs text-civic-stone">
        <p>
          <strong className="text-civic-ink">Transparency:</strong> All sponsorship funds go toward
          platform infrastructure — server costs, AI processing, and data sourcing from official
          government sources. Civic Intelligence is politically neutral and does not accept
          sponsorship from political parties or politicians.
        </p>
      </div>
    </div>
  );
}
