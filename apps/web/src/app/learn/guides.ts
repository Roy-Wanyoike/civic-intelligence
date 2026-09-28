/**
 * Civic Education Hub — guide data.
 * =================================
 *
 * SINGLE source of truth for every published guide at `/learn/<slug>`.
 *
 * This file is plain TypeScript data — no JSX, no per-guide component files.
 * The renderer at `apps/web/src/app/learn/[slug]/page.tsx` walks the
 * `sections[]` array of a guide and emits the prose, the table of
 * contents, the sources list, the related-platform links, and the
 * "Was this helpful?" feedback widget. Adding a 6th guide is a 1-file
 * change here (append a `Guide` object to `guides`); no other code
 * moves.
 *
 * Content model
 * -------------
 * A `Guide` has:
 *   - `slug`            URL path fragment, also used as the SEO canonical fragment.
 *   - `title`           rendered as the page <h1> + <title>.
 *   - `description`     SEO meta description (≤160 chars) + landing card summary.
 *   - `category`         accent colour + icon on the landing card.
 *   - `readingTimeMinutes` shown on the card + guide header.
 *   - `lastReviewed`     ISO date shown so readers can judge currency.
 *   - `sources`          authoritative upstream URLs (kenyalaw.org, parliament.go.ke).
 *                         Rendered as the "Sources" section at the bottom.
 *   - `relatedLinks`     internal platform pages (/bills, /acts, /people, …).
 *                         Rendered as the "Related on the platform" grid.
 *   - `sections`         the body. Each section has:
 *       - `id`           anchor used by the TOC (#fragment).
 *       - `heading`      section <h2>.
 *       - `badge?`       optional pill next to the heading (e.g. "Stage 3").
 *       - `body`         array of paragraph strings. Each string becomes a <p>.
 *                        Inline markdown link syntax `[label](/path)` is
 *                        parsed by the renderer into <a> tags so paragraphs
 *                        can link to live platform pages without escaping JSX.
 *       - `source_links?`  per-section source URLs (e.g. a stage-specific /bills
 *                        filter or a specific kenyalaw.org article).
 *       - `related_bills?` per-section internal links to relevant Bills / Acts
 *                        / people on the platform.
 *
 * The `source_links` / `related_bills` per-section fields are the spec-mandated
 * hook (issue #292 §3) for in-context cross-links — e.g. the "Stage 4 —
 * Committee Stage" section of the Bill-becomes-law guide links to the
 * /participation page (where open calls live) without the reader having to
 * scroll to the bottom of the page.
 */

export type GuideCategory =
  | 'Legislation'
  | 'Parliament'
  | 'Constitution'
  | 'Civic Participation'
  | 'Reading Skills';

/** External authoritative source — kenyalaw.org, parliament.go.ke, etc. */
export interface GuideSource {
  label: string;
  url: string;
}

/** Internal platform link — /bills, /acts, /people/[id], … */
export interface GuideRelatedLink {
  label: string;
  href: string;
  description?: string;
}

export interface GuideSection {
  /** Anchor id — must be unique within the guide. Used as the #fragment
   * the TOC links to. Lowercase, hyphen-separated. */
  id: string;
  /** Section heading rendered as the page <h2>. */
  heading: string;
  /** Optional badge pill rendered next to the heading. Used by the
   * bill-becomes-law walkthrough for "Stage 1" … "Stage 8". */
  badge?: string;
  /** Body — array of paragraph strings. Each becomes a <p> inside the
   * prose wrapper. Supports inline `[label](href)` link syntax. */
  body: string[];
  /** Optional per-section authoritative sources (e.g. a specific Bill
   * page on kenyalaw.org). Rendered as a small inline link list under
   * the section body, before the next section. */
  source_links?: GuideSource[];
  /** Optional per-section internal links to relevant Bills/Acts/people
   * on the platform. Rendered as a small "See also" link list under
   * the section body. */
  related_bills?: GuideRelatedLink[];
}

export interface Guide {
  slug: string;
  title: string;
  /** SEO meta description (≤160 chars). Also used as the landing card summary. */
  description: string;
  category: GuideCategory;
  /** Estimated reading time in minutes — shown on the card + guide header. */
  readingTimeMinutes: number;
  /** ISO date (YYYY-MM-DD) the guide was last reviewed. */
  lastReviewed: string;
  /** Authoritative upstream sources — rendered as the "Sources" section. */
  sources: GuideSource[];
  /** Internal platform pages related to this guide — "Related on the platform". */
  relatedLinks: GuideRelatedLink[];
  /** Ordered body sections. The TOC is derived from this list automatically. */
  sections: GuideSection[];
}

// ---------------------------------------------------------------------------
// Guides
// ---------------------------------------------------------------------------

