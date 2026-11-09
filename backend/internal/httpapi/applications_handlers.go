package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
)

// ApplicationsHandler serves /api/v1/applications. It embeds BaseHandler for
// the shared plumbing and depends only on the repo interface, so it runs
// identically over postgres in production and memory in tests.
type ApplicationsHandler struct {
	BaseHandler
	repo repo.ApplicationRepo
}

// NewApplicationsHandler wires the module.
func NewApplicationsHandler(base BaseHandler, r repo.ApplicationRepo) *ApplicationsHandler {
	return &ApplicationsHandler{BaseHandler: base, repo: r}
}

// Register implements RouteRegistrar.
func (h *ApplicationsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/applications", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/applications", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/applications/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/applications/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/applications/{id}", protected(http.HandlerFunc(h.delete)))
}

func (h *ApplicationsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := repo.ApplicationFilter{
		Q:      q.Get("q"),
		Status: q.Get("status"),
		Page:   h.page(r),
	}
	items, err := h.repo.List(r.Context(), userID, f)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ApplicationsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var a domain.Application
	if err := decodeJSON(w, r, &a); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, r, &a) {
		return
	}
	a.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &a); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, a)
}

func (h *ApplicationsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	a, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, a)
}

func (h *ApplicationsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.Application
	if err := decodeJSON(w, r, &in); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, r, &in) {
		return
	}
	existing, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	in.BaseRecord = existing.BaseRecord // identity and audit fields are immutable
	in.Touch(h.Now())
	if err := h.repo.Update(r.Context(), &in); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, in)
}

func (h *ApplicationsHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), userID, id); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
