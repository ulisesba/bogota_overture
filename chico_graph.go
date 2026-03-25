package main

import (
	"flag"
	"fmt"
	"log"

	"bogota_overture/internal/graph"
)

// CLI entrypoint.
// Examples:
//   go run chico_graph.go
//   go run chico_graph.go -latA 4.67 -lonA -74.05 -latB 4.68 -lonB -74.04
func main() {
	var (
		geojsonPath   = flag.String("geojson", "chico_streets.geojson", "Path to Overture streets GeoJSON")
		latA          = flag.Float64("latA", 0, "Latitude of point A")
		lonA          = flag.Float64("lonA", 0, "Longitude of point A")
		latB          = flag.Float64("latB", 0, "Latitude of point B")
		lonB          = flag.Float64("lonB", 0, "Longitude of point B")
		maxSnapMeters = flag.Float64("maxSnapMeters", 300, "Reject/ignore snaps farther than this (0 disables)")
	)
	flag.Parse()

	g, err := graph.FromGeoJSON(*geojsonPath)
	if err != nil {
		log.Fatal("failed to load graph:", err)
	}
	log.Printf("Graph loaded (%d nodes, %d edges).", len(g.Nodes), len(g.Edges))

	// If any point is missing, only print info.
	if *latA == 0 && *lonA == 0 && *latB == 0 && *lonB == 0 {
		fmt.Println("Tip: run with -latA -lonA -latB -lonB to compute route distance.")
		return
	}

	A := graph.Location{Lon: *lonA, Lat: *latA}
	B := graph.Location{Lon: *lonB, Lat: *latB}

	distM, path, snapA, snapB, err := g.RouteBetweenPoints(A, B, *maxSnapMeters)
	if err != nil {
		log.Fatal("route error:", err)
	}

	fmt.Printf("Shortest path distance: %.2f meters (%.3f km)\n", distM, distM/1000.0)
	fmt.Printf("Snap distances: snapA=%.2f m snapB=%.2f m\n", snapA, snapB)
	fmt.Printf("Path nodes: %d\n", len(path))
}

