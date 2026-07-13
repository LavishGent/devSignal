package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"devsignal/internal/dashboard"
	"devsignal/internal/recommend"
	"devsignal/internal/reports"
	"devsignal/internal/search"
	"devsignal/internal/snapshot"
)

// InsightsHandler serves the cross-module endpoints: dashboard summary,
// recommendations, reports (JSON and CSV), and global search.
type InsightsHandler struct {
	BaseHandler
	loader   snapshot.Loader
	engine   *recommend.Engine
	searcher *search.Service
}

// NewInsightsHandler wires the insight endpoints.
func NewInsightsHandler(base BaseHandler, loader snapshot.Loader, engine *recommend.Engine, searcher *search.Service) *InsightsHandler {
	return &InsightsHandler{BaseHandler: base, loader: loader, engine: engine, searcher: searcher}
}

// Register implements RouteRegistrar.
func (h *InsightsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/dashboard", protected(http.HandlerFunc(h.dashboard)))
	mux.Handle("GET /api/v1/recommendations", protected(http.HandlerFunc(h.recommendations)))
	mux.Handle("GET /api/v1/reports", protected(http.HandlerFunc(h.reportList)))
	mux.Handle("GET /api/v1/reports/{id}", protected(http.HandlerFunc(h.report)))
	mux.Handle("GET /api/v1/search", protected(http.HandlerFunc(h.search)))
}

func (h *InsightsHandler) dashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	snap, err := h.loader.Load(r.Context(), userID)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, dashboard.Build(snap, h.Now()))
}

func (h *InsightsHandler) recommendations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	snap, err := h.loader.Load(r.Context(), userID)
	if err != nil {
		h.repoError(w, err)
		return
	}
	recs := h.engine.Run(snap, h.Now())
	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && limit > 0 && limit < len(recs) {
		recs = recs[:limit]
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": recs})
}

func (h *InsightsHandler) reportList(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.userID(w, r); !ok {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": reports.Catalog()})
}

func (h *InsightsHandler) report(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	build, found := reports.ByID(r.PathValue("id"))
	if !found {
		respondError(w, http.StatusNotFound, "report not found")
		return
	}
	snap, err := h.loader.Load(r.Context(), userID)
	if err != nil {
		h.repoError(w, err)
		return
	}
	rep := build(snap, h.Now())

	if r.URL.Query().Get("format") == "csv" {
		data, err := rep.CSV()
		if err != nil {
			h.repoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+rep.ID+`.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	respondJSON(w, http.StatusOK, rep)
}

func (h *InsightsHandler) search(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		respondError(w, http.StatusBadRequest, "query parameter q is required")
		return
	}
	groups, err := h.searcher.Search(r.Context(), userID, q)
	if err != nil {
		h.repoError(w, err)
		return
	}
	if groups == nil {
		groups = []search.Group{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"query": q, "groups": groups})
}
