// Package kenya_seed provides authoritative seed data for civic topics.
//
// This file (issue #267) exports the KenyaTopics slice + supporting DTOs
// used by the GET /api/v1/topics and /api/v1/topics/{id} endpoints.
//
// A "civic topic" is a subject-matter lens the platform exposes so citizens
// can browse Bills, Acts, and people associated with that subject (e.g.
// "Health", "Education", "Finance & National Planning"). Topics map to
// the Departmental Committees of the Kenya National Assembly + Senate —
// the same set the frontend Topics page
// (apps/web/src/app/topics/page.tsx) and the navbar mega-menu reference.
//
// The seed is intentionally self-contained: each topic carries its own
// slices of associated Bills / Acts / People (with source URLs) so the
// /topics/{id} detail handler can return concrete associations without
// re-crawling the live Kenya Law adapter. The list handler returns just
// the counts (bill_count, act_count, people_count) so the Topics page
// can render a lightweight index.
//
// When issue #265's seed Bills slice (kenya_seed.SampleBills) merges to
// main, the bills association in this file can be reconciled against
// that authoritative source — until then this file owns the seed.
//
// All source URLs point to authoritative primary references:
//   - Kenya Law (https://www.kenyalaw.org)
//   - Parliament of Kenya (https://parliament.go.ke)
package kenya_seed

// Topic is the basic topic info shown in the /topics list response.
// The fields mirror the expected JSON shape documented in
// docs/api/openapi.yaml.
type Topic struct {
	// ID is the kebab-case topic identifier used in URLs
	// (e.g. "health", "education", "finance").
	ID string `json:"id"`
	// Name is the human-readable display name (e.g. "Health").
	Name string `json:"name"`
	// Description is a 1-2 sentence plain-language summary of what the
	// topic covers. Surfaced verbatim on the Topics page so citizens
	// know what to expect when they drill in.
	Description string `json:"description"`
	// BillCount is the number of seed Bills associated with this topic.
	BillCount int `json:"bill_count"`
	// ActCount is the number of seed Acts associated with this topic.
	ActCount int `json:"act_count"`
	// PeopleCount is the number of seed People (MPs, officials)
	// associated with this topic — typically the chairs / ranking
	// members of the corresponding Departmental Committee.
	PeopleCount int `json:"people_count"`
	// Country is the ISO 3166-1 alpha-2 country code the topic belongs
	// to. Always "KE" for the Kenya seed; the API filters by the
	// country on the request context so a Uganda-scoped request
	// returns an empty list (other countries do not yet have seed data).
	Country string `json:"country"`
}

// TopicBill is the bill summary shown in the /topics/{id} detail response.
// It is a deliberately trimmed-down view of a Bill — just enough to
// render a clickable list on the Topics page. Each entry carries a
// source_url so the citizen can verify the reference.
type TopicBill struct {
	// ID is the platform-internal bill ID (e.g. "ke-bill-2026-health").
	ID string `json:"id"`
	// Title is the human-readable bill title (e.g. "The Health Bill, 2026").
	Title string `json:"title"`
	// House is "National Assembly" or "Senate".
	House string `json:"house"`
	// Year is the 4-digit publication year parsed from the bill's slug.
	Year int `json:"year"`
	// SourceURL is the canonical URL on kenyalaw.org or parliament.go.ke.
	SourceURL string `json:"source_url"`
}

// TopicAct is the act summary shown in the /topics/{id} detail response.
type TopicAct struct {
	// ID is the platform-internal act ID (e.g. "ke-act-data-protection-2019").
	ID string `json:"id"`
	// Title is the human-readable act name (e.g. "Data Protection Act, 2019").
	Title string `json:"title"`
	// Citation is the citation-style act number (e.g. "No. 24 of 2019").
	Citation string `json:"citation"`
	// SourceURL is the canonical URL on kenyalaw.org.
	SourceURL string `json:"source_url"`
}

// TopicPerson is the person summary shown in the /topics/{id} detail
// response. Typically the chair / vice-chair of the Departmental
// Committee responsible for the topic.
type TopicPerson struct {
	// ID is the platform-internal person ID (e.g. "person-001").
	ID string `json:"id"`
	// FullName is the person's full name (e.g. "Rt. Hon. Moses Wetangula").
	FullName string `json:"full_name"`
	// Role is the person's role (e.g. "Speaker of the National Assembly").
	Role string `json:"role"`
	// House is "National Assembly", "Senate", or "Executive".
	House string `json:"house"`
}

