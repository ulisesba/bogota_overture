package main

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"bogota_overture/internal/graph"
)

//go:embed web/*
var webAssets embed.FS

type pointReq struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type routeReq struct {
	A              pointReq `json:"A"`
	B              pointReq `json:"B"`
	MaxSnapMeters float64  `json:"maxSnapMeters"`
}

type routeResp struct {
	DistanceM float64        `json:"distance_m"`
	SnapA_M   float64        `json:"snapA_m"`
	SnapB_M   float64        `json:"snapB_m"`
	Path      []graph.Location `json:"path"`
}

func main() {
	geojsonPath := "chico_streets.geojson"
	g, err := graph.FromGeoJSON(geojsonPath)
	if err != nil {
		log.Fatal("failed to load graph: ", err)
	}
	log.Printf("Graph loaded (%d nodes, %d edges).", len(g.Nodes), len(g.Edges))

	mux := http.NewServeMux()

	// Serve static web files (index + app.js).
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		serveEmbeddedFile(w, r, "web/index.html")
	})
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		// Example: /static/app.js
		rel := r.URL.Path[len("/static/"):]
		serveEmbeddedFile(w, r, "web/"+rel)
	})

	// Edges for rendering on map.
	mux.HandleFunc("/api/graph/edges", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		geo := buildEdgesGeoJSON(g)
		_ = json.NewEncoder(w).Encode(geo)
	})

	// Routing endpoint.
	mux.HandleFunc("/api/route", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req routeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		A := graph.Location{Lon: req.A.Lon, Lat: req.A.Lat}
		B := graph.Location{Lon: req.B.Lon, Lat: req.B.Lat}

		start := time.Now()
		distM, path, snapA, snapB, err := g.RouteBetweenPoints(A, B, req.MaxSnapMeters)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		resp := routeResp{
			DistanceM: distM,
			SnapA_M:   snapA,
			SnapB_M:   snapB,
			Path:      g.PathToLonLat(path),
		}

		_ = json.NewEncoder(w).Encode(resp)
		log.Printf("route computed in %s, dist=%.2fm", time.Since(start), distM)
	})

	addr := ":8080"
	log.Printf("Server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// serveEmbeddedFile serves one embedded asset from webAssets.
func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, name string) {
	b, err := webAssets.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Minimal content-type handling.
	if len(name) >= 3 && name[len(name)-3:] == "js" {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	} else if len(name) >= 4 && name[len(name)-4:] == "html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.Write(b)
}

// buildEdgesGeoJSON converts the internal graph edges into a GeoJSON FeatureCollection.
// Each edge is a LineString with 2 coordinates: [[lonA,latA],[lonB,latB]]
func buildEdgesGeoJSON(g *graph.Graph) map[string]any {
	features := make([]any, 0, len(g.Edges))
	for _, e := range g.Edges {
		a := g.Nodes[e.From].Location
		b := g.Nodes[e.To].Location
		features = append(features, map[string]any{
			"type": "Feature",
			"geometry": map[string]any{
				"type": "LineString",
				"coordinates": [][]float64{
					{a.Lon, a.Lat},
					{b.Lon, b.Lat},
				},
			},
			"properties": map[string]any{},
		})
	}

	return map[string]any{
		"type":     "FeatureCollection",
		"features": features,
	}
}

