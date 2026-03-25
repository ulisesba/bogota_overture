package graph

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"sort"
)

// Location is a geographic coordinate in GeoJSON order: (lon, lat).
type Location struct {
	Lon float64
	Lat float64
}

// NodeID is the index/ID of a node in Graph.Nodes.
type NodeID int

type Node struct {
	ID       NodeID
	Location Location
}

// Edge represents a street segment between two nodes.
// The graph is undirected, but we store one Edge record per undirected pair.
type Edge struct {
	From         NodeID
	To           NodeID
	WeightMeters float64
}

// Graph is an in-memory transportation network.
// nodes are points, edges are consecutive segment pieces, and adjacency is fast lookup.
type Graph struct {
	Nodes     []Node
	Edges     []Edge
	Relations map[NodeID][]int // node -> indexes into Edges slice
}

type nodeCandidate struct {
	ID       NodeID
	DistToPt float64
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		Relations: make(map[NodeID][]int),
	}
}

// haversineMeters computes the great-circle distance between two lon/lat points.
func haversineMeters(lon1, lat1, lon2, lat2 float64) float64 {
	const R = 6371000.0
	phi1 := lat1 * math.Pi / 180.0
	phi2 := lat2 * math.Pi / 180.0
	dphi := (lat2 - lat1) * math.Pi / 180.0
	dlambda := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dphi/2.0)*math.Sin(dphi/2.0) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dlambda/2.0)*math.Sin(dlambda/2.0)
	return 2.0 * R * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
}

// connectedComponents computes connected-component ids on the undirected graph.
func connectedComponents(g *Graph) []int {
	comp := make([]int, len(g.Nodes))
	for i := range comp {
		comp[i] = -1
	}
	curr := 0

	queue := make([]NodeID, 0)
	for start := range g.Nodes {
		if comp[start] != -1 {
			continue
		}
		comp[start] = curr
		queue = queue[:0]
		queue = append(queue, NodeID(start))

		for len(queue) > 0 {
			u := queue[0]
			queue = queue[1:]
			for _, eidx := range g.Relations[u] {
				e := g.Edges[eidx]
				v := e.From
				if v == u {
					v = e.To
				}
				if comp[v] == -1 {
					comp[v] = curr
					queue = append(queue, v)
				}
			}
		}
		curr++
	}
	return comp
}

// NearestNodeID returns the closest node to point p (lon/lat).
func (g *Graph) NearestNodeID(p Location) NodeID {
	best := NodeID(-1)
	bestDist := math.Inf(1)
	for _, n := range g.Nodes {
		d := haversineMeters(p.Lon, p.Lat, n.Location.Lon, n.Location.Lat)
		if d < bestDist {
			bestDist = d
			best = n.ID
		}
	}
	return best
}

func nearestKNodeCandidates(g *Graph, p Location, k int) []nodeCandidate {
	if k <= 0 {
		return nil
	}
	cands := make([]nodeCandidate, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		d := haversineMeters(p.Lon, p.Lat, n.Location.Lon, n.Location.Lat)
		cands = append(cands, nodeCandidate{ID: n.ID, DistToPt: d})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].DistToPt < cands[j].DistToPt })
	if len(cands) > k {
		return cands[:k]
	}
	return cands
}

type pqItem struct {
	Node NodeID
	Dist float64
}

type minPQ []*pqItem

func (pq minPQ) Len() int           { return len(pq) }
func (pq minPQ) Less(i, j int) bool { return pq[i].Dist < pq[j].Dist }
func (pq minPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *minPQ) Push(x any)        { *pq = append(*pq, x.(*pqItem)) }
func (pq *minPQ) Pop() any          { old := *pq; n := len(old); item := old[n-1]; *pq = old[:n-1]; return item }

