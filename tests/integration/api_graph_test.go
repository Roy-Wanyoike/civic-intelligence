// Package integration — api_graph_test.go exercises the Civic Knowledge
// Graph API endpoints (ENG-I1, Wave 9). The Kenya seed data populates the
// graph at startup with acts, bills, articles, administrations, borrowing
// agreements, sample institutions, sample committees, and sample people.
//
// Endpoints covered:
//
//      GET /api/v1/graph                       -- summary (total nodes + edges)
//      GET /api/v1/graph/nodes?type=...&limit=20 -- list nodes (optionally filtered)
//      GET /api/v1/graph/node/{id}             -- single node + direct neighbours
//      GET /api/v1/graph/relationships?id=...&depth=1|2|3 -- multi-hop traversal
//      GET /api/v1/graph/search?q=...          -- search nodes by label
//      GET /api/v1/graph/paths?from=...&to=... -- shortest path (BFS)
//
// The tests assert the documented response shape (nodes / edges / metadata)
// and the FACT reality_layer tag on the summary endpoint.
package integration

import (
	"net/http"
	"strings"
	"testing"
)

// graphResponse mirrors the GraphResponse struct from services/api/cmd/graph.go.
type graphResponse struct {
	Nodes    []map[string]any `json:"nodes"`
	Edges    []map[string]any `json:"edges"`
	Metadata struct {
		TotalNodes int `json:"total_nodes"`
		TotalEdges int `json:"total_edges"`
		Depth      int `json:"depth"`
	} `json:"metadata"`
}

// TestGraph_RootSummaryReturnsFact verifies the bare /graph endpoint
// returns a summary tagged with reality_layer=FACT so citizens landing on
// the API root immediately see the platform's evidence-first promise.
func TestGraph_RootSummaryReturnsFact(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/graph"), &resp)
	assertStatus(t, "/graph", http.StatusOK, status)

	if resp["reality_layer"] != "FACT" {
		t.Errorf("graph: expected reality_layer FACT, got %v", resp["reality_layer"])
	}
	totalNodes, _ := resp["total_nodes"].(float64)
	if totalNodes == 0 {
		t.Error("graph: expected total_nodes > 0")
	}
	totalEdges, _ := resp["total_edges"].(float64)
	if totalEdges == 0 {
		t.Error("graph: expected total_edges > 0")
	}
	nodeTypes, _ := resp["node_types"].(map[string]any)
	if len(nodeTypes) == 0 {
		t.Error("graph: expected non-empty node_types map in summary")
	}
}

// TestGraph_NodesListReturnsActs verifies the nodes endpoint returns Act
// nodes when filtered by type=act.
func TestGraph_NodesListReturnsActs(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/nodes?type=act&limit=200"), &resp)
	assertStatus(t, "/graph/nodes?type=act", http.StatusOK, status)

	if resp.Metadata.TotalNodes == 0 {
		t.Fatal("graph/nodes: expected acts, got 0")
	}
	for _, n := range resp.Nodes {
		if n["type"] != "act" {
			t.Errorf("graph/nodes: type filter violated, got node of type %v (id=%v)", n["type"], n["id"])
		}
	}
}

// TestGraph_NodeDetailReturnsNeighbours verifies the node detail endpoint
// returns the requested node plus its direct neighbours (depth=1).
func TestGraph_NodeDetailReturnsNeighbours(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/node/ke-act-constitution-2010"), &resp)
	assertStatus(t, "/graph/node/ke-act-constitution-2010", http.StatusOK, status)

	if resp.Metadata.TotalNodes < 2 {
		t.Errorf("graph/node: expected at least 2 nodes (target + 1 neighbour), got %d", resp.Metadata.TotalNodes)
	}
	if resp.Metadata.Depth != 1 {
		t.Errorf("graph/node: expected depth=1, got %d", resp.Metadata.Depth)
	}
	if resp.Metadata.TotalEdges == 0 {
		t.Error("graph/node: expected at least 1 edge in detail response")
	}
}

// TestGraph_RelationshipsDepth1 verifies the relationships endpoint
// expands the neighbourhood to depth=1 and returns nodes + edges.
func TestGraph_RelationshipsDepth1(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/relationships?id=admin-uhuru-kenyatta&depth=1"), &resp)
	assertStatus(t, "/graph/relationships?id=admin-uhuru-kenyatta&depth=1", http.StatusOK, status)

	if resp.Metadata.TotalNodes < 2 {
		t.Errorf("graph/relationships: expected at least 2 nodes, got %d", resp.Metadata.TotalNodes)
	}
	if resp.Metadata.TotalEdges == 0 {
		t.Error("graph/relationships: expected at least 1 edge")
	}
}

// TestGraph_RelationshipsDepth3 verifies the relationships endpoint
// expands the neighbourhood to depth=3 (clamped from depth=10).
func TestGraph_RelationshipsDepth3(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/relationships?id=admin-uhuru-kenyatta&depth=10"), &resp)
	assertStatus(t, "/graph/relationships?depth=10", http.StatusOK, status)

	if resp.Metadata.Depth != 3 {
		t.Errorf("graph/relationships: expected depth clamped to 3, got %d", resp.Metadata.Depth)
	}
}