export const howABillBecomesLaw: Guide = {
  slug: 'how-a-bill-becomes-law',
  title: 'How a Bill becomes a law in Kenya',
  description:
    'An 8-stage walkthrough of how a Bill becomes a law in Kenya: first reading, second reading, committee, report stage, third reading, assent, publication, and commencement.',
  category: 'Legislation',
  readingTimeMinutes: 9,
  lastReviewed: '2025-01-15',
  sources: [
    {
      label: 'Constitution of Kenya, 2010 — Articles 109–115',
      url: 'https://www.kenyalaw.org/kl/index.php?id=398',
    },
    {
      label: 'National Assembly Standing Orders (current edition)',
      url: 'https://www.parliament.go.ke/the-national-assembly/standing-orders',
    },
    {
      label: 'Kenya Law — Bills Tracker',
      url: 'https://www.kenyalaw.org/kl/index.php?id=4686',
    },
  ],
  relatedLinks: [
    {
      label: 'Track live Bills',
      href: '/bills',
      description: 'See every Bill currently before Parliament, filtered by stage.',
    },
    {
      label: 'Browse enacted Acts',
      href: '/acts',
      description: 'Bills that completed all 8 stages and became law.',
    },
    {
      label: 'Find your MP',
      href: '/people',
      description: 'MPs vote on Bills at Second, Report, and Third Reading.',
    },
    {
      label: 'Public participation opportunities',
      href: '/participation',
      description: 'When committees invite submissions — usually between Second Reading and Committee Stage.',
    },
  ],
  sections: [
    {
      id: 'introduction',
      heading: 'Why the 8 stages matter',
      body: [
        'A Bill does not become law the day it is published. Every Bill must pass through a deliberately slow, public, eight-stage process before it binds anyone. The slowness is a feature, not a bug — it gives MPs, committees, and the public time to scrutinise the text before it acquires the force of law.',
        'The eight stages are set by Articles 109 to 115 of the Constitution of Kenya, 2010 and amplified by the Standing Orders of each House. A Money Bill (which only the National Assembly may originate) follows the same path with extra rules about Senate involvement. A Constitutional Amendment Bill adds a fourth reading in each House and may need a referendum.',
        'Use the live [Bills tracker](/bills) to see which Kenyan Bill is at which stage today.',
      ],
      related_bills: [
        {
          label: 'Bills before Parliament right now',
          href: '/bills',
          description: 'Each Bill card shows its current stage — from First Reading to Assent.',
        },
      ],
    },
    {
      id: 'stage-1-publication',
      heading: 'Publication (Gazettement)',
      badge: 'Stage 1',
      body: [
        'A Bill is published in the Gazette as a Kenya Gazette Notice, accompanied by the long title and the explanatory memorandum. Publication is the official "this Bill exists" moment — before publication, the text may be a draft circulating within the Attorney-General\'s office or a sponsor\'s chambers, and has no formal standing.',
        'Once gazetted, the Bill is given an identifier like `NA Bill No. 23 of 2024` and added to the order paper. The sponsor (a Cabinet Secretary for a government Bill, or an MP for a private member\'s Bill) formally introduces it for First Reading.',
        'See new Bills as soon as they are gazetted on the [Kenya Gazette](/gazette) page, or subscribe to [Gazette Alerts](/gazette/alerts) to be notified when a keyword you care about appears.',
      ],
    },
    {
      id: 'stage-2-first-reading',
      heading: 'First Reading',
      badge: 'Stage 2',
      body: [
        'The Clerk reads the Bill\'s long title aloud in the House. No debate takes place. The Bill is then committed to the relevant departmental committee (for a National Assembly Bill) or standing committee (for a Senate Bill).',
        'The committee stage that follows is where the real scrutiny begins: committees invite written submissions from the public, hold public hearings, and call the sponsor to defend the Bill clause by clause.',
      ],
    },
    {
      id: 'stage-3-second-reading',
      heading: 'Second Reading — debate on principles',
      badge: 'Stage 3',
      body: [
        'The House debates the principles and policy of the Bill, not its wording. MPs who support or oppose the Bill speak; amendments to specific clauses are NOT permitted at this stage — that comes later, in committee and at report stage.',
        'If the House votes against the Bill at Second Reading, the Bill dies. If it votes in favour, the Bill proceeds to committee for clause-by-clause scrutiny.',
        'The committee\'s report (and any proposed amendments) is what the House will consider at Report Stage.',
      ],
    },
    {
      id: 'stage-4-committee-stage',
      heading: 'Committee Stage — clause by clause',
      badge: 'Stage 4',
      body: [
        'The departmental committee goes through the Bill clause by clause, receiving evidence from the sponsor, ministries, experts, and the public. The committee may propose amendments — to fix drafting errors, align the Bill with existing law, or change policy.',
        'This is the stage at which public participation is most meaningful: committees publish calls for submissions in the Gazette and on parliament.go.ke, with a deadline typically 14 days out. See [open participation opportunities](/participation) on this platform.',
      ],
      source_links: [
        {
          label: 'Constitution of Kenya, 2010 — Article 118 (public access)',
          url: 'https://www.kenyalaw.org/kl/index.php?id=398',
        },
      ],
      related_bills: [
        {
          label: 'Public participation opportunities',
          href: '/participation',
          description: 'Open calls for submissions on Bills currently in committee.',
        },
        {
          label: 'Find the committee handling a Bill',
          href: '/committees',
          description: 'Departmental committees are listed with their current chairs.',
        },
      ],
    },
    {
      id: 'stage-5-report-stage',
      heading: 'Report Stage',
      badge: 'Stage 5',
      body: [
        'The committee tables its report — including any amendments it recommends — and the House considers each amendment individually. MPs may move further amendments at this stage, but only to clauses the committee has already touched (Standing Order limitation, designed to keep the debate focused).',
        'After all amendments are voted on, the Bill is set down for Third Reading.',
      ],
    },
    {
      id: 'stage-6-third-reading',
      heading: 'Third Reading — final vote',
      badge: 'Stage 6',
      body: [
        'The House debates the Bill one last time — typically on whether the Bill, as amended, should pass. No new substantive amendments are permitted. A simple majority is enough for ordinary Bills; Constitutional Amendment Bills require a two-thirds supermajority.',
        'Once passed by one House, a Bill is transmitted to the other House (National Assembly ↔ Senate) where it goes through the same stages. Money Bills skip the Senate except where Article 114 of the Constitution requires consultation.',
      ],
    },
    {
      id: 'stage-7-presidential-assent',
      heading: 'Presidential Assent',
      badge: 'Stage 7',
      body: [
        'The Bill is presented to the President, who within 14 days must either: (a) assent — sign the Bill into law; (b) refer it back to Parliament with proposed amendments (a one-time veto with conditions); or (c) refer it to the Supreme Court for an advisory opinion on constitutionality.',
        'If Parliament overrides a referral by a two-thirds vote in both Houses, the President must assent within seven days.',
        'See recent [assented Acts](/acts) on the platform — the `Assent` event is the moment a Bill becomes an Act.',
      ],
      related_bills: [
        {
          label: 'Recently assented Acts',
          href: '/acts',
          description: 'Acts of Parliament that received Presidential Assent.',
        },
      ],
    },
    {
      id: 'stage-8-publication-and-commencement',
      heading: 'Publication & Commencement',
      badge: 'Stage 8',
      body: [
        'After assent, the Act is published in the Gazette as a Special Issue. The Act comes into force either: (a) on the date of publication (default); (b) on a date specified in the Act itself; or (c) on a date to be appointed later by the Cabinet Secretary via a legal notice (a "commencement order").',
        'The Act only binds Kenyans from its commencement date — an Act that has been assented to but has not yet commenced is not yet enforceable. This is why commencement orders matter: they decide when a new law starts affecting you.',
        'Browse the full lineage of any Act — from Bill to assent to amendments — on its [Act page](/acts).',
      ],
    },
    {
      id: 'special-cases',
      heading: 'Special cases',
      body: [
        'Money Bills (defined by Article 114) can only originate in the National Assembly. The Senate may recommend amendments but cannot block passage.',
        'Constitutional Amendment Bills require a two-thirds supermajority in both Houses. Some amendments (affecting the Bill of Rights, the territory of Kenya, or the term of Parliament) must also be approved by popular vote in a referendum.',
        'Government Bills are drafted by the Office of the Attorney-General on instruction from Cabinet. Private member\'s Bills are drafted by individual MPs and are increasingly common — for example, several recent data-protection amendments originated in the Senate.',
      ],
    },
  ],
};

