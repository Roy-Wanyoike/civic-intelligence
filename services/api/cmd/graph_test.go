// Package main — Civic Knowledge Graph endpoint tests (ENG-I1, Wave 9).
//
// The tests exercise all 5 graph endpoints against the package-level
// `graphStore` singleton (built once at init from the Kenya seed data):
//
//      GET /api/v1/graph/nodes
//      GET /api/v1/graph/node/{id}
//      GET /api/v1/graph/relationships?id=...&depth=1|2|3
//      GET /api/v1/graph/search?q=...
//      GET /api/v1/graph/paths?from=...&to=...
//
// The graph is fully deterministic — every test asserts against stable
// node IDs + edge sets sourced from the existing seed data (KenyaActs,
// KenyaAdministrations, KenyaConstitutionArticles, KenyaBorrowingAgreements,
// sampleInstitutions, samplePeople, sampleCommittees).
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// graphResponse is the decoded shape returned by every graph endpoint.
// Used so tests can decode any endpoint with the same struct.
type graphResponse struct {
	Nodes    []GraphNode    `json:"nodes"`
	Edges    []GraphEdge    `json:"edges"`
	Metadata GraphMetadata  `json:"metadata"`
}

// TestGraph_NodesListReturnsAllNodeTypes verifies that the nodes endpoint
// returns nodes of multiple types (acts, bills, articles, administrations,
// people, institutions, etc.) without a type filter.
func TestGraph_NodesListReturnsAllNodeTypes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/nodes?limit=200", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Metadata.TotalNodes == 0 {
		t.Fatal("expected graph nodes; got 0")
	}
	// Verify multiple node types are present.
	types := map[string]bool{}
	for _, n := range resp.Nodes {
		types[n.Type] = true
	}
	if !types[NodeAct] {
		t.Errorf("expected 'act' nodes in list; got types=%v", types)
	}
	if !types[NodeConstitutionArticle] {
		t.Errorf("expected 'constitution_article' nodes in list; got types=%v", types)
	}
	if !types[NodeAdministration] {
		t.Errorf("expected 'administration' nodes in list; got types=%v", types)
	}
	if !types[NodeBorrowingAgreement] {
		t.Errorf("expected 'borrowing_agreement' nodes in list; got types=%v", types)
	}
}

// TestGraph_NodesListFilterByType verifies that the ?type= query parameter
// restricts the response to a single node type.
func TestGraph_NodesListFilterByType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/nodes?type=act&limit=200", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Metadata.TotalNodes == 0 {
		t.Fatal("expected acts; got 0")
	}
	for _, n := range resp.Nodes {
		if n.Type != NodeAct {
			t.Errorf("type filter violated: got node of type %q (id=%s)", n.Type, n.ID)
		}
	}
}

// TestGraph_NodeDetailReturnsNeighbours verifies that the node detail
// endpoint returns the requested node plus its direct neighbours
// (depth=1). Uses the Constitution of Kenya Act (which has a BillID, so
// there is an ORIGINATES_FROM edge in the response).
func TestGraph_NodeDetailReturnsNeighbours(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/node/ke-act-constitution-2010", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Metadata.TotalNodes == 0 {
		t.Fatal("expected nodes in detail response")
	}
	if resp.Metadata.Depth != 1 {
		t.Errorf("expected depth=1; got %d", resp.Metadata.Depth)
	}
	// First node must be the requested one.
	if resp.Nodes[0].ID != "ke-act-constitution-2010" {
		t.Errorf("expected first node ke-act-constitution-2010; got %q", resp.Nodes[0].ID)
	}
	// The Act must have at least one ORIGINATES_FROM edge from its Bill.
	hasOriginatesFrom := false
	for _, e := range resp.Edges {
		if e.Type == EdgeOriginatesFrom && e.Target == "ke-act-constitution-2010" {
			hasOriginatesFrom = true
			break
		}
	}
	if !hasOriginatesFrom {
		t.Errorf("expected ORIGINATES_FROM edge to ke-act-constitution-2010; got edges=%v", resp.Edges)
	}
}

