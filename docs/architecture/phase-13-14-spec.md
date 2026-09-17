The next logical step is to move from a **Civic Knowledge Network** into a true **Civic Intelligence Operating System**: a platform that can continuously reason over time, institutions, laws, evidence, events, and citizen questions while preserving provenance and historical correctness.

# Phase 13 — Civic Knowledge Network & Institutional Intelligence

## Mission

Transform the global civic data platform into a continuously evolving, machine-readable **Civic Knowledge Network** connecting:

* Institutions
* Jurisdictions
* People
* Political parties
* Bills
* Amendments
* Acts
* Regulations
* Policies
* Court decisions
* Committee reports
* Parliamentary proceedings
* Government decisions
* Documents
* Evidence
* Claims
* Events
* Topics
* Organizations
* Cross-border institutions
* International agreements

The platform must answer not only:

> “What happened?”

but also:

> “How is this connected?”

> “What changed over time?”

> “What did this replace?”

> “What did this lead to?”

> “Which institutions were involved?”

> “What evidence supports this relationship?”

> “What was true at a particular point in time?”

The core principle is:

> **Every piece of civic knowledge must have provenance, temporal validity, and an explainable relationship to authoritative evidence.**

Do not turn the system into an opaque AI knowledge base.

AI explains the knowledge network.

The knowledge network does not become true because AI said it was true.

---

# 1. Discovery & Baseline

Before implementing anything, perform a complete repository and architecture discovery.

Inspect:

* Existing architecture
* Domain services
* Database schemas
* Migrations
* Civic entities
* Evidence model
* Trust model
* Provenance
* Global jurisdiction model
* Country adapters
* Ingestion
* Documents
* Search
* Graph capabilities
* AI/RAG
* Research workspace
* APIs
* Events
* Temporal workflows
* NATS
* Notifications
* Developer platform
* UI
* Observability
* Security
* Existing tests
* Existing GitHub issues
* Existing PRs
* Documentation
* ADRs

Search the entire repository for:

* duplicated graph models
* relationship tables
* entity resolution
* provenance
* lineage
* temporal fields
* evidence references
* institution models
* person history
* research queries
* saved searches
* graph traversal
* AI-generated relationships
* hard-coded jurisdiction assumptions

Do not create duplicate architecture where an existing implementation can be extended safely.

Review all existing GitHub issues before creating new ones.

Consolidate overlapping work.

Create new issues only for genuinely missing capabilities.

No direct pushes to main.

Every implementation must follow:

**Issue → implementation branch → PR → tests → review → merge → issue closure**

Never close an issue before its PR is merged and validated.

---

# 2. Architectural Objective

Create the following logical architecture:

```text
                 CIVIC KNOWLEDGE NETWORK
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   Knowledge Objects   Relationships      Temporal History
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                    Provenance Layer
                           │
                    Evidence Network
                           │
              ┌────────────┼────────────┐
              │            │            │
           Search        Graph       Research
              │            │            │
              └────────────┼────────────┘
                           │
                  Intelligence Layer
                           │
                    Civic Orchestrator
                           │
                    Citizen / Researcher
```

The knowledge layer must remain independent from the AI layer.

---

# 3. Knowledge Object Model

Create a generalized first-class `KnowledgeObject`.

Potential types:

```text
COUNTRY
JURISDICTION
INSTITUTION
ORGANIZATION
PERSON
PARTY
OFFICE
COMMITTEE
BILL
AMENDMENT
ACT
REGULATION
POLICY
GOVERNMENT_DECISION
COURT_DECISION
MOTION
PETITION
COMMITTEE_REPORT
HANSARD
ORDER_PAPER
VOTE
DOCUMENT
SOURCE
CLAIM
EVIDENCE
EVENT
TOPIC
AGREEMENT
PROGRAM
PROJECT
```

Each object should support:

* stable ID
* object type
* jurisdiction
* institution
* external identifier
* name
* aliases
* description
* source references
* verification state
* created_at
* observed_at
* valid_from
* valid_to
* superseded_by
* metadata

Do not force every object type to share meaningless fields.

Use strongly typed domain structures where appropriate.

---

# 4. Temporal Civic Knowledge

Make time a first-class architectural property.

The system must distinguish:

```text
occurred_at
observed_at
published_at
effective_from
effective_to
valid_from
valid_to
superseded_at
```

Support historical reconstruction.

A query such as:

> “What was the status of this Bill on 12 March 2024?”

must not simply return today's status.

It must reconstruct the state applicable at that time.

Implement temporal queries for:

* institutions
* people
* roles
* bills
* legislation
* regulations
* policies
* relationships
* source records
* claims
* evidence

---

# 5. Institutional Intelligence

Create first-class institutional profiles.

An institution profile should expose:

* identity
* jurisdiction
* government level
* mandate
* responsibilities
* parent institution
* subordinate institutions
* committees
* offices
* official sources
* documents
* matters handled
* legislation associated with it
* regulations
* policies
* proceedings
* historical changes
* leadership/role relationships where authoritative data exists
* source health
* coverage
* verification status

Example:

```text
Institution
 ├── Mandate
 ├── Jurisdiction
 ├── Parent
 ├── Offices
 ├── Committees
 ├── People
 ├── Documents
 ├── Bills
 ├── Acts
 ├── Regulations
 ├── Events
 ├── Sources
 └── Historical Timeline
```

Do not infer institutional responsibilities from an institution's name.

Use authoritative evidence.

---

# 6. Person & Role History

Model people independently from their roles.

A person may have:

```text
Person
   ↓
Role
   ↓
Institution
   ↓
Jurisdiction
   ↓
valid_from / valid_to
```

Support:

* role history
* institution history
* committee membership
* office membership
* party affiliation where officially sourced
* appointment events
* resignation events
* election/appointment records where available
* document participation
* speeches/proceedings where available

Never overwrite historical roles.

Use temporal records.

---

# 7. Legal Lineage

Build a machine-readable legal lineage graph.

Examples:

```text
Bill
 ↓
Bill Version
 ↓
Amendment
 ↓
Passed Bill
 ↓
Assent
 ↓
Act
 ↓
Regulation
 ↓
Court Interpretation
```

Support relationships such as:

```text
AMENDS
REPLACES
SUPERSEDES
IMPLEMENTS
REFERENCES
DERIVES_FROM
INTERPRETS
CHALLENGES
INVALIDATES
EXTENDS
EXPIRES
```

Every relationship requires provenance.

Do not create a legal relationship simply because two documents mention similar terminology.

---

# 8. Matter Genealogy

Create a generalized `CivicMatter` relationship network.

A matter can connect:

```text
Bill
Policy
Regulation
Committee Report
Hansard
Government Decision
Court Decision
Public Participation
Petition
Act
```

Example:

```text
Housing Topic
      │
      ├── Government Policy
      │
      ├── Bill
      │     ├── Amendment
      │     └── Committee Report
      │
      ├── Act
      │     └── Regulation
      │
      └── Court Decision
```

The platform must be able to trace this chain.

---

# 9. Relationship Model

Create first-class relationships.

Each relationship should include:

```text
relationship_id
source_object
target_object
relationship_type
jurisdiction
source
evidence
confidence
verification_state
created_at
valid_from
valid_to
superseded_at
created_by
```

Examples:

```text
BILL → AMENDS → ACT
BILL → INTRODUCED_BY → PERSON
BILL → CONSIDERED_BY → COMMITTEE
ACT → IMPLEMENTED_BY → REGULATOR
REGULATION → DERIVED_FROM → ACT
COURT_DECISION → INTERPRETS → ACT
DOCUMENT → SUPPORTS → CLAIM
CLAIM → SUPPORTED_BY → EVIDENCE
PERSON → HELD_ROLE → INSTITUTION
INSTITUTION → RESPONSIBLE_FOR → POLICY
```

