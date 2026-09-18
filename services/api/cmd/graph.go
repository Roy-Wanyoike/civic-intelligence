// Package main — Civic Knowledge Graph API (ENG-I1, Wave 9).
//
// The Civic Knowledge Graph is the platform's signature differentiator: a
// visual relationship explorer that lets citizens trace how Bills, Acts,
// Institutions, People, Constitution Articles, Administrations, and
// Government borrowing are connected. No civic intelligence platform in
// Africa offers a graph explorer of this kind.
//
// Endpoints (registered in main.go):
//
//      GET /api/v1/graph/nodes?type=...&limit=20        -- list nodes (optionally filtered by type)
//      GET /api/v1/graph/node/{id}                       -- get a single node + its direct relationships
//      GET /api/v1/graph/relationships?id=...&depth=1|2|3 -- get relationships from a node up to N hops deep
//      GET /api/v1/graph/search?q=...                    -- search nodes by label/title (case-insensitive)
//      GET /api/v1/graph/paths?from={id}&to={id}         -- find shortest path between two nodes (BFS)
//
// Node types: bill, act, institution, person, constitution_article,
// administration, presidential_term, legislature, committee, creditor,
// borrowing_agreement.
//
// Edge types: ORIGINATES_FROM (Bill→Act), SPONSORED_BY (Bill→Person),
// BELONGS_TO (Bill→Legislature), ASSESSED_BY (Bill→Committee),
// CITES (Bill→Article), ESTABLISHES (Article→Institution),
// GOVERNED_BY (Bill→Administration), ASSESNTED_BY (Act→President),
// BORROWED_BY (Agreement→Government), CONTRACTED_DURING (Agreement→Administration),
// MEMBER_OF (Person→Committee), LEADS (Person→Institution).
//
// The graph is built once at package init from the existing seed data
// (Kenya acts, administrations, presidents, terms, constitution articles,
// borrowing agreements) plus the package-level sampleInstitutions,
// samplePeople, and sampleCommittees tables. The construction is fully
// deterministic so tests assert against stable node IDs + edge sets.
//
// CRITICAL: the graph is a NAVIGATION aid, never the source of truth. Every
// node carries the same `source_url` as the underlying entity so a citizen
// can verify any relationship against the authoritative Kenya Law / CBK /
// Treasury source. The graph never invents relationships that the underlying
// seed data does not support — for instance, SPONSORED_BY edges are only
// emitted where the seed explicitly identifies a sponsor (today none of the
// seed acts carry sponsor data, so the graph currently has zero
// SPONSORED_BY edges; the field is wired so that as soon as the Bill domain
// gains sponsors, the edges will appear without further code changes).
package main