export const understandingParliament: Guide = {
  slug: 'understanding-parliament',
  title: "Understanding Kenya's Parliament",
  description:
    'A plain-language guide to Kenya\'s bicameral Parliament: the National Assembly, the Senate, the committee system, and how a Bill moves between the two Houses.',
  category: 'Parliament',
  readingTimeMinutes: 8,
  lastReviewed: '2025-01-15',
  sources: [
    {
      label: 'Constitution of Kenya, 2010 — Chapter 8 (Parliament)',
      url: 'https://www.kenyalaw.org/kl/index.php?id=398',
    },
    {
      label: 'Parliament of Kenya — About Parliament',
      url: 'https://www.parliament.go.ke/about-parliament',
    },
    {
      label: 'Senate of Kenya — Role of the Senate',
      url: 'https://www.parliament.go.ke/the-senate',
    },
  ],
  relatedLinks: [
    {
      label: 'Browse legislatures',
      href: '/legislatures',
      description: 'Each Parliament (11th, 12th, 13th) is a separate legislature with its own Bills and committees.',
    },
    {
      label: 'Standing committees',
      href: '/committees',
      description: 'Where most of the substantive scrutiny of Bills actually happens.',
    },
    {
      label: 'MPs and Senators',
      href: '/people',
      description: 'Find your representative and their voting record.',
    },
    {
      label: 'Read the Constitution',
      href: '/constitution',
      description: 'Chapter 8 establishes Parliament; Chapter 9 sets out the legislative process.',
    },
  ],
  sections: [
    {
      id: 'why-bicameral',
      heading: 'Why two Houses?',
      body: [
        'The 2010 Constitution replaced Kenya\'s pre-2010 unicameral National Assembly with a bicameral Parliament made up of two Houses: the National Assembly (349 elected members plus 12 nominated, representing the people directly) and the Senate (47 elected senators — one per county — plus nominated members, representing the counties).',
        'The reason for two Houses is devolution. Once Kenya created 47 county governments with real powers and budgets, the counties needed a voice at the national level to defend their interests against central encroachment. The Senate is that voice.',
      ],
    },
    {
      id: 'unicameral-vs-bicameral',
      heading: 'Unicameral vs. bicameral — a brief comparison',
      body: [
        'Before the 2010 Constitution, Kenya was unicameral: a single National Assembly passed all laws. Unicameral systems are simpler and faster, but they offer only one round of scrutiny. A second House forces a Bill to be examined twice — by different people, with different incentives — before it binds anyone.',
        'The trade-off is speed vs. deliberation. A bicameral Parliament can take longer to pass a Bill because both Houses must agree on the final text. Where the two Houses disagree, the Constitution provides a mediation committee (see "Mediation" below).',
      ],
    },
    {
      id: 'national-assembly',
      heading: 'The National Assembly',
      body: [
        'The National Assembly represents the people directly. Its responsibilities include: legislation (ordinary Bills can originate in either House, but Money Bills — taxes, budget, public debt — originate only in the National Assembly); budget approval (the Division of Revenue Act and the Appropriation Act originate here); oversight (the Assembly summons Cabinet Secretaries and reviews the conduct of the Executive); and removing a President (impeachment is initiated in the National Assembly and tried in the Senate).',
        'The Assembly is led by the Speaker of the National Assembly, elected by members. The leader of the largest party or coalition becomes the Leader of the Majority Party; the runner-up becomes the Leader of the Official Opposition.',
        'See your constituency\'s MP on the [Constituencies](/constituencies) page.',
      ],
      related_bills: [
        {
          label: 'National Assembly Bills',
          href: '/bills',
          description: 'Filter the Bills tracker by House to see NA-only Bills.',
        },
        {
          label: 'Find your MP',
          href: '/people',
          description: 'Filter by House = National Assembly.',
        },
      ],
    },
    {
      id: 'senate',
      heading: 'The Senate',
      body: [
        'The Senate represents the 47 counties. Its core responsibilities are: county revenue allocation (the Senate determines how national revenue is shared among counties each year — the County Allocation of Revenue Act); county oversight (the Senate scrutinises county budgets and considers county-governance-related Bills); impeachment trial (a Governor is impeached by the Senate after the county assembly votes to remove them); and legislation affecting counties (any Bill that touches on county government must be considered by the Senate).',
        'The Senate is led by the Speaker of the Senate. Each county\'s senator is elected directly by the county\'s voters; there are also nominated senators representing women, youth, and persons with disabilities.',
        'See senators on the [People](/people) page, filtered by house.',
      ],
      related_bills: [
        {
          label: 'Senate Bills',
          href: '/bills',
          description: 'Filter the Bills tracker by House = Senate.',
        },
      ],
    },
    {
      id: 'when-houses-disagree',
      heading: 'When the two Houses disagree — mediation',
      body: [
        'If one House passes a Bill and the other rejects it or passes it with amendments the first House won\'t accept, the Bill goes to a Mediation Committee — a joint committee of equal members from both Houses.',
        'The committee attempts to produce a version both Houses can accept. If mediation fails, the Bill lapses — it does not become law. This is a deliberate brake that forces the two Houses to negotiate rather than the larger House simply overriding the smaller.',
        'Money Bills are an exception: the National Assembly can pass a Money Bill even if the Senate does not agree, but the Senate must be heard.',
      ],
    },
    {
      id: 'committee-system',
      heading: 'The committee system — where the real work happens',
      body: [
        'Plenary debates in the Chamber get the headlines, but the substantive scrutiny of Bills happens in committees. Each House has two kinds of committees: Departmental / Standing Committees (which mirror the work of a ministry — e.g. the Departmental Committee on Health scrutinises the Ministry of Health\'s Bills and budget; these committees take evidence, hold public hearings, and recommend amendments clause-by-clause) and Select Committees (set up for a specific purpose — e.g. the Powers and Privileges Committee, the Committee on Implementation of the Constitution).',
        'Committee membership is sized so that every MP sits on at least one committee. Committees meet weekly when Parliament is sitting and may meet during recess by special permission.',
        'See all committees and their current chairs on the [Committees](/committees) page.',
      ],
      related_bills: [
        {
          label: 'Standing committees',
          href: '/committees',
          description: 'List of departmental and select committees with their current chairs.',
        },
      ],
    },
    {
      id: 'how-a-bill-moves',
      heading: 'How a Bill moves between the Houses',
      body: [
        'For ordinary Bills (not Money Bills), the Bill can originate in either House. After the originating House passes the Bill, it is transmitted to the other House for concurrence. The second House can: (a) pass the Bill as-is — the Bill goes to the President; (b) pass the Bill with amendments — the Bill returns to the originating House for reconsideration; or (c) reject the Bill — triggering mediation.',
        'See the full lifecycle in our ["How a Bill becomes a law" guide](/learn/how-a-bill-becomes-law).',
      ],
    },
    {
      id: 'sittings-and-calendar',
      heading: 'Sittings and the parliamentary calendar',
      body: [
        'A typical parliamentary year runs from February to November, with recess periods in April, August, and December. Plenary sittings are on Tuesday afternoons, Wednesday mornings and afternoons, and Thursday afternoons.',
        'Committee sittings happen almost daily during session weeks. Follow the [civic calendar](/calendar) to see what is scheduled this week.',
      ],
      related_bills: [
        {
          label: 'Civic calendar',
          href: '/calendar',
          description: 'Sessions, committee meetings, and gazette dates on one calendar.',
        },
      ],
    },
  ],
};