---

# 10. Relationship Trust

Do not allow AI-generated relationships to become indistinguishable from verified relationships.

Relationship states:

```text
DISCOVERED
EXTRACTED
VALIDATING
VERIFIED
CONFLICTED
CORRECTED
SUPERSEDED
REJECTED
```

Relationship origin:

```text
AUTHORITATIVE
EXTRACTED
DERIVED
AI_SUGGESTED
HUMAN_REVIEWED
```

AI suggestions must enter a validation workflow.

---

# 11. Evidence Graph

Extend the existing evidence architecture into an evidence graph.

Example:

```text
Claim
 ↓
Citation
 ↓
Evidence
 ↓
Document Snapshot
 ↓
Source
 ↓
Institution
 ↓
Jurisdiction
```

A relationship should be explainable through the same chain.

Example:

```text
Bill X
   ↓
AMENDS
   ↓
Act Y
   ↓
Evidence
   ↓
Official Document
   ↓
Page 14
```

Users must be able to inspect this chain.

---

# 12. Provenance Explorer

Create a dedicated provenance experience.

For any knowledge object:

```text
Object
  ↓
Facts
  ↓
Relationships
  ↓
Evidence
  ↓
Documents
  ↓
Snapshots
  ↓
Official Sources
```

Allow users to answer:

> Where did this information come from?

> Which document supports this?

> Was this relationship extracted or verified?

> Was the source later replaced?

> What was known at the time?

---

# 13. Civic Timeline Engine

Create a deterministic timeline engine.

Timeline events must distinguish:

```text
EVENT_OCCURRED
EVENT_PUBLISHED
EVENT_OBSERVED
EVENT_INFERRED
```

Never present inferred events as historical facts.

Timeline generation must be deterministic from canonical events.

AI may provide explanations such as:

> “This amendment changed the provision concerning…”

but the underlying event must come from canonical data.

---

# 14. Topic Intelligence

Create first-class global topics.

Examples:

```text
Housing
Healthcare
Education
Agriculture
Employment
Taxation
Transport
AI
Digital Services
Environment
Energy
Trade
Finance
Infrastructure
```

Topics should connect to:

* bills
* acts
* regulations
* policies
* institutions
* court decisions
* events
* research
* jurisdictions

Support hierarchical topics:

```text
Digital Government
 ├── Digital Identity
 ├── Digital Payments
 ├── Data Protection
 ├── AI Regulation
 └── Cybersecurity
```

Topic assignments must be explainable and versioned.

---

# 15. Cross-Domain Intelligence

Enable queries such as:

> “Show everything connected to this Bill.”

> “What Act resulted from this Bill?”

> “Which regulations implement this Act?”

> “Which institutions are responsible for implementing it?”

> “Which court decisions have interpreted it?”

> “Which committee reports discussed it?”

> “What changed between the original proposal and final Act?”

> “Which official documents support each part of this timeline?”

---

# 16. Temporal Graph Queries

Implement graph traversal with temporal filters.

Example:

```text
Find:
Bill X
  → amendments
  → final Act
  → implementing regulations
  → court interpretations

WHERE:
valid_at = 2025-01-01
jurisdiction = KE
verification_state = VERIFIED
```

The graph must not return relationships that were not valid during the requested period.

---

# 17. Graph Storage Strategy

Do not introduce a graph database simply because the product now has a graph.

First evaluate:

* PostgreSQL relational representation
* recursive queries
* materialized graph projections
* pgvector
* search indexes

Measure actual requirements.

Introduce a dedicated graph database only if evidence demonstrates that PostgreSQL cannot satisfy:

* traversal depth
* latency
* relationship volume
* concurrency
* operational requirements
* query complexity

If a graph database is introduced, PostgreSQL remains the canonical transactional source unless architecture evidence justifies otherwise.

Never create two competing sources of truth.

