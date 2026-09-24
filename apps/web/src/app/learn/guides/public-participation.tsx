import type { Guide } from './types';

/**
 * Guide 5 of 5 — How to participate in public participation.
 *
 * Public participation is a constitutional value (Article 10) and a
 * procedural requirement (Article 118) — Parliament must facilitate
 * public input before passing most Bills. This guide explains when
 * participation happens, how committees invite it, and what a good
 * submission looks like.
 */
export const publicParticipation: Guide = {
  slug: 'public-participation',
  title: 'How to participate in public participation',
  summary:
    'When and how Parliament invites public input on Bills and county plans — plus what a good submission looks like.',
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
      title: 'Why public participation matters',
      body: (
        <>
          <p>
            Public participation is not optional in Kenya. Article 10 of the
            Constitution lists &ldquo;participation of the people&rdquo; as a
            national value, and Article 118 obliges Parliament to{' '}
            <strong>facilitate public participation and involvement in the
            legislative and other business of Parliament and its
            committees</strong>.
          </p>
          <p>
            This means a Bill passed without meaningful public participation
            can be challenged in court. The Supreme Court has held that
            participation must be <em>meaningful</em> — not just a box-ticking
            exercise: the public must be given enough time, accessible
            information, and a real opportunity to influence the outcome.
          </p>
        </>
      ),
    },
    {
      id: 'when-parliament-invites-input',
      title: 'When Parliament invites public input',
      body: (
        <>
          <p>
            Public participation is invited at three points in a Bill&rsquo;s
            lifecycle:
          </p>
          <ul>
            <li>
              <strong>Pre-publication</strong> — when a department is drafting
              a Bill, it may publish a &ldquo;concept note&rdquo; or a
              &ldquo;pre-publication draft&rdquo; and invite comments. This is
              the earliest — and often the most influential — moment to weigh
              in.
            </li>
            <li>
              <strong>Between Second Reading and Committee Stage</strong> —
              after the House has voted to support the principles of the Bill
              in Second Reading, the Bill is committed to a departmental
              committee. The committee publishes a call for written
              submissions, usually with a 14-day window. This is the most
              common participation moment.
            </li>
            <li>
              <strong>Public hearings</strong> — for Bills of national
              importance, committees hold in-person hearings in selected
              counties. Anyone may request to make oral representations.
            </li>
          </ul>
          <p>
            Calls for submissions are published in the Kenya Gazette and on
            parliament.go.ke. On this platform, you can see open calls on the{' '}
            <a href="/participation">Public Participation page</a> and
            subscribe to alerts via{' '}
            <a href="/gazette/alerts">Gazette Alerts</a>.
          </p>
        </>
      ),
    },
    {
      id: 'how-calls-work',
      title: 'How calls for submissions work',
      body: (
        <>
          <p>
            A formal call for submissions typically includes:
          </p>
          <ul>
            <li>
              The <strong>Bill identifier and title</strong> (e.g.
              <em> The Housing Bill, 2024 — NA Bill No. 23 of 2024</em>).
            </li>
            <li>
              The <strong>committee</strong> receiving submissions (e.g. the
              Departmental Committee on Lands).
            </li>
            <li>
              The <strong>deadline</strong> — usually 14 days from the date
              of the call. Late submissions are rarely considered.
            </li>
            <li>
              The <strong>submission channels</strong> — usually an email
              address (e.g.
              <code> clerk.lands@parliament.go.ke</code>) and a physical
              address. Some committees now accept submissions via an online
              form.
            </li>
            <li>
              A <strong>hearing date</strong> for anyone who wants to make
              oral representations in person.
            </li>
          </ul>
          <p>
            You do not need to be a lawyer, an NGO, or a registered
            organisation to make a submission. Individual citizens are
            expressly invited, and committees have a statutory duty to
            consider every submission received.
          </p>
        </>
      ),
    },
    {
      id: 'writing-a-submission',
      title: 'How to write a submission that gets read',
      body: (
        <>
          <h3>1. Be specific about which clause you are addressing</h3>
          <p>
            Quote the clause number — &ldquo;Clause 14 of the Bill proposes
            to&hellip;&rdquo;. A submission that addresses a specific clause is
            far more likely to influence the committee&rsquo;s
            clause-by-clause deliberation than a general statement of
            opposition or support.
          </p>
          <h3>2. State your position clearly</h3>
          <p>
            Open with: &ldquo;I support / oppose / propose an amendment to
            Clause X, for the following reasons.&rdquo; The committee clerk
            is reading hundreds of submissions — your position should be
            obvious from the first sentence.
          </p>
          <h3>3. Give evidence</h3>
          <p>
            Anecdotes are accepted, but evidence is more persuasive. Cite
            data, reports, or personal experience. If you have a study or
            report, attach it. The platform&rsquo;s{' '}
            <a href="/bills">Bill pages</a> link to all source documents you
            might want to reference.
          </p>
          <h3>4. Propose specific wording</h3>
          <p>
            If you want the committee to amend a clause, propose the exact
            wording you would like. Committees are drafting bodies; they
            prefer concrete amendment text over abstract objections.
          </p>
          <h3>5. Be civil, brief, and on time</h3>
          <p>
            Submissions should be civil and respectful — committees will
            disregard intemperate submissions. Aim for 2 to 4 pages. Always
            submit before the deadline.
          </p>
        </>
      ),
    },
    {
      id: 'formats-and-accessibility',
      title: 'Formats and accessibility',
      body: (
        <>
          <p>
            Submissions may be made in writing (email or post), in person at
            a public hearing, or — increasingly — through an online form.
            Each committee specifies its accepted channels in the call.
          </p>
          <p>
            Submissions may be in <strong>English or Kiswahili</strong>. A
            committee will arrange interpretation for persons with
            disabilities on request (per the Public Participation
            Guidelines). If you need accommodation, contact the committee
            clerk at least 7 days before the hearing.
          </p>
          <p>
            Submissions are <strong>public documents</strong>. Once received,
            they form part of the committee&rsquo;s record and may be
            published in the committee&rsquo;s report. Do not include
            information you want to keep confidential.
          </p>
        </>
      ),
    },
    {
      id: 'beyond-parliament',
      title: 'Beyond Parliament — county-level participation',
      body: (
        <>
          <p>
            Public participation does not stop at Parliament. County
            governments have their own participation duties under the
            County Governments Act, 2012 — county budgets, county plans, and
            county Bills must all go through public participation before
            the county assembly votes.
          </p>
          <p>
            The participation cycle at the county level is annual: each
            county publishes a County Integrated Development Plan (CIDP), an
            Annual Development Plan (ADP), and a Budget Review and Outlook
            Paper (CBROP). Each document goes through a public participation
            window before the county assembly approves it.
          </p>
          <p>
            Your county&rsquo;s official website is the place to find
            county-level calls. National-level calls appear on the{' '}
            <a href="/gazette">Kenya Gazette</a> page.
          </p>
        </>
      ),
    },
    {
      id: 'what-happens-next',
      title: 'What happens to your submission',
      body: (
        <>
          <p>
            Once the participation window closes, the committee clerk compiles
            a <strong>public participation report</strong> summarising the
            submissions received and the issues raised. The committee
            considers this report alongside the Bill during clause-by-clause
            deliberation.
          </p>
          <p>
            The committee&rsquo;s final report (tabled at Report Stage)
            summarises how the committee responded to public input: which
            submissions it accepted, which it rejected, and why. This report
            is a public document — you can request a copy from the committee
            clerk or from the Parliamentary Hansard library.
          </p>
          <p>
            If a committee does not adequately respond to public input, the
            Bill is vulnerable to a constitutional challenge on the grounds
            that participation was not meaningful. Several Kenyan statutes
            have been struck down on this basis — the doctrine of meaningful
            participation has real legal teeth.
          </p>
        </>
      ),
    },
  ],
};