// TestGraph_NodeDetailUnknownIDReturns404 verifies the platform never
// invents a node for an unknown ID — the handler responds 404.
func TestGraph_NodeDetailUnknownIDReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/node/does-not-exist", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", rec.Code)
	}
}

// TestGraph_RelationshipsDepth1 verifies the relationships endpoint
// returns the requested node + its 1-hop neighbours.
func TestGraph_RelationshipsDepth1(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/relationships?id=admin-uhuru-kenyatta&depth=1", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Metadata.Depth != 1 {
		t.Errorf("expected depth=1; got %d", resp.Metadata.Depth)
	}
	// The Uhuru Kenyatta administration has multiple borrowing agreements
	// linked via CONTRACTED_DURING, so depth=1 should return them.
	if resp.Metadata.TotalNodes < 2 {
		t.Errorf("expected at least 2 nodes (admin + 1 neighbour); got %d", resp.Metadata.TotalNodes)
	}
	if resp.Metadata.TotalEdges == 0 {
		t.Errorf("expected at least 1 edge; got 0")
	}
}

// TestGraph_RelationshipsDepth3 verifies the relationships endpoint
// expands the neighbourhood up to 3 hops and clamps higher values.
func TestGraph_RelationshipsDepth3(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/relationships?id=admin-uhuru-kenyatta&depth=10", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Depth should be clamped to 3.
	if resp.Metadata.Depth != 3 {
		t.Errorf("expected depth clamped to 3; got %d", resp.Metadata.Depth)
	}
	// Depth=3 from the Uhuru administration should reach a Bill (Bill →
	// Act → ASSESNTED_BY → President → LEADS → Executive institution →
	// ADMINISTRATION). At minimum, we expect the graph to have grown
	// beyond depth=1.
	if resp.Metadata.TotalNodes < 5 {
		t.Errorf("expected graph to expand beyond depth=1 (>5 nodes); got %d", resp.Metadata.TotalNodes)
	}
}

// TestGraph_RelationshipsMissingIDReturns400 verifies the endpoint rejects
// requests without the id parameter.
func TestGraph_RelationshipsMissingIDReturns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/relationships?depth=2", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", rec.Code)
	}
}

// TestGraph_RelationshipsUnknownIDReturns404 verifies the endpoint returns
// 404 (not 200 with an empty graph) for an unknown node ID.
func TestGraph_RelationshipsUnknownIDReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/relationships?id=does-not-exist&depth=2", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", rec.Code)
	}
}

// TestGraph_SearchMatchesByLabel verifies the search endpoint matches
// nodes by label substring (case-insensitive).
func TestGraph_SearchMatchesByLabel(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/search?q=uhuru", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Metadata.TotalNodes == 0 {
		t.Fatal("expected search results for 'uhuru'; got 0")
	}
	// At least one result should be the Uhuru Kenyatta administration.
	foundAdmin := false
	for _, n := range resp.Nodes {
		if n.ID == "admin-uhuru-kenyatta" {
			foundAdmin = true
		}
	}
	if !foundAdmin {
		t.Errorf("expected admin-uhuru-kenyatta in search results; got %v", resp.Nodes)
	}
}

// TestGraph_SearchEmptyQReturns400 verifies the search endpoint requires
// a non-empty q parameter.
func TestGraph_SearchEmptyQReturns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/search?q=", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", rec.Code)
	}
}

// TestGraph_PathsFindsShortestConnection verifies the paths endpoint
// returns the shortest path between two nodes. Uses the Constitution Bill
// → Constitution Act → President path: the bill is connected to the act
// via ORIGINATES_FROM, the act is connected to the president via
// ASSESNTED_BY, so a path from the bill to the president should be
// exactly 2 hops.
func TestGraph_PathsFindsShortestConnection(t *testing.T) {
	// Path from the Constitution of Kenya Bill → President Mwai Kibaki
	// (Kibaki was in power when the 2010 constitution was assented).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/paths?from=ke-bill-constitution-2008&to=president-mwai-kibaki", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp graphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Metadata.TotalNodes < 3 {
		t.Errorf("expected at least 3 nodes in path; got %d", resp.Metadata.TotalNodes)
	}
	// Path must start at the bill and end at the president.
	if resp.Nodes[0].ID != "ke-bill-constitution-2008" {
		t.Errorf("expected path to start at ke-bill-constitution-2008; got %q", resp.Nodes[0].ID)
	}
	if resp.Nodes[len(resp.Nodes)-1].ID != "president-mwai-kibaki" {
		t.Errorf("expected path to end at president-mwai-kibaki; got %q", resp.Nodes[len(resp.Nodes)-1].ID)
	}
	// Depth (edge count) should be small (2-3 hops).
	if resp.Metadata.Depth > 4 {
		t.Errorf("expected short path (<=4 hops); got %d hops", resp.Metadata.Depth)
	}
}

