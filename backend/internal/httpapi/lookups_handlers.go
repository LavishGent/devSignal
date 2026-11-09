package httpapi

import "net/http"

// LookupsHandler serves GET /api/v1/options: every fixed-choice list the UI
// needs to populate its dropdowns and filters. The lists come from the
// lookup_values table, so an operator extends them with an INSERT.
type LookupsHandler struct {
	BaseHandler
}

// NewLookupsHandler wires the module.
func NewLookupsHandler(base BaseHandler) *LookupsHandler {
	return &LookupsHandler{BaseHandler: base}
}

// Register implements RouteRegistrar.
func (h *LookupsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/options", protected(http.HandlerFunc(h.options)))
}

func (h *LookupsHandler) options(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.userID(w, r); !ok {
		return
	}
	lookups, err := h.Lookups.All(r.Context())
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, lookups)
}
