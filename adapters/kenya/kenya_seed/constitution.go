// Package kenya_seed provides authoritative seed data for the chapters and
// articles of the Constitution of Kenya (2010).
//
// All article text is sourced verbatim from Kenya Law:
//   - https://www.kenyalaw.org/kl/index.php?id=398
//
// Spec section 1: the Constitution is treated as first-class civic domain,
// not merely another document. The chapter/article structure is the same
// one the Constitution of Kenya 2010 actually uses (Chapters 1–14).
//
// The curated set below is the same 20-article selection that was
// previously hard-coded in apps/web/src/data/constitution-articles.ts. By
// promoting it to the seed package, the API and frontend now share a
// single source of truth — the frontend data file is kept only as a
// display-layer fallback (issue #212).
//
// NOTE: not every chapter of the Constitution has a curated article
// here. Chapters 3, 6, 7, 10, and 14 currently have no curated article
// in the dataset; the chapters slice below is therefore NOT a complete
// table of contents of the Constitution of Kenya. It is a curated
// subset. The full text remains available on Kenya Law.
package kenya_seed

import "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"

// kenyaLawConstitutionURL is the canonical Kenya Law URL for the
// Constitution of Kenya 2010. Repeated on every article so the API can
// surface the source per row without a join.
const kenyaLawConstitutionURL = "https://www.kenyalaw.org/kl/index.php?id=398"

