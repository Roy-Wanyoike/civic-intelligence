# ADR-0012: Server-Sent Events for streaming AI responses

## Status

Accepted — 2026-09-09

## Context

The "Ask about this Bill" feature streams the AI's response to the citizen as it's generated. Options:

1. **WebSockets** — bidirectional, but overkill for our use case (we only need server → client streaming).
2. **Server-Sent Events (SSE)** — unidirectional, simple, works over HTTP, native browser support.
3. **Long polling** — works everywhere but inefficient.

## Decision

Adopt **Server-Sent Events (SSE)** for streaming AI responses from the BFF to the browser.

- Endpoint: `POST /api/v1/ai/questions/stream`
- Response content type: `text/event-stream`
- Events: `answer` (partial answer chunk), `citations` (final citation list), `done` (final metadata: validation status, failures).

## Consequences

- **Positive**: simpler than WebSockets — no protocol upgrade, no framing, no connection state.
- **Positive**: works through proxies and CDNs out of the box.
- **Positive**: native `EventSource` API in browsers — no client library needed.
- **Positive**: each event is a self-contained JSON object — easy to test.
- **Negative**: unidirectional — the client can't send mid-stream messages. Acceptable: the client sends the question up front via POST, then listens.
- **Negative**: some proxies buffer SSE aggressively. Mitigated by setting `X-Accel-Buffering: no` and `Cache-Control: no-cache`.

## Fallback

If SSE is unavailable (proxy issue, old browser), the client falls back to `POST /api/v1/ai/questions` (non-streaming) which returns the complete response as a single JSON object.

## References

- `services/ai/app/main.py` (`/v1/questions/stream` endpoint)
- `apps/web/src/components/bill-ask-panel.tsx`
