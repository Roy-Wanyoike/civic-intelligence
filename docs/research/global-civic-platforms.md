# Research — Global Civic Intelligence Platforms

**Date:** 2026-09-09
**Purpose:** Analyze existing civic/parliamentary monitoring platforms globally to inform the Civic Intelligence Platform's expansion strategy.

---

## Summary

After researching 15+ platforms across Africa, Europe, and North America, the key findings are:

1. **Most platforms are MP-centric** — TheyWorkForYou, Mzalendo, Parliament Watch — they track what politicians do, not what legislation means
2. **Few use AI** — most are data aggregators without plain-language explanations
3. **None are cross-country** — every platform is country-specific
4. **Civic Intelligence's differentiation** is: AI explanations + evidence grounding + cross-country architecture + political neutrality

---

## Platforms Analyzed

### Africa

| Platform | Country | URL | Key Feature |
|----------|---------|-----|-------------|
| Mzalendo | Kenya | mzalendo.com | MP scoresheets, Hansard extraction |
| Trackminster | Kenya | trackminster.com | Constituency-centric, accountability |
| Parliament Watch Uganda | Uganda | parliamentwatch.ug | Bills tracker, MP profiles |
| People's Assembly | South Africa | peoplesassembly.org.za | MP + committee tracking |
| PMG (Parliamentary Monitoring Group) | South Africa | pmg.org.za | Committee reports, detailed tracking |
| Odekro | Ghana | odekro.org | Parliament monitoring, Hansard |

### Europe

| Platform | Country | URL | Key Feature |
|----------|---------|-----|-------------|
| TheyWorkForYou | UK | theyworkforyou.com | MP voting records, speeches |
| PublicWhip | UK | publicwhip.org.uk | Voting analysis |
| ParlTrack | EU | parltrack.org | EU legislation tracking |
| HowTheyVote | Switzerland | howtheyvote.eu | Swiss parliament votes |
| Abgeordnetenwatch | Germany | abgeordnetenwatch.de | MP profiles + voting |

### North America

| Platform | Country | URL | Key Feature |
|----------|---------|-----|-------------|
| GovTrack | US | govtrack.us | Federal legislation tracker |
| LegiScan | US | legiscan.com | State + federal legislation |
| BillTrack50 | US | billtrack50.com | State legislation tracking |
| OpenCongress | US | (defunct) | Was the original bill tracker |
| Countable | US | countable.us | Civic engagement + bill explanations |
| Parliament of Canada | Canada | parl.ca | Official parliament data |

### Australia

| Platform | Country | URL | Key Feature |
|----------|---------|-----|-------------|
| TheyVoteForYou | Australia | theyvoteforyou.org.au | MP voting records |

---

## Key Insights for Civic Intelligence Platform

### What we can learn

1. **TheyWorkForYou (UK)**: Excellent MP-centric UX. Hansard parsing is a core capability. Their data model (Person → Speech → Vote → Bill) is proven.

2. **GovTrack (US)**: Best-in-class bill tracking. They show bill status, sponsors, cosponsors, related bills, and committee assignments. Their RSS/API feeds are excellent.

3. **Mzalendo (Kenya)**: Local Kenyan platform. MP scoresheets (how often they attend, vote, speak). They have Hansard data and committee reports.

4. **Parliament Watch Uganda**: Bills tracker with status tracking. Good model for our Uganda adapter.

5. **Countable (US)**: Closest to our vision — they explain bills in plain language and let citizens "vote" on them. BUT they are politically opinionated (progressive lean).

### What makes us different

| Feature | Civic Intelligence | Others |
|---------|-------------------|--------|
| AI explanations | ✅ Yes (citation-validated) | ❌ None use AI |
| Evidence grounding | ✅ Every claim traceable | ❌ No evidence chains |
| Cross-country | ✅ Adapter architecture | ❌ All are country-specific |
| Political neutrality | ✅ Enforced by architecture | ⚠️ Many are opinionated |
| Immutable versioning | ✅ BillVersions never overwritten | ❌ None track document history |
| Contradiction engine | ✅ Surfaces source conflicts | ❌ None detect conflicts |
| M-Pesa + Card payments | ✅ Africa-first payments | ❌ Western payment only |
| Loans/grants tracker | ✅ Government debt tracking | ❌ None track sovereign debt |

### Recommended features to adopt

From GovTrack:
- Bill cosponsor tracking
- Related bills (similar legislation across years)
- RSS feeds for each Bill

From TheyWorkForYou:
- MP attendance + speaking frequency
- Vote records with party-line deviation

From Mzalendo:
- MP scoresheets (but politically neutral — just facts)
- Committee attendance

From Countable:
- Plain-language bill explanations (we already have this)
- Citizen "support/oppose" (but without lobbying)

---

## Country Expansion Priority

Based on research, the recommended expansion order:

1. **Kenya** ✅ (live — 50 real bills)
2. **Uganda** (parliament.go.ug has Bills + Bill Tracker + Hansard)
3. **Tanzania** (parliament.go.tz — Bunge)
4. **Ghana** (parliament.gh + Odekro as reference)
5. **Nigeria** (nass.gov.ng — National Assembly, 36 states)
6. **South Africa** (parliament.gov.za + PMG as reference)
7. **Rwanda** (parliament.gov.rw — Parliament of Rwanda)

### Uganda (first expansion)

**Sources:**
- parliament.go.ug — official Parliament of Uganda website
- parliamentwatch.ug — aggregator (reference only, not authoritative)
- Parliament has: Bills, Bill Tracker, Hansard, Order Papers, Committee Reports

**Legislative structure:**
- Unicameral: Parliament of Uganda (no Senate)
- 556 members (including ex-officio)
- Bills: First Reading → Second Reading → Committee → Report → Third Reading → Assent
- President assents to Bills

**Key difference from Kenya:** Uganda is unicameral (no Senate). The adapter must handle this without breaking the core domain.
