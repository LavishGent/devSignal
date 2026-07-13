package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
)

// ProjectsHandler serves /api/v1/projects. It embeds BaseHandler for the
// shared plumbing and depends only on the repo interface, so it runs
// identically over postgres in production and memory in tests.
type ProjectsHandler struct {
	BaseHandler
	repo repo.ProjectRepo
}

// NewProjectsHandler wires the module.
func NewProjectsHandler(base BaseHandler, r repo.ProjectRepo) *ProjectsHandler {
	return &ProjectsHandler{BaseHandler: base, repo: r}
}

// Register implements RouteRegistrar.
func (h *ProjectsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/projects", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/projects", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/projects/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/projects/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/projects/{id}", protected(http.HandlerFunc(h.delete)))
}

func (h *ProjectsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := repo.ProjectFilter{
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

func (h *ProjectsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var p domain.Project
	if err := decodeJSON(w, r, &p); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, &p) {
		return
	}
	p.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &p); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, p)
}

func (h *ProjectsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	p, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, p)
}

func (h *ProjectsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.Project
	if err := decodeJSON(w, r, &in); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, &in) {
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

func (h *ProjectsHandler) delete(w http.ResponseWriter, r *http.Request) {
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