---

# 18. Research Operating System

Expand Research Workspace into a full civic research environment.

Research Project:

```text
Research Project
 ├── Questions
 ├── Searches
 ├── Documents
 ├── Evidence
 ├── Claims
 ├── Relationships
 ├── Timelines
 ├── Comparisons
 ├── Notes
 ├── AI analyses
 ├── Saved queries
 ├── Dataset versions
 ├── Research runs
 └── Exports
```

Support:

* save evidence
* annotate evidence
* build timelines
* compare documents
* construct relationship maps
* save searches
* create claim collections
* export citations
* reproduce research

---

# 19. Reproducible Research

Every research run should record:

```text
query
filters
jurisdictions
date range
retrieval configuration
dataset versions
document snapshots
evidence IDs
claim IDs
AI model
prompt version
retrieval configuration
generated_at
```

A researcher should be able to rerun a historical analysis and determine:

```text
same result
changed source
changed document
changed evidence
changed dataset
changed model
changed prompt
```

Never silently overwrite historical research results.

---

# 20. Knowledge APIs

Create APIs for the knowledge network.

Examples:

```http
GET /api/v1/knowledge/{type}/{id}

GET /api/v1/knowledge/{type}/{id}/relationships

GET /api/v1/knowledge/{type}/{id}/timeline

GET /api/v1/knowledge/{type}/{id}/lineage

GET /api/v1/knowledge/{type}/{id}/evidence

GET /api/v1/knowledge/{type}/{id}/history

POST /api/v1/knowledge/traverse

POST /api/v1/knowledge/query

POST /api/v1/research/run
```

API responses must preserve:

* provenance
* verification
* temporal validity
* jurisdiction
* source
* evidence
* confidence

---

# 21. AI + Knowledge Graph

Upgrade RAG into graph-aware retrieval.

Pipeline:

```text
Question
 ↓
Intent Detection
 ↓
Jurisdiction Detection
 ↓
Entity Resolution
 ↓
Knowledge Graph Retrieval
 ↓
Keyword Retrieval
 ↓
Vector Retrieval
 ↓
Temporal Filtering
 ↓
Evidence Retrieval
 ↓
Reranking
 ↓
Claim Construction
 ↓
Citation Validation
 ↓
Answer
```

AI must not invent graph edges.

If an edge is unavailable:

```text
UNKNOWN
```

not:

```text
PROBABLY RELATED
```

unless clearly labelled as inference.

---

# 22. “What Was True?” Engine

Build a dedicated historical truth query capability.

Examples:

> “What was the status of this Bill on a specific date?”

> “Who held this office in 2022?”

> “Which regulation was in force at this time?”

> “Which version of the Bill existed before the amendment?”

> “What did the official source say before it was updated?”

Every answer must identify:

* temporal scope
* source snapshot
* evidence
* verification state
* uncertainty

---

# 23. Institutional Change Detection

Track institutional evolution.

Detect:

* renamed institutions
* merged institutions
* dissolved institutions
* reorganized agencies
* changed mandates
* changed jurisdictions
* new offices
* abolished offices
* transferred responsibilities

Represent these as explicit events.

Never interpret organizational changes without evidence.

---

# 24. Knowledge Corrections

Extend the trust correction workflow.

Correction flow:

```text
Detected Issue
 ↓
Review
 ↓
Evidence Verification
 ↓
Relationship/Facts Decision
 ↓
Correction
 ↓
Historical Record Preserved
 ↓
Indexes Updated
 ↓
AI Outputs Revalidated
```

When a knowledge relationship changes, identify dependent:

* claims
* summaries
* timelines
* research runs
* search results
* notifications
* AI outputs

Invalidate or regenerate dependent outputs when required.

---

# 25. Knowledge Dependency Graph

Create dependency tracking.

Example:

```text
Official Document
       ↓
Extracted Fact
       ↓
Bill Version
       ↓
Relationship
       ↓
Timeline
       ↓
AI Summary
       ↓
Notification
```

