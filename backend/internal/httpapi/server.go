package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Server assembles middleware, handlers, and routes into one http.Handler.
type Server struct {
	logger     *slog.Logger
	production bool
	staticDir  string
	authH      *AuthHandler
	limiter    *RateLimiter
	modules    []RouteRegistrar
}

// RouteRegistrar is implemented by every module handler; the server treats
// all modules polymorphically through this interface.
type RouteRegistrar interface {
	Register(mux *http.ServeMux, protected Middleware)
}

// NewServer builds the API server shell. Module handlers are attached with
// AddModule before Routes is called.
func NewServer(logger *slog.Logger, production bool, staticDir string, authH *AuthHandler) *Server {
	return &Server{
		logger:     logger,
		production: production,
		staticDir:  staticDir,
		authH:      authH,
		limiter:    NewRateLimiter(10, time.Minute),
	}
}

// AddModule attaches a module's routes to the server.
func (s *Server) AddModule(m RouteRegistrar) { s.modules = append(s.modules, m) }

// Routes builds the full handler tree.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	protected := s.authH.RequireAuth()
	limited := s.limiter.Middleware()

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.authH.Register(mux, limited, protected)
	for _, m := range s.modules {
		m.Register(mux, protected)
	}

	// Any unknown /api path is a JSON 404, not an HTML page.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusNotFound, "endpoint not found")
	})

	if s.staticDir != "" {
		mux.Handle("/", spaHandler(s.staticDir))
	}

	return Chain(mux,
		Recover(s.logger),
		SecurityHeaders(s.production),
		RequestLogger(s.logger),
	)
}

// spaHandler serves the built frontend, falling back to index.html for
// client-side routes (e.g., /applications) without leaking path traversal.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "..") {
			clean = "index.html"
		}
		if _, err := os.Stat(filepath.Join(dir, clean)); err != nil {
			r.URL.Path = "/" // unknown path → SPA entry point
		}
		fs.ServeHTTP(w, r)
	})
}
