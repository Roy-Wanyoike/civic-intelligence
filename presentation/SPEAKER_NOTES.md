# Speaker Notes — droidcon Uganda 2026

> **Talk:** Building Civic Intelligence with Next.js + AI: Lessons from Kenya's Parliament
> **Speaker:** Roy Wanyoike
> **Length:** 30 minutes (≈ 26 min talk + 4 min Q&A)
> **Audience:** Junior-to-intermediate developers, mixed Android / web / backend background
> **Voice:** Engineer-to-engineer, not preacher. Concrete. Say "I", not "we" unless literally true.

---

## Slide 1 — Title

> "Hi, I'm Roy. I'm here to talk about a year-long project I've been building in Nairobi — the Civic Intelligence Platform. It's an open-source effort to make East African parliamentary information legible to ordinary citizens. I'm going to walk you through what we built, why we built it this way, and the lessons that I think will save you time if you ever try something similar."

**Pacing note:** 30 seconds. Don't read the title slide. Introduce yourself, then move on.

**Audience hook:** Mention this is a case study — they will see real production code, real numbers, real failures. Not a tutorial, not a pitch.

---

## Slide 2 — The problem

> "The problem is simple. Parliament publishes a lot — Bills, Hansard, Order Papers, committee reports, assent notices, the Kenya Gazette. It's all public. And it is almost impossible for a normal citizen to read."

Pause.

> "A typical Kenyan Bill runs 80 pages of legalese. The Kenya Law site lists 50+ live Bills at any moment. The National Assembly and Senate each publish their own Hansard. By the time a Bill becomes an Act, even people who follow politics for a living have lost the thread."

> "And the worst part: the information is technically there. You can find it. You just can't understand it. So the gap between 'public' and 'accessible' is where the harm happens."

**Pacing:** 2 minutes. Stop on the second-to-last bullet — let the audience feel the gap.

**If they ask later "why not just use ChatGPT?"** — this is the setup. The answer is: ChatGPT is the gap, not the bridge. Civic information is the one domain where a confident wrong answer is worse than no answer.

---

## Slide 3 — The opportunity

> "But here's the thing. The sources are out there. parliament.go.ke, kenyalaw.org, president.go.ke — they publish HTML, PDF, RSS. It's structured enough to scrape, even if it's ugly. So the opportunity is: what if we treat every authoritative source as a first-class citizen, ground every claim in one of them, and let AI do the explaining instead of the deciding?"

> "That's the whole pitch. Show people what happened. Explain what it means. Show them the evidence. Let them decide what to think. We are not a political platform. We don't rank politicians, we don't endorse, we don't tell you what to think. We just make the public record legible."

**Pacing:** 1.5 minutes. This is the mission slide. Don't linger — it's not a TED talk.

---

## Slide 4 — Architecture overview

> "OK, so this is the system. It's a lot. Let me walk you through it."

Point to the diagram.

> "On the left, official sources — Parliament, Kenya Law, the President's office. Each country has an adapter that knows how to discover, fetch, and parse its own sources."

> "In the middle, a Go BFF — services/api. It exposes the REST endpoints the frontend hits. Behind it are domain services: legislation owns Bills, Acts, stages; ingestion crawls; documents parses PDFs and HTML; evidence owns citations; intelligence runs candidate-fact validation; notifications handles follows; simulation is the what-if engine."

> "On the right, the Next.js frontend — 36 routes, App Router, TanStack Query. And the Python AI service is a separate FastAPI app — 12 capabilities, RAG pipeline, citation validation."

> "Two things I want you to notice. First — Python and Go live side by side. We did not try to do the AI in Go, and we did not try to do the Bill state machine in Python. Each language does what it's good at. The boundary is HTTP, not a library call."

> "Second — the LLM is behind a gateway. The frontend never calls OpenAI directly. Every request goes through the gateway, which enforces budgets, retries, fallback, and routes by capability — `bill_summarizer`, `civic_question_answerer`, `timeline_extractor` — not by provider."

**Pacing:** 3 minutes. This is the densest slide. Pause on the Python/Go split — that's the trade-off most teams get wrong.

---

## Slide 5 — Evidence-first AI pipeline

> "OK, here's the RAG pipeline. It's standard RAG with one twist at the end."

Walk through the pipeline diagram.

> "Question in. Intent classification — is this a summarize, a timeline, a stage, an impact question? Query expansion. Hybrid search — FTS plus pgvector. Rerank. Evidence selection. Context construction with citations. Then the LLM. Then — and this is the part that took the longest — claim extraction and citation validation."