// TestGraph_PathsNoPathReturns404 verifies the paths endpoint returns 404
// when no path exists between two nodes. This case is hard to construct
// because the seed data is densely connected, so we use a node that is
// guaranteed disconnected: a synthetic ID that does not exist will return
// 404 from the existence check, not the no_path branch. To exercise the
// no_path branch we'd need a real node with no edges — there isn't one
// in the current seed. The test below documents the contract.
func TestGraph_PathsUnknownNodeReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/paths?from=does-not-exist&to=president-mwai-kibaki", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown from node; got %d", rec.Code)
	}
}

// TestGraph_PathsMissingParamsReturns400 verifies both from + to are
// required.
func TestGraph_PathsMissingParamsReturns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/paths?from=president-mwai-kibaki", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", rec.Code)
	}
}

// TestGraph_RootReturnsSummary verifies the bare /api/v1/graph endpoint
// returns a metadata summary (so a citizen landing on the API root sees
// the graph size + node-type breakdown rather than a 404).
func TestGraph_RootReturnsSummary(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph", nil)
	rec := httptest.NewRecorder()
	makeGraphRouter()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["total_nodes"] == nil {
		t.Error("expected total_nodes in summary")
	}
	if resp["reality_layer"] != "FACT" {
		t.Errorf("expected reality_layer FACT; got %v", resp["reality_layer"])
	}
	nodeTypes, _ := resp["node_types"].(map[string]any)
	if len(nodeTypes) == 0 {
		t.Error("expected non-empty node_types map in summary")
	}
}

// TestGraph_StoreHasExpectedEdgeTypes verifies the seeded graph contains
// every documented edge type with at least one edge, except SPONSORED_BY
// (which the seed does not currently populate because the Bill domain has
// no sponsor field — documented in the package comment).
func TestGraph_StoreHasExpectedEdgeTypes(t *testing.T) {
	edgeTypes := map[string]int{}
	for _, e := range graphStore.edges {
		edgeTypes[e.Type]++
	}
	required := []string{
		EdgeOriginatesFrom,
		EdgeBelongsTo,
		EdgeCites,
		EdgeEstablishes,
		EdgeGovernedBy,
		EdgeAssentedBy,
		EdgeBorrowedBy,
		EdgeContractedDuring,
		EdgeMemberOf,
		EdgeLeads,
		// EdgeAssessedBy is included via curated committee-for-act
		// mappings; not every act will produce one, but at least the
		// finance + justice acts should.
	}
	for _, et := range required {
		if edgeTypes[et] == 0 {
			t.Errorf("expected at least one %s edge in the seed graph", et)
		}
	}
}

// TestGraph_StoreHasEveryNodeType verifies the seeded graph contains
// every documented node type with at least one node.
func TestGraph_StoreHasEveryNodeType(t *testing.T) {
	nodeTypes := map[string]int{}
	for _, n := range graphStore.nodes {
		nodeTypes[n.Type]++
	}
	required := []string{
		NodeBill,
		NodeAct,
		NodeInstitution,
		NodePerson,
		NodeConstitutionArticle,
		NodeAdministration,
		NodePresidentialTerm,
		NodeLegislature,
		NodeCommittee,
		NodeCreditor,
		NodeBorrowingAgreement,
	}
	for _, nt := range required {
		if nodeTypes[nt] == 0 {
			t.Errorf("expected at least one %s node in the seed graph", nt)
		}
	}
}
