# Research — Kenya Legislative Sources

**Date:** 2026-09-09
**Purpose:** Identify credible sources for the Civic Intelligence Platform's Kenya ingestion pipeline.

---

## 1. Kenya Law (new.kenyalaw.org/bills/)

**URL:** https://new.kenyalaw.org/bills/

**What it provides:**
- Official Bills tracker with 932+ documents (as of search date)
- Browseable by year: 2026, 2025, 2024, 2023, 2022
- Each Bill links to its full text + metadata
- Examples found: County Governments Retirement Scheme Bill, Basic Education Bill, Council for Law Reporting Bill

**Authority level:** Primary — this is the official Kenya Law Reports site maintained by the National Council for Law Reporting (a state corporation under the Office of the Attorney General).

**Integration approach:**
- Crawl `https://new.kenyalaw.org/bills/` for Bill listings
- Parse the year-based filter to discover all Bills for a given year
- Follow each Bill link to its detail page for metadata (title, identifier, sponsor, house, stage, assent date, commencement date)
- Download the full text PDF/HTML for archival + parsing

**Adapter:** `adapters/kenya/kenya_law/` (issue #29 — already created as a stub)

---

## 2. Parliament of Kenya (parliament.go.ke)

**URL:** https://www.parliament.go.ke

**What it provides:**
- Bills page with current and historical Bills
- Hansard (debate transcripts)
- Order Papers (daily agenda)
- Votes & Proceedings
- Committee documents and reports
- MP/Senator profiles

**Authority level:** Primary — official Parliament of Kenya website.

**Integration approach:**
- Crawl `parliament.go.ke/the-national-assembly` and `parliament.go.ke/the-senate`
- Parse Bill tracker for stage transitions
- Cross-reference with Kenya Law for assent + commencement dates

**Adapter:** `adapters/kenya/parliament/` (issue #24 — already created as a stub)

---

## 3. President of Kenya (president.go.ke)

**URL:** https://www.president.go.ke

**What it provides:**
- Presidential assent announcements — when the President signs Bills into law
- Press releases with the list of Bills assented to

**Authority level:** Primary — official State House website.

**Recent findings (from web search on 2026-09-09):**
- **Sep 8, 2026:** President Ruto assented to 4 Bills at State House, including:
  - Kenya National Council for Population and Development Bill
  - (3 others covering finance, trusts, air charges — details to be confirmed)
- **Dec 4, 2024:** President assented to Division of Revenue (Amendment) Bill 2024, National Rating Bill 2022, Water (Amendment) Bill 2024
- **Dec 12, 2024:** Signed 7 items of legislation including 2 financial bills

**Integration approach:**
- Crawl `president.go.ke` for assent announcements
- Parse the Bill names + assent dates
- Emit `bill.stage_changed` events (stage = PRESIDENTIAL_ASSENT)
- Cross-reference with Parliament + Kenya Law to link the Bill record

**Adapter:** `adapters/kenya/president/` (NEW — issue to be created)

---

## 4. Trackminster (trackminster.com)

**URL:** https://www.trackminster.com

**What it provides:**
- "Empowering Kenyan citizens with real parliamentary data. Open, transparent, and built for accountability."
- Data sourced from official public records
- Focuses on constituency-level data (e.g., Maragwa Constituency)

**Authority level:** Aggregator — third-party platform that republishes official parliamentary data. NOT a primary source.

**What we can learn from it:**
- **UX inspiration:** constituency-centric navigation (citizens find their MP first, then see Bills their MP sponsored/voted on)
- **Accountability framing:** "built for accountability" — surfaces MP voting records and attendance
- **Real-time focus:** "real parliamentary data" — emphasizes freshness

**Integration approach:**
- Do NOT ingest from Trackminster (it's an aggregator, not authoritative)
- DO use it as a UX benchmark for our own constituency/MP pages
- Track as a competitive reference in product documentation

---

## 5. Other credible sources identified

| Source | URL | What | Authority |
|--------|-----|------|-----------|
| People Daily | peopledaily.digital | News on assented bills | News (not primary) |
| The Star Kenya | the-star.co.ke | News coverage | News |
| Kenya Moja | kenyamoja.com | News aggregator | Aggregator |
| African Law Business | africanlawbusiness.com | Legal news | News |
| Quorum | quorum.us | Legislative tracking (US-focused) | Product reference |
| LegiScan | legiscan.com | Legislative tracking (US) | Product reference |
| BillTrack50 | billtrack50.com | Legislative tracking (US) | Product reference |
| StateScape | statescape.com | Legislative tracking (US) | Product reference |
| mySociety data | data.mysociety.org | Kenya National Assembly open data | Open data |
| OpenSanctions | opensanctions.org | Kenya MP data (JSON downloads) | Open data |

---

## Bill tracking workflow (the "how can we track all this" answer)

To track Bills from introduction to enactment — like the 4 Bills the President signed today — the platform needs:

### 1. Multi-source ingestion
```
parliament.go.ke     → Bill discovery + stage tracking
new.kenyalaw.org/bills/ → Bill metadata + full text
president.go.ke      → Assent events
kenyalaw.org/kl/     → Acts of Parliament (post-assent)
```

### 2. Event-driven pipeline
```
Crawler discovers new Bill
  → emit bill.discovered event
  → documents service fetches + archives
  → legislation service creates canonical Bill record

Crawler detects stage change (e.g., Second Reading → Committee)
  → emit bill.stage_changed event
  → legislation service updates current_stage
  → notifications service alerts followers

Crawler detects presidential assent on president.go.ke
  → emit bill.stage_changed (stage=PRESIDENTIAL_ASSENT)
  → evidence service links assent announcement
  → notifications service alerts followers
  → daily briefing includes the assent
```

### 3. Real-time analysis
```
For each Bill:
  - Plain-language summary (AI, citation-validated)
  - Verified timeline (every event linked to a source)
  - Stage explanation (Kenya adapter terminology)
  - Impact analysis (citizen, small business, farmer, etc.)
  - Related entities (sponsor, committee, house)
  - Source documents (archived immutably)
```

### 4. Citizen alerts
```
User follows a Bill
  → When stage changes: push notification + email
  → When amendment tabled: push + email
  → When assented: breaking alert
  → When commenced: final alert + link to the Act
```

This is the full vision. Issues #24-#31 cover the ingestion adapters. Issues #39-#41 cover the monitoring + alerts. Issues #32-#35 cover the analysis.