> "The citation validator runs five checks on every claim. Does the cited source exist? Does the citation point to the right document? Does the snippet support the claim — meaning, do the claim and the snippet share at least one significant word? Is the claim stronger than the evidence? Is the source authoritative?"

> "If any of those fail, the response is not silently passed through. We mark it `validation_status = FAILED` and the UI surfaces it as: `[This response could not be fully verified from available authoritative sources.]`"

> "Is the heuristic perfect? No. We've had false positives where a perfectly good claim got flagged because the citation used a synonym. But under-claiming is the feature, not the bug. Civic information is the one domain where being wrong confidently is the worst failure mode."

**Pacing:** 3 minutes. The pipeline is on screen — point to each stage. Don't read the bullet list aloud; let the diagram do work.

**Honesty beat:** Mention the false-positive trade-off. Don't pretend it's solved.

---

## Slide 6 — The six reality labels

> "Here's the part I'm most proud of. Every piece of content on the platform carries a reality label. Six of them."

Point to the labels.

> "FACT — verified civic information from an authoritative source. EVIDENCE — authoritative supporting material, traceable to a source. ASSUMPTION — an explicit scenario input that has not been observed. SIMULATION — a modeled outcome, NOT an observed civic fact. UNKNOWN — insufficient evidence, the platform does not fabricate precision. LIMITATION — a known limitation of the model or scenario."

> "These are not just UI tags. They are enforced at the type level. The Go domain model has a `RealityLayer` enum: `OBSERVED`, `HYPOTHETICAL`, `MODELED`, `UNKNOWN`. Every record in the simulation domain is tagged `HYPOTHETICAL` or `MODELED` — never `OBSERVED`. Observed records belong to the legislation service, not the simulation service. You cannot accidentally save a simulated outcome as if it were a fact — the type system stops you."

> "And in the React layer, every scenario page opens with a disclaimer banner that uses the same vocabulary. So a user landing on a 'what if this Bill becomes law' page immediately sees: this is HYPOTHETICAL. The results are SIMULATED. Reality is observed; scenarios are constructed; assumptions are explicit; evidence remains traceable."

**Pacing:** 2.5 minutes. Show the TypeScript `RealityBadge` component briefly — the code is small, which is the point.

**Audience takeaway:** The visual language and the type language are the same language. That's the trick. You can't have one without the other.

---

## Slide 7 — Scenario engine

> "OK, the scenario engine. This is the riskiest part of the platform — because it's where AI could most easily fabricate civic facts. So we built very strict guardrails."

> "A scenario has a baseline — observed reality, with sources. It has assumptions — explicit, each one tagged by type: OBSERVED_INPUT, USER_DEFINED, MODEL_ASSUMPTION, HISTORICAL_REFERENCE, ESTIMATE, or UNKNOWN. Unknown assumptions cannot carry a value — because that would be fabricated precision. It has variables — typed: integer, decimal, percentage, currency, date, duration, boolean, categorical, geographic, entity reference. It has constraints — a small expression grammar: `<`, `<=`, `>`, `>=`, `==`, `!=`, `&&`, `||`, `!`, parentheses. It has a model. And it has a time horizon."

> "Before a scenario runs, the validation pipeline walks through: inputs, evidence, assumptions, model, units, time horizon, constraints. If any step fails, the scenario doesn't run."

> "Two engines ship today: deterministic rules, and Monte Carlo. The Monte Carlo engine produces percentiles — median, p10, p90, min, max — never a single point estimate. Because a single number out of a model is a lie. A distribution is honest."

Show the Go snippet.

> "Notice the comment: `Modeled output. Values are simulated percentiles, NOT observed facts.` That string is in the response payload. The UI cannot strip it. Even if someone tried, the type system would prevent the result from being saved as an observed fact."

**Pacing:** 3 minutes. This is the deepest technical slide — pause on the validation pipeline order. Emphasize the UNKNOWN cannot carry a value rule.

---

## Slide 8 — Multi-country adapter architecture

> "OK, multi-country. This is the part that I think generalizes beyond civic tech. Every country's parliament is different. Kenya is bicameral — National Assembly plus Senate. Uganda is unicameral. Nigeria is bicameral but with different houses. South Africa has a National Council of Provinces. Tanzania has a unicameral Bunge."

