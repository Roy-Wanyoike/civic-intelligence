# 01 — Platform Overview

This document is the entry point to the architecture series. It describes what the Civic Intelligence Platform is, who it serves, what it deliberately is not, and how the rest of the documentation is structured. If you read only one document in `docs/architecture/`, read this one — then follow the cross-links to go deeper on the part you care about.

## What the platform is

The Civic Intelligence Platform is a system that takes civic information — Bills, Acts, Regulations, Hansard, Committee Reports, Gazettes, Policies, Government Decisions — and turns it into something a citizen can actually understand. "Understand" here means three things: see what is happening (a bill was introduced, a vote was taken, a regulation was gazetted), see why it matters (a plain-language explanation grounded in the source), and see where the explanation came from (a citation that resolves to a specific page and section of a specific document).

The platform is built around one architectural invariant — *evidence before AI* — and one product vision — *"Ask Kenya"*. The invariant says no AI-generated statement is shown to a citizen without a backing citation. The vision says the platform should eventually answer any citizen question about civic life by drawing evidence from across the entire civic record. The first product (Bill Intelligence / Bill Summarizer) is a wedge into that vision, not the final shape of the product.

## Who it serves

The platform serves five audiences, in priority order:

1. **Citizens.** The primary audience. A citizen should be able to land on the homepage, find a bill that affects them, read a plain-language summary, see the source, and decide what to think. The citizen is not assumed to have any legal training, any political affiliation, or any prior context. The platform explains; it does not persuade.
2. **Students.** Secondary school and university students studying civics, law, political science, or journalism. They use the platform to understand how legislation actually works in their country, with the source documents available for primary research.
3. **Journalists.** Reporters covering parliament, regulation, and government decision-making. They use the platform to find stories (a bill that is moving fast, an amendment that contradicts a prior commitment, a vote that broke along unusual lines) and to back their reporting with citations.
4. **Researchers.** Academics and policy analysts studying legislative behavior over time. They use the platform's structured data (BillVersions, amendments, votes, timelines) and the API for longitudinal analysis.
5. **Developers.** Engineers building on top of the platform — civic-tech NGOs, newsroom dev teams, government digital services. They consume the API and the open data.

The platform does not optimize for politicians, political parties, lobbyists, or government press offices. They are welcome to use it, but design decisions favor the citizen.

## What the platform is not

Equally important is what the platform deliberately does not do:

- **Not a political recommender.** The platform does not tell citizens who to vote for, what to support, or which position is correct. It explains what is happening and why it matters; it does not take sides. A bill's summary describes the bill, not whether the bill is good.
- **Not legal advice.** Plain-language explanations are educational, not advisory. A citizen who needs legal advice on how a law applies to their specific situation should consult a lawyer. The platform's terms of service make this explicit.
- **Not a vote-decider.** The platform does not aggregate citizen opinions, run polls, or measure support. It does not show "73% of users support this bill." Voting happens at the ballot box, not in the platform.
- **Not a primary source.** The platform is a layer *over* primary sources. Every fact links back to its source document. Where the source and the platform disagree, the source is correct and the platform is buggy; report it.
- **Not a substitute for parliament.** The platform does not draft bills, propose amendments, or participate in the legislative process. It observes and explains; it does not legislate.

These negative constraints are design constraints, not limitations to be removed later. They keep the platform trustworthy.

## How the documentation is structured

The `docs/architecture/` series is numbered so it can be read in order. Each document links to the next and to relevant ADRs.

| # | Document | What it covers |
| --- | --- | --- |
| 01 | This document | What the platform is, who it serves, what it is not. |
| 02 | [Services](./02-services.md) | The 9 Go services + Python AI service, with responsibilities and boundaries. |
| 03 | [Domain model](./03-domain-model.md) | The full civic domain model: Bills, Acts, People, Committees, Evidence, AI outputs. |
| 04 | [Country adapters](./04-country-adapters.md) | The adapter pattern and how to add a new country. |
| 05 | [Evidence system](./05-evidence-system.md) | Claim → Evidence → Document → Page/Section → Source chain, contradiction engine. |
| 06 | [AI gateway + RAG](./06-ai-gateway-rag.md) | The RAG pipeline, the 12 capabilities, the eval framework. |
| 07 | [Events + workflows](./07-events-workflows.md) | Full event catalog and Temporal workflows. |
| 08 | [Security](./08-security.md) | Deep security: OIDC, RBAC, SSRF, prompt injection, audit, CI gates. |
| 09 | [Observability](./09-observability.md) | OpenTelemetry, metrics catalog, dashboards, alerting. |
| 10 | [Testing](./10-testing.md) | Unit, integration, contract, e2e, AI-eval layers. |
| 11 | [Data model](./11-data-model-architecture.md) | Single Postgres cluster, logical schemas, pgvector, JSONB, FTS. |
| 12 | [Deployment](./12-deployment.md) | 5 deployables, Docker Compose, Helm, Argo CD, environments. |
| 13 | [Roadmap](./13-roadmap.md) | The 8 phases from Foundation to Global Expansion. |
| 14 | [Product vision](./14-product-vision.md) | "Ask Kenya," the CivicMatter abstraction, the cross-domain intelligence map. |

When in doubt about where something is documented, start here and follow links.

## The wedge and the north star

The Bill Summarizer is the wedge — the smallest useful product that delivers real value and exercises the full pipeline (ingestion → documents → evidence → intelligence → search → API → web). The north star is "Ask Kenya": a citizen types a question and gets an evidence-backed answer that draws from Bills, Acts, Regulations, Hansard, Committees, Policies, and Decisions together. The wedge proves the architecture; the north star proves the vision. Phase 3 of the roadmap delivers the wedge; Phase 8 delivers the north star across multiple countries. See [`docs/architecture/13-roadmap.md`](./13-roadmap.md) and [`docs/architecture/14-product-vision.md`](./14-product-vision.md).