export const howToReadABill: Guide = {
  slug: 'how-to-read-a-bill',
  title: 'How to read a Bill',
  description:
    'A plain-language guide to the structure of a Kenyan Bill: short title, long title, explanatory memorandum, clauses, schedules, and how to read each part without a law degree.',
  category: 'Reading Skills',
  readingTimeMinutes: 7,
  lastReviewed: '2025-01-15',
  sources: [
    {
      label: 'Kenya Law — How a Bill is drafted (legislative drafting manual)',
      url: 'https://www.kenyalaw.org/kl/index.php?id=4686',
    },
    {
      label: 'Office of the Attorney-General & Department of Justice',
      url: 'https://www.statelaw.go.ke/',
    },
    {
      label: 'Constitution of Kenya, 2010 — Article 115 (assent)',
      url: 'https://www.kenyalaw.org/kl/index.php?id=398',
    },
  ],
  relatedLinks: [
    {
      label: 'Read live Bills',
      href: '/bills',
      description: 'Apply these reading tips to a real Bill before Parliament right now.',
    },
    {
      label: 'Bill versions and amendments',
      href: '/bills',
      description: 'Every Bill has multiple published versions — track what changed between them.',
    },
    {
      label: 'Ask a question about a Bill',
      href: '/ask',
      description: 'The platform\'s evidence-grounded AI explains clauses in plain language.',
    },
  ],
  sections: [
    {
      id: 'anatomy',
      heading: 'The anatomy of a Bill',
      body: [
        'Every Kenyan Bill follows the same template, set by the Office of the Attorney-General\'s drafting office. Knowing the structure means you can skip straight to the parts you care about — most readers do not need to read a Bill cover-to-cover.',
        'The typical Bill has, in order: (1) the short title — the name you will cite the Act by; (2) the long title — one sentence describing the Bill\'s purpose; (3) the enacting formula — the formal "Enacted by the Parliament of Kenya"; (4) the explanatory memorandum — a plain-language summary of each clause; (5) the arrangement of clauses — the table of contents; (6) the clauses (also called sections once enacted) — the substantive law; (7) the schedules — annexes of forms, lists, or detailed rules; and (8) a memorandum of objects and reasons — signed by the sponsor.',
      ],
    },
    {
      id: 'short-title',
      heading: '1. The short title',
      body: [
        'The short title is the name you will use to refer to the Act once it is enacted. For example, The Data Protection Act, 2019 is the short title — and the citation form lawyers use is `No. 24 of 2019` (Act number 24 of 2019).',
        'Short titles always end with a year — the year of assent, not the year of drafting. The year tells you which year the Bill became law, which is important when an older Act was amended.',
      ],
    },
    {
      id: 'long-title',
      heading: '2. The long title',
      body: [
        'The long title is one sentence describing what the Bill does. It always begins with the word "An" — as in, "An Act of Parliament to provide for the regulation of…".',
        'The long title is more than a label: it sets the constitutional scope of the Bill. Courts use the long title to decide whether a clause falls outside the Bill\'s stated purpose (and is therefore ultra vires).',
      ],
    },
    {
      id: 'explanatory-memorandum',
      heading: '3. The explanatory memorandum',
      body: [
        'The explanatory memorandum (sometimes called the "memorandum of objects and reasons") is a plain-language summary of each clause. It is written by the sponsor and circulated with the Bill.',
        'Read this first. It will not bind a court (only the clauses do), but it tells you what the sponsor was trying to do — which is the fastest way to understand a Bill before you read the legal text.',
        'On this platform, every Bill\'s In plain language section is an AI-generated, citation-validated rewrite of the memorandum. See an example on any [live Bill page](/bills).',
      ],
      related_bills: [
        {
          label: 'Read a live Bill',
          href: '/bills',
          description: 'Each Bill page has an AI-generated plain-language summary near the top.',
        },
      ],
    },
    {
      id: 'clauses',
      heading: '4. The clauses (the law itself)',
      body: [
        'Each clause is a self-contained unit of law, numbered `1.`, `2.`, `3.`, and so on. A typical clause has: a headnote (the bold text above the clause, like "Short title and commencement" — headnotes are a guide, not the law; the text of the clause is what binds); a chapeau (the opening sentence that the sub-clauses hang off of); sub-clauses (labelled `(1)`, `(2)`, `(3)`); paragraphs inside sub-clauses (labelled `(a)`, `(b)`, `(c)`); and sub-paragraphs (labelled `(i)`, `(ii)`, `(iii)`).',
        'This nesting is the single biggest reason legal text feels complicated. Once you know the pattern `(1)(a)(i)`, you can read it as if it were an outline.',
        'Clause 1 is almost always the short title and commencement clause. Clause 2 is almost always the interpretation clause — a glossary of defined terms used throughout the Bill.',
      ],
    },
    {
      id: 'schedules',
      heading: '5. The schedules',
      body: [
        'Schedules are annexes attached at the end of the Bill. They contain material too detailed to fit in the body — for example: the form of an oath or certificate; the list of bodies to which an Act applies; the numerical fees, rates, or quotas the Act sets; or the text of consequential amendments to other Acts.',
        'Schedules are part of the law — a court will treat a schedule the same as a clause. Always check the schedules for the "consequential amendments" schedule, which lists the other Acts this Bill will change.',
      ],
    },
    {
      id: 'reading-tips',
      heading: 'Plain-language reading tips',
      body: [
        'Read the memorandum first, then skip to the clauses you care about. Use the explanatory memorandum as a map. Once you know which clause does what, you only need to read the two or three clauses that affect you — not the entire Bill.',
        'Trace the defined terms. Defined terms are written with a capital letter: "the Authority", "personal data". Look them up in the interpretation clause (usually clause 2) before reading further — the Bill\'s meaning depends on these definitions.',
        'Watch out for "may" vs. "shall". "Shall" in a Bill means must — it imposes a duty. "May" means is permitted to — it confers a power or a discretion. A single word can change the entire effect of a clause.',
        'Read the "consequential amendments" schedule. This is where a Bill quietly changes other Acts. A small clause in a Bill can rewrite large parts of an existing Act — and the consequential amendments schedule is where you find out.',
        'Use the platform\'s version comparison. Every Bill has multiple published versions — the original, the committee-amended text, and the as-enacted text. Use the [Versions tab](/bills) on any Bill to see exactly what changed between versions.',
      ],
      related_bills: [
        {
          label: 'Bill versions',
          href: '/bills',
          description: 'Each Bill page has a Versions tab showing what changed between drafts.',
        },
      ],
    },
  ],
};