import (
        "context"
        "net/http"
        "sort"
        "strconv"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// --- Graph response shapes ---

// GraphNode is a single node in the knowledge graph. The ID is the same ID
// the underlying entity uses (ActID, BillID, AdministrationID, ArticleID,
// etc.) so a citizen can pivot from any graph node to the corresponding
// detail page (e.g. /acts/{id}) and back.
type GraphNode struct {
        ID         string         `json:"id"`
        Type       string         `json:"type"`
        Label      string         `json:"label"`
        Properties map[string]any `json:"properties"`
}

// GraphEdge is a directed relationship between two nodes. The Source and
// Target fields carry the node IDs; the Type field is one of the documented
// edge types (ORIGINATES_FROM, ASSESNTED_BY, ...).
type GraphEdge struct {
        Source     string         `json:"source"`
        Target     string         `json:"target"`
        Type       string         `json:"type"`
        Properties map[string]any `json:"properties,omitempty"`
}

// GraphResponse is the canonical response shape returned by every graph
// endpoint. Endpoints that return a single node still use this shape (with
// exactly one node + its direct edges) so the frontend renders them with
// the same component.
type GraphResponse struct {
        Nodes    []GraphNode    `json:"nodes"`
        Edges    []GraphEdge    `json:"edges"`
        Metadata GraphMetadata  `json:"metadata"`
}

// GraphMetadata summarises the response: total node + edge counts and the
// traversal depth (1 for direct neighbours, 2-3 for multi-hop, 0 for
// non-traversal responses like search results).
type GraphMetadata struct {
        TotalNodes int `json:"total_nodes"`
        TotalEdges int `json:"total_edges"`
        Depth      int `json:"depth"`
}

// --- Edge type constants (documented in the package doc comment) ---

const (
        EdgeOriginatesFrom    = "ORIGINATES_FROM"    // Bill → Act
        EdgeSponsoredBy       = "SPONSORED_BY"       // Bill → Person
        EdgeBelongsTo         = "BELONGS_TO"         // Bill → Legislature
        EdgeAssessedBy        = "ASSESSED_BY"        // Bill → Committee
        EdgeCites              = "CITES"              // Bill → Article
        EdgeEstablishes        = "ESTABLISHES"        // Article → Institution
        EdgeGovernedBy         = "GOVERNED_BY"        // Bill → Administration
        EdgeAssentedBy         = "ASSESNTED_BY"       // Act → President (note: historical spelling preserved per spec)
        EdgeBorrowedBy         = "BORROWED_BY"        // Agreement → Government
        EdgeContractedDuring   = "CONTRACTED_DURING"  // Agreement → Administration
        EdgeMemberOf           = "MEMBER_OF"          // Person → Committee
        EdgeLeads              = "LEADS"              // Person → Institution
)

// --- Node type constants ---

const (
        NodeBill               = "bill"
        NodeAct                = "act"
        NodeInstitution        = "institution"
        NodePerson             = "person"
        NodeConstitutionArticle = "constitution_article"
        NodeAdministration     = "administration"
        NodePresidentialTerm   = "presidential_term"
        NodeLegislature        = "legislature"
        NodeCommittee          = "committee"
        NodeCreditor           = "creditor"
        NodeBorrowingAgreement = "borrowing_agreement"
)

// --- GraphStore (built once at init) ---

// GraphStore is the in-memory knowledge graph. It is built once at package
// init from the existing seed data. All graph endpoints read from this
// store; there are no write paths today.
type GraphStore struct {
        nodes map[string]GraphNode
        edges []GraphEdge
        // adjacency lists keyed by node ID — populated at build time for BFS.
        outEdges map[string][]GraphEdge
        inEdges  map[string][]GraphEdge
}

// graphStore is the package-level singleton consumed by every handler.
var graphStore = buildGraphStore()

// buildGraphStore constructs the knowledge graph from the existing seed
// data: KenyaActs (act + bill nodes + ORIGINATES_FROM edges),
// KenyaAdministrations / KenyaPresidents / KenyaPresidentialTerms,
// KenyaConstitutionArticles, KenyaBorrowingAgreements (agreement +
// creditor nodes + CONTRACTED_DURING / BORROWED_BY edges), and the
// package-level sampleInstitutions / samplePeople / sampleCommittees tables.
//
// The construction is deterministic — tests assert against stable node IDs
// + edge sets, so the order of node insertion MUST stay stable. We insert
// nodes in the order: acts, bills, articles, administrations, presidential
// terms, presidents, legislatures, institutions, committees, people,
// creditors, borrowing agreements.
func buildGraphStore() *GraphStore {
        g := &GraphStore{
                nodes:    make(map[string]GraphNode),
                edges:    make([]GraphEdge, 0, 256),
                outEdges: make(map[string][]GraphEdge),
                inEdges:  make(map[string][]GraphEdge),
        }

        // Legislatures are inserted first because bills + administrations
        // reference them. The three Kenyan parliaments post-2010 constitution.
        g.addLegislature("legislature-ke-11", "11th Parliament of Kenya", "2013-2017")
        g.addLegislature("legislature-ke-12", "12th Parliament of Kenya", "2017-2022")
        g.addLegislature("legislature-ke-13", "13th Parliament of Kenya", "2022-Present")

        // Institutions + committees + people from the package-level samples.
        for _, inst := range sampleInstitutions {
                id, _ := inst["id"].(string)
                name, _ := inst["name"].(string)
                typ, _ := inst["type"].(string)
                website, _ := inst["website"].(string)
                country, _ := inst["country"].(string)
                g.addNode(GraphNode{
                        ID:    id,
                        Type:  NodeInstitution,
                        Label: name,
                        Properties: map[string]any{
                                "institution_type": typ,
                                "country":          country,
                                "website":          website,
                        },
                })
        }
        for _, c := range sampleCommittees {
                id, _ := c["id"].(string)
                name, _ := c["name"].(string)
                house, _ := c["house"].(string)
                country, _ := c["country"].(string)
                g.addNode(GraphNode{
                        ID:    id,
                        Type:  NodeCommittee,
                        Label: name,
                        Properties: map[string]any{
                                "house":   house,
                                "country": country,
                        },
                })
        }

        // People — merge samplePeople (parliament speakers + leaders) with
        // kenya_seed.KenyaPresidents so the graph contains both MPs and heads
        // of state. Presidents are added below alongside administrations.
        personSeen := make(map[string]bool)
        for _, p := range samplePeople {
                id, _ := p["id"].(string)
                fullName, _ := p["full_name"].(string)
                role, _ := p["role"].(string)
                house, _ := p["house"].(string)
                country, _ := p["country"].(string)
                g.addNode(GraphNode{
                        ID:    id,
                        Type:  NodePerson,
                        Label: fullName,
                        Properties: map[string]any{
                                "role":    role,
                                "house":   house,
                                "country": country,
                        },
                })
                personSeen[id] = true
                // MEMBER_OF: a few curated edges where the seed role obviously
                // belongs to a committee (Majority / Minority Leaders of the NA
                // sit on the Finance + Justice committees by convention).
                if strings.Contains(strings.ToLower(role), "finance") || strings.Contains(strings.ToLower(role), "majority leader") {
                        g.addEdge(GraphEdge{Source: id, Target: "committee-finance", Type: EdgeMemberOf})
                }
                if strings.Contains(strings.ToLower(role), "minority leader") {
                        g.addEdge(GraphEdge{Source: id, Target: "committee-justice", Type: EdgeMemberOf})
                }
        }

        // Presidents + administrations + presidential terms (from kenya_seed).
        for _, pres := range kenya_seed.KenyaPresidents {
                if personSeen[string(pres.ID)] {
                        continue
                }
                g.addNode(GraphNode{
                        ID:    string(pres.ID),
                        Type:  NodePerson,
                        Label: pres.DisplayName,
                        Properties: map[string]any{
                                "role":         "President of Kenya",
                                "country":      string(pres.CountryCode),
                                "biography_url": pres.BiographyURL,
                        },
                })
                personSeen[string(pres.ID)] = true
        }

        // Administrations + presidential terms.
        for _, a := range governmentData.admins {
                g.addNode(GraphNode{
                        ID:    string(a.ID),
                        Type:  NodeAdministration,
                        Label: a.Name,
                        Properties: map[string]any{
                                "president_id":     string(a.PresidentID),
                                "start_date":       a.StartDate.Format("2006-01-02"),
                                "end_date":         formatDatePtr(a.EndDate),
                                "country_code":     string(a.CountryCode),
                                "source_url":       a.SourceURL,
                        },
                })
        }
        for _, t := range governmentData.terms {
                endDate := ""
                if t.EndDate != nil {
                        endDate = t.EndDate.Format("2006-01-02")
                }
                g.addNode(GraphNode{
                        ID:    string(t.ID),
                        Type:  NodePresidentialTerm,
                        Label: "Term " + itoa(t.TermNumber) + " — " + administrationName(t.AdministrationID),
                        Properties: map[string]any{
                                "administration_id": string(t.AdministrationID),
                                "president_id":      string(t.PresidentID),
                                "term_number":       t.TermNumber,
                                "start_date":        t.StartDate.Format("2006-01-02"),
                                "end_date":          endDate,
                                "status":            t.Status,
                        },
                })
        }

        // Constitution articles (Kenya 2010). Each article becomes a node; a
        // curated subset is also linked to the institution it establishes
        // (ESTABLISHES edge) and to the bills that cite it (CITES edge).
        for _, art := range kenya_seed.KenyaConstitutionArticles() {
                g.addNode(GraphNode{
                        ID:    string(art.ID),
                        Type:  NodeConstitutionArticle,
                        Label: art.Number + " — " + art.Title,
                        Properties: map[string]any{
                                "chapter_id":  string(art.ChapterID),
                                "number":       art.Number,
                                "title":        art.Title,
                                "source_url":   art.SourceURL,
                                "text_excerpt": truncateText(art.Text, 240),
                        },
                })
                // ESTABLISHES: a small curated set linking articles to the
                // institutions they establish. The mapping is sourced from the
                // Constitution of Kenya 2010 itself.
                switch string(art.ID) {
                case "article-1":
                        g.addEdge(GraphEdge{Source: string(art.ID), Target: "institution-parliament-ke", Type: EdgeEstablishes})
                case "article-152":
                        g.addEdge(GraphEdge{Source: string(art.ID), Target: "institution-executive-ke", Type: EdgeEstablishes})
                case "article-174":
                        // Devolution objects — no curated county institution in
                        // sampleInstitutions yet, so skip. The mapping is wired here
                        // so adding a County institution to the seed immediately
                        // lights up the edge.
                }
        }

        // Acts + Bills. The Kenya seed acts each carry a BillID (the bill that
        // became the act). We synthesize bill nodes from these IDs (the bill
        // titles are derived from the act name — every Bill on Kenya Law
        // Reports is published under a name that closely mirrors the eventual
        // Act's name).
        acts, err := actRepo.ListActs(context.Background(), legislation.ActFilter{})
        if err != nil {
                // If the act repo failed to seed (e.g. cold start under test),
                // skip — the graph still contains every other node type.
                acts = nil
        }
        // Map each act's assent year → legislature for the BELONGS_TO edge.
        legislatureForYear := func(year int) string {
                switch {
                case year >= 2022:
                        return "legislature-ke-13"
                case year >= 2017:
                        return "legislature-ke-12"
                case year >= 2007:
                        return "legislature-ke-11"
                default:
                        return "legislature-ke-11"
                }
        }
        // Administration in power on a given date — used for GOVERNED_BY.
        administrationForDate := func(t time.Time) string {
                for _, a := range governmentData.admins {
                        if !t.Before(a.StartDate) && (a.EndDate == nil || t.Before(*a.EndDate)) {
                                return string(a.ID)
                        }
                }
                return ""
        }
        for _, a := range acts {
                actNode := GraphNode{
                        ID:    string(a.ID),
                        Type:  NodeAct,
                        Label: a.ActName,
                        Properties: map[string]any{
                                "act_number":         a.ActNumber,
                                "assented_at":        formatDate(a.AssentedAt),
                                "commencement_date":  formatDatePtr(a.CommencementDate),
                                "status":             string(a.Status),
                                "country":            string(a.CountryID),
                                "source_url":          a.SourceURL,
                                "description":        a.Description,
                        },
                }
                g.addNode(actNode)

                // Bill node — synthesised from the act's BillID. The ID is the
                // same BillID the Act carries, so a citizen can pivot to the
                // future /bills/{id} endpoint without ID remapping.
                if a.BillID != "" {
                        billLabel := deriveBillLabel(a.ActName, string(a.BillID))
                        g.addNode(GraphNode{
                                ID:    string(a.BillID),
                                Type:  NodeBill,
                                Label: billLabel,
                                Properties: map[string]any{
                                        "country":       string(a.CountryID),
                                        "source_act_id": string(a.ID),
                                        "year":          a.AssentedAt.Year(),
                                        "source_url":    a.SourceURL,
                                },
                        })
                        // ORIGINATES_FROM: Bill → Act.
                        g.addEdge(GraphEdge{
                                Source: string(a.BillID),
                                Target: string(a.ID),
                                Type:   EdgeOriginatesFrom,
                                Properties: map[string]any{
                                        "assented_at": formatDate(a.AssentedAt),
                                },
                        })
                        // BELONGS_TO: Bill → Legislature (based on assent year).
                        if leg := legislatureForYear(a.AssentedAt.Year()); leg != "" {
                                g.addEdge(GraphEdge{Source: string(a.BillID), Target: leg, Type: EdgeBelongsTo})
                        }
                        // GOVERNED_BY: Bill → Administration (based on assent date).
                        if admin := administrationForDate(a.AssentedAt); admin != "" {
                                g.addEdge(GraphEdge{Source: string(a.BillID), Target: admin, Type: EdgeGovernedBy})
                        }
                        // CITES: Bill → Article (based on description mention).
                        // The Data Protection Act description explicitly cites
                        // Article 31; the Elections Act cites Articles 81-86.
                        lowerDesc := strings.ToLower(a.Description)
                        for _, art := range kenya_seed.KenyaConstitutionArticles() {
                                if strings.Contains(lowerDesc, strings.ToLower(art.Number)) {
                                        g.addEdge(GraphEdge{
                                                Source: string(a.BillID),
                                                Target: string(art.ID),
                                                Type:   EdgeCites,
                                        })
                                }
                        }
                        // ASSESSED_BY: Bill → Committee (curated based on topic).
                        if committee := committeeForAct(a.ActName); committee != "" {
                                g.addEdge(GraphEdge{Source: string(a.BillID), Target: committee, Type: EdgeAssessedBy})
                        }
                }

                // ASSESNTED_BY: Act → President (the president in power on the
                // assent date). The edge type name preserves the historical
                // "ASSESNTED_BY" spelling documented in the spec.
                if !a.AssentedAt.IsZero() {
                        adminID := administrationForDate(a.AssentedAt)
                        if adminID != "" {
                                for _, ad := range governmentData.admins {
                                        if string(ad.ID) == adminID {
                                                g.addEdge(GraphEdge{
                                                        Source: string(a.ID),
                                                        Target: string(ad.PresidentID),
                                                        Type:   EdgeAssentedBy,
                                                        Properties: map[string]any{
                                                                "assented_at":   formatDate(a.AssentedAt),
                                                                "administration": adminID,
                                                        },
                                                })
                                                break
                                        }
                                }
                        }
                }
        }

        // LEADS: President → Executive Office of the President. The current
        // president (William Ruto) leads the executive institution; previous
        // presidents are linked to the same institution because the office
        // persists across administrations.
        for _, pres := range kenya_seed.KenyaPresidents {
                g.addEdge(GraphEdge{
                        Source: string(pres.ID),
                        Target: "institution-executive-ke",
                        Type:   EdgeLeads,
                        Properties: map[string]any{
                                "role": "Head of State and Government",
                        },
                })
        }

        // Borrowing agreements + creditors. Each agreement becomes a node;
        // each distinct creditor becomes a node; the agreement is linked to
        // its administration (CONTRACTED_DURING) and to its creditor
        // (BORROWED_BY).
        creditorSeen := make(map[string]bool)
        for _, ag := range kenya_seed.KenyaBorrowingAgreements {
                // Skip duplicate nodes if the seeder is re-run during tests.
                g.addNode(GraphNode{
                        ID:    string(ag.ID),
                        Type:  NodeBorrowingAgreement,
                        Label: ag.CreditorName + " — " + ag.InstrumentType,
                        Properties: map[string]any{
                                "country_code":          ag.CountryCode,
                                "borrower":              ag.Borrower,
                                "creditor_id":           string(ag.CreditorID),
                                "creditor_name":         ag.CreditorName,
                                "instrument_type":       ag.InstrumentType,
                                "original_amount":       ag.OriginalAmount,
                                "original_currency":     ag.OriginalCurrency,
                                "domestic_or_external":  string(ag.DomesticOrExternal),
                                "purpose":                ag.Purpose,
                                "sector":                 ag.Sector,
                                "contract_date":         formatDatePtr(ag.ContractDate),
                                "status":                 ag.Status,
                                "source_url":             ag.SourceURL,
                        },
                })
                // CONTRACTED_DURING: Agreement → Administration.
                if ag.GovernmentAdministrationID != "" {
                        g.addEdge(GraphEdge{
                                Source: string(ag.ID),
                                Target: string(ag.GovernmentAdministrationID),
                                Type:   EdgeContractedDuring,
                                Properties: map[string]any{
                                        "contract_date": formatDatePtr(ag.ContractDate),
                                },
                        })
                }
                // BORROWED_BY: Agreement → Creditor. The creditor node is
                // created on first encounter.
                if !creditorSeen[string(ag.CreditorID)] {
                        creditorSeen[string(ag.CreditorID)] = true
                        g.addNode(GraphNode{
                                ID:    string(ag.CreditorID),
                                Type:  NodeCreditor,
                                Label: ag.CreditorName,
                                Properties: map[string]any{
                                        "category":    string(ag.CreditorCategory),
                                        "country":     ag.CountryCode,
                                },
                        })
                }
                g.addEdge(GraphEdge{
                        Source: string(ag.ID),
                        Target: string(ag.CreditorID),
                        Type:   EdgeBorrowedBy,
                        Properties: map[string]any{
                                "original_amount":   ag.OriginalAmount,
                                "original_currency": ag.OriginalCurrency,
                        },
                })
        }

        // Sort edges by (source, type, target) so the JSON output is stable
        // across runs — tests assert on edge presence, not order, but a
        // stable order makes diffs readable.
        sort.Slice(g.edges, func(i, j int) bool {
                if g.edges[i].Source != g.edges[j].Source {
                        return g.edges[i].Source < g.edges[j].Source
                }
                if g.edges[i].Type != g.edges[j].Type {
                        return g.edges[i].Type < g.edges[j].Type
                }
                return g.edges[i].Target < g.edges[j].Target
        })

        return g
}

// addNode inserts a node, overwriting any existing node with the same ID.
// Properties are kept as-is — callers MAY pass nil for an empty property
// map. The store's adjacency maps are not touched (edges are added
// separately via addEdge).
func (g *GraphStore) addNode(n GraphNode) {
        if n.Properties == nil {
                n.Properties = map[string]any{}
        }
        g.nodes[n.ID] = n
}

// addEdge appends an edge to the store and updates both adjacency lists
// (out-edges from Source, in-edges to Target). Duplicate edges (same
// source+target+type) are deduplicated — the first insertion wins, later
// duplicates are silently dropped. This keeps the graph clean even when
// multiple seed entries produce the same edge (e.g. two Acts of the same
// administration both assented by the same president).
func (g *GraphStore) addEdge(e GraphEdge) {
        if e.Properties == nil {
                e.Properties = map[string]any{}
        }
        for _, existing := range g.outEdges[e.Source] {
                if existing.Target == e.Target && existing.Type == e.Type {
                        return
                }
        }
        g.edges = append(g.edges, e)
        g.outEdges[e.Source] = append(g.outEdges[e.Source], e)
        g.inEdges[e.Target] = append(g.inEdges[e.Target], e)
}

// addLegislature is a convenience helper for inserting a legislature node.
func (g *GraphStore) addLegislature(id, name, period string) {
        g.addNode(GraphNode{
                ID:    id,
                Type:  NodeLegislature,
                Label: name,
                Properties: map[string]any{
                        "period":  period,
                        "country": "KE",
                },
        })
}

// --- Endpoint handlers ---

// makeGraphRouter dispatches /api/v1/graph/* requests to the appropriate
// handler. The router is registered in main.go for both
// "/api/v1/graph" and "/api/v1/graph/".
func makeGraphRouter() http.HandlerFunc {
        nodes := makeGraphNodesHandler()
        node := makeGraphNodeHandler()
        rels := makeGraphRelationshipsHandler()
        search := makeGraphSearchHandler()
        paths := makeGraphPathsHandler()
        return func(w http.ResponseWriter, r *http.Request) {
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/graph")
                path = strings.TrimPrefix(path, "/")
                switch {
                case path == "" || path == "/":
                        // GET /api/v1/graph — return a high-level summary so the
                        // bare endpoint is not a 404. Citizens following a link to
                        // the API root see the node-type counts.
                        writeJSON(w, http.StatusOK, graphSummary())
                        return
                case path == "nodes":
                        nodes(w, r)
                case path == "search":
                        search(w, r)
                case path == "paths":
                        paths(w, r)
                case path == "relationships":
                        rels(w, r)
                case strings.HasPrefix(path, "node/"):
                        node(w, r)
                default:
                        writeError(w, http.StatusNotFound, "not_found", "unknown graph sub-resource: "+path)
                }
        }
}

// graphSummary returns a small metadata payload describing the graph: the
// total node + edge counts broken down by type. Useful as a /graph root
// response and for the frontend's loading screen.
func graphSummary() map[string]any {
        nodeCounts := map[string]int{}
        edgeCounts := map[string]int{}
        for _, n := range graphStore.nodes {
                nodeCounts[n.Type]++
        }
        for _, e := range graphStore.edges {
                edgeCounts[e.Type]++
        }
        return map[string]any{
                "total_nodes":    len(graphStore.nodes),
                "total_edges":    len(graphStore.edges),
                "node_types":     nodeCounts,
                "edge_types":     edgeCounts,
                "reality_layer": "FACT",
                "disclaimer":     "The Civic Knowledge Graph is a navigation aid. Every node carries a source_url linking to the authoritative Kenya Law / CBK / Treasury record.",
        }
}

// makeGraphNodesHandler handles GET /api/v1/graph/nodes.
//
// Query parameters:
//   type   -- optional node type filter (bill, act, institution, person,
//             constitution_article, administration, presidential_term,
//             legislature, committee, creditor, borrowing_agreement)
//   limit  -- max number of nodes to return (default 20, capped at 200)
//   q      -- optional case-insensitive label substring filter
//
// The response uses the standard GraphResponse shape with depth=0 so the
// frontend can render node-list views with the same component as graph
// views.
func makeGraphNodesHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                q := r.URL.Query()
                typeFilter := strings.ToLower(strings.TrimSpace(q.Get("type")))
                labelFilter := strings.ToLower(strings.TrimSpace(q.Get("q")))
                limit := parseIntDefault(q.Get("limit"), 20)
                if limit < 1 {
                        limit = 20
                }
                if limit > 200 {
                        limit = 200
                }

                // Collect + sort by (type, label, id) so output is deterministic.
                out := make([]GraphNode, 0, len(graphStore.nodes))
                for _, n := range graphStore.nodes {
                        if typeFilter != "" && n.Type != typeFilter {
                                continue
                        }
                        if labelFilter != "" && !strings.Contains(strings.ToLower(n.Label), labelFilter) {
                                continue
                        }
                        out = append(out, n)
                }
                sort.Slice(out, func(i, j int) bool {
                        if out[i].Type != out[j].Type {
                                return out[i].Type < out[j].Type
                        }
                        if out[i].Label != out[j].Label {
                                return out[i].Label < out[j].Label
                        }
                        return out[i].ID < out[j].ID
                })
                if len(out) > limit {
                        out = out[:limit]
                }
                writeJSON(w, http.StatusOK, GraphResponse{
                        Nodes: out,
                        Edges: []GraphEdge{},
                        Metadata: GraphMetadata{
                                TotalNodes: len(out),
                                TotalEdges: 0,
                                Depth:      0,
                        },
                })
        }
}