> "The naive approach is: write Kenya into the domain. Then write Uganda into the domain. Then realize Nigeria breaks everything. We did the opposite. The global domain model contains zero country-specific strings. No 'Senate', no 'National Assembly', no 'Second Reading'. Those are all values supplied by an adapter."

Point to the interface.

> "The adapter interface is seven methods. Discover, Fetch, Parse, NormalizeSourceItem, GetLegislativeStructure, GetStages, GetTerminology. That's it. Implement those seven methods and your country is onboarded. The domain doesn't change."

> "We have six adapters in the repo. Kenya is the reference implementation — full parliament, kenya_law, president, gazette, 84 tests. Uganda is ready — 11 tests, unicameral, 8 stages. Tanzania, Ghana, Nigeria, South Africa are scaffolded — the contract tests exist, the structure is there, the data is partial."

> "The contract tests are the secret. Every adapter has a `contract_test.go` that asserts it satisfies the interface — and that its stages form a valid transition graph, and that its terminology has at least 25 terms with sources. You can't merge an adapter that breaks the contract. CI catches it."

**Pacing:** 2.5 minutes. This is the slide that most applies to Android devs — the adapter pattern is universal.

---

## Slide 9 — Uganda adapter deep dive

> "Let me show you the Uganda adapter. It's small — that's the point."

Show the Go code on slide.

> "UgandaAdapter implements `contracts.LegislativeSourceAdapter`. It has a parliament sub-adapter that does the actual HTTP. It returns `CountryCode() = "UG"`. It supports URLs that contain `parliament.go.ug` or `parliamentwatch.ug`. It delegates Discover, Fetch, Parse to its parliament adapter. And it returns Uganda-specific data — 8 stages, 25 terms, a unicameral house with 556 members — from the internal data package."

> "Notice the compile-time assertion at the bottom: `var _ contracts.LegislativeSourceAdapter = (*UgandaAdapter)(nil)`. If the adapter ever drifts from the interface — say, we add a method to the interface and forget to implement it — the build fails. Not the test. The build."

> "And here are the stages. First Reading, Second Reading, Committee Stage, Report Stage, Third Reading, Presidential Assent, Commencement, Rejected. Each one has a `SimpleExplanation` — plain language. Each one has `AllowedNext` — what stages can legally follow. The state machine is data, not code."

> "The terminology has 25 terms with sources — most pointing at parliament.go.ug. Royal Assent is in there as `Not applicable in Uganda — see Presidential Assent`. That's a small detail but it matters — the same concept has a different name across Commonwealth countries, and the adapter handles it."

**Pacing:** 2.5 minutes. The demo runs in the next slide; here, focus on the architecture of the adapter.

---

## Slide 10 — The reality/simulation boundary

> "I want to spend a minute on this because it's the part that took the longest to get right. How do you prevent AI from fabricating civic facts?"

> "Three layers. First — the type system. The simulation domain has a `RealityLayer` enum. Every record is tagged. The `Scenario.Validate()` method rejects a scenario whose reality layer is not HYPOTHETICAL, or whose baseline is not OBSERVED. You cannot save a malformed record."

> "Second — the validation pipeline. The simulation engine runs `ValidatePipeline` before it executes. Inputs, evidence, assumptions, model, units, time horizon, constraints — seven gates, in order. If any fails, the scenario stays in DRAFT or returns FAILED."

> "Third — the API. Every response from `/api/v1/scenarios/*` carries a disclaimer. The create endpoint says `All scenarios are HYPOTHETICAL. They are NOT observed civic facts.` The run endpoint says `These results are SIMULATED. They are NOT observed civic facts.` The methodology endpoint says `This scenario is HYPOTHETICAL. Outputs are SIMULATED.` Even if a malicious client tried to strip the disclaimer, the type system would prevent the simulation result from being saved to an `OBSERVED` table."

> "And the database enforces it too. The `intelligence.candidate_facts` table has a CHECK constraint: `(validated = FALSE) = (accepted_at IS NULL)`. AI cannot promote a candidate fact to canonical state without going through validation. That constraint is in the schema — not in the application code. You can't bypass it by calling the wrong method."

Show the CHECK constraint snippet.

**Pacing:** 2.5 minutes. This is the trust slide. Slow down on the schema constraint — that's the "I didn't know you could do that in Postgres" moment for the audience.

---

## Slide 11 — Tech stack + lessons learned

> "OK, lessons. Three things worked, three didn't."

