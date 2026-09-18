# droidcon Uganda 2026 — Talk Proposal

> **Conference:** droidcon Uganda 2026 — October 28–29, 2026, Kampala
> **Speaker:** Roy Wanyoike
> **Submitted:** 2026-09-15
> **Format:** Session (30 minutes)
> **Level:** Intermediate

---

## Title

**Building Civic Intelligence with Next.js + AI: Lessons from Kenya's Parliament**

---

## Abstract

Parliament publishes a lot — Bills, Hansard, Order Papers, committee reports, assent notices, gazettes. It's all public. It is also almost impossible for an ordinary citizen to read. A Bill in Kenya runs to 80 pages of legalese; the Kenya Law site lists 50+ live Bills at any time; the Senate and the National Assembly each publish their own Hansard. By the time a Bill becomes an Act, even keen followers have lost the thread.

I've spent the last year building the Civic Intelligence Platform — an open-source project that turns this firehose into something a citizen can actually use: plain-language Bill summaries, verified timelines, evidence-grounded Q&A, and "what if this Bill becomes law?" scenarios. The platform is live for Kenya, with an adapter ready for Uganda and four more African countries on the roadmap. The codebase is roughly 350 source files: a Next.js App Router frontend (36 routes), a Go BFF and microservices (legislation, ingestion, evidence, simulation, intelligence, notifications), a Python FastAPI AI service (12 RAG capabilities), Postgres with pgvector, NATS JetStream, and Temporal. It has 456 Go tests across 40 files and 22 Python tests including a permanent AI evaluation dataset that blocks any prompt change from reaching production.

This talk is a case study, not a tutorial. I'll walk through three concrete engineering decisions and the trade-offs behind them.

First, the **service boundary**: why we put the LLM in a separate Python service behind a provider-agnostic gateway, instead of calling OpenAI from inside the Next.js server components. The answer is partly cost (we can swap providers per capability), partly safety (every response runs through a citation validator that fails closed), and partly because Go and Python make different trade-offs for different jobs — and we want both.

Second, **evidence-first AI**: every AI response must be grounded in a retrieved document, every claim must cite its source, and every citation is checked against the snippet it points at. A response that cannot be grounded is surfaced as `[This response could not be fully verified]` — not silently passed through. The frontend uses a six-label visual language (FACT / EVIDENCE / ASSUMPTION / SIMULATION / UNKNOWN / LIMITATION) so users can see at a glance which parts of a page are observed and which are modeled. I'll show the small TypeScript component that implements it and the Go types that enforce the same boundary on the backend.

Third, the **country adapter pattern**: the same platform supports Kenya (bicameral, 11 stages) and Uganda (unicameral, 8 stages) without any country-specific strings in the global domain model. Every stage name, every term, every committee is data shipped by an adapter that implements a single Go interface. Adding Uganda was an engineering task — not a rewrite.

I'll close with what didn't work: where the citation validator was too strict and we had to relax it, where we initially leaked country knowledge into the domain and had to refactor, and what the next 6 months look like — including the Uganda live demo.

---

## After this talk, you'll learn…

1. **How to architect a multi-service civic platform with Next.js App Router, a Go BFF, and a Python AI service** — including the trade-offs at each boundary, why the LLM lives in its own service behind a provider-agnostic gateway, and how to keep the frontend honest when the backend mixes observed facts with AI-generated explanations.

2. **A practical pattern for "evidence-first AI"**: how to ground LLM outputs in authoritative sources so users can verify every claim, with the 6-label visual language (FACT / EVIDENCE / ASSUMPTION / SIMULATION / UNKNOWN / LIMITATION) we use to separate reality from simulation in the UI — and the matching Go types that enforce the same boundary server-side.

3. **How to build a jurisdiction-agnostic adapter system** that lets the same platform work across Kenya, Uganda, Tanzania, Ghana, Nigeria, and South Africa — and the data-modeling decisions that make onboarding a new country an engineering task, not a rewrite. The talk includes a live demo of the Uganda adapter against `parliament.go.ug`.

---

## Speaker bio

Roy Wanyoike is a software engineer based in Nairobi. He builds the Civic Intelligence Platform — an open-source effort to make Kenyan and East African parliamentary information legible to ordinary citizens. His focus is on civic tech that respects users: evidence-grounded AI instead of confident-sounding chatbots, immutable legislative history instead of silent edits, and a country-adapter architecture that lets the same codebase serve multiple jurisdictions without rewriting the core. Before this, he worked on backend systems in Go and Python. He writes about civic-tech engineering trade-offs in public on GitHub and is happiest when a pull request ships with tests, an ADR, and a documented limitation.

---

## Notes for reviewers

This talk fits droidcon Uganda specifically because Uganda is one of the six countries the platform already supports — we ship a working Uganda adapter (unicameral Parliament, 8 stages, 25 parliamentary terms) with contract tests, and the talk includes a live demo against `parliament.go.ug`. The audience in Kampala will see a real production codebase — 350+ files, 456 Go tests, 22 Python tests including a permanent AI evaluation dataset — not a toy example or a slide full of buzzwords. Even though the talk is not Android-specific, the architecture lessons transfer directly: the evidence-first AI pipeline, the provider-agnostic gateway, the country-adapter pattern, and the six-label reality UI are all patterns an Android engineer can apply when building apps that talk to LLMs or that need to work across multiple jurisdictions. The talk also contributes back: anyone who wants to extend the Uganda adapter, write the Tanzania one, or build a Ugandan-language UI for the platform can do it on a documented path, and I'll point to the contributing guide and the contract test suite at the end.
