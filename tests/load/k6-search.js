/**
 * k6 load test — /api/v1/search
 *
 * Search is more expensive than the bills list (FTS + ranking). The SLO is
 * the same as for bills, but we run at a lower RPS (50/s) to reflect that
 * search should not be hammered at 100 RPS in production — search results
 * are cached aggressively client-side.
 *
 * SLOs:
 *   p50  <  200 ms
 *   p95  <  500 ms
 *   p99  < 1000 ms
 *   error rate < 1%
 *
 * Run:
 *   k6 run tests/load/k6-search.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const DURATION = __ENV.K6_DURATION || '5m';
const RATE = parseInt(__ENV.K6_RATE || '50', 10);

const searchLatency = new Trend('search_latency', true);
const searchErrors = new Rate('search_errors');

// A pool of representative queries. Mixing short / long / rare queries
// exercises both the FTS index and the fallback trigram path.
const QUERIES = [
  'housing',
  'affordable housing',
  'counties revenue allocation',
  'health insurance',
  'education bursary',
  'public debt',
  'constitution amendment',
  'slum upgrading',
  'coffee farmer prices',
  'gazette notice',
];

export const options = {
  scenarios: {
    sustained: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 100,
      maxVUs: 300,
    },
  },
  thresholds: {
    http_req_duration: ['p(50)<200', 'p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    search_errors: ['rate<0.01'],
  },
};

const params = {
  headers: {
    'Accept': 'application/json',
    'User-Agent': 'k6-load/1.0',
  },
  timeout: '5s',
};

export default function () {
  const q = QUERIES[__ITER % QUERIES.length];
  const url = `${BASE}/api/v1/search?q=${encodeURIComponent(q)}&limit=10`;

  const res = http.get(url, params);
  searchLatency.add(res.timings.duration);

  const ok = check(res, {
    'status is 200': (r) => r.status === 200,
    'has results array': (r) => {
      try {
        const b = r.json();
        return Array.isArray(b?.results) || Array.isArray(b?.bills);
      } catch {
        return false;
      }
    },
  });
  searchErrors.add(!ok);

  sleep(Math.random() * 0.05);
}
