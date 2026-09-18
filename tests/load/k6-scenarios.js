/**
 * k6 load test — /api/v1/scenarios (run scenario)
 *
 * Scenario execution is the heaviest read path: it composes evidence +
 * comparable cases + an AI synthesis. The SLO is therefore more generous
 * at the tail:
 *
 *   p50  <  500 ms   (heavyweight — multi-agent synthesis)
 *   p95  < 1500 ms
 *   p99  < 3000 ms
 *   error rate < 2%
 *
 * Run:
 *   k6 run tests/load/k6-scenarios.js
 *
 * Notes:
 *   - This test RUNS a scenario (POST /api/v1/scenarios/{id}/run), not just
 *     lists them, so it exercises the full agent network + AI gateway.
 *   - In dev/CI the AI gateway returns deterministic stub responses, so the
 *     latency profile reflects orchestration, not real model latency.
 *   - Production SLOs against real models will be looser; re-baseline after
 *     the first prod load test.
 */
import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const DURATION = __ENV.K6_DURATION || '5m';
const RATE = parseInt(__ENV.K6_RATE || '10', 10); // scenarios/sec

const runLatency = new Trend('scenario_run_latency', true);
const runErrors = new Rate('scenario_run_errors');
const listLatency = new Trend('scenario_list_latency', true);
const runsStarted = new Counter('scenario_runs_started');

export const options = {
  scenarios: {
    sustained: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 50,
      maxVUs: 150,
    },
  },
  thresholds: {
    // Scenario run SLOs (heavier than list).
    scenario_run_latency: ['p(50)<500', 'p(95)<1500', 'p(99)<3000'],
    scenario_run_errors: ['rate<0.02'],
    // List endpoint must still meet the standard read SLO.
    scenario_list_latency: ['p(50)<200', 'p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.02'],
  },
};

const listParams = {
  headers: { Accept: 'application/json', 'User-Agent': 'k6-load/1.0' },
  timeout: '5s',
};

const runParams = {
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    'User-Agent': 'k6-load/1.0',
  },
  timeout: '10s',  // scenarios can take longer; cap at 10s.
};

// A rotating pool of pre-seeded scenario IDs. Override via env for prod.
const SCENARIO_IDS = (__ENV.K6_SCENARIO_IDS || '')
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean);

export default function () {
  let scenarioId = SCENARIO_IDS[__ITER % SCENARIO_IDS.length];

  // If no IDs supplied, list first to discover one.
  if (!scenarioId) {
    group('list scenarios', () => {
      const listRes = http.get(`${BASE}/api/v1/scenarios?limit=20`, listParams);
      listLatency.add(listRes.timings.duration);
      check(listRes, { 'list 200': (r) => r.status === 200 });
      try {
        const body = listRes.json();
        const arr = body?.scenarios || body?.data || [];
        if (Array.isArray(arr) && arr.length > 0) {
          scenarioId = arr[0].id || arr[0].uuid;
        }
      } catch {
        // fall through — we'll skip the run step.
      }
    });
  }

  if (!scenarioId) {
    sleep(0.5);
    return;
  }

  group('run scenario', () => {
    const url = `${BASE}/api/v1/scenarios/${encodeURIComponent(scenarioId)}/run`;
    const body = JSON.stringify({
      inputs: {
        population: 50_000_000,
        gdp_growth: 0.05,
      },
      audience: 'general',
      timeout_seconds: 30,
    });

    const res = http.post(url, body, runParams);
    runLatency.add(res.timings.duration);
    runsStarted.add(1);

    const ok = check(res, {
      'status is 200 or 202': (r) => r.status === 200 || r.status === 202,
      'has run_id or scenario_run_id': (r) => {
        try {
          const b = r.json();
          return Boolean(b?.run_id || b?.scenario_run_id || b?.id);
        } catch {
          return false;
        }
      },
    });
    runErrors.add(!ok);
  });

  sleep(Math.random() * 0.5);
}

export function handleSummary(data) {
  const m = data.metrics || {};
  const run = m.scenario_run_latency?.values || {};
  const lst = m.scenario_list_latency?.values || {};
  const started = m.scenario_runs_started?.values?.count ?? 0;
  const errRate = m.scenario_run_errors?.values?.rate ?? 0;

  return {
    stdout: `
=== /api/v1/scenarios load test ===
runs started:        ${started}
scenario_run_latency
  p50: ${ms(run['p(50)'])}    p95: ${ms(run['p(95)'])}    p99: ${ms(run['p(99)'])}
scenario_list_latency
  p50: ${ms(lst['p(50)'])}    p95: ${ms(lst['p(95)'])}    p99: ${ms(lst['p(99)'])}
scenario_run_errors: ${(errRate * 100).toFixed(2)}%

SLO check (run):
  p50<500ms:   ${(run['p(50)'] ?? 0) < 500 ? 'PASS' : 'FAIL'}
  p95<1500ms:  ${(run['p(95)'] ?? 0) < 1500 ? 'PASS' : 'FAIL'}
  p99<3000ms:  ${(run['p(99)'] ?? 0) < 3000 ? 'PASS' : 'FAIL'}
  err<2%:      ${errRate < 0.02 ? 'PASS' : 'FAIL'}
`,
  };
}

function ms(v) {
  if (v === undefined || v === null) return 'n/a';
  return `${v.toFixed(1)} ms`;
}