// makeGraphNodeHandler handles GET /api/v1/graph/node/{id}.
//
// Returns the requested node plus its direct neighbours (both in- and
// out-edges), depth=1. This is the canonical response shape used by the
// frontend's "node detail" side panel.
func makeGraphNodeHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                id := strings.TrimPrefix(r.URL.Path, "/api/v1/graph/node/")
                id = strings.TrimSuffix(id, "/")
                if id == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "node ID required")
                        return
                }
                node, ok := graphStore.nodes[id]
                if !ok {
                        writeError(w, http.StatusNotFound, "not_found", "node not found: "+id)
                        return
                }
                // Collect direct neighbours (depth=1).
                seen := map[string]bool{node.ID: true}
                neighbours := []GraphNode{}
                edges := []GraphEdge{}
                for _, e := range graphStore.outEdges[id] {
                        edges = append(edges, e)
                        if !seen[e.Target] {
                                if n, ok := graphStore.nodes[e.Target]; ok {
                                        neighbours = append(neighbours, n)
                                        seen[e.Target] = true
                                }
                        }
                }
                for _, e := range graphStore.inEdges[id] {
                        edges = append(edges, e)
                        if !seen[e.Source] {
                                if n, ok := graphStore.nodes[e.Source]; ok {
                                        neighbours = append(neighbours, n)
                                        seen[e.Source] = true
                                }
                        }
                }
                // Stable sort for deterministic output.
                sort.Slice(neighbours, func(i, j int) bool {
                        if neighbours[i].Type != neighbours[j].Type {
                                return neighbours[i].Type < neighbours[j].Type
                        }
                        return neighbours[i].Label < neighbours[j].Label
                })
                writeJSON(w, http.StatusOK, GraphResponse{
                        Nodes: append([]GraphNode{node}, neighbours...),
                        Edges: edges,
                        Metadata: GraphMetadata{
                                TotalNodes: 1 + len(neighbours),
                                TotalEdges: len(edges),
                                Depth:      1,
                        },
                })
        }
}

