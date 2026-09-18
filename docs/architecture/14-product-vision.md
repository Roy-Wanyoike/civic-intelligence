# 14 — Product Vision

This document describes the platform's product vision: the "Ask Kenya" north star, the `CivicMatter` abstraction that makes it possible, why the Bill Summarizer is the wedge and not the product, and the eventual cross-domain intelligence map that the platform will offer. This is the document to read when the team is asking "why are we building this?" — every architectural and product decision traces back to a sentence here.

## The north star: "Ask Kenya"

The platform's north star is a single product experience: a citizen opens the platform, types a question in plain language, and gets an evidence-backed answer. "Will this bill affect school fees?" "When did my MP last vote, and on what?" "What did the committee say about the budget?" "Has the regulation on import duties changed since last year?" The answer comes back in seconds, written in plain language, with citations that resolve to the specific page and section of the specific document that supports each claim. The citizen can click any citation and read the source for themselves.

This is "Ask Kenya" — and eventually "Ask Uganda," "Ask Tanzania," "Ask Ghana," and so on. It is the product the platform exists to deliver. Every phase of the roadmap, every architectural choice, every capability built, is a step toward this north star.

The north star is not a chatbot. A chatbot answers; it does not cite. A chatbot is helpful; it is not trustworthy. The platform's answer is trustworthy because every claim is grounded in evidence that the citizen can verify. If the platform cannot find evidence for a claim, it says so — "we could not find a source for this" — rather than fabricating one. This is the "evidence before AI" principle made product: the AI is the explainer, the evidence is the authority.

The north star is not a search engine. A search engine returns documents; the citizen reads them. The platform returns explanations; the documents are the backing. Search is a building block (the citizen can search if they want), but the primary experience is "ask and receive an explained, cited answer," not "search and read."

The north star is not legal advice. The platform explains what the law says and what is happening in the legislative process; it does not advise a citizen on what to do about their specific situation. A citizen who needs legal advice is told to consult a lawyer. This is a deliberate scope limit, not a gap to be filled later.

## The CivicMatter abstraction