If the document is corrected:

```text
Document correction
       ↓
affected facts
       ↓
affected relationships
       ↓
affected AI outputs
       ↓
affected notifications
```

This prevents stale intelligence.

---

# 26. Knowledge Quality Engine

Measure:

* object completeness
* relationship completeness
* provenance coverage
* evidence coverage
* temporal coverage
* entity resolution quality
* duplicate rate
* contradiction rate
* stale relationship rate
* unsupported relationship rate
* source authority
* source freshness

Create a Knowledge Quality Scorecard internally.

Do not present an unexplained numerical “truth score” to citizens.

---

# 27. Global Civic Graph

Expand the graph to support:

```text
Country
Jurisdiction
Institution
Person
Party
Bill
Act
Regulation
Policy
Court Decision
Committee
Document
Source
Claim
Evidence
Event
Topic
International Organization
Agreement
```

Support:

* national relationships
* county/state relationships
* cross-border relationships
* regional organizations
* international agreements
* shared policy frameworks

Examples:

```text
Kenya
 ↓
East African Community
 ↓
Regional Agreement
 ↓
National Implementation
 ↓
Kenyan Act
```

Every cross-border relationship must have explicit provenance.

---

# 28. Global Research Queries

Support questions such as:

> “Compare how different jurisdictions approached AI regulation.”

> “Trace digital tax legislation across several countries.”

> “Find institutions responsible for data protection across selected jurisdictions.”

> “Show how a regional agreement was implemented nationally.”

Comparison must remain descriptive.

Do not produce unsupported political judgments or rankings.

---

# 29. UI / UX

Create a modern knowledge-driven experience.

Primary navigation:

```text
Logo
Ask
Explore
Topics
Countries
Research
Search
Notifications
Profile
```

Avoid overcrowded navigation.

---

## Knowledge Object Page

Example:

```text
Bill X

Status
Current stage

What happened?
Plain-language explanation

Timeline
────────────────

Connected matters
Bill → Amendment → Act → Regulation

Institutions
────────────────

People / roles
────────────────

Evidence
────────────────

Documents
────────────────

Provenance
────────────────
```

---

# 30. Relationship Explorer

Build an interactive relationship explorer.

Users should be able to select:

```text
Bill
Institution
Person
Act
Regulation
Court Decision
Policy
Topic
```

and see connected objects.

Important:

Do not create a visually impressive graph that becomes unreadable.

Provide:

* filters
* relationship types
* date ranges
* jurisdictions
* verification state
* evidence availability
* expand/collapse
* search within graph

---

# 31. Institutional Page

Institution page:

```text
Institution
Mandate

Jurisdiction

What it does

Current structure

People / roles

Bills

Acts

Regulations

Policies

Proceedings

Documents

Sources

Historical changes

Evidence

Coverage
```

---

# 32. Legal Lineage UI

Create:

```text
Original Bill
      ↓
Version 1
      ↓
Amendments
      ↓
Version 2
      ↓
Passed
      ↓
Act
      ↓
Regulations
      ↓
Court Decisions
```

Allow users to inspect the evidence supporting each transition.

---

# 33. Research Workspace UX

Research workspace should support:

* split-pane research
* saved evidence
* citations
* timeline builder
* graph explorer
* notes
* comparison
* AI assistance
* reproducibility
* exports

AI assistance must never obscure source material.

---

# 34. Developer Ecosystem

Expose knowledge-network APIs through the developer platform.

Add products:

```text
Knowledge API
Relationship API
Timeline API
Lineage API
Institution API
Research API
Graph Query API
Evidence API
```

Every response must preserve provenance.

---

# 35. Event Model

Introduce knowledge events:

```text
knowledge.object.created
knowledge.object.updated
knowledge.relationship.created
knowledge.relationship.verified
knowledge.relationship.invalidated
knowledge.relationship.corrected
knowledge.timeline.updated
knowledge.entity.merged
knowledge.entity.unmerged
knowledge.provenance.updated
knowledge.dependency.invalidated
research.run.created
research.run.completed
research.run.invalidated
```

All events require:

* event ID
* schema version
* jurisdiction
* object ID
* occurred_at
* observed_at
* provenance
* correlation ID

Maintain at-least-once delivery and idempotent consumers.

---

# 36. Security

Threat-model:

* relationship poisoning
* fake entity relationships
* source poisoning
* graph traversal abuse
* unauthorized research access
* cross-tenant graph leakage
* API abuse
* export abuse
* sensitive metadata exposure
* prompt injection
* indirect prompt injection
* malicious documents
* SSRF
* authorization bypass
* IDOR
* graph query denial of service

Never allow arbitrary graph traversal to bypass authorization.

---

# 37. Performance

Establish measured targets.

Benchmark:

* entity lookup
* relationship lookup
* 1-hop traversal
* 3-hop traversal
* deep traversal
* temporal queries
* provenance lookup
* evidence lookup
* institution pages
* legal lineage
* research queries
* cross-jurisdiction search

Test:

```text
100 concurrent users
500
1,000
5,000
10,000+
```

Measure:

* p50
* p95
* p99
* throughput
* CPU
* memory
* DB load
* cache hit rate
* graph query cost
* search latency

Do not invent performance claims.

---

# 38. Resilience

Test:

* graph projection failure
* database failure
* NATS restart
* Temporal worker failure
* search outage
* AI provider failure
* source outage
* evidence service outage
* cache outage
* object storage outage

The core civic truth layer must remain usable when AI or graph projections fail.

---

# 39. Consistency & Reconciliation

Build reconciliation between:

```text
PostgreSQL
NATS
Temporal
Search
Graph projection
Redis
Object Storage
Evidence
AI outputs
Notifications
Research
```

Detect:

* missing relationships
* orphaned evidence
* stale projections
* missing events
* duplicate objects
* invalid graph edges
* stale AI outputs
* broken provenance chains

Provide repair workflows.

---

# 40. Agent Dispatch

Dispatch specialized engineering agents.

### Distinguished Architect

Own:

* knowledge architecture
* temporal model
* graph strategy
* service boundaries
* scalability
* compatibility

### Knowledge Graph Engineers

Own:

* object model
* relationships
* temporal graph
* traversal
* projections

### Civic Domain Engineers

Own:

* institutions
* legislation
* legal lineage
* matter genealogy
* procedures

### Trust Engineers

Own:

* provenance
* evidence
* verification
* correction
* relationship trust

### Data Engineers

Own:

* normalization
* entity resolution
* dependency tracking
* quality

### AI Engineers

Own:

* graph-aware RAG
* temporal retrieval
* reasoning
* claim extraction
* citation validation

### Research Platform Engineers

Own:

* research workspaces
* reproducibility
* evidence collections
* exports

### API Engineers

Own:

* knowledge APIs
* graph APIs
* research APIs
* developer contracts

### Security Engineers

Own:

* graph security
* authorization
* poisoning
* traversal abuse

### SRE

Own:

* reliability
* capacity
* observability
* disaster recovery

### UI/UX

Own:

* knowledge exploration
* institutional pages
* graph UX
* legal lineage
* research workspace

### QA

Own:

* golden knowledge datasets
* temporal correctness
* graph correctness
* provenance validation
* regression
* chaos
* load testing

---

# 41. GitHub Issue Plan

Create or reconcile issues around:

```text
CIVIC-1301 Knowledge architecture baseline
CIVIC-1302 KnowledgeObject model
CIVIC-1303 Temporal knowledge model
CIVIC-1304 Institution intelligence
CIVIC-1305 Person role history
CIVIC-1306 Legal lineage
CIVIC-1307 Matter genealogy
CIVIC-1308 Relationship model
CIVIC-1309 Relationship verification
CIVIC-1310 Evidence graph
CIVIC-1311 Provenance explorer
CIVIC-1312 Civic timeline engine
CIVIC-1313 Topic intelligence
CIVIC-1314 Cross-domain intelligence
CIVIC-1315 Temporal graph queries
CIVIC-1316 Graph storage evaluation
CIVIC-1317 Research OS
CIVIC-1318 Reproducible research
CIVIC-1319 Knowledge APIs
CIVIC-1320 Graph-aware AI retrieval
CIVIC-1321 Historical truth engine
CIVIC-1322 Institutional change detection
CIVIC-1323 Knowledge correction propagation
CIVIC-1324 Knowledge dependency graph
CIVIC-1325 Knowledge quality engine
CIVIC-1326 Global civic graph
CIVIC-1327 Cross-jurisdiction research
CIVIC-1328 Knowledge event model
CIVIC-1329 Knowledge security
CIVIC-1330 Knowledge performance
CIVIC-1331 Knowledge resilience
CIVIC-1332 Knowledge reconciliation
CIVIC-1333 Relationship explorer
CIVIC-1334 Institutional UX
CIVIC-1335 Legal lineage UX
CIVIC-1336 Research workspace UX
CIVIC-1337 Knowledge developer APIs
CIVIC-1338 Golden knowledge datasets
CIVIC-1339 Production validation
CIVIC-1340 Phase 13 production readiness
```

Before creating these, inspect existing issues and consolidate duplicates.

---

# 42. Critical End-to-End Validation

Build a golden scenario around a real civic matter.

Example:

```text
Official Bill
 ↓
Bill Version
 ↓
Amendment
 ↓
Committee Report
 ↓
Proceeding
 ↓
Passed Bill
 ↓
Act
 ↓
Regulation
 ↓
Court Decision
```

Verify:

* every object exists
* every relationship has provenance
* every relationship has temporal validity
* every document is immutable
* every claim is evidence-backed
* every transition is explainable
* every correction propagates
* every projection is eventually consistent
* historical queries return historically correct state
* AI cannot invent missing relationships

---

# 43. Historical Reconstruction Test

Select a matter with multiple changes.

Ask:

```text
What was known at T1?
What was known at T2?
What changed between T1 and T2?
Which source caused the change?
Which version was valid at each point?
```

The answers must be reproducible from stored snapshots and events.

---

# 44. Dependency Invalidation Test

Simulate:

```text
Official source corrected
 ↓
Document snapshot updated
 ↓
Fact corrected
 ↓
Relationship invalidated
 ↓
Timeline updated
 ↓
AI output invalidated
 ↓
Search projection updated
 ↓
Affected notification identified
```

No stale information may remain silently published.

---

# 45. Research Reproducibility Test

Researcher:

```text
Search
 ↓
Collect evidence
 ↓
Create claims
 ↓
Build timeline
 ↓
Run AI analysis
 ↓
Export research
```

Later rerun the research.

System must show exactly what changed:

```text
SOURCE_CHANGED
DOCUMENT_CHANGED
EVIDENCE_CHANGED
DATASET_CHANGED
MODEL_CHANGED
PROMPT_CHANGED
NO_CHANGE
```

---

# 46. Cross-Jurisdiction Test

Use at least three jurisdictions with different:

* institutions
* terminology
* legislative procedures
* document structures
* government levels

Verify that the global core remains unchanged.

A country-specific difference must be implemented through:

```text
configuration
adapter
procedure definition
ontology extension
validation policy
```

not duplicated global logic.

---

# 47. AI Failure Test

Disable the AI provider.

Verify that users can still:

* search
* inspect Bills
* inspect institutions
* inspect timelines
* inspect evidence
* inspect documents
* inspect provenance
* traverse relationships
* conduct deterministic research

AI is an intelligence layer, not a dependency for civic truth.

---

# 48. Final Acceptance Gates

Phase 13 is complete only when all gates pass.

