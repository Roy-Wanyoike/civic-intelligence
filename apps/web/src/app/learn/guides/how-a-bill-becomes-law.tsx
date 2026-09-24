import type { Guide } from './types';

/**
 * Guide 1 of 5 — How a Bill becomes a law in Kenya.
 *
 * Walkthrough of the canonical 8-stage progression a Bill follows from
 * drafting to commencement. Each section is one stage; the `badge` field
 * is rendered beside the heading as a "Stage N" pill so the reader can
 * see at a glance where they are in the lifecycle.
 *
 * Sources cite the Constitution of Kenya 2010 (Art. 109–115) and the
 * National Assembly Standing Orders. Every stage links to the live
 * `/bills` tracker and to the per-stage Hansard feed so the reader can
 * move from explanation → observation in one click.
 */
export const howABillBecomesLaw: Guide = {
  slug: 'how-a-bill-becomes-law',
  title: 'How a Bill becomes a law in Kenya',
  summary:
    'Follow a Bill through its 8-stage journey — from first reading to commencement — and see where public participation fits in.',
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
      title: 'Why the 8 stages matter',
      body: (
        <>
          <p>
            A Bill does not become law the day it is published. Every Bill
            must pass through a deliberately slow, public, eight-stage process
            before it binds anyone. The slowness is a feature, not a bug —
            it gives MPs, committees, and the public time to scrutinise the
            text before it acquires the force of law.
          </p>
          <p>
            The eight stages are set by{' '}
            <strong>Articles 109 to 115 of the Constitution of Kenya, 2010</strong>{' '}
            and amplified by the Standing Orders of each House. A Money Bill
            (which only the National Assembly may originate) follows the same
            path with extra rules about Senate involvement. A Constitutional
            Amendment Bill adds a fourth reading in each House and may need a
            referendum.
          </p>
          <p>
            Use the live <a href="/bills">Bills tracker</a> to see which Kenyan
            Bill is at which stage today.
          </p>
        </>
      ),
    },
    {
      id: 'stage-1-publication',
      title: 'Publication (Gazettement)',
      badge: 'Stage 1',
      body: (
        <>
          <p>
            A Bill is published in the <em>Gazette</em> as a Kenya Gazette
            Notice, accompanied by the long title and the explanatory
            memorandum. Publication is the official &ldquo;this Bill exists&rdquo;
            moment — before publication, the text may be a draft circulating
            within the Attorney-General&rsquo;s office or a sponsor&rsquo;s
            chambers, and has no formal standing.
          </p>
          <p>
            Once gazetted, the Bill is given an identifier like
            <code> NA Bill No. 23 of 2024</code> and added to the order paper.
            The sponsor (a Cabinet Secretary for a government Bill, or an MP
            for a private member&rsquo;s Bill) formally introduces it for
            First Reading.
          </p>
          <p>
            See new Bills as soon as they are gazetted on the{' '}
            <a href="/gazette">Kenya Gazette</a> page, or subscribe to{' '}
            <a href="/gazette/alerts">Gazette Alerts</a> to be notified when a
            keyword you care about appears.
          </p>
        </>
      ),
    },
    {
      id: 'stage-2-first-reading',
      title: 'First Reading',
      badge: 'Stage 2',
      body: (
        <>
          <p>
            The Clerk reads the Bill&rsquo;s long title aloud in the House.
            No debate takes place. The Bill is then committed to the relevant
            departmental committee (for a National Assembly Bill) or standing
            committee (for a Senate Bill).
          </p>
          <p>
            The committee stage that follows is where the real scrutiny begins:
            committees invite written submissions from the public, hold public
            hearings, and call the sponsor to defend the Bill clause by
            clause.
          </p>
        </>
      ),
    },
    {
      id: 'stage-3-second-reading',
      title: 'Second Reading — debate on principles',
      badge: 'Stage 3',
      body: (
        <>
          <p>
            The House debates the <strong>principles and policy</strong> of the
            Bill, not its wording. MPs who support or oppose the Bill speak;
            amendments to specific clauses are NOT permitted at this stage —
            that comes later, in committee and at report stage.
          </p>
          <p>
            If the House votes against the Bill at Second Reading, the Bill
            dies. If it votes in favour, the Bill proceeds to committee for
            clause-by-clause scrutiny.
          </p>
          <p>
            The committee&rsquo;s report (and any proposed amendments) is what
            the House will consider at Report Stage.
          </p>
        </>
      ),
    },
    {
      id: 'stage-4-committee-stage',
      title: 'Committee Stage — clause by clause',
      badge: 'Stage 4',
      body: (
        <>
          <p>
            The departmental committee goes through the Bill clause by clause,
            receiving evidence from the sponsor, ministries, experts, and the
            public. The committee may propose amendments — to fix drafting
            errors, align the Bill with existing law, or change policy.
          </p>
          <p>
            This is the stage at which <strong>public participation</strong>{' '}
            is most meaningful: committees publish calls for submissions in
            the Gazette and on parliament.go.ke, with a deadline typically
            14 days out. See <a href="/participation">open participation
            opportunities</a> on this platform.
          </p>
        </>
      ),
    },
    {
      id: 'stage-5-report-stage',
      title: 'Report Stage',
      badge: 'Stage 5',
      body: (
        <>
          <p>
            The committee tables its report — including any amendments it
            recommends — and the House considers each amendment
            individually. MPs may move further amendments at this stage,
            but only to clauses the committee has already touched (Standing
            Order limitation, designed to keep the debate focused).
          </p>
          <p>
            After all amendments are voted on, the Bill is set down for
            Third Reading.
          </p>
        </>
      ),
    },
    {
      id: 'stage-6-third-reading',
      title: 'Third Reading — final vote',
      badge: 'Stage 6',
      body: (
        <>
          <p>
            The House debates the Bill one last time — typically on whether
            the Bill, as amended, should pass. No new substantive amendments
            are permitted. A simple majority is enough for ordinary Bills;
            Constitutional Amendment Bills require a two-thirds supermajority.
          </p>
          <p>
            Once passed by one House, a Bill is transmitted to the other
            House (National Assembly ↔ Senate) where it goes through the
            same stages. Money Bills skip the Senate except where Article
            114 of the Constitution requires consultation.
          </p>
        </>
      ),
    },
    {
      id: 'stage-7-presidential-assent',
      title: 'Presidential Assent',
      badge: 'Stage 7',
      body: (
        <>
          <p>
            The Bill is presented to the President, who within 14 days must
            either:
          </p>
          <ul>
            <li>assent — sign the Bill into law; or</li>
            <li>
              refer it back to Parliament with proposed amendments (a one-time
              veto with conditions); or
            </li>
            <li>
              refer it to the Supreme Court for an advisory opinion on
              constitutionality.
            </li>
          </ul>
          <p>
            If Parliament overrides a referral by a two-thirds vote in both
            Houses, the President must assent within seven days.
          </p>
          <p>
            See recent <a href="/acts">assented Acts</a> on the platform —
            the <code>Assent</code> event is the moment a Bill becomes an Act.
          </p>
        </>
      ),
    },
    {
      id: 'stage-8-publication-and-commencement',
      title: 'Publication & Commencement',
      badge: 'Stage 8',
      body: (
        <>
          <p>
            After assent, the Act is published in the Gazette as a{' '}
            <strong>Special Issue</strong>. The Act comes into force either:
          </p>
          <ul>
            <li>on the date of publication (default); or</li>
            <li>on a date specified in the Act itself; or</li>
            <li>
              on a date to be appointed later by the Cabinet Secretary via a
              legal notice (a &ldquo;commencement order&rdquo;).
            </li>
          </ul>
          <p>
            The Act only binds Kenyans from its commencement date — an Act
            that has been assented to but has not yet commenced is not yet
            enforceable. This is why commencement orders matter: they decide
            <em>when</em> a new law starts affecting you.
          </p>
          <p>
            Browse the full lineage of any Act — from Bill to assent to
            amendments — on its <a href="/acts">Act page</a>.
          </p>
        </>
      ),
    },
    {
      id: 'special-cases',
      title: 'Special cases',
      body: (
        <>
          <h3>Money Bills</h3>
          <p>
            Money Bills (defined by Article 114) can only originate in the
            National Assembly. The Senate may recommend amendments but cannot
            block passage.
          </p>
          <h3>Constitutional Amendment Bills</h3>
          <p>
            Amendments to the Constitution require a two-thirds supermajority
            in both Houses. Some amendments (affecting the Bill of Rights,
            the territory of Kenya, or the term of Parliament) must also be
            approved by popular vote in a referendum.
          </p>
          <h3>Cabinet-origination vs. private member&rsquo;s Bills</h3>
          <p>
            Government Bills are drafted by the Office of the
            Attorney-General on instruction from Cabinet. Private
            member&rsquo;s Bills are drafted by individual MPs and are
            increasingly common — for example, several recent data-protection
            amendments originated in the Senate.
          </p>
        </>
      ),
    },
  ],
};
