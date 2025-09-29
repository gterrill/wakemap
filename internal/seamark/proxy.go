package seamark

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// Handler proxies OpenSeaMap tiles with caching, while avoiding double CORS headers.
// CORS is handled by the outer middleware; we DO NOT forward upstream Access-Control-* headers.
func Handler() http.Handler {
	client := &http.Client{Timeout: 10 * time.Second}

	hopByHop := map[string]struct{}{
		"Connection":          {},
		"Keep-Alive":          {},
		"Proxy-Authenticate":  {},
		"Proxy-Authorization": {},
		"Te":                  {},
		"Trailer":             {},
		"Transfer-Encoding":   {},
		"Upgrade":             {},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream := "https://tiles.openseamap.org/seamark" + r.URL.Path // /{z}/{x}/{y}.png

		req, _ := http.NewRequest(http.MethodGet, upstream, nil)
		req.Header.Set("User-Agent", "wakemap-dev/1.0")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy safe headers only; skip CORS + hop-by-hop
		for k, vv := range resp.Header {
			if _, skip := hopByHop[k]; skip {
				continue
			}
			if strings.HasPrefix(http.CanonicalHeaderKey(k), "Access-Control-") {
				continue
			}
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Cache-Control", "public, max-age=86400") // keep upstream ETag/Last-Modified; add:

		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})
}
