let canvas;
let ctx;

let edges = []; // [{a:{lon,lat}, b:{lon,lat}}]
let nodes = []; // [{lon,lat}] unique endpoints (from edges)

let bbox = null; // {minLon,maxLon,minLat,maxLat}

let clickA = null; // {lat, lon}
let clickB = null; // {lat, lon}
let routePath = null; // [{lon,lat}, ...]

function setOut(text) {
  document.getElementById("out").textContent = text;
}

function normalizeLonLat(pt) {
  // Go encodes points as { "Lon": ..., "Lat": ... }.
  // In our app we store and use { lon: ..., lat: ... }.
  if (pt == null) return null;
  const lon = pt.lon ?? pt.Lon;
  const lat = pt.lat ?? pt.Lat;
  if (typeof lon !== "number" || typeof lat !== "number") return null;
  return { lon, lat };
}

function lonLatToCanvasXY(lon, lat) {
  // Project lon/lat linearly into the canvas using bbox.
  // x grows with lon, y grows with lat but canvas y increases downward, so we flip.
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  const { minLon, maxLon, minLat, maxLat } = bbox;

  const x = ((lon - minLon) / (maxLon - minLon)) * w;
  const y = ((maxLat - lat) / (maxLat - minLat)) * h;
  return { x, y };
}

function canvasXYToLonLat(x, y) {
  // Inverse of lonLatToCanvasXY.
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  const { minLon, maxLon, minLat, maxLat } = bbox;

  const lon = minLon + (x / w) * (maxLon - minLon);
  const lat = maxLat - (y / h) * (maxLat - minLat);
  return { lat, lon };
}

function computeBBoxFromEdges() {
  let minLon = Infinity;
  let maxLon = -Infinity;
  let minLat = Infinity;
  let maxLat = -Infinity;

  for (const e of edges) {
    for (const p of [e.a, e.b]) {
      minLon = Math.min(minLon, p.lon);
      maxLon = Math.max(maxLon, p.lon);
      minLat = Math.min(minLat, p.lat);
      maxLat = Math.max(maxLat, p.lat);
    }
  }

  bbox = { minLon, maxLon, minLat, maxLat };
}

function resizeCanvas() {
  // Keep drawing crisp with devicePixelRatio.
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  canvas.width = Math.floor(w * dpr);
  canvas.height = Math.floor(h * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
}

function clear() {
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  ctx.clearRect(0, 0, w, h);
}

function drawEdges() {
  ctx.lineWidth = 1;
  ctx.strokeStyle = "#444";
  ctx.globalAlpha = 0.65;

  for (const e of edges) {
    const p1 = lonLatToCanvasXY(e.a.lon, e.a.lat);
    const p2 = lonLatToCanvasXY(e.b.lon, e.b.lat);
    ctx.beginPath();
    ctx.moveTo(p1.x, p1.y);
    ctx.lineTo(p2.x, p2.y);
    ctx.stroke();
  }

  ctx.globalAlpha = 1.0;
}

function drawNodes() {
  ctx.fillStyle = "#111";
  // Very small dots so it doesn't overwhelm the edges.
  for (const n of nodes) {
    const p = lonLatToCanvasXY(n.lon, n.lat);
    ctx.fillRect(p.x, p.y, 1, 1);
  }
}

function drawMarker(pt, color) {
  if (!pt) return;
  const p = lonLatToCanvasXY(pt.lon, pt.lat);
  ctx.fillStyle = color;
  ctx.beginPath();
  ctx.arc(p.x, p.y, 5, 0, Math.PI * 2);
  ctx.fill();
}

function drawRoute() {
  if (!routePath || routePath.length < 2) return;

  ctx.lineWidth = 3;
  ctx.strokeStyle = "orange";
  ctx.beginPath();
  const first = normalizeLonLat(routePath[0]);
  if (!first) return;
  let p = lonLatToCanvasXY(first.lon, first.lat);
  ctx.moveTo(p.x, p.y);

  for (let i = 1; i < routePath.length; i++) {
    const pt = normalizeLonLat(routePath[i]);
    if (!pt) continue;
    p = lonLatToCanvasXY(pt.lon, pt.lat);
    ctx.lineTo(p.x, p.y);
  }
  ctx.stroke();
}

function redraw() {
  if (!bbox) return;
  clear();
  drawEdges();
  drawNodes();
  drawRoute();
  drawMarker(clickA, "red");
  drawMarker(clickB, "blue");
}

function reset() {
  clickA = null;
  clickB = null;
  routePath = null;
  setOut("Distancia: (haz click)");
  redraw();
}

async function requestRoute() {
  if (!clickA || !clickB) return;

  setOut("Distancia: calculando...");

  const payload = {
    A: clickA,
    B: clickB,
    maxSnapMeters: 300,
  };

  const res = await fetch("/api/route", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  const data = await res.json();
  if (!res.ok) {
    setOut("Error: " + (data.error || "unknown"));
    return;
  }

  routePath = data.path; // [{lon,lat}, ...]
  setOut(
    `Distancia: ${data.distance_m.toFixed(2)} m (${(data.distance_m / 1000).toFixed(
      3
    )} km) | snapA=${data.snapA_m.toFixed(1)}m snapB=${data.snapB_m.toFixed(1)}m`
  );
  redraw();
}

async function loadEdges() {
  const res = await fetch("/api/graph/edges");
  const geo = await res.json();

  // Convert GeoJSON features -> edge list.
  edges = [];
  const nodeMap = new Map();
  for (const f of geo.features) {
    const coords = f.geometry.coordinates; // [[lon,lat],[lon,lat]]
    if (!coords || coords.length !== 2) continue;
    const a = coords[0];
    const b = coords[1];
    const aPt = { lon: a[0], lat: a[1] };
    const bPt = { lon: b[0], lat: b[1] };
    edges.push({ a: aPt, b: bPt });

    // Deduplicate nodes by rounding to reduce float-key noise.
    const keyA = `${aPt.lon.toFixed(7)},${aPt.lat.toFixed(7)}`;
    const keyB = `${bPt.lon.toFixed(7)},${bPt.lat.toFixed(7)}`;
    nodeMap.set(keyA, aPt);
    nodeMap.set(keyB, bPt);
  }

  computeBBoxFromEdges();
  nodes = Array.from(nodeMap.values());
}

function installClickHandler() {
  canvas.addEventListener("click", async (e) => {
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    const pt = canvasXYToLonLat(x, y);

    if (!clickA) {
      clickA = { lat: pt.lat, lon: pt.lon };
      setOut("Elegiste A. Ahora haz click en B.");
      redraw();
      return;
    }
    if (!clickB) {
      clickB = { lat: pt.lat, lon: pt.lon };
      setOut("Elegiste B. Calculando distancia...");
      await requestRoute();
      return;
    }

    // If already have A and B, next click becomes new A (and clears previous route).
    clickA = { lat: pt.lat, lon: pt.lon };
    clickB = null;
    routePath = null;
    setOut("Elegiste nueva A. Ahora haz click en B.");
    redraw();
  });
}

async function init() {
  canvas = document.getElementById("c");
  ctx = canvas.getContext("2d");

  await loadEdges();

  resizeCanvas();
  redraw();

  window.addEventListener("resize", () => {
    resizeCanvas();
    redraw();
  });

  installClickHandler();
  document.getElementById("reset").addEventListener("click", reset);
}

init().catch((e) => {
  console.error(e);
  setOut("Error cargando el grafo: " + e);
});

