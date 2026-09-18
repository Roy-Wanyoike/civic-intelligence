// Constitution of Kenya 2010 — selected articles for the Constitution Spotlight feature.
// Source: https://www.kenyalaw.org/kl/index.php?id=398
// Each article has: number, title, chapter, text, type (fact/awareness)

export interface ConstitutionArticle {
  number: string;
  title: string;
  chapter: string;
  text: string;
  type: 'fact' | 'awareness';
  source: string;
}

export const constitutionArticles: ConstitutionArticle[] = [
  {
    number: 'Article 1',
    title: 'Sovereignty of the People',
    chapter: 'Chapter 1 — Sovereignty of the People and the Supremacy of the Constitution',
    text: 'All sovereign power belongs to the people of Kenya and shall be exercised only in accordance with this Constitution. The people may exercise their sovereign power either directly or through their democratically elected representatives.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 10',
    title: 'National Values and Principles of Governance',
    chapter: 'Chapter 2 — The Republic',
    text: 'The national values and principles of governance include patriotism, national unity, sharing and devolution of power, the rule of law, democracy and participation of the people, human dignity, equity, social justice, inclusiveness, equality, human rights, non-discrimination, protection of the marginalised, good governance, integrity, transparency and accountability.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 19',
    title: 'Rights and Fundamental Freedoms',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'The Bill of Rights is an integral part of Kenya\'s democratic State and is the framework for social, economic and cultural policies. The purpose of recognising and protecting rights and fundamental freedoms is to preserve the dignity of individuals and communities and to promote social justice and the realisation of the potential of all human beings.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 26',
    title: 'Right to Life',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person has the right to life. The life of a person begins at conception. A person shall not be deprived of life intentionally, except to the extent authorised by this Constitution or other written law.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 27',
    title: 'Equality and Freedom from Discrimination',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person is equal before the law and has the right to equal protection and equal benefit of the law. Equality includes the full and equal enjoyment of all rights and fundamental freedoms. The State shall not discriminate directly or indirectly against any person on any ground.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 28',
    title: 'Human Dignity',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person has inherent dignity and the right to have that dignity respected and protected.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 31',
    title: 'Privacy',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person has the right to privacy, which includes the right not to have their person, home or property searched; their possessions seized; information relating to their family or private affairs unnecessarily required or revealed; or the privacy of their communications infringed.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 33',
    title: 'Freedom of Expression',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person has the right to freedom of expression, which includes freedom to seek, receive or impart information or ideas; freedom of artistic creativity; and academic freedom and freedom of scientific research.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 35',
    title: 'Access to Information',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every citizen has the right of access to information held by the State; and information held by another person and required for the exercise or protection of any right or fundamental freedom. The State shall publish and publicise any important information affecting the nation.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 38',
    title: 'Political Rights',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every citizen is free to make political choices, which includes the right to form, or participate in forming, a political party; to participate in the activities of, or recruit members for, a political party; or to campaign for a political party or cause.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 41',
    title: 'Labour Relations',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person has the right to fair labour practices, including the right to fair remuneration; to reasonable working conditions; to form, join or participate in the activities and programmes of a trade union; and to go on strike.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 43',
    title: 'Economic and Social Rights',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every person has the right to the highest attainable standard of health; to accessible and adequate housing and to reasonable standards of sanitation; to be free from hunger, and to have adequate food of acceptable quality; to clean and safe water in adequate quantities; to social security; and to education.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 46',
    title: 'Consumer Rights',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Consumers have the right to goods and services of reasonable quality; to the information necessary for them to gain full benefit from goods and services; to the protection of their health, safety and economic interests; and to compensation for loss or injury arising from defects in goods or services.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 53',
    title: 'Children\'s Rights',
    chapter: 'Chapter 4 — The Bill of Rights',
    text: 'Every child has the right to a name and nationality from birth; to free and compulsory basic education; to basic nutrition, shelter and health care; to be protected from abuse, neglect, harmful cultural practices, all forms of violence, inhuman treatment and punishment, and hazardous or exploitative labour.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 60',
    title: 'Principles of Land Policy',
    chapter: 'Chapter 5 — Land and Environment',
    text: 'Land in Kenya shall be held, used and managed in a manner that is equitable, efficient, productive and sustainable, and in accordance with principles including equitable access, security of land rights, sustainable and productive management, and transparency.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 118',
    title: 'Public Access and Participation',
    chapter: 'Chapter 8 — The Legislature',
    text: 'Parliament shall facilitate public participation and involvement in the legislative and other business of Parliament and its committees. Parliament may not exclude the public, or the media, from any sitting unless in exceptional circumstances the Speaker has determined otherwise.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 152',
    title: 'The Cabinet',
    chapter: 'Chapter 9 — The Executive',
    text: 'The Cabinet consists of the President, the Deputy President, the Attorney-General and not fewer than fourteen and not more than twenty-two Cabinet Secretaries. The President shall nominate and, with the approval of the National Assembly, appoint Cabinet Secretaries.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 174',
    title: 'Objects of Devolved Government',
    chapter: 'Chapter 11 — Devolved Government',
    text: 'The objects of the devolution of government are to promote democratic and accountable exercise of power; foster national unity by recognising diversity; give powers of self-governance to the people; recognise the right of communities to manage their own affairs; and protect and promote the interests and rights of minorities and marginalised communities.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 201',
    title: 'Principles of Public Finance',
    chapter: 'Chapter 12 — Public Finance',
    text: 'Public finance shall promote an equitable society, and in particular the burden of taxation shall be shared fairly; revenue raised nationally shall be shared equitably; and expenditure shall promote the equitable development of the country, including making special provision for marginalised groups and areas.',
    type: 'awareness',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
  {
    number: 'Article 232',
    title: 'Values and Principles of Public Service',
    chapter: 'Chapter 13 — The Public Service',
    text: 'The values and principles of public service include high standards of professional ethics; efficient, effective and economic use of resources; responsive, prompt, effective, impartial and equitable provision of services; public involvement in the process of policy making; and accountability for administrative acts.',
    type: 'fact',
    source: 'https://www.kenyalaw.org/kl/index.php?id=398',
  },
];
