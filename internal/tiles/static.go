package tiles

import "net/http"

// Serve SPA & static assets from ./web/dist (prod) or ./web/public (dev)
func SPA() http.Handler {
	return http.FileServer(http.Dir("web/public"))
}