> "Worked: putting the LLM behind a provider-agnostic gateway. We started on the Stub provider — no API key, no network — and the entire pipeline ran in CI. When we plugged in OpenAI, the only change was one environment variable. When we want to switch to Anthropic or a local model, the only change is the gateway factory."

> "Worked: the country adapter pattern. Uganda was added in a week. The domain didn't change. The frontend didn't change. The DB schema didn't change. We wrote `adapters/uganda/` and added a row to `legislation.countries`."

> "Worked: immutable Bill versions. ADR-0011. We never UPDATE — we INSERT. History is preserved forever. When a Bill's stage changes, the old stage is still there, with the date it was recorded. This turned out to matter more than I expected — it's how the timeline feature works."

> "Didn't work: the citation validator's word-overlap heuristic. It catches hallucinations, but it also catches legitimate synonyms. 'MP' vs 'Member of Parliament' fails the check. We had to add a synonym table. We're still tuning."

> "Didn't work: initially leaking Kenya-specific knowledge into the domain. The first version of the Bill state machine had 'Second Reading' hardcoded. When we onboarded Uganda, we had to refactor. The ADR-0004 country adapter pattern came out of that mistake."

> "Didn't work: trying to ship 15 microservices on day one. The first production deploy was a modular monolith — five deployables, not fifteen. The boundaries exist in code, the deployment can split later. ADR-0001."

**Pacing:** 3 minutes. This is the most useful slide for the audience — they'll remember the failures more than the successes.

---

## Slide 12 — Live demo

> "OK, demo time. Let me show you the platform."

**Demo checklist (5 minutes):**

1. Open the homepage. Point at the search bar. Type: `What is happening with housing?` Show the results page.
2. Click into a Bill. Show the plain-language summary, the verified timeline, the citation links. Click a citation — show it opens the original source.
3. Open the "Ask about this Bill" chat. Ask a question. Show the AI response with inline citations. Show what happens when a question can't be verified — the `[This response could not be fully verified]` notice.
4. Switch to the Scenarios section. Open a "What if this Bill becomes law?" scenario. Show the HYPOTHETICAL banner. Show the Monte Carlo output — median, p10, p90, min, max.
5. Switch country: open the Uganda page. Show the Uganda Bill stages. Show the Uganda terminology. Open `parliament.go.ug` in another tab to show the source.
6. Final: open the GitHub repo. Show the test count. Show the adapter folder structure.

**If the demo fails (backup):** Switch to the screenshots in the slides. Say: "This is the part of the talk where I pretend the WiFi works. It doesn't. Let me show you screenshots instead." Audience will laugh. Move on.

**Pacing:** 5 minutes. Hard stop at 5 — don't let the demo eat the rest of the talk.

---

## Slide 13 — Open source + how to contribute

> "Everything I've shown you is open source. MIT. On GitHub — github.com/Roy-Wanyoike/civic-intelligence. 350+ files, 456 Go tests, 22 Python tests, 17 SQL migrations, 14 ADRs."

> "If you want to contribute, three paths. First — extend the Uganda adapter. The structure is there, the contract tests are there, the data is partial. The biggest gap is real Bill discovery against parliament.go.ug — the discovery scaffold exists, the parser needs to handle their HTML."

> "Second — write the Tanzania adapter. The contract tests exist as a stub. Tanzania is unicameral, Bunge, different stages. The 7-method interface is your contract."

> "Third — improve the citation validator. The word-overlap heuristic is too strict. If you have NLP background, this is the place. We need a similarity check that handles synonyms, abbreviations, and partial matches without opening the door to hallucinations."

> "The contributing guide is in CONTRIBUTING.md. The ADRs are in docs/adr/. The architecture doc is ARCHITECTURE.md. If you want to add a country, the onboarding checklist is in ADR-0004."

**Pacing:** 1.5 minutes. End on the call to action — Uganda is the audience's home turf. They are the best people to extend the Uganda adapter.

---

## Slide 14 — Q&A

> "Thank you. I have time for questions. I'm Roy, you can find me on GitHub at Roy-Wanyoike. The repo is github.com/Roy-Wanyoike/civic-intelligence. The slides and a longer write-up are linked in the description."

**Q&A prep — likely questions and answers:**

**Q: Why not just use ChatGPT directly?**
A: Two reasons. First, ChatGPT hallucinates — it invents Bills, dates, MPs, votes. For civic information, that's not a small bug, it's the whole game. Second, ChatGPT can't cite its sources. Every claim on our platform links to a real document we fetched and hashed. If the link is dead, you can see the snapshot we archived.

