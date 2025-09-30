import 'maplibre-gl/dist/maplibre-gl.css';
import maplibregl, { type LngLatBoundsLike } from 'maplibre-gl';

async function safeFetchJSON(url: string) {
  const r = await fetch(url, { credentials: 'omit' });
  if (!r.ok) throw new Error(`${url} -> ${r.status}`);
  return r.json();
}

type SourceInit = Parameters<maplibregl.Map['addSource']>[1];

let map: maplibregl.Map;
let startMarker: maplibregl.Marker | null = null;
let finishMarker: maplibregl.Marker | null = null;
const DRAWER_BREAKPOINT = 960;
let drawerMediaQuery: MediaQueryList | null = null;
let drawerModeInitialized = false;
let seamarkReady: Promise<boolean> | null = null;

function isDrawerOpen() {
  return document.body.classList.contains('drawer-open');
}

function isDrawerPermanent() {
  return document.body.classList.contains('drawer-permanent');
}

function setDrawerOpen(open: boolean) {
  document.body.classList.toggle('drawer-open', open);
  const toggle = document.getElementById('drawerToggle');
  if (toggle) toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
}

function setupDrawer() {
  const toggle = document.getElementById('drawerToggle');
  const close = document.getElementById('drawerClose');
  const overlay = document.getElementById('drawerOverlay');
  toggle?.addEventListener('click', () => setDrawerOpen(!isDrawerOpen()));
  close?.addEventListener('click', () => setDrawerOpen(false));
  overlay?.addEventListener('click', () => setDrawerOpen(false));
}

function applyDrawerMode(permanent: boolean) {
  document.body.classList.toggle('drawer-permanent', permanent);
  if (!drawerModeInitialized) {
    setDrawerOpen(permanent);
    drawerModeInitialized = true;
    return;
  }
  if (!permanent && isDrawerOpen()) setDrawerOpen(false);
}

function initResponsiveDrawer() {
  drawerMediaQuery = window.matchMedia(`(min-width: ${DRAWER_BREAKPOINT}px)`);
  const apply = (mq: MediaQueryList | MediaQueryListEvent) => applyDrawerMode(mq.matches);
  apply(drawerMediaQuery);
  if (typeof drawerMediaQuery.addEventListener === 'function') {
    drawerMediaQuery.addEventListener('change', apply);
  } else if (typeof drawerMediaQuery.addListener === 'function') {
    drawerMediaQuery.addListener(apply);
  }
}

function fmtCoord(lon: number, lat: number) {
  return `${lat.toFixed(5)}°, ${lon.toFixed(5)}°`;
}

function upsertMarker(
  markerRef: maplibregl.Marker | null,
  lon: number,
  lat: number,
  color: string,
  titleHTML: string
): maplibregl.Marker {
  if (!markerRef) {
    // default pin with color; add a small offset so popup doesn’t cover the pin
    const m = new maplibregl.Marker({ color })
      .setLngLat([lon, lat])
      .setPopup(new maplibregl.Popup({ offset: 12 }).setHTML(titleHTML))
      .addTo(map);
    return m;
  }
  markerRef.setLngLat([lon, lat]);
  // replace popup content each time in case titleHTML changed
  const p = markerRef.getPopup() ?? new maplibregl.Popup({ offset: 12 });
  p.setHTML(titleHTML);
  markerRef.setPopup(p);
  return markerRef;
}

function removeTrackMarkers() {
  if (startMarker) { startMarker.remove(); startMarker = null; }
  if (finishMarker) { finishMarker.remove(); finishMarker = null; }
}

function mapReady(): Promise<void> {
  return new Promise((resolve) => {
    if (map && map.isStyleLoaded && map.isStyleLoaded()) return resolve();
    map.once('load', () => resolve());
  });
}

async function pickStyle(): Promise<string> {
  // Prefer local copy to avoid DNS/404 issues; fallback to tiles host if available.
  const local = '/styles/openfreemap.json';
  try {
    const r = await fetch(local, { cache: 'no-store' });
    if (r.ok) return local;
  } catch { }
  // If you have a working tiles host exposing style.json, use it here:
  return 'http://tiles.local:8081/styles/openfreemap/style.json';
}

