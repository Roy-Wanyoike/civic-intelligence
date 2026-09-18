/**
 * k6 load test — /api/v1/bills
 *
 * SLOs:
 *   p50  <  200 ms
 *   p95  <  500 ms
 *   p99  < 1000 ms
 *   error rate < 1%
 *
 * Run:
 *   k6 run tests/load/k6-bills.js
 *
 * With env overrides:
 *   BASE_URL=http://staging.civic.internal k6 run tests/load/k6-bills.js
 *   K6_DURATION=10m K6_RATE=200 k6 run tests/load/k6-bills.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const DURATION = __ENV.K6_DURATION || '5m';
const RATE = parseInt(__ENV.K6_RATE || '100', 10); // requests per second
const PRE_AUTH_TOKEN = __ENV.K6_AUTH_TOKEN || '';   // optional

const billsLatency = new Trend('bills_latency', true);
const billsErrors = new Rate('bills_errors');

export const options = {
  scenarios: {
    sustained_100rps: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 200,
      maxVUs: 500,
    },
  },
  thresholds: {
    // SLOs — the build fails if any are breached.
    http_req_duration: ['p(50)<200', 'p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    bills_errors: ['rate<0.01'],
  },
};

const params = {
  headers: {
    'Accept': 'application/json',
    'User-Agent': 'k6-load/1.0',
    ...(PRE_AUTH_TOKEN ? { Authorization: `Bearer ${PRE_AUTH_TOKEN}` } : {}),
  },
  timeout: '5s',
};

export default function () {
  // Vary the page so we exercise the cache + DB evenly.
  const page = (__ITER % 10) + 1;
  const url = `${BASE}/api/v1/bills?page=${page}&limit=20`;

  const res = http.get(url, params);

  billsLatency.add(res.timings.duration);

  const ok = check(res, {
    'status is 200': (r) => r.status === 200,
    'has bills array': (r) => {
      try {
        const body = r.json();
        return Array.isArray(body?.bills) || Array.isArray(body?.data);
      } catch {
        return false;
      }
    },
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  billsErrors.add(!ok);

  // Small jitter so the arrival pattern isn't perfectly uniform.
  sleep(Math.random() * 0.05);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data),
    'tests/load/results/bills-summary.json': JSON.stringify(data, null, 2),
  };
}

// Tiny built-in reporter so we don't depend on an external summary module.
function textSummary(data) {
  const m = data.metrics || {};
  const dur = m.http_req_duration?.values || {};
  const failed = m.http_req_failed?.values?.rate ?? 0;
  const errs = m.bills_errors?.values?.rate ?? 0;
  return `
=== /api/v1/bills load test ===
duration:       ${data.state?.testRunDurationMs ?? '?'} ms
iterations:    ${m.iterations?.values?.count ?? 0}

http_req_duration
  p50:  ${ms(dur['p(50)'])}
  p95:  ${ms(dur['p(95)'])}
  p99:  ${ms(dur['p(99)'])}
  min:  ${ms(dur.min)}
  max:  ${ms(dur.max)}

http_req_failed rate: ${(failed * 100).toFixed(2)}%
bills_errors rate:     ${(errs * 100).toFixed(2)}%

SLO check:
  p50<200ms:  ${(dur['p(50)'] ?? 0) < 200 ? 'PASS' : 'FAIL'}
  p95<500ms:  ${(dur['p(95)'] ?? 0) < 500 ? 'PASS' : 'FAIL'}
  p99<1000ms: ${(dur['p(99)'] ?? 0) < 1000 ? 'PASS' : 'FAIL'}
  err<1%:     ${failed < 0.01 ? 'PASS' : 'FAIL'}
`;
}

function ms(v) {
  if (v === undefined || v === null) return 'n/a';
  return `${v.toFixed(1)} ms`;
}