The `CivicMatter` abstraction (described in [`03-domain-model.md`](./03-domain-model.md#civicmatter-the-long-term-abstraction)) is the technical enabler of the north star. A `CivicMatter` is any unit of civic life that the platform tracks: a `Bill`, an `Act`, a `Regulation`, a `Policy`, a `GovernmentDecision`, eventually a `Judgment`. Each `CivicMatter` has a title, a status, a set of versions, a set of events, a set of citations, and a set of AI explanations. The concrete subtypes add their own fields, but the shared shape lets the platform treat them uniformly — and, crucially, lets the `Ask` capability retrieve across all of them in a single query.

Without the `CivicMatter` abstraction, "Ask Kenya" would be a federation of capability-specific searches: one for bills, one for acts, one for Hansard, one for committee reports, glued together with brittle orchestration. With the abstraction, the RAG pipeline retrieves from a uniform space of civic matters, ranks them by relevance to the question, and composes an answer from across them. The abstraction is what makes the north star tractable.

The abstraction is designed now, even though only the `Bill` subtype is implemented in Phase 3. This is deliberate: designing the abstraction after building five subtype-specific implementations would require a painful refactor; designing it first means each new subtype is a clean addition. The cost of designing the abstraction up front is a small amount of premature generality in the `Bill` implementation; the cost of not designing it is a multi-quarter refactor in Phase 6. We pay the small cost.

## Why the Bill Summarizer is the wedge, not the product

The Bill Summarizer is the smallest useful product that exercises the full pipeline (ingestion → documents → evidence → intelligence → search → API → web) and delivers real value to a citizen. It is the wedge — the thin end of the platform — not the product itself.

Why bills, and not acts or regulations or Hansard? Three reasons:

1. **Bills are legible.** A bill has a clear lifecycle (introduced → debated → passed → assented), a clear structure (clauses, schedules, a memorandum of objects), and a clear question for the citizen ("what does this bill do, and will it affect me?"). Acts are legible too, but they are also settled — a citizen's question about an act is "what does the law say?" which is more reference-like than explanation-like. Bills are alive; they are happening now; the citizen can affect their passage.
2. **Bills have a clear value proposition.** "What does this bill do, in plain language, with the source?" is a question a citizen has today and cannot easily answer. The Kenya parliament website publishes bills as PDFs; reading them requires legal training and patience. The platform collapses that effort into a summary with citations. The value is immediate and obvious.
3. **Bills exercise the full pipeline.** A bill must be fetched (ingestion), parsed (documents), cited (evidence), explained (intelligence), searched (search), served (API), and rendered (web). The Bill Summarizer product touches every service and every boundary. Shipping it proves the architecture end-to-end; if it works, the platform works.

The Bill Summarizer is not the product because the platform is bigger than bills. The product is "Ask Kenya" — a citizen's ability to ask any question about civic life and get an evidence-backed answer. The Bill Summarizer is the first concrete instance of that product; the `Ask` capability is the general instance. The wedge proves the general instance is possible; the general instance is the platform's reason to exist.

## The cross-domain intelligence map

The eventual product is a cross-domain intelligence map: the platform can answer questions that span Bills, Acts, Regulations, Hansard, Committees, Policies, Government Decisions, and (eventually) Judgments. A citizen asking "how did this bill become law?" gets a timeline that draws from the bill (introduced, debated), the Hansard (what was said in debate), the committee report (what was recommended), the order paper (when it was scheduled), the gazette (when it was assented and commenced), and the act (the final text). Each piece is a citation; the answer is a composed explanation.

This cross-domain intelligence is what makes the platform qualitatively different from a single-source civic portal. A parliament website shows bills; a gazette portal shows notices; a Hansard archive shows speeches. None of them show the connections. The platform's value is the connections: the ability to say "this speech in Hansard is about this clause in this bill, which became this section in this act, which was amended by this regulation." The connections are the intelligence.

The map is built incrementally. Phase 3 delivers the bill-level map (bill → timeline → versions → summary). Phase 6 delivers the bill-act-regulation-hansard map. Phase 8 delivers the cross-country map. Each phase adds a new source type to the `CivicMatter` abstraction and a new capability to the AI catalog; the user experience evolves from "explain this bill" to "answer any question about civic life."

## Who the platform is for (and not for)

The platform is for citizens first. Every design decision optimizes for the citizen's experience: the homepage is simple, the bill page is plain-language, the search is forgiving, the answer is cited. The platform is also for students, journalists, researchers, and developers — but the citizen is the primary audience, and the others are well-served by a citizen-first design (a journalist wants the same plain-language summary a citizen wants, plus the citations to dig deeper).

The platform is not for politicians, political parties, lobbyists, or government press offices. They are welcome to use it, but the platform does not optimize for them. The platform does not help a politician draft a bill; it does not help a party track its members' votes; it does not help a lobbyist identify amendment opportunities. These are not anti-features — they are simply not the platform's purpose. Other tools can serve those audiences; the platform serves the citizen.

## The long-term bet

The long-term bet is that citizens, given trustworthy explanations of what their government is doing, will engage more constructively with civic life. Not more partisan, not more polarized, but more informed. The platform does not assume citizens will agree on what the government should do; it assumes that an informed disagreement is better than an uninformed one. The bet is that evidence-backed explanations, with citations that anyone can verify, raise the floor of civic discourse.

This bet is unprovable in advance. The platform's success metrics are not "did citizens change their votes" (we do not measure that) but "did citizens use the platform, did they return, did they follow bills, did they ask questions, did they click citations." Engagement is the proxy for trust; trust is the proxy for the bet. If the platform is widely used and trusted, the bet is paying off, even if we cannot measure the downstream civic effect.

The platform's exit condition — the state at which we would say "we have built the thing we set out to build" — is "Ask [Country]" working across multiple countries, with citizens using it regularly, with the platform cited in newsrooms and classrooms as a trusted source. That is the north star, and every architectural choice in this repo is in service of it.