**Q: How do you handle source conflicts — when Parliament says one thing and Kenya Law says another?**
A: We don't silently resolve. ADR-0013. We record a SourceConflict with both sources and metadata, and flag it for manual review. The UI surfaces it: "Source A says stage=Committee, source B says stage=Second Reading." We show the user the conflict; we don't pick a winner.

**Q: Why Go and Python? Why not all TypeScript?**
A: Each language does what it's good at. Go is great for the BFF, the ingestion workers, the domain state machines — typed, fast, deployable as a single binary. Python is great for the AI layer — FastAPI, Pydantic, the rich ecosystem of ML libraries. The boundary is HTTP. ADR-0002 documents the trade-off.

**Q: How much does it cost to run?**
A: With the Stub provider, free. With OpenAI, our daily budget is set in the gateway config — if we exceed it, the gateway raises `GatewayBudgetExceededError` and the request fails closed. We track cost per request via `_estimate_openai_cost_cents`. Real cost: a few dollars a day at current traffic.

**Q: How does this apply to Android developers?**
A: Three patterns transfer directly. First, the provider-agnostic gateway — any Android app that calls an LLM should have one, so you can swap providers without rewriting the app. Second, the country-adapter pattern — if your app needs to work across multiple jurisdictions (payments, taxes, identity), the adapter interface is the right abstraction. Third, the reality-label visual language — if your app mixes observed data with AI-generated content, label them differently. Users will trust you more.

**Q: What about privacy? Do you store user questions?**
A: Questions are processed in the AI service. We log the question text and the response for audit — that's necessary for the eval dataset. We don't sell or share the data. The identity service has standard RBAC: citizen, researcher, editor, admin. PII handling is documented in SECURITY.md.

**Q: Have you talked to the Kenyan Parliament about this?**
A: Not yet. The platform is independent — not affiliated with the Government of Kenya, as the disclaimer says. We'd welcome a conversation. The codebase is open-source under MIT, so if Parliament wanted to run their own instance, they could.

**Q: How long did it take to build?**
A: About a year, part-time. Phase 1 — foundation, monorepo, CI, DB, domain — was three months. Phase 2 — Kenya adapter, 50 real Bills, ingestion — was four months. Phase 3 — AI, RAG, citation validation, eval dataset — was three months. Phase 4 onward — simulations, government, post-assent, multi-country — is the current work.

**Pacing:** 4 minutes of Q&A. If they have no questions, say: "Let me leave you with one thought — the gap between 'public' and 'accessible' is where the harm happens. If you build anything civic, build it so the evidence is traceable. Thank you."

---

## Final reminders

- **Don't apologize.** If the demo fails, laugh and move on. Don't say "sorry, the WiFi is bad." Say "this is the part where I pretend the WiFi works."
- **Don't read the slides.** The slides are for the audience. You're the talk; the slides are the backdrop.
- **Don't say "leverage."** Or "delve into." Or "in today's fast-paced world." Or "unlock the power of." Engineers will stop listening.
- **Do say "I."** This is your project. Own it. "We" is fine when you mean "the codebase" or "the architecture" — but the decisions were yours.
- **Do mention what didn't work.** The failure stories are the most useful part. The audience will remember them.
- **Do point to the repo.** Repeatedly. The whole point is that they go look at it after the talk.

---

## Timing summary

| Section             | Slide(s) | Time   | Running |
|---------------------|----------|--------|---------|
| Title + intro       | 1        | 0:30   | 0:30    |
| Problem             | 2        | 2:00   | 2:30    |
| Opportunity         | 3        | 1:30   | 4:00    |
| Architecture         | 4        | 3:00   | 7:00    |
| Evidence-first AI   | 5        | 3:00   | 10:00   |
| Reality labels      | 6        | 2:30   | 12:30   |
| Scenario engine     | 7        | 3:00   | 15:30   |
| Multi-country       | 8        | 2:30   | 18:00   |
| Uganda adapter      | 9        | 2:30   | 20:30   |
| Reality boundary    | 10       | 2:30   | 23:00   |
| Lessons             | 11       | 3:00   | 26:00   |
| Demo                | 12       | 5:00   | (overlap with 11 in practice — demo runs from 22:00–27:00) |
| Open source         | 13       | 1:30   | 28:30   |
| Q&A                 | 14       | 4:00   | 32:30   |

> Total: ~30 min + 2 min overrun buffer for Q&A. Trim Q&A or the demo if running long.