/* package main

import (
	"flag"
	"fmt"
	"log"

	"bogota_overture/internal/graph"
)

// CLI entrypoint:
//   go run chico_graph.go -geojson chico_streets.geojson
//   go run chico_graph.go -latA 4.67 -lonA -74.05 -latB 4.68 -lonB -74.04
func main() {
	var (
		geojsonPath   = flag.String("geojson", "chico_streets.geojson", "Path to Overture streets GeoJSON")
		latA          = flag.Float64("latA", 0, "Latitude of point A")
		lonA          = flag.Float64("lonA", 0, "Longitude of point A")
		latB          = flag.Float64("latB", 0, "Latitude of point B")
		lonB          = flag.Float64("lonB", 0, "Longitude of point B")
		maxSnapMeters = flag.Float64("maxSnapMeters", 300, "Reject/ignore snaps farther than this (0 disables)")
	)
	flag.Parse()

	g, err := graph.FromGeoJSON(*geojsonPath)
	if err != nil {
		log.Fatal("failed to load graph:", err)
	}
	log.Printf("Graph loaded (%d nodes, %d edges).", len(g.Nodes), len(g.Edges))

	// If any point is missing, only print info.
	if *latA == 0 && *lonA == 0 && *latB == 0 && *lonB == 0 {
		fmt.Println("Tip: run with -latA -lonA -latB -lonB to compute route distance.")
		return
	}

	A := graph.Location{Lon: *lonA, Lat: *latA}
	B := graph.Location{Lon: *lonB, Lat: *latB}

	distM, path, snapA, snapB, err := g.RouteBetweenPoints(A, B, *maxSnapMeters)
	if err != nil {
		log.Fatal("route error:", err)
	}

	fmt.Printf("Shortest path distance: %.2f meters (%.3f km)\n", distM, distM/1000.0)
	fmt.Printf("Snap distances: snapA=%.2f m snapB=%.2f m\n", snapA, snapB)
	fmt.Printf("Path nodes: %d\n", len(path))
}

package main

import (
	"flag"
	"fmt"
	"log"

	"bogota_overture/internal/graph"
)

// CLI entrypoint:
//   go run chico_graph.go
//   go run chico_graph.go -latA 4.67 -lonA -74.05 -latB 4.68 -lonB -74.04
func main() {
	var (
		geojsonPath   = flag.String("geojson", "chico_streets.geojson", "Path to Overture streets GeoJSON")
		latA          = flag.Float64("latA", 0, "Latitude of point A")
		lonA          = flag.Float64("lonA", 0, "Longitude of point A")
		latB          = flag.Float64("latB", 0, "Latitude of point B")
		lonB          = flag.Float64("lonB", 0, "Longitude of point B")
		maxSnapMeters = flag.Float64("maxSnapMeters", 300, "Reject/ignore snaps farther than this (0 disables)")
	)
	flag.Parse()

	g, err := graph.FromGeoJSON(*geojsonPath)
	if err != nil {
		log.Fatal("failed to load graph:", err)
	}

	log.Printf("Graph loaded (%d nodes, %d edges).", len(g.Nodes), len(g.Edges))

	// If any point is missing (still 0), only print info.
	if *latA == 0 && *lonA == 0 && *latB == 0 && *lonB == 0 {
		fmt.Println("Tip: run with -latA -lonA -latB -lonB to compute route distance.")
		return
	}

	A := graph.Location{Lon: *lonA, Lat: *latA}
	B := graph.Location{Lon: *lonB, Lat: *latB}

	distM, path, snapA, snapB, err := g.RouteBetweenPoints(A, B, *maxSnapMeters)
	if err != nil {
		log.Fatal("route error:", err)
	}

	fmt.Printf("Shortest path distance: %.2f meters (%.3f km)\n", distM, distM/1000.0)
	fmt.Printf("Snap distances: snapA=%.2f m snapB=%.2f m\n", snapA, snapB)
	fmt.Printf("Path nodes: %d\n", len(path))
}

package main


import (
	"container/heap"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
)

// File-level GeoJSON shape.
type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
	Geometry   geometry        `json:"geometry"`
}

// Geometry is decoded by type because coordinates shape changes by geometry type.
type geometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// Point stores one coordinate pair in lon/lat order (GeoJSON standard).
type Point struct {
	Lon float64
	Lat float64
}

type NodeID int

type Node struct {
	ID    NodeID
	Point Point
}

type Edge struct {
	From         NodeID
	To           NodeID
	WeightMeters float64
}

type Graph struct {
	Nodes []Node
	Edges []Edge
	Relations map[NodeID][]int // node -> indexes in Edges

	nodeByKey map[pointKey]NodeID
	edgeSet   map[edgeKey]struct{}
}

type pointKey struct {
	LonBits uint64
	LatBits uint64
}

type edgeKey struct {
	A NodeID
	B NodeID
}

func newGraph() *Graph {
	return &Graph{
		Relations: make(map[NodeID][]int),
		nodeByKey: make(map[pointKey]NodeID),
		edgeSet:   make(map[edgeKey]struct{}),
	}
}

func (g *Graph) addNode(p Point) NodeID {
	key := pointKey{
		LonBits: math.Float64bits(p.Lon),
		LatBits: math.Float64bits(p.Lat),
	}
	if id, ok := g.nodeByKey[key]; ok {
		return id
	}

	id := NodeID(len(g.Nodes))
	g.Nodes = append(g.Nodes, Node{ID: id, Point: p})
	g.nodeByKey[key] = id
	return id
}

func normalizeEdge(a, b NodeID) edgeKey {
	if a < b {
		return edgeKey{A: a, B: b}
	}
	return edgeKey{A: b, B: a}
}

func (g *Graph) addUndirectedEdge(a, b NodeID) {
	if a == b {
		return
	}
	key := normalizeEdge(a, b)
	if _, exists := g.edgeSet[key]; exists {
		return
	}
	g.edgeSet[key] = struct{}{}

	pa := g.Nodes[a].Point
	pb := g.Nodes[b].Point
	w := haversineMeters(pa.Lon, pa.Lat, pb.Lon, pb.Lat)

	idx := len(g.Edges)
	g.Edges = append(g.Edges, Edge{From: a, To: b, WeightMeters: w})
	g.Relations[a] = append(g.Relations[a], idx)
	g.Relations[b] = append(g.Relations[b], idx)
}

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

func decodeLineString(raw json.RawMessage) ([]Point, error) {
	var coords [][]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return nil, err
	}
	out := make([]Point, 0, len(coords))
	for _, c := range coords {
		if len(c) < 2 {
			continue
		}
		out = append(out, Point{Lon: c[0], Lat: c[1]})
	}
	return out, nil
}

func decodeMultiLineString(raw json.RawMessage) ([][]Point, error) {
	var coords [][][]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return nil, err
	}
	out := make([][]Point, 0, len(coords))
	for _, line := range coords {
		pts := make([]Point, 0, len(line))
		for _, c := range line {
			if len(c) < 2 {
				continue
			}
			pts = append(pts, Point{Lon: c[0], Lat: c[1]})
		}
		out = append(out, pts)
	}
	return out, nil
}

func addPolylineToGraph(g *Graph, pts []Point) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		a := g.addNode(pts[i])
		b := g.addNode(pts[i+1])
		g.addUndirectedEdge(a, b)
	}
}

func buildGraphFromGeoJSON(path string) (*Graph, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fc featureCollection
	if err := json.Unmarshal(b, &fc); err != nil {
		return nil, err
	}
	if fc.Type != "FeatureCollection" {
		return nil, fmt.Errorf("expected FeatureCollection, got %q", fc.Type)
	}

	g := newGraph()
	for _, f := range fc.Features {
		if f.Geometry.Type == "" || len(f.Geometry.Coordinates) == 0 {
			continue
		}

		switch f.Geometry.Type {
		case "LineString":
			pts, err := decodeLineString(f.Geometry.Coordinates)
			if err != nil {
				return nil, err
			}
			addPolylineToGraph(g, pts)
		case "MultiLineString":
			lines, err := decodeMultiLineString(f.Geometry.Coordinates)
			if err != nil {
				return nil, err
			}
			for _, line := range lines {
				addPolylineToGraph(g, line)
			}
		default:
			// Ignore unsupported geometry types for this street graph task.
			continue
		}
	}
	return g, nil
}

func nearestNodeID(g *Graph, p Point) NodeID {
	best := NodeID(-1)
	bestDist := math.Inf(1)
	for _, n := range g.Nodes {
		d := haversineMeters(p.Lon, p.Lat, n.Point.Lon, n.Point.Lat)
		if d < bestDist {
			bestDist = d
			best = n.ID
		}
	}
	return best
}

type nodeCandidate struct {
	ID       NodeID
	DistToPt float64
}

func nearestKNodeCandidates(g *Graph, p Point, k int) []nodeCandidate {
	if k <= 0 {
		return nil
	}
	cands := make([]nodeCandidate, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		d := haversineMeters(p.Lon, p.Lat, n.Point.Lon, n.Point.Lat)
		cands = append(cands, nodeCandidate{ID: n.ID, DistToPt: d})
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].DistToPt < cands[j].DistToPt
	})
	if len(cands) > k {
		return cands[:k]
	}
	return cands
}

func componentIDs(g *Graph) []int {
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

type pqItem struct {
	Node NodeID
	Dist float64
	Idx  int
}

type minPQ []*pqItem

func (pq minPQ) Len() int { return len(pq) }
func (pq minPQ) Less(i, j int) bool {
	return pq[i].Dist < pq[j].Dist
}
func (pq minPQ) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Idx = i
	pq[j].Idx = j
}
func (pq *minPQ) Push(x any) {
	item := x.(*pqItem)
	item.Idx = len(*pq)
	*pq = append(*pq, item)
}
func (pq *minPQ) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Idx = -1
	*pq = old[:n-1]
	return item
}

func shortestPathDijkstra(g *Graph, src, dst NodeID) (float64, []NodeID, error) {
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
			v := e.To
			if e.From == u {
				v = e.To
			} else {
				v = e.From
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
	// Reverse in-place.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return dist[dst], path, nil
}

func parseArgsPoint(latStr, lonStr string) (Point, error) {
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return Point{}, fmt.Errorf("invalid latitude %q: %w", latStr, err)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return Point{}, fmt.Errorf("invalid longitude %q: %w", lonStr, err)
	}
	return Point{Lon: lon, Lat: lat}, nil
}

func main() {
	graphPath := "chico_streets.geojson"
	g, err := buildGraphFromGeoJSON(graphPath)
	if err != nil {
		fmt.Println("Error building graph:", err)
		os.Exit(1)
	}

	fmt.Println("Graph built from", graphPath)
	fmt.Println("Number of nodes:", len(g.Nodes))
	fmt.Println("Number of edges:", len(g.Edges))

	// Optional routing mode:
	// go run chico_graph.go <latA> <lonA> <latB> <lonB>
	if len(os.Args) == 5 {
		A, err := parseArgsPoint(os.Args[1], os.Args[2])
		if err != nil {
			fmt.Println("Error parsing point A:", err)
			os.Exit(1)
		}
		B, err := parseArgsPoint(os.Args[3], os.Args[4])
		if err != nil {
			fmt.Println("Error parsing point B:", err)
			os.Exit(1)
		}

		src := nearestNodeID(g, A)
		dst := nearestNodeID(g, B)
		comp := componentIDs(g)

		var (
			dist float64
			path []NodeID
		)

		// First try direct nearest nodes.
		dist, path, err = shortestPathDijkstra(g, src, dst)
		if err != nil {
			// Fallback: try multiple nearby candidates and pick a connected pair.
			const K = 25
			candA := nearestKNodeCandidates(g, A, K)
			candB := nearestKNodeCandidates(g, B, K)

			bestScore := math.Inf(1)
			bestFound := false
			var bestSrc, bestDst NodeID
			var bestDist float64
			var bestPath []NodeID
			var bestSnapA, bestSnapB float64

			for _, ca := range candA {
				for _, cb := range candB {
					if comp[ca.ID] != comp[cb.ID] {
						continue
					}
					dPath, pNodes, e2 := shortestPathDijkstra(g, ca.ID, cb.ID)
					if e2 != nil {
						continue
					}
					// Score includes both path distance and snapping penalty.
					score := dPath + ca.DistToPt + cb.DistToPt
					if score < bestScore {
						bestScore = score
						bestFound = true
						bestSrc = ca.ID
						bestDst = cb.ID
						bestDist = dPath
						bestPath = pNodes
						bestSnapA = ca.DistToPt
						bestSnapB = cb.DistToPt
					}
				}
			}

			if !bestFound {
				fmt.Println("Routing error: no connected path found near A/B candidates.")
				fmt.Println("Nearest A node component:", comp[src], "Nearest B node component:", comp[dst])
				os.Exit(1)
			}

			src = bestSrc
			dst = bestDst
			dist = bestDist
			path = bestPath
			fmt.Printf("Info: nearest A/B nodes were disconnected; used connected alternatives (snap A=%.2fm, snap B=%.2fm)\n", bestSnapA, bestSnapB)
		}

		fmt.Printf("A (lat,lon): %.7f, %.7f\n", A.Lat, A.Lon)
		fmt.Printf("B (lat,lon): %.7f, %.7f\n", B.Lat, B.Lon)
		fmt.Println("Nearest source node ID:", src)
		fmt.Println("Nearest destination node ID:", dst)
		fmt.Printf("Shortest path distance: %.2f meters (%.3f km)\n", dist, dist/1000.0)
		fmt.Println("Path node count:", len(path))
		fmt.Println("Path node IDs:", path)
		return
	}

	fmt.Println("Tip: to route between 2 points run:")
	fmt.Println("go run chico_graph.go <latA> <lonA> <latB> <lonB>")
}

*/

