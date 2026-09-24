import type { Guide } from './types';

/**
 * Guide 2 of 5 — Understanding Kenya's Parliament.
 *
 * Explains the bicameral structure (National Assembly + Senate), the
 * division of responsibilities between the two Houses, and how the
 * committee system does most of the substantive work. Links to the
 * `/legislatures`, `/committees`, and `/people` pages so the reader can
 * drill into a specific legislature or committee.
 */
export const understandingParliament: Guide = {
  slug: 'understanding-parliament',
  title: "Understanding Kenya's Parliament",
  summary:
    'Kenya has a bicameral Parliament — the National Assembly and the Senate. Learn what each House does, where they overlap, and how committees carry the real work.',
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
      title: 'Why two Houses?',
      body: (
        <>
          <p>
            The 2010 Constitution replaced Kenya&rsquo;s pre-2010 unicameral
            National Assembly with a <strong>bicameral Parliament</strong> made
            up of two Houses:
          </p>
          <ul>
            <li>
              <strong>The National Assembly</strong> — 349 elected members
              plus 12 nominated, representing the people directly.
            </li>
            <li>
              <strong>The Senate</strong> — 47 elected senators (one per
              county) plus nominated members, representing the counties.
            </li>
          </ul>
          <p>
            The reason for two Houses is devolution. Once Kenya created 47
            county governments with real powers and budgets, the counties
            needed a voice at the national level to defend their interests
            against central encroachment. The Senate is that voice.
          </p>
        </>
      ),
    },
    {
      id: 'unicameral-vs-bicameral',
      title: 'Unicameral vs. bicameral — a brief comparison',
      body: (
        <>
          <p>
            Before the 2010 Constitution, Kenya was unicameral: a single
            National Assembly passed all laws. Unicameral systems are simpler
            and faster, but they offer only one round of scrutiny. A second
            House forces a Bill to be examined twice — by different people,
            with different incentives — before it binds anyone.
          </p>
          <p>
            The trade-off is speed vs. deliberation. A bicameral Parliament
            can take longer to pass a Bill because both Houses must agree on
            the final text. Where the two Houses disagree, the Constitution
            provides a mediation committee (see <em>Mediation</em> below).
          </p>
        </>
      ),
    },
    {
      id: 'national-assembly',
      title: 'The National Assembly',
      body: (
        <>
          <p>
            The National Assembly represents the people directly. Its
            responsibilities include:
          </p>
          <ul>
            <li>
              <strong>Legislation</strong> — ordinary Bills can originate in
              either House, but <em>Money Bills</em> (taxes, budget, public
              debt) originate only in the National Assembly.
            </li>
            <li>
              <strong>Budget approval</strong> — the Division of Revenue Act
              and the Appropriation Act originate here.
            </li>
            <li>
              <strong>Oversight</strong> — the Assembly summons Cabinet
              Secretaries and reviews the conduct of the Executive.
            </li>
            <li>
              <strong>Removing a President</strong> — impeachment is initiated
              in the National Assembly and tried in the Senate.
            </li>
          </ul>
          <p>
            The Assembly is led by the <strong>Speaker of the National
            Assembly</strong>, elected by members. The leader of the largest
            party or coalition becomes the <strong>Leader of the Majority
            Party</strong>; the runner-up becomes the <strong>Leader of the
            Official Opposition</strong>.
          </p>
          <p>
            See your constituency&rsquo;s MP on the{' '}
            <a href="/constituencies">Constituencies</a> page.
          </p>
        </>
      ),
    },
    {
      id: 'senate',
      title: 'The Senate',
      body: (
        <>
          <p>
            The Senate represents the 47 counties. Its core responsibilities
            are:
          </p>
          <ul>
            <li>
              <strong>County revenue allocation</strong> — the Senate
              determines how national revenue is shared among counties each
              year (the County Allocation of Revenue Act).
            </li>
            <li>
              <strong>County oversight</strong> — the Senate scrutinises
              county budgets and considers countyGovernance-related Bills.
            </li>
            <li>
              <strong>Impeachment trial</strong> — a Governor is impeached by
              the Senate after the county assembly votes to remove them.
            </li>
            <li>
              <strong>Legislation affecting counties</strong> — any Bill that
              touches on county government must be considered by the Senate.
            </li>
          </ul>
          <p>
            The Senate is led by the <strong>Speaker of the Senate</strong>.
            Each county&rsquo;s senator is elected directly by the
            county&rsquo;s voters; there are also nominated senators
            representing women, youth, and persons with disabilities.
          </p>
          <p>
            See senators on the <a href="/people">People</a> page, filtered
            by house.
          </p>
        </>
      ),
    },
    {
      id: 'when-houses-disagree',
      title: 'When the two Houses disagree — mediation',
      body: (
        <>
          <p>
            If one House passes a Bill and the other rejects it or passes it
            with amendments the first House won&rsquo;t accept, the Bill goes
            to a <strong>Mediation Committee</strong> — a joint committee of
            equal members from both Houses.
          </p>
          <p>
            The committee attempts to produce a version both Houses can
            accept. If mediation fails, the Bill lapses — it does not become
            law. This is a deliberate brake that forces the two Houses to
            negotiate rather than the larger House simply overriding the
            smaller.
          </p>
          <p>
            Money Bills are an exception: the National Assembly can pass a
            Money Bill even if the Senate does not agree, but the Senate
            must be heard.
          </p>
        </>
      ),
    },
    {
      id: 'committee-system',
      title: 'The committee system — where the real work happens',
      body: (
        <>
          <p>
            Plenary debates in the Chamber get the headlines, but the
            substantive scrutiny of Bills happens in committees. Each House
            has two kinds of committees:
          </p>
          <ul>
            <li>
              <strong>Departmental / Standing Committees</strong> — mirror
              the work of a ministry (e.g. the Departmental Committee on
              Health scrutinises the Ministry of Health&rsquo;s Bills and
              budget). These committees take evidence, hold public hearings,
              and recommend amendments clause-by-clause.
            </li>
            <li>
              <strong>Select Committees</strong> — set up for a specific
              purpose (e.g. the Powers and Privileges Committee, the
              Committee on Implementation of the Constitution).
            </li>
          </ul>
          <p>
            Committee membership is sized so that every MP sits on at least
            one committee. Committees meet weekly when Parliament is sitting
            and may meet during recess by special permission.
          </p>
          <p>
            See all committees and their current chairs on the{' '}
            <a href="/committees">Committees</a> page.
          </p>
        </>
      ),
    },
    {
      id: 'how-a-bill-moves',
      title: 'How a Bill moves between the Houses',
      body: (
        <>
          <p>
            For ordinary Bills (not Money Bills), the Bill can originate in
            either House. After the originating House passes the Bill, it is
            transmitted to the other House for concurrence. The second House
            can:
          </p>
          <ul>
            <li>pass the Bill as-is — the Bill goes to the President;</li>
            <li>
              pass the Bill with amendments — the Bill returns to the
              originating House for reconsideration; or
            </li>
            <li>reject the Bill — triggering mediation.</li>
          </ul>
          <p>
            See the full lifecycle in our{' '}
            <a href="/learn/how-a-bill-becomes-law">&ldquo;How a Bill becomes
            a law&rdquo; guide</a>.
          </p>
        </>
      ),
    },
    {
      id: 'sittings-and-calendar',
      title: 'Sittings and the parliamentary calendar',
      body: (
        <>
          <p>
            A typical parliamentary year runs from February to November, with
            recess periods in April, August, and December. Plenary sittings
            are on Tuesday afternoons, Wednesday mornings and afternoons, and
            Thursday afternoons.
          </p>
          <p>
            Committee sittings happen almost daily during session weeks.
            Follow the <a href="/calendar">civic calendar</a> to see what is
            scheduled this week.
          </p>
        </>
      ),
    },
  ],
};
