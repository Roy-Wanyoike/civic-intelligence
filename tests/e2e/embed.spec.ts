import { test, expect } from '@playwright/test';

/**
 * Embeddable widgets — issue #291.
 *
 * Three chrome-less, iframe-able widgets live under /embed/*:
 *   /embed/mp-finder     — constituency lookup → MP scorecard link
 *   /embed/bill-tracker  — Bill current-stage timeline (?bill_id=…)
 *   /embed/today         — today's parliament calendar events
 *
 * Each widget must:
 *   - return 200 OK
 *   - render its expected widget chrome (the "Powered by Civic
 *     Intelligence" footer link is the contract — it's present on
 *     every render path, including the no-bill_id prompt state)
 *   - render WITHOUT the main app's navbar/footer (the root layout
 *     skips the chrome for /embed/* routes via the x-civic-embed
 *     middleware marker; we assert the absence of the navbar's
 *     skip-link to make sure that branch fired)
 *   - be served with `Content-Security-Policy: frame-ancestors *`
 *     so any third-party origin may embed it via <iframe>
 *
 * The widgets fetch from the Go BFF (port 9000) and fall back to a
 * small built-in mock roster when the BFF is offline — so the tests
 * are deterministic without requiring a live backend.
 */

const EXPECTED_FOOTER = 'Powered by Civic Intelligence';

test('TestEmbedMPFinder_ReturnsWidget', async ({ request }) => {
  // --- HTTP-only smoke check ---
  const resp = await request.get('/embed/mp-finder');
  expect(resp.status(), 'GET /embed/mp-finder returns 200').toBe(200);
  const body = await resp.text();
  expect(body, 'renders the "Powered by" footer').toContain(EXPECTED_FOOTER);
  expect(body, 'renders the widget headline').toContain('Find your MP');
  expect(body, 'renders the search form label').toContain('Constituency name');
  expect(body, 'renders the search input').toContain('name="q"');

  // --- Embed-mode contract: NO main-app chrome ---
  // The root layout renders `<a href="#main" class="skip-link">…</a>`
  // on every non-embed route. If we see it, the embed branch didn't
  // fire and the iframe would include the navbar.
  expect(body, 'skip-link is absent (root layout is in embed mode)').not.toContain('skip-link');

  // --- CSP contract: must allow any origin to iframe the widget ---
  const csp = resp.headers()['content-security-policy'] ?? '';
  expect(csp, 'CSP frame-ancestors permits any origin').toMatch(/frame-ancestors\s*\*/);

  // --- Render path: with a query that should match a mock fallback ---
  const respQ = await request.get('/embed/mp-finder?q=Westlands');
  expect(respQ.status()).toBe(200);
  const bodyQ = await respQ.text();
  expect(bodyQ, 'search form preserves the query in the input').toContain('Westlands');
  // The mock fallback returns "Tim Wanyonyi" for Westlands — the live
  // BFF returns the actual sitting MP, which we don't assert here
  // (the contract is "MP name + party + scorecard link render", not
  // a specific name).
  expect(bodyQ, 'renders an MP scorecard CTA').toContain('View MP scorecard');
});

test('TestEmbedBillTracker_ReturnsWidget', async ({ request }) => {
  // --- No bill_id → friendly "how to use" state, still 200 + footer ---
  const respNoBill = await request.get('/embed/bill-tracker');
  expect(respNoBill.status(), 'GET /embed/bill-tracker (no bill_id) returns 200').toBe(200);
  const bodyNoBill = await respNoBill.text();
  expect(bodyNoBill, 'renders the "Powered by" footer').toContain(EXPECTED_FOOTER);
  expect(bodyNoBill, 'renders the widget headline').toContain('Bill Tracker');
  expect(bodyNoBill, 'prompts the embedder to pass bill_id').toContain('?bill_id=');

  // --- With bill_id → renders the 8-stage horizontal timeline ---
  // The mock fallback has a Bill with this ID (mockBills[0] —
  // "The Housing Bill, 2024" at current_stage "Committee Stage").
  const resp = await request.get(
    '/embed/bill-tracker?bill_id=00000000-0000-0000-0000-000000000001',
  );
  expect(resp.status(), 'GET /embed/bill-tracker?bill_id=… returns 200').toBe(200);
  const body = await resp.text();
  expect(body, 'renders the "Powered by" footer').toContain(EXPECTED_FOOTER);
  expect(body, 'renders the Bill title').toContain('The Housing Bill, 2024');
  // The 8 canonical stages must all appear in the tracker.
  for (const stage of [
    'First Reading',
    'Second Reading',
    'Committee Stage',
    'Report Stage',
    'Third Reading',
    'Assent',
    'Publication',
    'Commencement',
  ]) {
    expect(body, `renders the "${stage}" stage`).toContain(stage);
  }
  // The current stage marker ("Committee Stage" for the mock Bill)
  // must be marked current in the DOM so the widget renders a
  // "you are here" cue, not just a flat list of stages.
  expect(body, 'marks the current stage with aria-current').toContain('aria-current="step"');

  // --- CSP contract ---
  const csp = resp.headers()['content-security-policy'] ?? '';
  expect(csp, 'CSP frame-ancestors permits any origin').toMatch(/frame-ancestors\s*\*/);

  // --- Embed-mode contract: NO main-app chrome ---
  expect(body, 'skip-link is absent (root layout is in embed mode)').not.toContain('skip-link');
});

test('TestEmbedToday_ReturnsWidget', async ({ request }) => {
  const resp = await request.get('/embed/today');
  expect(resp.status(), 'GET /embed/today returns 200').toBe(200);
  const body = await resp.text();
  expect(body, 'renders the "Powered by" footer').toContain(EXPECTED_FOOTER);
  expect(body, 'renders the widget headline').toContain('Today in Parliament');
  expect(body, 'renders the long-form date').toMatch(
    /Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday/,
  );
  expect(body, 'renders the events list').toContain('class="events"');

  // --- CSP contract ---
  const csp = resp.headers()['content-security-policy'] ?? '';
  expect(csp, 'CSP frame-ancestors permits any origin').toMatch(/frame-ancestors\s*\*/);

  // --- Embed-mode contract: NO main-app chrome ---
  expect(body, 'skip-link is absent (root layout is in embed mode)').not.toContain('skip-link');
});

/**
 * Negative-control test: a non-embed route is NOT served with the
 * embed-specific CSP — `frame-ancestors` should be absent (or at
 * least NOT `*`) so the main site remains protected against
 * clickjacking. This guards against an accidental "set frame-ancestors
 * * globally" regression in next.config.mjs.
 */
test('Non-embed routes are NOT embeddable (clickjacking guard)', async ({ request }) => {
  const resp = await request.get('/');
  expect(resp.status()).toBe(200);
  const csp = resp.headers()['content-security-policy'] ?? '';
  // Either CSP is absent, or it does NOT permit frame-ancestors *.
  // (Next.js does not set CSP by default — so we only assert the
  // negative: frame-ancestors * must NOT appear.)
  expect(csp, 'main site does not opt into global iframe embedding').not.toMatch(
    /frame-ancestors\s*\*/,
  );
});