export const constitutionalRights: Guide = {
  slug: 'your-constitutional-rights',
  title: 'Your constitutional rights',
  description:
    'A plain-language guide to the Bill of Rights in the Constitution of Kenya, 2010 — Articles 19, 26, 27, 28, 31, 33, 35, 43, 46, and 50 — each with the constitutional text and an AI explanation.',
  category: 'Constitution',
  readingTimeMinutes: 10,
  lastReviewed: '2025-01-15',
  sources: [
    {
      label: 'Constitution of Kenya, 2010 — Chapter 4 (The Bill of Rights)',
      url: 'https://www.kenyalaw.org/kl/index.php?id=398',
    },
    {
      label: 'Kenya National Commission on Human Rights',
      url: 'https://www.knchr.org/',
    },
    {
      label: 'Read the Constitution on this platform',
      url: '/constitution',
    },
  ],
  relatedLinks: [
    {
      label: 'The Constitution, article by article',
      href: '/constitution',
      description: 'Browse every article of the Constitution with the original kenyalaw.org text.',
    },
    {
      label: 'How a Bill becomes a law',
      href: '/learn/how-a-bill-becomes-law',
      description: 'A Constitutional Amendment Bill needs a supermajority — see how in Guide 1.',
    },
    {
      label: 'Ask a civic question',
      href: '/ask',
      description: 'The AI grounds every answer in the constitutional text and cites its sources.',
    },
  ],
  sections: [
    {
      id: 'introduction',
      heading: 'What the Bill of Rights is',
      body: [
        'The Bill of Rights is Chapter 4 of the Constitution of Kenya, 2010 (Articles 19 to 59). It is the part of the Constitution that protects the freedoms and entitlements of every person in Kenya — citizens and non-citizens alike.',
        'The Bill of Rights is the framework for social, economic, and cultural policy: every law, every state action, and every government programme must be measured against it. A law that unconstitutionally limits a right is invalid to the extent of the inconsistency.',
        'Each section below shows the constitutional text (a verbatim quote from kenyalaw.org — a FACT), followed by a plain-language explanation generated by the platform\'s AI and citation-validated (an EXPLANATION).',
      ],
    },
    {
      id: 'article-19',
      heading: 'Article 19 — Rights and Fundamental Freedoms',
      body: [
        'Constitutional text (FACT): "The Bill of Rights is an integral part of Kenya\'s democratic State and is the framework for social, economic and cultural policies. The purpose of recognising and protecting rights and fundamental freedoms is to preserve the dignity of individuals and communities and to promote social justice and the realisation of the potential of all human beings."',
        'AI explanation (EXPLANATION): Article 19 tells us the Bill of Rights is not just a list of freedoms — it is the lens through which the State must design every policy. When the government drafts a budget, a Bill, or a regulation, it must justify how that policy preserves human dignity. This is why a court can strike down a law even if no individual was harmed: the test is whether the law itself is consistent with the Bill of Rights.',
      ],
      source_links: [
        {
          label: 'Constitution of Kenya, 2010 — Article 19',
          url: 'https://www.kenyalaw.org/kl/index.php?id=398',
        },
      ],
    },
    {
      id: 'article-26',
      heading: 'Article 26 — Right to Life',
      body: [
        'Constitutional text (FACT): "Every person has the right to life. The life of a person begins at conception. A person shall not be deprived of life intentionally, except to the extent authorised by this Constitution or other written law."',
        'AI explanation (EXPLANATION): Article 26 is one of the most-cited articles in Kenyan litigation. Two elements deserve attention. First, the Constitution expressly says life begins at conception — a phrase that has been the basis of much litigation around abortion law (Article 26(4) permits abortion only when, in the opinion of a trained health professional, the mother\'s life or health is in danger). Second, the State may not take life except as the Constitution itself authorises — meaning the death penalty, while on the statute books, is constrained by this article.',
      ],
    },
    {
      id: 'article-27',
      heading: 'Article 27 — Equality and Freedom from Discrimination',
      body: [
        'Constitutional text (FACT): "Every person is equal before the law and has the right to equal protection and equal benefit of the law. Equality includes the full and equal enjoyment of all rights and fundamental freedoms. The State shall not discriminate directly or indirectly against any person on any ground, including race, sex, pregnancy, marital status, health status, ethnic or social origin, colour, age, disability, religion, conscience, belief, culture, dress, language or birth."',
        'AI explanation (EXPLANATION): Article 27 lists the protected grounds of non-discrimination. Two features make this article unusually powerful. First, the list is non-exhaustive ("including") — courts can recognise new grounds. Second, both direct and indirect discrimination are prohibited: a rule that looks neutral on its face but disadvantages a protected group in practice is also unconstitutional. Article 27(8) further requires the State to take legislative and other measures to ensure that not more than two-thirds of elective bodies are of the same gender — the "two-thirds gender rule".',
      ],
    },
    {
      id: 'article-28',
      heading: 'Article 28 — Human Dignity',
      body: [
        'Constitutional text (FACT): "Every person has inherent dignity and the right to have that dignity respected and protected."',
        'AI explanation (EXPLANATION): Article 28 is the shortest article in the Bill of Rights and one of the most consequential. Dignity is treated as inherent — meaning it is not granted by the State and cannot be taken away. Courts have used Article 28 to limit police powers (e.g. to prohibit degrading treatment of detainees), to require humane conditions in prisons, and to ground the right to a name and identity. Article 28 also underpins Article 19: the entire purpose of the Bill of Rights is to preserve human dignity.',
      ],
    },
    {
      id: 'article-31',
      heading: 'Article 31 — Privacy',
      body: [
        'Constitutional text (FACT): "Every person has the right to privacy, which includes the right not to have their person, home or property searched; their possessions seized; information relating to their family or private affairs unnecessarily required or revealed; or the privacy of their communications infringed."',
        'AI explanation (EXPLANATION): Article 31 is the constitutional foundation of data-protection law in Kenya. The Data Protection Act, 2019 (which you can browse on the [Acts page](/acts)) gives effect to this article by setting out how organisations may collect, store, and process personal data. Article 31 also constrains State surveillance: the police cannot search a home, seize property, or intercept communications without legal authority (typically a court order).',
      ],
      related_bills: [
        {
          label: 'Data Protection Act, 2019',
          href: '/acts',
          description: 'The Act that operationalises Article 31 — search Acts for "Data Protection".',
        },
      ],
    },
    {
      id: 'article-33',
      heading: 'Article 33 — Freedom of Expression',
      body: [
        'Constitutional text (FACT): "Every person has the right to freedom of expression, which includes freedom to seek, receive or impart information or ideas; freedom of artistic creativity; and academic freedom and freedom of scientific research."',
        'AI explanation (EXPLANATION): Article 33 protects expression in three forms — seeking, receiving, and imparting information — across all media. The right is not absolute: Article 33(2) permits limitations for propaganda for war, incitement to violence, hate speech, and advocacy of hatred. The Supreme Court has held that the burden of justifying any limitation on free expression falls on the State, and the limitation must be proportionate. This article is the basis for press freedom in Kenya and is invoked in challenges to media regulations and content takedown orders.',
      ],
    },
    {
      id: 'article-35',
      heading: 'Article 35 — Access to Information',
      body: [
        'Constitutional text (FACT): "Every citizen has the right to access information held by the State; and information held by another person and required for the exercise or protection of any right."',
        'AI explanation (EXPLANATION): Article 35 is the backbone of the platform you are reading right now. The Constitution obliges the State to publish and publicise any important information affecting the nation. The Access to Information Act, 2016 operationalises this right — citizens can request information from any public body, and the body must respond within 21 days. This is also why official sources (like kenyalaw.org and parliament.go.ke) are the foundation of every claim this platform surfaces.',
      ],
    },
    {
      id: 'article-43',
      heading: 'Article 43 — Economic and Social Rights',
      body: [
        'Constitutional text (FACT): "Every person has the right to the highest attainable standard of health; accessible and adequate housing and to reasonable standards of sanitation; to be free from hunger, and to have adequate food of acceptable quality; to clean and safe water in adequate quantities; to social security; and to education."',
        'AI explanation (EXPLANATION): Article 43 protects "positive" rights — things the State must do (provide health, housing, food, water, social security, education) rather than merely refrain from interfering with. These rights are subject to progressive realisation (Article 21(2)): the State must take steps "to the maximum of available resources". This makes Article 43 harder to enforce in court than the civil-political rights above, but it anchors every budget debate — including the housing levy litigation that reached the Court of Appeal and the High Court.',
      ],
    },
    {
      id: 'article-46',
      heading: 'Article 46 — Consumer Rights',
      body: [
        'Constitutional text (FACT): "Consumers have the right to goods and services of reasonable quality; to the information necessary for them to gain full benefit from goods and services; to the protection of their health, safety and economic interests; and to compensation for loss or injury arising from defects in goods or services."',
        'AI explanation (EXPLANATION): Article 46 constitutionalises consumer protection — previously a matter of statute alone. The Consumer Protection Act, 2012 gives effect to this article. Article 46 has been invoked in challenges to misleading advertising, unsafe products, and unfair contract terms. It also entitles consumers to compensation for defective goods, which strengthens the remedies available under ordinary contract and tort law.',
      ],
    },
    {
      id: 'article-50',
      heading: 'Article 50 — Fair Hearing',
      body: [
        'Constitutional text (FACT): "Every person has the right to have any dispute that can be resolved by the application of law decided in a fair and public hearing before a court or, where appropriate, another independent and impartial tribunal or body."',
        'AI explanation (EXPLANATION): Article 50 is the constitutional source of due-process rights in Kenya: the right to be informed of the charge, to be presumed innocent, to be tried in public, to have adequate time and facilities to prepare a defence, to call and cross-examine witnesses, and to be represented by a lawyer of one\'s choice. Article 50(2) enumerates the rights of an accused person in a criminal trial; Article 50(1) is broader, applying to any justiciable dispute. A court that violates Article 50 renders its decision voidable on appeal.',
      ],
    },
    {
      id: 'limits-and-enforcement',
      heading: 'Limits, duties, and enforcement',
      body: [
        'Rights are not absolute — but limits must be justified. Article 24 permits the State to limit a right only if the limitation is "reasonable and justifiable in an open and democratic society based on human dignity, equality and freedom". The State carries the burden of proving that a limitation meets this test.',
        'The State has a duty to realise rights. Article 21 imposes a fundamental duty on the State and every State organ to observe, respect, protect, promote, and fulfil the rights in the Bill of Rights. "Fulfil" means the State must take positive steps to realise rights — not merely refrain from violating them.',
        'Enforcement — the High Court. Article 23 gives the High Court jurisdiction to hear matters alleging a right has been denied, violated, infringed, or threatened. The court may award compensation, declare a law unconstitutional, or order the State to take action. Article 22 further provides that any person — not only the victim — may institute court proceedings to enforce the Bill of Rights.',
        'Read all 59 articles of the Bill of Rights on the [Constitution page](/constitution).',
      ],
    },
  ],
};