// ShortestPathDijkstra runs Dijkstra using edge weights (meters).
func (g *Graph) ShortestPathDijkstra(src, dst NodeID) (float64, []NodeID, error) {
	if src < 0 || src >= NodeID(len(g.Nodes)) || dst < 0 || dst >= NodeID(len(g.Nodes)) {
		return 0, nil, errors.New("source or destination node out of range")
	}
	if src == dst {
		return 0, []NodeID{src}, nil
	}

	dist := make([]float64, len(g.Nodes))
	prev := make([]NodeID, len(g.Nodes))
	seen := make([]bool, len(g.Nodes))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = NodeID(-1)
	}
	dist[src] = 0

	pq := &minPQ{}
	heap.Init(pq)
	heap.Push(pq, &pqItem{Node: src, Dist: 0})

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*pqItem)
		u := item.Node
		if seen[u] {
			continue
		}
		seen[u] = true
		if u == dst {
			break
		}

		for _, eidx := range g.Relations[u] {
			e := g.Edges[eidx]
			v := e.From
			if v == u {
				v = e.To
			}
			if seen[v] {
				continue
			}
			alt := dist[u] + e.WeightMeters
			if alt < dist[v] {
				dist[v] = alt
				prev[v] = u
				heap.Push(pq, &pqItem{Node: v, Dist: alt})
			}
		}
	}

	if math.IsInf(dist[dst], 1) {
		return 0, nil, errors.New("no path found between source and destination")
	}

	// Reconstruct path from dst back to src.
	path := make([]NodeID, 0)
	for at := dst; at != NodeID(-1); at = prev[at] {
		path = append(path, at)
		if at == src {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return dist[dst], path, nil
}

// RouteBetweenPoints computes shortest-path distance between two points (lon/lat),
// snapping them to the nearest graph nodes. If the nearest nodes are disconnected,
// it tries multiple candidates near A and near B to find a connected pair.
//
// Returns:
// - distMeters: shortest path length in meters
// - path: node ids along the path
// - snapAMeters / snapBMeters: distance from input points to snapped nodes
func (g *Graph) RouteBetweenPoints(A, B Location, maxSnapMeters float64) (distMeters float64, path []NodeID, snapAMeters float64, snapBMeters float64, err error) {
	src0 := g.NearestNodeID(A)
	dst0 := g.NearestNodeID(B)

	// Compute exact snapping distances for reporting.
	snapAMeters = haversineMeters(A.Lon, A.Lat, g.Nodes[src0].Location.Lon, g.Nodes[src0].Location.Lat)
	snapBMeters = haversineMeters(B.Lon, B.Lat, g.Nodes[dst0].Location.Lon, g.Nodes[dst0].Location.Lat)

	// Optional reject if points are too far from the graph.
	if maxSnapMeters > 0 {
		if snapAMeters > maxSnapMeters || snapBMeters > maxSnapMeters {
			return 0, nil, snapAMeters, snapBMeters, fmt.Errorf("snap too far (A=%.2fm, B=%.2fm > maxSnapMeters=%.2fm)", snapAMeters, snapBMeters, maxSnapMeters)
		}
	}

	// Fast path: try nearest nodes.
	distMeters, path, err = g.ShortestPathDijkstra(src0, dst0)
	if err == nil {
		return distMeters, path, snapAMeters, snapBMeters, nil
	}

	// If disconnected, try K nearest candidates around A and B.
	K := 25
	candA := nearestKNodeCandidates(g, A, K)
	candB := nearestKNodeCandidates(g, B, K)

	comps := connectedComponents(g)

	bestScore := math.Inf(1)
	bestFound := false
	var bestSrc, bestDst NodeID
	var bestPath []NodeID
	var bestDist float64
	var bestSnapA, bestSnapB float64

	for _, ca := range candA {
		for _, cb := range candB {
			// If maxSnapMeters is enabled, ignore far candidates too.
			if maxSnapMeters > 0 && (ca.DistToPt > maxSnapMeters || cb.DistToPt > maxSnapMeters) {
				continue
			}

			if comps[ca.ID] != comps[cb.ID] {
				continue
			}

			d, p, e := g.ShortestPathDijkstra(ca.ID, cb.ID)
			if e != nil {
				continue
			}

			// Score: shortest path + snapping penalty (keeps choices near the original clicks).
			score := d + ca.DistToPt + cb.DistToPt
			if score < bestScore {
				bestScore = score
				bestFound = true
				bestSrc = ca.ID
				bestDst = cb.ID
				bestPath = p
				bestDist = d
				bestSnapA = ca.DistToPt
				bestSnapB = cb.DistToPt
			}
		}
	}

	if !bestFound {
		return 0, nil, snapAMeters, snapBMeters, errors.New("no connected path found near A/B candidates")
	}

	// keep values consistent with selected pair
	_ = bestSrc
	_ = bestDst
	return bestDist, bestPath, bestSnapA, bestSnapB, nil
}

// PathToLonLat converts a path of node IDs into a polyline of (lon,lat) points.
func (g *Graph) PathToLonLat(path []NodeID) []Location {
	out := make([]Location, 0, len(path))
	for _, id := range path {
		out = append(out, g.Nodes[id].Location)
	}
	return out
}