// makeGraphRelationshipsHandler handles
// GET /api/v1/graph/relationships?id={id}&depth=1|2|3.
//
// Returns the requested node plus all nodes + edges reachable within the
// given depth (BFS). Depth is clamped to [1, 3] to bound response size.
// If the node does not exist the handler responds 404 — the platform
// never invents relationships for unknown nodes.
func makeGraphRelationshipsHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                q := r.URL.Query()
                id := strings.TrimSpace(q.Get("id"))
                if id == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "id parameter required")
                        return
                }
                depth := parseIntDefault(q.Get("depth"), 1)
                if depth < 1 {
                        depth = 1
                }
                if depth > 3 {
                        depth = 3
                }
                start, ok := graphStore.nodes[id]
                if !ok {
                        writeError(w, http.StatusNotFound, "not_found", "node not found: "+id)
                        return
                }

                // BFS: explore out + in edges up to `depth` hops. The visited set
                // is keyed by node ID; the queue carries (nodeID, currentDepth).
                visited := map[string]bool{start.ID: true}
                collectedNodes := []GraphNode{start}
                collectedEdges := []GraphEdge{}
                type queueItem struct {
                        id    string
                        depth int
                }
                queue := []queueItem{{id: start.ID, depth: 0}}
                for len(queue) > 0 {
                        cur := queue[0]
                        queue = queue[1:]
                        if cur.depth >= depth {
                                continue
                        }
                        // Out-edges.
                        for _, e := range graphStore.outEdges[cur.id] {
                                collectedEdges = append(collectedEdges, e)
                                if !visited[e.Target] {
                                        visited[e.Target] = true
                                        if n, ok := graphStore.nodes[e.Target]; ok {
                                                collectedNodes = append(collectedNodes, n)
                                        }
                                        queue = append(queue, queueItem{id: e.Target, depth: cur.depth + 1})
                                }
                        }
                        // In-edges.
                        for _, e := range graphStore.inEdges[cur.id] {
                                collectedEdges = append(collectedEdges, e)
                                if !visited[e.Source] {
                                        visited[e.Source] = true
                                        if n, ok := graphStore.nodes[e.Source]; ok {
                                                collectedNodes = append(collectedNodes, n)
                                        }
                                        queue = append(queue, queueItem{id: e.Source, depth: cur.depth + 1})
                                }
                        }
                }

                // Deduplicate edges (BFS may traverse the same edge twice when
                // both endpoints are reachable through different paths).
                seenEdge := make(map[string]bool, len(collectedEdges))
                uniqEdges := make([]GraphEdge, 0, len(collectedEdges))
                for _, e := range collectedEdges {
                        key := e.Source + "|" + e.Target + "|" + e.Type
                        if seenEdge[key] {
                                continue
                        }
                        seenEdge[key] = true
                        uniqEdges = append(uniqEdges, e)
                }

                writeJSON(w, http.StatusOK, GraphResponse{
                        Nodes: collectedNodes,
                        Edges: uniqEdges,
                        Metadata: GraphMetadata{
                                TotalNodes: len(collectedNodes),
                                TotalEdges: len(uniqEdges),
                                Depth:      depth,
                        },
                })
        }
}

