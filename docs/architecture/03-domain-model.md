# 03 — Domain Model

This document describes the canonical civic domain model. Every entity below lives in `services/legislation/` (or, for evidence and AI outputs, in `services/evidence/` and `services/intelligence/` respectively) and is backed by a table in the `legislation` (or `evidence`, `intelligence`) schema in the shared PostgreSQL cluster. The domain is country-agnostic: a `Bill` has the same shape in Kenya, Uganda, or Ghana. Country-specific concerns (sources, stages, terminology) live in the adapter, not the domain.

The long-term abstraction that ties these together is the **CivicMatter** — a generalized type that encompasses `Bill`, `Act`, `Regulation`, `Policy`, `GovernmentDecision`, and eventually `Judgment`. See [§CivicMatter](#civicmatter-the-long-term-abstraction) at the end of this document and [`docs/architecture/14-product-vision.md`](./14-product-vision.md).

## Legislative structure

### Country

A `Country` is the top-level scope of the platform. Every other entity belongs to exactly one country (or to the platform globally, for cross-country reference data). The country record carries ISO codes, the default language, the default currency, and a pointer to its adapter module. Adding a country to the platform is a `Country` row plus an `adapters/<country>/` directory; nothing in the domain code changes.

### Institution

An `Institution` is any organization that participates in civic life: a parliament, a senate, a county assembly, a ministry, a regulatory agency, a constitutional commission, a court. Institutions are the actors that introduce, debate, pass, gazette, and enforce civic matters. An institution has a type (legislative, executive, judicial, regulatory), a country, and a parent institution (optional — a ministry reports to a presidency; a committee reports to a house).

### Legislature

A `Legislature` is a special kind of institution: one that legislates. It owns the legislative calendar, the houses it contains, and the bills under its consideration. In Kenya, the Parliament of Kenya is a `Legislature` containing two houses: the National Assembly and the Senate. A `Legislature` may have a single house (unicameral) or multiple (bicameral); the domain does not assume either.

### House

A `House` is a chamber within a `Legislature`. Bills are introduced, debated, and voted on in houses. A bill in a bicameral legislature passes through both houses (the order is country-specific and defined in the adapter). A `House` has members (People), committees, and a presiding officer. The Kenya insight — *the Senate is just another House* — is what lets the domain treat the National Assembly and the Senate uniformly: both are houses of the Parliament of Kenya, and a bill's stage transitions through each are recorded the same way.

### Committee

A `Committee` is a subgroup of a `House` (or occasionally of a `Legislature` directly) that takes detailed consideration of bills, conduct oversight, or investigate specific matters. A committee has a chair, members, a remit, and a calendar. Bills are typically committed to a committee after second reading for clause-by-clause consideration. Committee reports are first-class documents (see `CommitteeReport`).

### Person

A `Person` is a human being who appears in the civic record: a member of parliament, a senator, a cabinet secretary, a committee chair, a presiding officer, a petitioner. A person has a name, identifiers (country-specific — Kenya uses the parliament member ID), a current affiliation (PoliticalParty), and a history of affiliations. People introduce bills, vote on bills, sit on committees, and speak in Hansard.

### PoliticalParty

A `PoliticalParty` is an organization that fields candidates for political office. A party has a name, an abbreviation, a founding date, and a current status (active, deregistered, merged). A person's party affiliation is recorded as a time-ranged relationship (a person can switch parties); the current affiliation is the latest record. Parties matter for understanding vote patterns and committee composition.

### Constituency

A `Constituency` is an electoral geographic unit. In Kenya, members of the National Assembly are elected from single-member constituencies; senators and women representatives are elected from counties. A constituency has a name, a boundary (recorded as GeoJSON, optional), a current representative (Person), and an election history.

### County

A `County` is a sub-national administrative unit. In Kenya, the 47 counties are the units of devolved government; each has a governor, a county assembly, and a budget. Counties are the devolved counterpart to the national government; the domain supports county-level legislation (county bills, county acts) the same way it supports national legislation.

## Legislative content

### Bill

A `Bill` is a proposed law under consideration by a `Legislature`. It has a title, a long title (the descriptive sentence stating the bill's purpose), a bill number (country-specific), a sponsor (Person), a type (government bill, private member's bill, county bill), a house of origin, the current stage, and a list of versions. A bill is the canonical record; its content lives in `BillVersion`s.

### BillVersion

A `BillVersion` is an immutable snapshot of a bill's text at a point in time. Bills evolve — they are amended in committee, recommitted, reprinted — and each significant revision is a new version. Versions are immutable: once published, they never change. This is the core invariant that lets the platform show "what the bill said on day X" and detect contradictions between versions. See [ADR-0011](../adr/ADR-0011-immutable-bill-versions.md).

### BillStage

A `BillStage` is a named step in a bill's lifecycle: `FirstReading`, `SecondReading`, `CommitteeOfTheWholeHouse`, `ThirdReading`, `PresidentialAssent`, `Commencement`. The set of stages is country-specific (defined in the adapter); the transition rules (which stage can follow which) are domain rules. The current stage of a bill is the latest `BillStageTransition` recorded against it.

### BillEvent

A `BillEvent` is a dated occurrence in a bill's lifecycle: introduced on date X, debated on date Y, voted on date Z, assented on date W. Events are the timeline of the bill; they are immutable once recorded (they are historical facts). A `BillEvent` has a date, an event type, a description, and a link to the source document (OrderPaper, Hansard, Gazette) that records it.

### Amendment

An `Amendment` is a proposed change to a bill's text, tabled by a person and considered in committee. An amendment targets a specific clause (or schedule) of a specific bill version, proposes the new text (or deletion), and has a status (proposed, accepted, rejected, withdrawn). Amendments are first-class because the diff between versions is built from accepted amendments.

### Clause

A `Clause` is a numbered section of a bill's text. Bills are structured into clauses (and schedules, which are themselves structured into paragraphs). Clauses are the unit of amendment — an amendment targets a clause — and the unit of citation — an evidence citation can resolve to a specific clause. Clause structure is preserved across versions so clause 14 of version 1 and clause 14 of version 2 are comparable.

### Act

An `Act` is a bill that has completed the legislative process and been assented to. An Act has a chapter number (country-specific — Kenya uses Cap. numbers), a commencement date (which may differ from the assent date), and a list of versions (Acts are amended over time, producing new consolidated versions). An Act is a `CivicMatter` that has graduated from Bill status.

### Regulation

A `Regulation` is a subordinate instrument issued under the authority of an Act. Regulations are made by ministries or regulators, not by the legislature, but they have legal force. Regulations cite the parent Act and the specific section that authorizes them. They are gazetted (see `GazetteNotice`) and may be challenged in court.

### Policy

A `Policy` is a non-binding government statement of intent: a sessional paper, a strategic plan, a cabinet memo. Policies do not have legal force but they explain government direction. The platform tracks policies because they often precede bills and explain legislative intent — a citizen asking "why is this bill being introduced?" may find the answer in a policy document.

### GovernmentDecision

A `GovernmentDecision` is a recorded decision by the executive: a cabinet decision, a presidential directive, a ministerial ruling. Decisions are tracked because they often have civic consequences (a decision to allocate budget, to deploy a service, to appoint an official) even when they are not legislative.

### HansardDocument

A `HansardDocument` is the official verbatim record of proceedings in a house. Hansard is the source of truth for what was said in parliament — by whom, on what topic, in response to what. It is the substrate for many AI capabilities (quote extraction, sentiment analysis, topic clustering) and for evidence (a claim that "the sponsor said X" cites a Hansard page).

### OrderPaper

An `OrderPaper` is the published agenda of a house for a sitting day. It lists the bills to be considered, the questions to be asked, the motions to be moved. Order papers are the source of truth for the schedule of legislative activity; they are cited by `BillEvent`s ("the bill was listed for second reading on the order paper of date X").

### CommitteeReport

A `CommitteeReport` is a published document by a committee, typically after considering a bill or conducting an inquiry. Committee reports are first-class documents because they often contain the substantive reasoning behind amendments and the committee's recommendations. They are cited by amendments ("as recommended in the committee report on bill X") and by AI capabilities (CommitteeReportSummarizer).

### Vote

A `Vote` is a recorded vote in a house: ayes, nays, abstentions, absent. A vote is taken on a motion (a bill's second reading, an amendment, a committee recommendation) and recorded per member. The aggregate vote determines the outcome; the per-member vote (`VotesProceeding`) is the substrate for accountability analysis.

### VotesProceeding

A `VotesProceeding` is a single member's vote in a single `Vote`: aye, nay, abstain, absent. VotesProceedings are the per-member granularity that lets the platform answer "how did my MP vote on bill X?" and "how often does this MP vote with their party?"

### GazetteNotice

A `GazetteNotice` is an official publication in the country's gazette. Gazette notices cover a huge range: acts assented, regulations issued, appointments made, dates of commencement, public appointments, legal notices. They are the legal-publication substrate; many canonical events (assent, commencement, regulation issuance) cite a gazette notice.

### PublicParticipation

A `PublicParticipation` is a recorded instance of citizen input into the legislative process: a submission on a bill, a hearing appearance, a memorandum. Public participation is constitutionally required in Kenya for certain bills; the platform tracks it because it is the formal channel for citizen input that the platform itself is not.

## Provenance and AI outputs

### Source

A `Source` is a registered external origin of civic information: the Kenya parliament website, the Kenya Gazette portal, a county assembly website. A source has a URL, a fetch policy (cadence, robots respect), an adapter module, and a trust level (authoritative, official, third-party). Every fetched document references a source. See `services/ingestion/`.

### SourceDocument

A `SourceDocument` is a specific fetched artifact: a PDF of bill number X, an HTML page of the order paper for date Y, a JSON payload from an API. It has a content hash, a fetch timestamp, the source it came from, and a pointer to the raw bytes in blob storage. A `SourceDocument` is the bridge between the external world and the platform.

### SourceSnapshot

A `SourceSnapshot` is a point-in-time capture of a source page. Sources change — a bill's page is updated as it moves through stages; a gazette page may be corrected — and snapshots let the platform detect changes (by comparing snapshot hashes), establish provenance (this fact was true as of this snapshot), and roll back if a fetch was wrong.

### Citation

A `Citation` is a resolved pointer from a claim to the specific page/section of a specific `SourceDocument` that supports it. A citation is verified at write time (the cited page/section exists and contains the cited text) and re-verified periodically (in case documents are re-parsed). A claim with zero verified citations is rejected.

### Evidence

`Evidence` is the entity that ties a `Claim` to one or more `Citation`s. A claim may have multiple pieces of evidence (a bill's title is supported by the bill document, the order paper, and the gazette notice — all three). Evidence has a strength score (primary source vs. secondary, direct vs. inferred) and is the unit the contradiction engine compares.

### Claim

A `Claim` is a single factual assertion in the canonical domain: "Bill X was introduced on date Y," "Senator Z voted aye on bill W," "Act A commenced on date B." Claims are the atoms of the domain; every canonical entity is essentially a structured set of claims. Claims have evidence; claims without evidence are rejected.

### AIExplanation

An `AIExplanation` is a generated plain-language explanation attached to a canonical entity: the summary on a bill page, the explanation of a vote, the plain-language version of a clause. An AIExplanation is never shown without its citations; it is versioned (a new explanation supersedes an old one, but the old one is retained for audit). It is a projection, not a canonical fact — it can be regenerated without changing the underlying entity.

### AIClaim

An `AIClaim` is a factual assertion generated by the AI: "this bill will increase taxes on X," "this amendment contradicts clause 14 of the prior version." AIClaims are candidate facts until validated; they live in `intelligence.candidate_facts` with a status. When validated, they may become `Claim`s (if they assert a canonical fact) or remain `AIClaim`s (if they are interpretive).

### AIResponse

An `AIResponse` is a complete AI output returned to a user (or stored for later): the body of a bill summary, the answer to a citizen question, the result of a capability call. An `AIResponse` contains one or more `AIClaim`s, each with its citations, and metadata about the model, prompt version, and eval result. It is the unit of AI output.

## CivicMatter: the long-term abstraction

The long-term north star is "Ask Kenya," and the abstraction that enables it is the `CivicMatter`: a generalized type that encompasses `Bill`, `Act`, `Regulation`, `Policy`, `GovernmentDecision`, and eventually `Judgment`. A `CivicMatter` has a title, a status, a set of versions, a set of events, a set of citations, and a set of AI explanations. The concrete subtypes add their own fields (a `Bill` has a sponsor and a house; a `Regulation` has a parent Act; a `Judgment` has a court and parties), but the shared shape lets the search and AI capabilities treat them uniformly.

Phase 3 of the roadmap (Bill Intelligence) treats only `Bill` as a `CivicMatter`. Phase 6 (Civic Intelligence) adds `Act`, `Regulation`, and `GazetteNotice`. Phase 7 (Research Platform) adds `HansardDocument`, `CommitteeReport`, and `Policy`. Phase 8 (Global Expansion) adds `Judgment` (court decisions) and the cross-country `CivicMatter` view. The abstraction is designed now, even though only the `Bill` subtype is implemented; this avoids a painful refactor later. See [`docs/architecture/14-product-vision.md`](./14-product-vision.md).