function ensureTrackLayer() {
  if (!map.getSource('track')) {
    const init: SourceInit = {
      type: 'geojson',
      data: { type: 'FeatureCollection', features: [] },
    };
    map.addSource('track', init);
  }
  if (!map.getLayer('track-line')) {
    map.addLayer({
      id: 'track-line',
      type: 'line',
      source: 'track',
        layout: {
        'line-cap': "round",
        'line-join': "round"
      },
      paint: { 
        'line-width': 8, 
        'line-color': '#6084eb',
        // 'circle-radius': 4,
        // 'circle-color': "#fff",
        // 'circle-stroke-color': "#aaa",
        // 'circle-stroke-width': 1,
      }
    });

    const popup = new maplibregl.Popup({ closeButton: false, closeOnClick: false });
    map.on('mouseenter', 'track-points', () => map.getCanvas().style.cursor = 'pointer');
    map.on('mouseleave', 'track-points', () => { map.getCanvas().style.cursor = ''; popup.remove(); });
    // map.on('mousemove', 'track-points', (e) => {
    //   const f = e.features?.[0]; if (!f) return;
    //   popup.setLngLat((f.geometry as any).coordinates).setHTML(`Point #${f.properties?.idx}`).addTo(map);
    // });
    map.on('mousemove', 'track-points', (e) => {
      const f = e.features?.[0];
      if (!f) return;
      const [lon, lat] = (f.geometry as any).coordinates as [number, number];
      const sogKn = (typeof f.properties?.sog_kn === 'number') ? f.properties.sog_kn as number : null;

      const html = `
        <div style="font:12px/1.3 system-ui">
          <div><strong>GPS</strong> ${lat.toFixed(5)}°, ${lon.toFixed(5)}°</div>
          <div><strong>SOG</strong> ${sogKn !== null ? sogKn.toFixed(1) + ' kn' : '—'}</div>
        </div>
      `;

      popup
        .setLngLat([lon, lat])
        .setHTML(html)
        .addTo(map);
    });

  }
}

type BBox = [number, number, number, number];

function ensureTrackPointsLayer(map: maplibregl.Map) {
  if (!map.getSource('track-points')) {
    map.addSource('track-points', {
      type: 'geojson',
      data: { type: 'FeatureCollection', features: [] }
    } as any);

    map.addLayer({
      id: 'track-points',
      type: 'circle',
      source: 'track-points',
      minzoom: 8, // only show once we’re reasonably zoomed in
      paint: {
        'circle-radius': [
          'interpolate', ['linear'], ['zoom'],
          8, 2,
          12, 4,
          16, 6
        ],
        'circle-color': '#1769aa',
        'circle-stroke-width': 1,
        'circle-stroke-color': '#ffffff',
        'circle-opacity': 0.9
      },
      layout: { visibility: 'visible' }
    } as any);
  }
}

const MS_TO_KN = 1.943844492;

function coordsToPointsFC(gj: any): GeoJSON.FeatureCollection {
  let coords: any[] = [];
  if (gj?.type === 'Feature' && gj?.geometry?.type === 'LineString') {
    coords = gj.geometry.coordinates;
  } else if (gj?.type === 'FeatureCollection') {
    const f = gj.features?.find((x: any) => x?.geometry?.type === 'LineString');
    coords = f?.geometry?.coordinates ?? [];
  }

  const features = coords.map((c: any, i: number) => {
    const lon = c[0], lat = c[1];
    const sogKn = (typeof c[2] === 'number' && c[2] > 0) ? c[2] * MS_TO_KN : null;
    return {
      type: 'Feature',
      properties: { idx: i, sog_kn: sogKn },
      geometry: { type: 'Point', coordinates: [lon, lat] }
    };
  });

  // const features = coords.map(([lon, lat], i) => ({
  //   type: 'Feature',
  //   properties: { idx: i }, // if you later want to label/index
  //   geometry: { type: 'Point', coordinates: [lon, lat] }
  // }));

  return { type: 'FeatureCollection', features };
}

// function setPointsVisibility(map: maplibregl.Map, visible: boolean) {
//   if (!map.getLayer('track-points')) return;
//   map.setLayoutProperty('track-points', 'visibility', visible ? 'visible' : 'none');
// }