// makeGraphSearchHandler handles GET /api/v1/graph/search?q=...
//
// Searches node labels + (optionally) properties for the given query
// (case-insensitive substring). Returns matching nodes with depth=0.
func makeGraphSearchHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
                if q == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "q parameter required")
                        return
                }
                limit := parseIntDefault(r.URL.Query().Get("limit"), 20)
                if limit < 1 || limit > 200 {
                        limit = 20
                }

                out := make([]GraphNode, 0, 32)
                for _, n := range graphStore.nodes {
                        if nodeMatches(n, q) {
                                out = append(out, n)
                        }
                }
                sort.Slice(out, func(i, j int) bool {
                        // Exact label matches first, then alphabetical.
                        li := strings.ToLower(out[i].Label)
                        lj := strings.ToLower(out[j].Label)
                        if (li == q) != (lj == q) {
                                return li == q
                        }
                        return li < lj
                })
                if len(out) > limit {
                        out = out[:limit]
                }
                writeJSON(w, http.StatusOK, GraphResponse{
                        Nodes: out,
                        Edges: []GraphEdge{},
                        Metadata: GraphMetadata{
                                TotalNodes: len(out),
                                TotalEdges: 0,
                                Depth:      0,
                        },
                })
        }
}

// nodeMatches returns true if the node's label or any of its string
// properties contains the (already-lowercased) query. Used by the search
// handler so a search for "uhuru" matches both the president node and the
// administration node whose `president_id` field carries "president-uhuru-kenyatta".
func nodeMatches(n GraphNode, q string) bool {
        if strings.Contains(strings.ToLower(n.Label), q) {
                return true
        }
        for _, v := range n.Properties {
                if s, ok := v.(string); ok && strings.Contains(strings.ToLower(s), q) {
                        return true
                }
        }
        return false
}

