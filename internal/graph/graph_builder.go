package graph

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// DefaultMergeEpsMeters is the default vertex-merge distance when building from GeoJSON.
const DefaultMergeEpsMeters = 1.0

// BuildOptions configures GeoJSON graph construction.
type BuildOptions struct {
	// MergeEpsMeters merges distinct coordinates within this great-circle distance (meters).
	// Values <= 0 are replaced with DefaultMergeEpsMeters.
	MergeEpsMeters float64
}

// GeoJSON decoding types (internal).
type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
	Geometry   geometry        `json:"geometry"`
}

type geometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// gridKey is a spatial-hash cell index (vertices within merge tolerance map here).
type gridKey struct {
	I int64
	J int64
}

type edgeKey struct {
	A NodeID
	B NodeID
}

// geoBuilder holds transient dedupe state while loading GeoJSON; it is not part of Graph.
type geoBuilder struct {
	g         *Graph
	cellNodes map[gridKey][]NodeID
	edgeSet   map[edgeKey]struct{}

	mergeEpsMeters float64
	dLat           float64
	dLon           float64
	refSet         bool
}

func newGeoBuilder(epsMeters float64) *geoBuilder {
	if epsMeters <= 0 {
		epsMeters = DefaultMergeEpsMeters
	}
	return &geoBuilder{
		g: &Graph{
			Relations: make(map[NodeID][]int),
		},
		cellNodes:      make(map[gridKey][]NodeID),
		edgeSet:        make(map[edgeKey]struct{}),
		mergeEpsMeters: epsMeters,
	}
}

func (bb *geoBuilder) ensureGrid(p Location) {
	if bb.refSet {
		return
	}
	latRad := p.Lat * math.Pi / 180.0
	cosLat := math.Cos(latRad)
	if cosLat < 1e-6 {
		cosLat = 1e-6
	}
	bb.dLat = bb.mergeEpsMeters / 111320.0
	bb.dLon = bb.mergeEpsMeters / (111320.0 * cosLat)
	bb.refSet = true
}

func (bb *geoBuilder) cellIndex(p Location) gridKey {
	return gridKey{
		I: int64(math.Floor(p.Lat / bb.dLat)),
		J: int64(math.Floor(p.Lon / bb.dLon)),
	}
}

func (bb *geoBuilder) addNode(p Location) NodeID {
	bb.ensureGrid(p)
	ck := bb.cellIndex(p)
	eps := bb.mergeEpsMeters

	for di := int64(-1); di <= 1; di++ {
		for dj := int64(-1); dj <= 1; dj++ {
			key := gridKey{I: ck.I + di, J: ck.J + dj}
			for _, nid := range bb.cellNodes[key] {
				q := bb.g.Nodes[nid].Location
				if haversineMeters(p.Lon, p.Lat, q.Lon, q.Lat) <= eps {
					return nid
				}
			}
		}
	}

	g := bb.g
	id := NodeID(len(g.Nodes))
	g.Nodes = append(g.Nodes, Node{ID: id, Location: p})
	bb.cellNodes[ck] = append(bb.cellNodes[ck], id)
	return id
}

func normalizeEdgeKey(a, b NodeID) edgeKey {
	if a < b {
		return edgeKey{A: a, B: b}
	}
	return edgeKey{A: b, B: a}
}

func (bb *geoBuilder) addUndirectedEdge(from, to NodeID) {
	if from == to {
		return
	}
	key := normalizeEdgeKey(from, to)
	if _, exists := bb.edgeSet[key]; exists {
		return
	}
	bb.edgeSet[key] = struct{}{}

	g := bb.g
	pa := g.Nodes[from].Location
	pb := g.Nodes[to].Location
	w := haversineMeters(pa.Lon, pa.Lat, pb.Lon, pb.Lat)

	idx := len(g.Edges)
	g.Edges = append(g.Edges, Edge{From: from, To: to, WeightMeters: w})
	g.Relations[from] = append(g.Relations[from], idx)
	g.Relations[to] = append(g.Relations[to], idx)
}

func decodeLineString(raw json.RawMessage) ([]Location, error) {
	var coords [][]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return nil, err
	}
	out := make([]Location, 0, len(coords))
	for _, c := range coords {
		if len(c) < 2 {
			continue
		}
		out = append(out, Location{Lon: c[0], Lat: c[1]})
	}
	return out, nil
}

func decodeMultiLineString(raw json.RawMessage) ([][]Location, error) {
	var coords [][][]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return nil, err
	}
	out := make([][]Location, 0, len(coords))
	for _, line := range coords {
		pts := make([]Location, 0, len(line))
		for _, c := range line {
			if len(c) < 2 {
				continue
			}
			pts = append(pts, Location{Lon: c[0], Lat: c[1]})
		}
		out = append(out, pts)
	}
	return out, nil
}

func addPolylineToGraph(bb *geoBuilder, pts []Location) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		a := bb.addNode(pts[i])
		c := bb.addNode(pts[i+1])
		bb.addUndirectedEdge(a, c)
	}
}

// FromGeoJSON builds a street graph from a GeoJSON FeatureCollection of LineString/MultiLineString.
// Vertices closer than DefaultMergeEpsMeters are merged (see BuildOptions).
func FromGeoJSON(path string) (*Graph, error) {
	return FromGeoJSONWithOptions(path, BuildOptions{})
}

// FromGeoJSONWithOptions is like FromGeoJSON but allows tuning merge tolerance.
func FromGeoJSONWithOptions(path string, opts BuildOptions) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fc featureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}
	if fc.Type != "FeatureCollection" {
		return nil, fmt.Errorf("expected FeatureCollection, got %q", fc.Type)
	}

	eps := opts.MergeEpsMeters
	if eps <= 0 {
		eps = DefaultMergeEpsMeters
	}
	gb := newGeoBuilder(eps)
	for _, f := range fc.Features {
		switch f.Geometry.Type {
		case "LineString":
			pts, err := decodeLineString(f.Geometry.Coordinates)
			if err != nil {
				return nil, err
			}
			addPolylineToGraph(gb, pts)
		case "MultiLineString":
			lines, err := decodeMultiLineString(f.Geometry.Coordinates)
			if err != nil {
				return nil, err
			}
			for _, line := range lines {
				addPolylineToGraph(gb, line)
			}
		default:
			continue
		}
	}

	return gb.g, nil
}

// BuildFromGeoJSON is an alias for FromGeoJSON (keeps older call sites working).
func BuildFromGeoJSON(path string) (*Graph, error) {
	return FromGeoJSON(path)
}

// BuildFromGeoJSONWithOptions is an alias for FromGeoJSONWithOptions.
func BuildFromGeoJSONWithOptions(path string, opts BuildOptions) (*Graph, error) {
	return FromGeoJSONWithOptions(path, opts)
}
