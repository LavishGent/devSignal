package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
)

// GoalsHandler serves /api/v1/goals. It embeds BaseHandler for the shared
// plumbing and depends only on the repo interface, so it runs identically
// over postgres in production and memory in tests.
type GoalsHandler struct {
	BaseHandler
	repo repo.GoalRepo
}

// NewGoalsHandler wires the module.
func NewGoalsHandler(base BaseHandler, r repo.GoalRepo) *GoalsHandler {
	return &GoalsHandler{BaseHandler: base, repo: r}
}

// Register implements RouteRegistrar.
func (h *GoalsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/goals", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/goals", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/goals/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/goals/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/goals/{id}", protected(http.HandlerFunc(h.delete)))
}

func (h *GoalsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := repo.GoalFilter{
		Q:        q.Get("q"),
		Status:   q.Get("status"),
		Category: q.Get("category"),
		Page:     h.page(r),
	}
	items, err := h.repo.List(r.Context(), userID, f)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *GoalsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var g domain.Goal
	if err := decodeJSON(w, r, &g); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, r, &g) {
		return
	}
	g.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &g); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, g)
}

func (h *GoalsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	g, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, g)
}

func (h *GoalsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.Goal
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

func (h *GoalsHandler) delete(w http.ResponseWriter, r *http.Request) {
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