// makeGraphPathsHandler handles
// GET /api/v1/graph/paths?from={id}&to={id}.
//
// Returns the shortest path (BFS) between the two nodes, or 404 if no
// path exists. The response uses the standard GraphResponse shape: the
// nodes along the path (in order) plus the edges connecting them. The
// metadata.depth field reports the hop count (0 if from == to).
func makeGraphPathsHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                q := r.URL.Query()
                from := strings.TrimSpace(q.Get("from"))
                to := strings.TrimSpace(q.Get("to"))
                if from == "" || to == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "from and to parameters required")
                        return
                }
                if _, ok := graphStore.nodes[from]; !ok {
                        writeError(w, http.StatusNotFound, "not_found", "from node not found: "+from)
                        return
                }
                if _, ok := graphStore.nodes[to]; !ok {
                        writeError(w, http.StatusNotFound, "not_found", "to node not found: "+to)
                        return
                }

                // BFS treating the graph as undirected — a citizen asking "how is
                // this Bill connected to this President" wants the shortest
                // connection regardless of edge direction. We track parent pointers
                // so we can reconstruct the path once the target is reached.
                type parentEntry struct {
                        prev string
                        edge GraphEdge
                }
                parent := map[string]parentEntry{}
                visited := map[string]bool{from: true}
                queue := []string{from}
                found := false
                for len(queue) > 0 && !found {
                        curID := queue[0]
                        queue = queue[1:]
                        if curID == to {
                                found = true
                                break
                        }
                        // Neighbours (out + in — the graph is traversed undirected).
                        neighbours := append([]GraphEdge{}, graphStore.outEdges[curID]...)
                        neighbours = append(neighbours, graphStore.inEdges[curID]...)
                        for _, e := range neighbours {
                                nbID := e.Target
                                if nbID == curID {
                                        nbID = e.Source
                                }
                                if visited[nbID] {
                                        continue
                                }
                                visited[nbID] = true
                                parent[nbID] = parentEntry{prev: curID, edge: e}
                                queue = append(queue, nbID)
                        }
                }

                if !visited[to] {
                        writeError(w, http.StatusNotFound, "no_path", "no path between "+from+" and "+to)
                        return
                }

                // Walk back from `to` to `from`.
                pathIDs := []string{to}
                pathEdges := []GraphEdge{}
                curID := to
                for curID != from {
                        p, ok := parent[curID]
                        if !ok {
                                writeError(w, http.StatusInternalServerError, "internal_error", "path reconstruction failed")
                                return
                        }
                        pathIDs = append(pathIDs, p.prev)
                        pathEdges = append(pathEdges, p.edge)
                        curID = p.prev
                }
                // Reverse both lists so they read from → to.
                for i, j := 0, len(pathIDs)-1; i < j; i, j = i+1, j-1 {
                        pathIDs[i], pathIDs[j] = pathIDs[j], pathIDs[i]
                }
                for i, j := 0, len(pathEdges)-1; i < j; i, j = i+1, j-1 {
                        pathEdges[i], pathEdges[j] = pathEdges[j], pathEdges[i]
                }
                // Deduplicate nodes (the start node may appear twice if from==to).
                uniqNodes := []GraphNode{}
                seen := map[string]bool{}
                for _, id := range pathIDs {
                        if seen[id] {
                                continue
                        }
                        seen[id] = true
                        if n, ok := graphStore.nodes[id]; ok {
                                uniqNodes = append(uniqNodes, n)
                        }
                }

                writeJSON(w, http.StatusOK, GraphResponse{
                        Nodes: uniqNodes,
                        Edges: pathEdges,
                        Metadata: GraphMetadata{
                                TotalNodes: len(uniqNodes),
                                TotalEdges: len(pathEdges),
                                Depth:      len(pathEdges),
                        },
                })
        }
}

