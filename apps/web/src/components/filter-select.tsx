'use client';

import { useRouter, useSearchParams } from 'next/navigation';

interface FilterSelectProps {
  label: string;
  name: string;
  value: string;
  options: readonly string[];
}

export function FilterSelect({ label, name, value, options }: FilterSelectProps) {
  const router = useRouter();
  const params = useSearchParams();

  return (
    <div>
      <label htmlFor={name} className="sr-only">{label}</label>
      <select
        id={name}
        name={name}
        defaultValue={value}
        className="w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm focus:outline-none focus:border-civic-leaf"
        onChange={(e) => {
          const next = new URLSearchParams(params.toString());
          if (e.target.value === 'all') {
            next.delete(name);
          } else {
            next.set(name, e.target.value);
          }
          router.push(`/bills?${next.toString()}`);
        }}
      >
        {options.map((o) => (
          <option key={o} value={o}>
            {label}: {o.replace('_', ' ')}
          </option>
        ))}
      </select>
    </div>
  );
}
