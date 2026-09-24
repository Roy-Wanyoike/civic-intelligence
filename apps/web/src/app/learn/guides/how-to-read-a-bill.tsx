import type { Guide } from './types';

/**
 * Guide 3 of 5 — How to read a Bill.
 *
 * Bills follow a strict structural template set by the Office of the
 * Attorney-General. This guide walks through each section of a typical
 * Bill (title, memorandum, clauses, schedules) and gives plain-language
 * tips for first-time readers.
 */
export const howToReadABill: Guide = {
  slug: 'how-to-read-a-bill',
  title: 'How to read a Bill',
  summary:
    'A Bill is a structured legal document — title, memorandum, clauses, schedules. Learn what each part does and where to focus first.',
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
      title: 'The anatomy of a Bill',
      body: (
        <>
          <p>
            Every Kenyan Bill follows the same template, set by the Office of
            the Attorney-General&rsquo;s drafting office. Knowing the
            structure means you can skip straight to the parts you care about
            — most readers do not need to read a Bill cover-to-cover.
          </p>
          <p>The typical Bill has, in order:</p>
          <ol>
            <li>The <strong>short title</strong> — the name you will cite the Act by.</li>
            <li>The <strong>long title</strong> — one sentence describing the Bill&rsquo;s purpose.</li>
            <li>The <strong>enacting formula</strong> — the formal &ldquo;Enacted by the Parliament of Kenya&rdquo;.</li>
            <li>The <strong>explanatory memorandum</strong> — a plain-language summary of each clause.</li>
            <li>The <strong>arrangement of clauses</strong> — the table of contents.</li>
            <li>The <strong>clauses</strong> (also called <em>sections</em> once enacted) — the substantive law.</li>
            <li>The <strong>schedules</strong> — annexes of forms, lists, or detailed rules.</li>
            <li>A <strong>memorandum of objects and reasons</strong> — signed by the sponsor.</li>
          </ol>
        </>
      ),
    },
    {
      id: 'short-title',
      title: '1. The short title',
      body: (
        <>
          <p>
            The short title is the name you will use to refer to the Act once
            it is enacted. For example, <em>The Data Protection Act, 2019</em>{' '}
            is the short title — and the citation form lawyers use is
            <code> No. 24 of 2019</code> (Act number 24 of 2019).
          </p>
          <p>
            Short titles always end with a year — the year of assent, not the
            year of drafting. The year tells you which year the Bill became
            law, which is important when an older Act was amended.
          </p>
        </>
      ),
    },
    {
      id: 'long-title',
      title: '2. The long title',
      body: (
        <>
          <p>
            The long title is one sentence describing what the Bill does. It
            always begins with the word <em>&ldquo;An&rdquo;</em> — as in,
            &ldquo;An Act of Parliament to provide for the regulation of&hellip;&rdquo;.
          </p>
          <p>
            The long title is more than a label: it sets the constitutional
            scope of the Bill. Courts use the long title to decide whether a
            clause falls outside the Bill&rsquo;s stated purpose (and is
            therefore <em>ultra vires</em>).
          </p>
        </>
      ),
    },
    {
      id: 'explanatory-memorandum',
      title: '3. The explanatory memorandum',
      body: (
        <>
          <p>
            The explanatory memorandum (sometimes called the &ldquo;memorandum
            of objects and reasons&rdquo;) is a plain-language summary of each
            clause. It is written by the sponsor and circulated with the Bill.
          </p>
          <p>
            <strong>Read this first.</strong> It will not bind a court (only
            the clauses do), but it tells you what the sponsor was trying to
            do — which is the fastest way to understand a Bill before you
            read the legal text.
          </p>
          <p>
            On this platform, every Bill&rsquo;s <em>In plain language</em>{' '}
            section is an AI-generated, citation-validated rewrite of the
            memorandum. See an example on any{' '}
            <a href="/bills">live Bill page</a>.
          </p>
        </>
      ),
    },
    {
      id: 'clauses',
      title: '4. The clauses (the law itself)',
      body: (
        <>
          <p>
            Each clause is a self-contained unit of law, numbered
            <code> 1.</code>, <code> 2.</code>, <code> 3.</code>, and so on.
            A typical clause has:
          </p>
          <ul>
            <li>
              A <strong>headnote</strong> — the bold text above the clause,
              like &ldquo;Short title and commencement.&rdquo; Headnotes are
              a guide, not the law; the text of the clause is what binds.
            </li>
            <li>
              A <strong>chapeau</strong> — the opening sentence that the
              sub-clauses hang off of.
            </li>
            <li>
              <strong>Sub-clauses</strong> — labelled
              <code> (1)</code>, <code> (2)</code>, <code> (3)</code>.
            </li>
            <li>
              <strong>Paragraphs</strong> inside sub-clauses — labelled
              <code> (a)</code>, <code> (b)</code>, <code> (c)</code>.
            </li>
            <li>
              <strong>Sub-paragraphs</strong> — labelled
              <code> (i)</code>, <code> (ii)</code>, <code> (iii)</code>.
            </li>
          </ul>
          <p>
            This nesting is the single biggest reason legal text feels
            complicated. Once you know the pattern
            <code> (1)(a)(i)</code>, you can read it as if it were an
            outline.
          </p>
          <p>
            Clause <strong>1</strong> is almost always the short title and
            commencement clause. Clause <strong>2</strong> is almost always
            the interpretation clause — a glossary of defined terms used
            throughout the Bill.
          </p>
        </>
      ),
    },
    {
      id: 'schedules',
      title: '5. The schedules',
      body: (
        <>
          <p>
            Schedules are annexes attached at the end of the Bill. They
            contain material too detailed to fit in the body — for example:
          </p>
          <ul>
            <li>the form of an oath or certificate;</li>
            <li>the list of bodies to which an Act applies;</li>
            <li>the numerical fees, rates, or quotas the Act sets; or</li>
            <li>the text of consequential amendments to other Acts.</li>
          </ul>
          <p>
            Schedules are <em>part of the law</em> — a court will treat a
            schedule the same as a clause. Always check the schedules for the
            &ldquo;consequential amendments&rdquo; schedule, which lists the
            other Acts this Bill will change.
          </p>
        </>
      ),
    },
    {
      id: 'reading-tips',
      title: 'Plain-language reading tips',
      body: (
        <>
          <h3>1. Read the memorandum first, then skip to the clauses you care about</h3>
          <p>
            Use the explanatory memorandum as a map. Once you know which
            clause does what, you only need to read the two or three clauses
            that affect you — not the entire Bill.
          </p>
          <h3>2. Trace the defined terms</h3>
          <p>
            Defined terms are written with a capital letter:
            <code> &ldquo;the Authority&rdquo;</code>,
            <code> &ldquo;personal data&rdquo;</code>. Look them up in the
            interpretation clause (usually clause 2) before reading further —
            the Bill&rsquo;s meaning depends on these definitions.
          </p>
          <h3>3. Watch out for &ldquo;may&rdquo; vs. &ldquo;shall&rdquo;</h3>
          <p>
            <em>&ldquo;Shall&rdquo;</em> in a Bill means <strong>must</strong>{' '}
            — it imposes a duty. <em>&ldquo;May&rdquo;</em> means{' '}
            <strong>is permitted to</strong> — it confers a power or a
            discretion. A single word can change the entire effect of a
            clause.
          </p>
          <h3>4. Read the &ldquo;consequential amendments&rdquo; schedule</h3>
          <p>
            This is where a Bill quietly changes other Acts. A small clause
            in a Bill can rewrite large parts of an existing Act — and the
            consequential amendments schedule is where you find out.
          </p>
          <h3>5. Use the platform&rsquo;s version comparison</h3>
          <p>
            Every Bill has multiple published versions — the original, the
            committee-amended text, and the as-enacted text. Use the{' '}
            <a href="/bills">Versions tab</a> on any Bill to see exactly what
            changed between versions.
          </p>
        </>
      ),
    },
  ],
};