function haversineM(a: [number, number], b: [number, number]): number {
  const R = 6371000; // meters
  const toRad = (x: number) => x * Math.PI / 180;
  const dLat = toRad(b[1] - a[1]);
  const dLon = toRad(b[0] - a[0]);
  const lat1 = toRad(a[1]);
  const lat2 = toRad(b[1]);
  const sin1 = Math.sin(dLat / 2), sin2 = Math.sin(dLon / 2);
  const h = sin1 * sin1 + Math.cos(lat1) * Math.cos(lat2) * sin2 * sin2;
  return 2 * R * Math.asin(Math.min(1, Math.sqrt(h)));
}

function formatDuration(sec: number): string {
  if (!isFinite(sec) || sec <= 0) return "—";
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = Math.floor(sec % 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

type TrackStats = { distanceNm: number; durationSec: number; avgKnots: number; name?: string };

function computeTrackStats(gj: any): TrackStats {
  // find first LineString coords
  let coords: [number, number][] | undefined;
  let name: string | undefined;

  if (gj?.type === 'FeatureCollection' && Array.isArray(gj.features)) {
    const f = gj.features.find((x: any) => x?.geometry?.type === 'LineString');
    if (f) {
      coords = f.geometry.coordinates as [number, number][];
      name = f.properties?.name ?? gj.properties?.name;
    }
  } else if (gj?.type === 'Feature' && gj.geometry?.type === 'LineString') {
    coords = gj.geometry.coordinates as [number, number][];
    name = gj.properties?.name;
  }

  let meters = 0;
  if (Array.isArray(coords) && coords.length > 1) {
    for (let i = 1; i < coords.length; i++) {
      meters += haversineM(coords[i - 1], coords[i]);
    }
  }

  // duration preference: top-level properties started_at/ended_at (unix sec)
  const started = gj?.properties?.started_at ?? gj?.features?.[0]?.properties?.started_at;
  const ended   = gj?.properties?.ended_at   ?? gj?.features?.[0]?.properties?.ended_at;

  let durationSec = 0;
  if (Number.isFinite(started) && Number.isFinite(ended)) {
    durationSec = Math.max(0, Number(ended) - Number(started));
  } else {
    // fallback: times array on the feature if present (ISO or epoch seconds)
    const f = gj?.features?.[0];
    const times: any[] = f?.properties?.times;
    if (Array.isArray(times) && times.length >= 2) {
      const t0 = +times[0], t1 = +times[times.length - 1];
      if (Number.isFinite(t0) && Number.isFinite(t1)) durationSec = Math.max(0, t1 - t0);
    }
  }

  const distanceNm = meters / 1852;
  const avgKnots = durationSec > 0 ? (meters / durationSec) * 1.943844492 : 0;

  return { distanceNm, durationSec, avgKnots, name };
}

function updateStatsUI(stats: TrackStats) {
  const byId = (id: string) => document.getElementById(id)!;
  document.getElementById('stats')?.classList.remove('hidden');
  byId('stat-name').textContent = stats.name ? stats.name : 'Track';
  byId('stat-distance').textContent = `Dist: ${stats.distanceNm.toFixed(1)} nm`;
  byId('stat-duration').textContent = `Time: ${formatDuration(stats.durationSec)}`;
  byId('stat-speed').textContent = `Avg: ${stats.avgKnots.toFixed(1)} kn`;
}

function updateTrackMarkersFromGeoJSON(gj: any) {
  // Accept Feature(LineString) or FC containing one LineString
  let coords: any[] = [];
  if (gj?.type === 'Feature' && gj?.geometry?.type === 'LineString') {
    coords = gj.geometry.coordinates;
  } else if (gj?.type === 'FeatureCollection') {
    const f = gj.features?.find((x: any) => x?.geometry?.type === 'LineString');
    coords = f?.geometry?.coordinates ?? [];
  }

  if (!Array.isArray(coords) || coords.length < 2) {
    removeTrackMarkers();
    return;
  }

  const start = coords[0];                       // [lon, lat, (optional sog_ms)]
  const finish = coords[coords.length - 1];

  const name = gj?.properties?.name ?? 'Track';
  const startedAt = gj?.properties?.started_at;  // unix seconds (if present)
  const endedAt = gj?.properties?.ended_at;

  const startHTML = `
    <div style="font:12px/1.3 system-ui">
      <div><strong>Start</strong> — ${name}</div>
      <div>${fmtCoord(start[0], start[1])}</div>
      ${startedAt ? `<div>${new Date(startedAt * 1000).toLocaleString()}</div>` : ''}
    </div>
  `;
  const finishHTML = `
    <div style="font:12px/1.3 system-ui">
      <div><strong>Finish</strong> — ${name}</div>
      <div>${fmtCoord(finish[0], finish[1])}</div>
      ${endedAt ? `<div>${new Date(endedAt * 1000).toLocaleString()}</div>` : ''}
    </div>
  `;

  startMarker  = upsertMarker(startMarker,  start[0],  start[1],  '#12b886', startHTML); // green
  finishMarker = upsertMarker(finishMarker, finish[0], finish[1], '#e03131', finishHTML); // red
}

export async function showTrack(id: string) {
  if (!id) return;

  await mapReady();                 // ensure the style is ready
  ensureTrackLayer(map);
  ensureTrackPointsLayer(map);    // NEW: points source + layer

  const gj = await safeFetchJSON(`/api/tracks/${id}.geojson`);
  (map.getSource('track') as maplibregl.GeoJSONSource).setData(gj);
  (map.getSource('track-points') as any).setData(coordsToPointsFC(gj));

  updateTrackMarkersFromGeoJSON(gj);

  // preserve your bbox-centering
  let bbox: BBox | null = null;
  if (Array.isArray(gj.bbox) && gj.bbox.length === 4) {
    bbox = [gj.bbox[0], gj.bbox[1], gj.bbox[2], gj.bbox[3]];
  }
  if (bbox) {
    map.fitBounds([[bbox[0], bbox[1]], [bbox[2], bbox[3]]], { padding: 48, linear: true });
  }

  // compute + show stats
  updateStatsUI(computeTrackStats(gj));
}

async function populateTrackSelect() {
  const sel = document.getElementById('trackSelect') as HTMLSelectElement;
  const list = await safeFetchJSON('/api/tracks?limit=50');
  const tracks = Array.isArray(list?.tracks) ? list.tracks : [];
  // wipe existing (keep placeholder)
  sel.length = 1;
  for (const t of tracks) {
    const opt = document.createElement('option');
    opt.value = t.id;
    const started = t.started_at ? new Date(t.started_at).toLocaleString() : '';
    const distNm = t.distance_m ? (t.distance_m / 1852).toFixed(1) + ' nm' : '';
    opt.textContent = `${t.name || t.id} — ${started} ${distNm}`;
    sel.appendChild(opt);
  }
  sel.onchange = () => {
    showTrack(sel.value);
    if (sel.value && !isDrawerPermanent()) setDrawerOpen(false);
  };
}

(async () => {
  setupDrawer();
  initResponsiveDrawer();
  const style = await pickStyle();
  map = new maplibregl.Map({
    container: 'map',
    style,
    center: [151.2, -33.86],
    zoom: 10,
  });
  
  map.addControl(new maplibregl.NavigationControl({
    showCompass: true,       // zoom only
    visualizePitch: true,
  }), 'top-right'); 

  // Add seamarks raster overlay through our Go proxy (CORS + cache)
  map.on('load', async () => {
    await populateTrackSelect();
    if (await checkSeamarkAvailable()) {
      map.addSource('seamarks', {
        type: 'raster',
        tiles: ['/seamark/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '© OpenSeaMap contributors',
      });
      map.addLayer({
        id: 'seamarks',
        type: 'raster',
        source: 'seamarks',
      });
    } else {
      console.info('Seamark tiles unavailable; skipping overlay.');
    }
  });
})();
async function checkSeamarkAvailable(): Promise<boolean> {
  if (!seamarkReady) {
    seamarkReady = (async () => {
      try {
        const resp = await fetch('/seamark/0/0/0.png', { method: 'HEAD' });
        return resp.ok;
      } catch (err) {
        console.debug('Seamark availability check failed:', err);
        return false;
      }
    })();
  }
  return seamarkReady;
}