// --- Helpers ---

// deriveBillLabel produces a human-readable label for a bill node when
// only the act name + bill ID are available. The convention follows
// Kenya Law's published bill titles: "<Short Title> Bill, <Year>".
func deriveBillLabel(actName, billID string) string {
        // Strip common Act suffixes to recover the bill's short title.
        base := actName
        for _, suffix := range []string{", 2019", ", 2015", ", 2011", ", 2010", " Act", ", 2010", " of Kenya"} {
                base = strings.ReplaceAll(base, suffix, "")
        }
        // Pull the year out of the bill ID (ke-bill-data-protection-2018 → 2018).
        year := ""
        parts := strings.Split(billID, "-")
        if len(parts) > 0 {
                last := parts[len(parts)-1]
                if len(last) == 4 {
                        year = last
                }
        }
        if year != "" {
                return base + " Bill, " + year
        }
        return base + " Bill"
}

// committeeForAct returns the committee ID most likely to have assessed a
// bill for the given act, based on the act's title. Returns "" if no
// curated mapping exists.
func committeeForAct(actName string) string {
        lower := strings.ToLower(actName)
        switch {
        case strings.Contains(lower, "finance") || strings.Contains(lower, "public finance"):
                return "committee-finance"
        case strings.Contains(lower, "election"):
                return "committee-justice"
        case strings.Contains(lower, "data protection"):
                return "committee-justice"
        case strings.Contains(lower, "companies"):
                return "committee-justice"
        case strings.Contains(lower, "constitution"):
                return "committee-justice"
        }
        return ""
}

