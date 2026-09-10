'use client';

import { useState } from 'react';
import { Loader2, CheckCircle2, AlertTriangle } from 'lucide-react';

type Status = 'idle' | 'loading' | 'success' | 'error';

export function MpesaSponsorForm() {
  const [status, setStatus] = useState<Status>('idle');
  const [message, setMessage] = useState('');

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setStatus('loading');
    const formData = new FormData(e.currentTarget);
    const phone = formData.get('phone');
    const amount = formData.get('amount');
    try {
      const resp = await fetch('/api/v1/sponsor/mpesa', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone, amount: Number(amount) }),
      });
      const data = await resp.json();
      if (resp.ok) {
        setStatus('success');
        setMessage('Check your phone for the M-Pesa prompt to confirm the payment.');
      } else {
        setStatus('error');
        setMessage(data.message || 'M-Pesa payment failed. Please try again.');
      }
    } catch {
      setStatus('error');
      setMessage('Network error. Please try again.');
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-3">
      <div>
        <label htmlFor="mpesa-phone-c" className="block text-xs font-medium text-civic-stone">
          Phone Number
        </label>
        <input
          type="tel"
          id="mpesa-phone-c"
          name="phone"
          placeholder="0712345678"
          required
          className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm focus:border-civic-leaf focus:outline-none"
        />
      </div>
      <div>
        <label htmlFor="mpesa-amount-c" className="block text-xs font-medium text-civic-stone">
          Amount (KES)
        </label>
        <input
          type="number"
          id="mpesa-amount-c"
          name="amount"
          placeholder="500"
          min="1"
          required
          className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm focus:border-civic-leaf focus:outline-none"
        />
      </div>
      <button
        type="submit"
        disabled={status === 'loading'}
        className="w-full rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white hover:bg-civic-leaf/90 disabled:opacity-50"
      >
        {status === 'loading' ? (
          <span className="flex items-center justify-center gap-2">
            <Loader2 className="h-4 w-4 animate-spin" /> Sending prompt...
          </span>
        ) : (
          'Pay with M-Pesa'
        )}
      </button>
      {status === 'success' && (
        <div className="flex items-start gap-2 rounded-md bg-civic-leaf/10 p-3 text-sm text-civic-leaf">
          <CheckCircle2 className="mt-0.5 h-4 w-4 flex-shrink-0" />
          <p>{message}</p>
        </div>
      )}
      {status === 'error' && (
        <div className="flex items-start gap-2 rounded-md bg-red-50 p-3 text-sm text-red-800">
          <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" />
          <p>{message}</p>
        </div>
      )}
    </form>
  );
}

export function CardSponsorForm() {
  const [status, setStatus] = useState<Status>('idle');
  const [message, setMessage] = useState('');

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setStatus('loading');
    const formData = new FormData(e.currentTarget);
    const amount = formData.get('amount');
    const email = formData.get('email');
    try {
      const resp = await fetch('/api/v1/sponsor/card', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ amount: Number(amount), email }),
      });
      const data = await resp.json();
      if (resp.ok && data.checkout_url) {
        window.location.href = data.checkout_url;
      } else {
        setStatus('error');
        setMessage(data.message || 'Card payment failed. Please try again.');
      }
    } catch {
      setStatus('error');
      setMessage('Network error. Please try again.');
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-3">
      <div>
        <label htmlFor="card-amount-c" className="block text-xs font-medium text-civic-stone">
          Amount (KES)
        </label>
        <input
          type="number"
          id="card-amount-c"
          name="amount"
          placeholder="500"
          min="1"
          required
          className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm focus:border-civic-acacia focus:outline-none"
        />
      </div>
      <div>
        <label htmlFor="card-email-c" className="block text-xs font-medium text-civic-stone">
          Email (for receipt)
        </label>
        <input
          type="email"
          id="card-email-c"
          name="email"
          placeholder="you@example.com"
          className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm focus:border-civic-acacia focus:outline-none"
        />
      </div>
      <button
        type="submit"
        disabled={status === 'loading'}
        className="w-full rounded-md bg-civic-acacia px-4 py-2 text-sm font-semibold text-civic-ink hover:bg-civic-acacia/90 disabled:opacity-50"
      >
        {status === 'loading' ? (
          <span className="flex items-center justify-center gap-2">
            <Loader2 className="h-4 w-4 animate-spin" /> Redirecting to Stripe...
          </span>
        ) : (
          'Pay with Card'
        )}
      </button>
      {status === 'error' && (
        <div className="flex items-start gap-2 rounded-md bg-red-50 p-3 text-sm text-red-800">
          <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" />
          <p>{message}</p>
        </div>
      )}
    </form>
  );
}