export const publicParticipation: Guide = {
  slug: 'public-participation',
  title: 'How to participate in public participation',
  description:
    'A guide to public participation in Kenya: when Parliament must invite public input, how calls for submissions work, and how to write a submission that the committee will actually read.',
  category: 'Civic Participation',
  readingTimeMinutes: 8,
  lastReviewed: '2025-01-15',
  sources: [
    {
      label: 'Constitution of Kenya, 2010 — Article 10 (national values) & Article 118 (public access)',
      url: 'https://www.kenyalaw.org/kl/index.php?id=398',
    },
    {
      label: 'Parliament of Kenya — Public Participation',
      url: 'https://www.parliament.go.ke/public-participation',
    },
    {
      label: 'Commission for the Implementation of the Constitution — Public Participation Guidelines',
      url: 'https://www.cickenya.or.ke/',
    },
  ],
  relatedLinks: [
    {
      label: 'Open participation opportunities',
      href: '/participation',
      description: 'When Parliament publishes a call for submissions, it appears here with the deadline.',
    },
    {
      label: 'Live Bills before Parliament',
      href: '/bills',
      description: 'Most calls for submissions open between Second Reading and Committee Stage.',
    },
    {
      label: 'How a Bill becomes a law',
      href: '/learn/how-a-bill-becomes-law',
      description: 'See where participation fits in the 8-stage Bill lifecycle.',
    },
    {
      label: 'Find your committee',
      href: '/committees',
      description: 'Most participation is via the relevant departmental committee, not plenary.',
    },
  ],
  sections: [
    {
      id: 'why-it-matters',
      heading: 'Why public participation matters',
      body: [
        'Public participation is not optional in Kenya. Article 10 of the Constitution lists "participation of the people" as a national value, and Article 118 obliges Parliament to facilitate public participation and involvement in the legislative and other business of Parliament and its committees.',
        'This means a Bill passed without meaningful public participation can be challenged in court. The Supreme Court has held that participation must be meaningful — not just a box-ticking exercise: the public must be given enough time, accessible information, and a real opportunity to influence the outcome.',
      ],
      source_links: [
        {
          label: 'Constitution of Kenya, 2010 — Article 10 (national values)',
          url: 'https://www.kenyalaw.org/kl/index.php?id=398',
        },
        {
          label: 'Constitution of Kenya, 2010 — Article 118 (public access)',
          url: 'https://www.kenyalaw.org/kl/index.php?id=398',
        },
      ],
    },
    {
      id: 'when-parliament-invites-input',
      heading: 'When Parliament invites public input',
      body: [
        'Public participation is invited at three points in a Bill\'s lifecycle. Pre-publication: when a department is drafting a Bill, it may publish a "concept note" or a "pre-publication draft" and invite comments — the earliest, and often the most influential, moment to weigh in. Between Second Reading and Committee Stage: after the House has voted to support the principles of the Bill in Second Reading, the Bill is committed to a departmental committee, which publishes a call for written submissions (usually with a 14-day window) — the most common participation moment. Public hearings: for Bills of national importance, committees hold in-person hearings in selected counties; anyone may request to make oral representations.',
        'Calls for submissions are published in the Kenya Gazette and on parliament.go.ke. On this platform, you can see open calls on the [Public Participation page](/participation) and subscribe to alerts via [Gazette Alerts](/gazette/alerts).',
      ],
      related_bills: [
        {
          label: 'Open participation opportunities',
          href: '/participation',
          description: 'Live list of open calls for submissions with deadlines.',
        },
        {
          label: 'Bills before Parliament',
          href: '/bills',
          description: 'Cross-reference a Bill you care about with its participation window.',
        },
      ],
    },
    {
      id: 'how-calls-work',
      heading: 'How calls for submissions work',
      body: [
        'A formal call for submissions typically includes: the Bill identifier and title (e.g. The Housing Bill, 2024 — NA Bill No. 23 of 2024); the committee receiving submissions (e.g. the Departmental Committee on Lands); the deadline (usually 14 days from the date of the call — late submissions are rarely considered); the submission channels (usually an email address like `clerk.lands@parliament.go.ke` and a physical address; some committees now accept submissions via an online form); and a hearing date for anyone who wants to make oral representations in person.',
        'You do not need to be a lawyer, an NGO, or a registered organisation to make a submission. Individual citizens are expressly invited, and committees have a statutory duty to consider every submission received.',
      ],
    },
    {
      id: 'writing-a-submission',
      heading: 'How to write a submission that gets read',
      body: [
        'Be specific about which clause you are addressing. Quote the clause number — "Clause 14 of the Bill proposes to…". A submission that addresses a specific clause is far more likely to influence the committee\'s clause-by-clause deliberation than a general statement of opposition or support.',
        'State your position clearly. Open with: "I support / oppose / propose an amendment to Clause X, for the following reasons." The committee clerk is reading hundreds of submissions — your position should be obvious from the first sentence.',
        'Give evidence. Anecdotes are accepted, but evidence is more persuasive. Cite data, reports, or personal experience. If you have a study or report, attach it. The platform\'s [Bill pages](/bills) link to all source documents you might want to reference.',
        'Propose specific wording. If you want the committee to amend a clause, propose the exact wording you would like. Committees are drafting bodies; they prefer concrete amendment text over abstract objections.',
        'Be civil, brief, and on time. Submissions should be civil and respectful — committees will disregard intemperate submissions. Aim for 2 to 4 pages. Always submit before the deadline.',
      ],
      related_bills: [
        {
          label: 'Bill source documents',
          href: '/bills',
          description: 'Each Bill page links to its gazetted text + memorandum.',
        },
      ],
    },
    {
      id: 'formats-and-accessibility',
      heading: 'Formats and accessibility',
      body: [
        'Submissions may be made in writing (email or post), in person at a public hearing, or — increasingly — through an online form. Each committee specifies its accepted channels in the call.',
        'Submissions may be in English or Kiswahili. A committee will arrange interpretation for persons with disabilities on request (per the Public Participation Guidelines). If you need accommodation, contact the committee clerk at least 7 days before the hearing.',
        'Submissions are public documents. Once received, they form part of the committee\'s record and may be published in the committee\'s report. Do not include information you want to keep confidential.',
      ],
    },
    {
      id: 'beyond-parliament',
      heading: 'Beyond Parliament — county-level participation',
      body: [
        'Public participation does not stop at Parliament. County governments have their own participation duties under the County Governments Act, 2012 — county budgets, county plans, and county Bills must all go through public participation before the county assembly votes.',
        'The participation cycle at the county level is annual: each county publishes a County Integrated Development Plan (CIDP), an Annual Development Plan (ADP), and a Budget Review and Outlook Paper (CBROP). Each document goes through a public participation window before the county assembly approves it.',
        'Your county\'s official website is the place to find county-level calls. National-level calls appear on the [Kenya Gazette](/gazette) page.',
      ],
    },
    {
      id: 'what-happens-next',
      heading: 'What happens to your submission',
      body: [
        'Once the participation window closes, the committee clerk compiles a public participation report summarising the submissions received and the issues raised. The committee considers this report alongside the Bill during clause-by-clause deliberation.',
        'The committee\'s final report (tabled at Report Stage) summarises how the committee responded to public input: which submissions it accepted, which it rejected, and why. This report is a public document — you can request a copy from the committee clerk or from the Parliamentary Hansard library.',
        'If a committee does not adequately respond to public input, the Bill is vulnerable to a constitutional challenge on the grounds that participation was not meaningful. Several Kenyan statutes have been struck down on this basis — the doctrine of meaningful participation has real legal teeth.',
      ],
    },
  ],
};