### Knowledge

* Every major civic entity has a canonical representation.
* Relationships are first-class.
* Relationships have provenance.
* Historical validity is preserved.
* Corrections preserve history.
* No silent entity merges exist.

### Temporal correctness

* Historical state can be reconstructed.
* Relationship validity is date-aware.
* Source snapshots are immutable.
* Superseded knowledge remains traceable.

### Trust

* Every important claim can reach evidence.
* Every relationship can reach supporting evidence where applicable.
* AI-generated relationships cannot silently become authoritative.
* Contradictions remain visible.

### Research

* Research runs are reproducible.
* Evidence collections are versioned.
* Dataset changes are detectable.
* AI/model/prompt changes are recorded.

### AI

* Graph-aware retrieval works.
* Temporal retrieval works.
* Citation validation works.
* Unsupported relationships are not fabricated.
* AI failure does not break core civic intelligence.

### API

* Knowledge APIs work.
* Relationship traversal works.
* Timeline APIs work.
* Lineage APIs work.
* Provenance remains available.

### Global

* Multiple jurisdictions work simultaneously.
* Different legislative procedures work.
* Different institutions work.
* Cross-jurisdiction relationships remain isolated and explainable.

### Security

* Authorization is enforced through graph traversal.
* Cross-tenant leakage tests pass.
* Graph abuse is controlled.
* Source poisoning defenses work.

### Performance

* Traversal performance is measured.
* Temporal queries are measured.
* Research workloads are measured.
* No unbounded graph queries exist.

### Resilience

* Projection failures recover.
* Events can be replayed.
* Reconciliation repairs divergence.
* AI outages do not affect canonical civic data.

### UX

* Citizens can understand relationships without understanding graph technology.
* Researchers can inspect evidence quickly.
* Institutional pages are understandable.
* Legal lineage is readable.
* Provenance is transparent without overwhelming normal users.

---

# 49. Final Product Validation

Run the following real-world questions:

> “What happened to this Bill?”

> “What changed between the original Bill and the final Act?”

> “Who was involved?”

> “Which institution is responsible?”

> “Which regulations came from this Act?”

> “What court decisions interpreted it?”

> “What did the law look like in 2023?”

> “Show me the evidence.”

> “What changed since I last researched this?”

> “How does this compare with another jurisdiction?”

The system must answer using:

```text
Knowledge
+
Temporal State
+
Relationships
+
Evidence
+
Provenance
+
Jurisdiction
+
Verified Sources
+
AI Explanation
```

Never:

```text
LLM memory
+
guessing
+
unstated assumptions
```

---

# Phase 13 Definition of Done

The platform should now behave like a **living civic knowledge network** rather than a collection of databases and AI features.

A citizen can move from:

```text
Question
 ↓
Matter
 ↓
Institution
 ↓
Timeline
 ↓
Relationships
 ↓
Documents
 ↓
Evidence
 ↓
Source
```

A researcher can move from:

```text
Question
 ↓
Search
 ↓
Evidence
 ↓
Claims
 ↓
Relationships
 ↓
Timeline
 ↓
Research
 ↓
Reproducible Result
```

And the system can move from:

```text
New official document
 ↓
Knowledge update
 ↓
Relationship update
 ↓
Temporal history
 ↓
Evidence validation
 ↓
Search/graph projection
 ↓
Affected intelligence
 ↓
Notifications
```

without losing historical truth.

The defining principle of Phase 13 is:

> **Do not merely store civic information. Model how civic reality is connected, how it changes over time, and exactly why the system believes each connection exists.**

The platform should now be capable of answering not only **“What happened?”**, but **“How do we know, what is it connected to, and what was true when?”**

This sets up the next major evolution: **Phase 14 can turn the knowledge network into a proactive Civic Intelligence Engine that continuously discovers emerging civic developments, detects meaningful changes, and prepares evidence-backed intelligence before a citizen explicitly asks.**