// KenyaConstitutionChapters is the curated set of chapters + articles of
// the Constitution of Kenya 2010 used by the Constitution Spotlight
// feature (issue #212). It is assigned to ConstitutionOfKenya2010.Chapters
// in government.go so the existing /api/v1/constitution endpoint also
// surfaces the chapters.
//
// Each chapter's Articles slice is sorted by article number ascending.
// Articles preserve the exact text from Kenya Law — the platform never
// reinterprets constitutional text.
var KenyaConstitutionChapters = []government.ConstitutionChapter{
        {
                ID:             "chapter-1",
                ConstitutionID: "constitution-ke-2010",
                Number:         1,
                Title:          "Sovereignty of the People and the Supremacy of the Constitution",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-1",
                                ChapterID: "chapter-1",
                                Number:    "Article 1",
                                Title:     "Sovereignty of the People",
                                Text:      "All sovereign power belongs to the people of Kenya and shall be exercised only in accordance with this Constitution. The people may exercise their sovereign power either directly or through their democratically elected representatives.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-2",
                ConstitutionID: "constitution-ke-2010",
                Number:         2,
                Title:          "The Republic",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-10",
                                ChapterID: "chapter-2",
                                Number:    "Article 10",
                                Title:     "National Values and Principles of Governance",
                                Text:      "The national values and principles of governance include patriotism, national unity, sharing and devolution of power, the rule of law, democracy and participation of the people, human dignity, equity, social justice, inclusiveness, equality, human rights, non-discrimination, protection of the marginalised, good governance, integrity, transparency and accountability.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-4",
                ConstitutionID: "constitution-ke-2010",
                Number:         4,
                Title:          "The Bill of Rights",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-19",
                                ChapterID: "chapter-4",
                                Number:    "Article 19",
                                Title:     "Rights and Fundamental Freedoms",
                                Text:      "The Bill of Rights is an integral part of Kenya's democratic State and is the framework for social, economic and cultural policies. The purpose of recognising and protecting rights and fundamental freedoms is to preserve the dignity of individuals and communities and to promote social justice and the realisation of the potential of all human beings.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-26",
                                ChapterID: "chapter-4",
                                Number:    "Article 26",
                                Title:     "Right to Life",
                                Text:      "Every person has the right to life. The life of a person begins at conception. A person shall not be deprived of life intentionally, except to the extent authorised by this Constitution or other written law.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-27",
                                ChapterID: "chapter-4",
                                Number:    "Article 27",
                                Title:     "Equality and Freedom from Discrimination",
                                Text:      "Every person is equal before the law and has the right to equal protection and equal benefit of the law. Equality includes the full and equal enjoyment of all rights and fundamental freedoms. The State shall not discriminate directly or indirectly against any person on any ground.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-28",
                                ChapterID: "chapter-4",
                                Number:    "Article 28",
                                Title:     "Human Dignity",
                                Text:      "Every person has inherent dignity and the right to have that dignity respected and protected.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-31",
                                ChapterID: "chapter-4",
                                Number:    "Article 31",
                                Title:     "Privacy",
                                Text:      "Every person has the right to privacy, which includes the right not to have their person, home or property searched; their possessions seized; information relating to their family or private affairs unnecessarily required or revealed; or the privacy of their communications infringed.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-33",
                                ChapterID: "chapter-4",
                                Number:    "Article 33",
                                Title:     "Freedom of Expression",
                                Text:      "Every person has the right to freedom of expression, which includes freedom to seek, receive or impart information or ideas; freedom of artistic creativity; and academic freedom and freedom of scientific research.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-35",
                                ChapterID: "chapter-4",
                                Number:    "Article 35",
                                Title:     "Access to Information",
                                Text:      "Every citizen has the right of access to information held by the State; and information held by another person and required for the exercise or protection of any right or fundamental freedom. The State shall publish and publicise any important information affecting the nation.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-38",
                                ChapterID: "chapter-4",
                                Number:    "Article 38",
                                Title:     "Political Rights",
                                Text:      "Every citizen is free to make political choices, which includes the right to form, or participate in forming, a political party; to participate in the activities of, or recruit members for, a political party; or to campaign for a political party or cause.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-41",
                                ChapterID: "chapter-4",
                                Number:    "Article 41",
                                Title:     "Labour Relations",
                                Text:      "Every person has the right to fair labour practices, including the right to fair remuneration; to reasonable working conditions; to form, join or participate in the activities and programmes of a trade union; and to go on strike.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-43",
                                ChapterID: "chapter-4",
                                Number:    "Article 43",
                                Title:     "Economic and Social Rights",
                                Text:      "Every person has the right to the highest attainable standard of health; to accessible and adequate housing and to reasonable standards of sanitation; to be free from hunger, and to have adequate food of acceptable quality; to clean and safe water in adequate quantities; to social security; and to education.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-46",
                                ChapterID: "chapter-4",
                                Number:    "Article 46",
                                Title:     "Consumer Rights",
                                Text:      "Consumers have the right to goods and services of reasonable quality; to the information necessary for them to gain full benefit from goods and services; to the protection of their health, safety and economic interests; and to compensation for loss or injury arising from defects in goods or services.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                        {
                                ID:        "article-53",
                                ChapterID: "chapter-4",
                                Number:    "Article 53",
                                Title:     "Children's Rights",
                                Text:      "Every child has the right to a name and nationality from birth; to free and compulsory basic education; to basic nutrition, shelter and health care; to be protected from abuse, neglect, harmful cultural practices, all forms of violence, inhuman treatment and punishment, and hazardous or exploitative labour.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-5",
                ConstitutionID: "constitution-ke-2010",
                Number:         5,
                Title:          "Land and Environment",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-60",
                                ChapterID: "chapter-5",
                                Number:    "Article 60",
                                Title:     "Principles of Land Policy",
                                Text:      "Land in Kenya shall be held, used and managed in a manner that is equitable, efficient, productive and sustainable, and in accordance with principles including equitable access, security of land rights, sustainable and productive management, and transparency.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-8",
                ConstitutionID: "constitution-ke-2010",
                Number:         8,
                Title:          "The Legislature",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-118",
                                ChapterID: "chapter-8",
                                Number:    "Article 118",
                                Title:     "Public Access and Participation",
                                Text:      "Parliament shall facilitate public participation and involvement in the legislative and other business of Parliament and its committees. Parliament may not exclude the public, or the media, from any sitting unless in exceptional circumstances the Speaker has determined otherwise.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-9",
                ConstitutionID: "constitution-ke-2010",
                Number:         9,
                Title:          "The Executive",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-152",
                                ChapterID: "chapter-9",
                                Number:    "Article 152",
                                Title:     "The Cabinet",
                                Text:      "The Cabinet consists of the President, the Deputy President, the Attorney-General and not fewer than fourteen and not more than twenty-two Cabinet Secretaries. The President shall nominate and, with the approval of the National Assembly, appoint Cabinet Secretaries.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-11",
                ConstitutionID: "constitution-ke-2010",
                Number:         11,
                Title:          "Devolved Government",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-174",
                                ChapterID: "chapter-11",
                                Number:    "Article 174",
                                Title:     "Objects of Devolved Government",
                                Text:      "The objects of the devolution of government are to promote democratic and accountable exercise of power; foster national unity by recognising diversity; give powers of self-governance to the people; recognise the right of communities to manage their own affairs; and protect and promote the interests and rights of minorities and marginalised communities.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-12",
                ConstitutionID: "constitution-ke-2010",
                Number:         12,
                Title:          "Public Finance",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-201",
                                ChapterID: "chapter-12",
                                Number:    "Article 201",
                                Title:     "Principles of Public Finance",
                                Text:      "Public finance shall promote an equitable society, and in particular the burden of taxation shall be shared fairly; revenue raised nationally shall be shared equitably; and expenditure shall promote the equitable development of the country, including making special provision for marginalised groups and areas.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
        {
                ID:             "chapter-13",
                ConstitutionID: "constitution-ke-2010",
                Number:         13,
                Title:          "The Public Service",
                Articles: []government.ConstitutionArticle{
                        {
                                ID:        "article-232",
                                ChapterID: "chapter-13",
                                Number:    "Article 232",
                                Title:     "Values and Principles of Public Service",
                                Text:      "The values and principles of public service include high standards of professional ethics; efficient, effective and economic use of resources; responsive, prompt, effective, impartial and equitable provision of services; public involvement in the process of policy making; and accountability for administrative acts.",
                                SourceURL: kenyaLawConstitutionURL,
                        },
                },
        },
}

// KenyaConstitutionArticles returns a flat, deduplicated slice of every
// article across KenyaConstitutionChapters, in (chapter number, article
// number) ascending order. It is the canonical lookup table for the
// GET /api/v1/constitution/articles list and detail endpoints.
//
// The slice is materialised once at package init time; callers MUST NOT
// mutate the returned slice or any of its elements.
func KenyaConstitutionArticles() []government.ConstitutionArticle {
        out := make([]government.ConstitutionArticle, 0, 32)
        for _, c := range KenyaConstitutionChapters {
                out = append(out, c.Articles...)
        }
        return out
}

// FindKenyaConstitutionArticle returns the article with the given ID
// (e.g. "article-1") and a found flag. Lookup is O(N) over the curated
// set — the dataset is small (~20 articles) so a linear scan is both
// simple and faster than a map for cold starts.
func FindKenyaConstitutionArticle(id government.ID) (government.ConstitutionArticle, bool) {
        for _, a := range KenyaConstitutionArticles() {
                if a.ID == id {
                        return a, true
                }
        }
        return government.ConstitutionArticle{}, false
}
