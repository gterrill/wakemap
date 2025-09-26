# WakeMap

<img src="wakemap-logo.svg" alt="WakeMap" width="480">

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](./LICENSE)
[![Status: alpha](https://img.shields.io/badge/status-alpha-orange.svg)](#status)
[![Made for boats](https://img.shields.io/badge/made_for-boats-1f75fe)](#)

Self-hosted, offline-first **moving map for boats**. Renders OpenFreeMap + OpenSeaMap + high‑res satellite imagery, shows live vessel position and nearby POIs, logs passages, and exports GPX. Optional Immich integration for geo‑tagged photo overlays.

## Features
- **Offline-first charts**: cached OpenFreeMap, OpenSeaMap, and hi‑res satellite imagery
- **Live vessel position**: NMEA 2000/0183 → position display with heading/COG/SOG
- **POIs**: nearby points of interest, filters, and detail panes
- **Passage logging**: automatic track recording, **GPX export**
- **Immich bridge (optional)**: overlay recent photos near current position
- **Self-hosted**: runs on-board; no internet required once tiles are cached

## Quickstart
```bash
# 1) Clone
git clone https://github.com/<you>/wakemap
cd wakemap

# 2) Start stack (example: compose)
docker compose up -d

# 3) Open
http://localhost:8080  # replace with your host or boat LAN address
```

> **Note**: See `./docs/config.md` for NMEA and tile-cache configuration (coming soon).

## Architecture (high-level)
- **core/** – server, tile-cache, POI/index logic, GPX exporter
- **ui/** – web UI (map, layers, POIs, track controls)
- **bridges/** – optional integrations (e.g., `immich-bridge`)
- **assets/** – logos/icons
- **docs/** – configuration and how-tos

## Status
Alpha. Expect rapid changes. PRs and issues welcome.

## License
Apache-2.0. See [LICENSE](./LICENSE). Attribution required via [NOTICE](./NOTICE).

---

© 2025 Gavin Terrill