// TestGraph_SearchMatchesByLabel verifies the search endpoint matches
// nodes by label substring (case-insensitive).
func TestGraph_SearchMatchesByLabel(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/search?q=uhuru"), &resp)
	assertStatus(t, "/graph/search?q=uhuru", http.StatusOK, status)

	if resp.Metadata.TotalNodes == 0 {
		t.Fatal("graph/search: expected results for 'uhuru', got 0")
	}
	foundAdmin := false
	for _, n := range resp.Nodes {
		if n["id"] == "admin-uhuru-kenyatta" {
			foundAdmin = true
		}
	}
	if !foundAdmin {
		t.Errorf("graph/search: expected admin-uhuru-kenyatta in results, got %v", resp.Nodes)
	}
}

// TestGraph_PathsFindsShortestConnection verifies the paths endpoint
// returns the shortest path between two nodes. Uses the Constitution
// Bill → Constitution Act → President path.
func TestGraph_PathsFindsShortestConnection(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/paths?from=ke-bill-constitution-2008&to=president-mwai-kibaki"), &resp)
	assertStatus(t, "/graph/paths", http.StatusOK, status)

	if resp.Metadata.TotalNodes < 3 {
		t.Errorf("graph/paths: expected at least 3 nodes in path, got %d", resp.Metadata.TotalNodes)
	}
	if resp.Metadata.Depth > 4 {
		t.Errorf("graph/paths: expected short path (<=4 hops), got %d", resp.Metadata.Depth)
	}
	if len(resp.Nodes) == 0 {
		t.Fatal("graph/paths: expected non-empty nodes list")
	}
	if resp.Nodes[0]["id"] != "ke-bill-constitution-2008" {
		t.Errorf("graph/paths: expected path to start at ke-bill-constitution-2008, got %v", resp.Nodes[0]["id"])
	}
	last := resp.Nodes[len(resp.Nodes)-1]
	if last["id"] != "president-mwai-kibaki" {
		t.Errorf("graph/paths: expected path to end at president-mwai-kibaki, got %v", last["id"])
	}
}

// TestGraph_UnknownNodeReturns404 verifies the platform never invents a
// node for an unknown ID — the detail endpoint responds 404.
func TestGraph_UnknownNodeReturns404(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/graph/node/does-not-exist"), &resp)
	if status != http.StatusNotFound {
		t.Errorf("graph/node/unknown: expected 404, got %d", status)
	}
}

// TestGraph_NoPathReturns404 verifies the paths endpoint responds 404
// (not 200 with an empty graph) when the from-node does not exist.
func TestGraph_NoPathReturns404(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/graph/paths?from=does-not-exist&to=president-mwai-kibaki"), &resp)
	if status != http.StatusNotFound {
		t.Errorf("graph/paths: expected 404 for unknown from, got %d", status)
	}
}

// TestGraph_SearchMissingQReturns400 verifies the search endpoint rejects
// empty queries with 400.
func TestGraph_SearchMissingQReturns400(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/graph/search"), &resp)
	if status != http.StatusBadRequest {
		t.Errorf("graph/search: expected 400 for empty q, got %d", status)
	}
}

// TestGraph_ResponseShapeHasAllFields verifies every node carries the
// documented id/type/label fields and every edge carries source/target/type.
func TestGraph_ResponseShapeHasAllFields(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/relationships?id=admin-uhuru-kenyatta&depth=2"), &resp)
	assertStatus(t, "/graph/relationships", http.StatusOK, status)

	for i, n := range resp.Nodes {
		for _, field := range []string{"id", "type", "label"} {
			v, ok := n[field]
			if !ok || v == nil || v == "" {
				t.Errorf("graph/relationships: nodes[%d] missing required field %q", i, field)
			}
		}
	}
	for i, e := range resp.Edges {
		for _, field := range []string{"source", "target", "type"} {
			v, ok := e[field]
			if !ok || v == nil || v == "" {
				t.Errorf("graph/relationships: edges[%d] missing required field %q", i, field)
			}
		}
	}
}

// TestGraph_NodeCarriesSourceURL verifies the FACT reality_layer promise:
// every node carries the source_url from the underlying seed entity so a
// citizen can verify any relationship against the authoritative source.
func TestGraph_NodeCarriesSourceURL(t *testing.T) {
	var resp graphResponse
	status := mustGet(t, apiURL("/graph/node/ke-act-constitution-2010"), &resp)
	assertStatus(t, "/graph/node", http.StatusOK, status)

	if len(resp.Nodes) == 0 {
		t.Fatal("graph/node: expected nodes in response")
	}
	props, _ := resp.Nodes[0]["properties"].(map[string]any)
	sourceURL, _ := props["source_url"].(string)
	if sourceURL == "" {
		t.Errorf("graph/node: expected source_url on act node, got %v", props)
	}
	if !strings.HasPrefix(sourceURL, "http") {
		t.Errorf("graph/node: source_url should be a URL, got %q", sourceURL)
	}
}