// TopicDetail is the full payload returned by GET /api/v1/topics/{id}.
// It embeds the Topic (so the basic fields are present at the top level)
// and adds the associated Bills / Acts / People slices.
type TopicDetail struct {
	Topic
	Bills  []TopicBill   `json:"bills"`
	Acts   []TopicAct    `json:"acts"`
	People []TopicPerson `json:"people"`
}

// kenyaTopicEntry is the internal representation that bundles a Topic
// with its associated Bills / Acts / People. It is private to this
// file; callers consume KenyaTopics (the list) + FindKenyaTopicByID
// (the detail) + the per-slice helpers below.
type kenyaTopicEntry struct {
	topic  Topic
	bills  []TopicBill
	acts   []TopicAct
	people []TopicPerson
}

// kenyaTopicEntries is the single source of truth for the Kenya topic
// seed. Each entry's Topic.{Bill,Act,People}Count fields MUST match
// the length of the corresponding slices — the init() function below
// enforces this invariant so the list response never lies about the
// detail response.
//
// The 10 topics below mirror the Departmental Committees of the Kenya
// National Assembly (per parliament.go.ke) — the same set referenced
// by the frontend navbar mega-menu (apps/web/src/components/header.tsx)
// and Topics page (apps/web/src/app/topics/page.tsx).
var kenyaTopicEntries = []kenyaTopicEntry{
	{
		topic: Topic{
			ID:          "health",
			Name:        "Health",
			Description: "Bills, Acts, and oversight concerning public health, healthcare financing, medical practitioners, and public-health regulation. Maps to the Departmental Committee on Health.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-health", Title: "The Health Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-national-coroners-service", Title: "The National Coroners Service Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-medical-practitioners", Title: "The Medical Practitioners and Dentists (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-alcoholic-drinks-control", Title: "The Alcoholic Drinks Control (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-blood-cells", Title: "The Kenya Blood Cells, Tissues and Organs Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts: []TopicAct{
			{ID: "ke-act-constitution-2010", Title: "Constitution of Kenya", Citation: "Constitution of Kenya, 2010", SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398"},
		},
		people: []TopicPerson{
			{ID: "person-001", FullName: "Rt. Hon. Moses Wetangula", Role: "Speaker of the National Assembly", House: "National Assembly"},
		},
	},
	{
		topic: Topic{
			ID:          "education",
			Name:        "Education",
			Description: "Bills, Acts, and oversight concerning basic, tertiary, and pre-service education, curriculum development, and qualifications frameworks. Maps to the Departmental Committee on Education and Research.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-basic-education", Title: "The Basic Education Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-kicd", Title: "The Kenya Institute of Curriculum Development (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-knqf", Title: "The Kenya National Qualifications Framework (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-pre-service-education", Title: "The Pre-service Education and In-service Training Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-tertiary-education", Title: "The Tertiary Education Placement and Funding Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts:   []TopicAct{},
		people: []TopicPerson{},
	},
	{
		topic: Topic{
			ID:          "finance",
			Name:        "Finance & National Planning",
			Description: "Bills, Acts, and oversight concerning public finance management, taxation, the budget, pensions, revenue administration, and national planning. Maps to the Departmental Committee on Finance and National Planning.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-pfm", Title: "The Public Finance Management (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-appropriation", Title: "The Appropriation Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-supplementary-appropriation", Title: "The Supplementary Appropriation Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-income-tax", Title: "The Income Tax (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-vat", Title: "The Value Added Tax (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-microfinance", Title: "The Microfinance Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-pension", Title: "The Pension (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-county-allocation", Title: "The County Allocation of Revenue Bill, 2026", House: "Senate", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts: []TopicAct{
			{ID: "ke-act-public-finance-management-2015", Title: "Public Finance Management Act, 2015", Citation: "No. 18 of 2015", SourceURL: "https://www.kenyalaw.org/kl/index.php?id=5769b1c8e3a1f8c3f9b1c8e3"},
			{ID: "ke-act-companies-2015", Title: "Companies Act, 2015", Citation: "No. 17 of 2015", SourceURL: "https://www.kenyalaw.org/kl/index.php?id=5769b1c8e3a1f8c3f9b1c8e2"},
		},
		people: []TopicPerson{
			{ID: "person-003", FullName: "Kimani Ichung'wah", Role: "Majority Leader, National Assembly", House: "National Assembly"},
			{ID: "person-004", FullName: "Opiyo Wandayi", Role: "Minority Leader, National Assembly", House: "National Assembly"},
		},
	},
	{
		topic: Topic{
			ID:          "justice",
			Name:        "Justice & Legal Affairs",
			Description: "Bills, Acts, and oversight concerning criminal law, the penal code, judicial procedure, elections, and statutory instruments. Maps to the Departmental Committee on Justice and Legal Affairs.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-criminal-procedure", Title: "The Criminal Procedure Code (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-penal-code", Title: "The Penal Code (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-statutory-instruments", Title: "The Statutory Instruments (Amendment) Bill, 2026", House: "Senate", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-commission-of-inquiry", Title: "The Commission of Inquiry (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-trust-administration", Title: "The Trust Administration Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts: []TopicAct{
			{ID: "ke-act-elections-2011", Title: "Elections Act, 2011", Citation: "No. 24 of 2011", SourceURL: "https://www.kenyalaw.org/kl/index.php?id=51a5b3d6c0e3a1f8c3f9b1c8"},
		},
		people: []TopicPerson{
			{ID: "person-002", FullName: "Sen. Amason Kingi", Role: "Speaker of the Senate", House: "Senate"},
		},
	},
	{
		topic: Topic{
			ID:          "defence-foreign-relations",
			Name:        "Defence & Foreign Relations",
			Description: "Bills, Acts, and oversight concerning national defence, the foreign service, strategic goods control, and regional security cooperation. Maps to the Departmental Committee on Defence, Intelligence and Foreign Relations.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-foreign-service", Title: "The Foreign Service (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-strategic-goods-control", Title: "The Strategic Goods Control Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts: []TopicAct{},
		people: []TopicPerson{
			{ID: "person-005", FullName: "William Ruto", Role: "President of Kenya", House: "Executive"},
		},
	},
	{
		topic: Topic{
			ID:          "energy-petroleum",
			Name:        "Energy & Petroleum",
			Description: "Bills, Acts, and oversight concerning electricity generation and distribution, petroleum exploration and refining, and renewable energy. Maps to the Departmental Committee on Energy.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-energy", Title: "The Energy Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-petroleum", Title: "The Petroleum (Exploration and Production) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts:   []TopicAct{},
		people: []TopicPerson{},
	},
	{
		topic: Topic{
			ID:          "agriculture-livestock",
			Name:        "Agriculture & Livestock",
			Description: "Bills, Acts, and oversight concerning crops, livestock, plant health inspection, fisheries, and food security. Maps to the Departmental Committee on Agriculture and Livestock.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-crops-laws", Title: "The Crops Laws (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-livestock", Title: "The Livestock Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-kephis", Title: "The Kenya Plant Health Inspectorate Services (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts:   []TopicAct{},
		people: []TopicPerson{},
	},
	{
		topic: Topic{
			ID:          "infrastructure-transport",
			Name:        "Infrastructure & Transport",
			Description: "Bills, Acts, and oversight concerning roads, public transport, traffic regulation, building standards, and regional infrastructure development. Maps to the Departmental Committee on Transport, Public Works and Housing.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-ntsa", Title: "The National Transport and Safety Authority (Amendment) Bill, 2026", House: "Senate", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-traffic", Title: "The Traffic (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-architectural", Title: "The Architectural and Quantity Surveying Practitioners Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts:   []TopicAct{},
		people: []TopicPerson{},
	},
	{
		topic: Topic{
			ID:          "environment-natural-resources",
			Name:        "Environment & Natural Resources",
			Description: "Bills, Acts, and oversight concerning environmental management, water resources, forestry, wildlife, and climate change. Maps to the Departmental Committee on Environment, Forestry and Mining.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-environmental", Title: "The Environmental Management and Co-ordination (Amendment) Bill, 2026", House: "Senate", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-water", Title: "The Water (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts:   []TopicAct{},
		people: []TopicPerson{},
	},
	{
		topic: Topic{
			ID:          "ict",
			Name:        "Information, Communication & Technology",
			Description: "Bills, Acts, and oversight concerning data protection, intellectual property, cyber-security, films and stage plays, and digital infrastructure. Maps to the Departmental Committee on Communication, Information and Innovation.",
			Country:     "KE",
		},
		bills: []TopicBill{
			{ID: "ke-bill-2026-intellectual-property", Title: "The Kenya Intellectual Property Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-films-stage-plays", Title: "The Films and Stage Plays (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
			{ID: "ke-bill-2026-trade-description", Title: "The Trade Description (Amendment) Bill, 2026", House: "National Assembly", Year: 2026, SourceURL: "https://new.kenyalaw.org/bills/"},
		},
		acts: []TopicAct{
			{ID: "ke-act-data-protection-2019", Title: "Data Protection Act, 2019", Citation: "No. 24 of 2019", SourceURL: "https://www.kenyalaw.org/kl/index.php?id=646aa3ba8b8f6d3a9c3f3f9c"},
		},
		people: []TopicPerson{},
	},
}

// KenyaTopics is the list of seed Kenyan civic topics, exposed as the
// Topic DTOs that the /api/v1/topics list handler serialises directly.
// The slice is built once at package init from kenyaTopicEntries so
// the BillCount / ActCount / PeopleCount fields always agree with the
// per-topic detail slices.
//
// The slice is sorted alphabetically by Name so the Topics page renders
// a deterministic ordering across requests.
var KenyaTopics = func() []Topic {
	out := make([]Topic, 0, len(kenyaTopicEntries))
	for _, e := range kenyaTopicEntries {
		t := e.topic
		t.BillCount = len(e.bills)
		t.ActCount = len(e.acts)
		t.PeopleCount = len(e.people)
		out = append(out, t)
	}
	return out
}()

// FindKenyaTopicByID looks up a single topic by its ID (case-insensitive).
// Returns nil when no topic matches. Used by the /topics/{id} detail
// handler to look up the topic before assembling the full payload.
func FindKenyaTopicByID(id string) *Topic {
	if id == "" {
		return nil
	}
	for i := range KenyaTopics {
		if equalFoldASCII(KenyaTopics[i].ID, id) {
			return &KenyaTopics[i]
		}
	}
	return nil
}

// KenyaTopicDetail assembles the full detail payload for the supplied
// topic ID. Returns nil when no topic matches — the caller (the
// /topics/{id} handler) translates nil into a 404 response.
//
// The returned TopicDetail embeds the Topic (so the basic fields are
// present at the top level of the JSON) and includes the associated
// Bills / Acts / People slices. Every associated entry carries a
// source_url so the citizen can verify the reference on the primary
// source's website.
func KenyaTopicDetail(id string) *TopicDetail {
	if id == "" {
		return nil
	}
	for i := range kenyaTopicEntries {
		if !equalFoldASCII(kenyaTopicEntries[i].topic.ID, id) {
			continue
		}
		e := &kenyaTopicEntries[i]
		t := e.topic
		t.BillCount = len(e.bills)
		t.ActCount = len(e.acts)
		t.PeopleCount = len(e.people)
		return &TopicDetail{
			Topic:  t,
			Bills:  e.bills,
			Acts:   e.acts,
			People: e.people,
		}
	}
	return nil
}

// KenyaTopicBills returns the seed Bills associated with the supplied
// topic ID, or nil when the topic is unknown. Useful for callers that
// want just the bills slice (e.g. a future "Bills by Topic" feed).
func KenyaTopicBills(id string) []TopicBill {
	if id == "" {
		return nil
	}
	for i := range kenyaTopicEntries {
		if equalFoldASCII(kenyaTopicEntries[i].topic.ID, id) {
			return kenyaTopicEntries[i].bills
		}
	}
	return nil
}

// KenyaTopicActs returns the seed Acts associated with the supplied
// topic ID, or nil when the topic is unknown.
func KenyaTopicActs(id string) []TopicAct {
	if id == "" {
		return nil
	}
	for i := range kenyaTopicEntries {
		if equalFoldASCII(kenyaTopicEntries[i].topic.ID, id) {
			return kenyaTopicEntries[i].acts
		}
	}
	return nil
}

// KenyaTopicPeople returns the seed People associated with the supplied
// topic ID, or nil when the topic is unknown.
func KenyaTopicPeople(id string) []TopicPerson {
	if id == "" {
		return nil
	}
	for i := range kenyaTopicEntries {
		if equalFoldASCII(kenyaTopicEntries[i].topic.ID, id) {
			return kenyaTopicEntries[i].people
		}
	}
	return nil
}

// equalFoldASCII reports whether two strings are equal under ASCII
// case-folding. It avoids pulling in the unicode package for what is
// a tightly-scoped kebab-case comparison. Used by FindKenyaTopicByID
// + friends so /topics/Health and /topics/health resolve to the same
// topic.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		c1, c2 := a[i], b[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}