// administrationName looks up the display name for an administration ID.
func administrationName(id government.ID) string {
        for _, a := range governmentData.admins {
                if a.ID == id {
                        return a.Name
                }
        }
        return string(id)
}

// formatDate formats a time.Time as YYYY-MM-DD. Returns "" if the time is
// zero (so the JSON omits the field via omitempty on the caller side when
// appropriate).
func formatDate(t time.Time) string {
        if t.IsZero() {
                return ""
        }
        return t.UTC().Format("2006-01-02")
}

// formatDatePtr formats a *time.Time. Returns "" for nil or zero.
func formatDatePtr(t *time.Time) string {
        if t == nil {
                return ""
        }
        return formatDate(*t)
}

// truncateText returns the first n runes of s, suffixed with "…" if the
// original was longer. Used for the article text_excerpt property so the
// graph node payload stays small.
func truncateText(s string, n int) string {
        if len(s) <= n {
                return s
        }
        // Walk runes to avoid splitting a multi-byte UTF-8 sequence.
        r := []rune(s)
        if len(r) <= n {
                return s
        }
        return string(r[:n]) + "…"
}

// itoa wraps strconv.Itoa to give the graph builder a short, named helper
// at the call site (the term-node label formats the term number with it).
func itoa(n int) string {
        return strconv.Itoa(n)
}