// ---------------------------------------------------------------------------
// Registry + helpers
// ---------------------------------------------------------------------------

export const guides: Guide[] = [
  howABillBecomesLaw,
  understandingParliament,
  howToReadABill,
  constitutionalRights,
  publicParticipation,
];

/** Quick slug → guide lookup used by the dynamic route. */
export function getGuideBySlug(slug: string): Guide | undefined {
  return guides.find((g) => g.slug === slug);
}

/** All slugs — used by generateStaticParams. */
export function getAllGuideSlugs(): string[] {
  return guides.map((g) => g.slug);
}

/**
 * Per-category accent colours. Each guide card on the landing page is
 * colour-coded by category so a reader can scan the grid and find what
 * they are looking for by hue. The classes are civic-* tokens so dark-mode
 * overrides in globals.css apply automatically.
 */
export const CATEGORY_STYLES: Record<
  GuideCategory,
  { badge: string; dot: string; ring: string }
> = {
  Legislation: {
    badge: 'bg-civic-leaf/10 text-civic-leaf border-civic-leaf/30',
    dot: 'bg-civic-leaf',
    ring: 'hover:border-civic-leaf/40',
  },
  Parliament: {
    badge: 'bg-civic-acacia/20 text-civic-ink border-civic-acacia/40',
    dot: 'bg-civic-acacia',
    ring: 'hover:border-civic-acacia/50',
  },
  Constitution: {
    badge: 'bg-civic-forest/10 text-civic-forest border-civic-forest/30',
    dot: 'bg-civic-forest',
    ring: 'hover:border-civic-forest/40',
  },
  'Civic Participation': {
    badge: 'bg-civic-clay/10 text-civic-clay border-civic-clay/30',
    dot: 'bg-civic-clay',
    ring: 'hover:border-civic-clay/40',
  },
  'Reading Skills': {
    badge: 'bg-civic-stone/15 text-civic-ink border-civic-border',
    dot: 'bg-civic-stone',
    ring: 'hover:border-civic-stone/50',
  },
};
