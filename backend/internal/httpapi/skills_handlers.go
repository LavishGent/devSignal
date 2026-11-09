package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
)

// SkillsHandler serves /api/v1/skills. It embeds BaseHandler for the shared
// plumbing and depends only on the repo interface, so it runs identically
// over postgres in production and memory in tests.
type SkillsHandler struct {
	BaseHandler
	repo repo.SkillRepo
}

// NewSkillsHandler wires the module.
func NewSkillsHandler(base BaseHandler, r repo.SkillRepo) *SkillsHandler {
	return &SkillsHandler{BaseHandler: base, repo: r}
}

// Register implements RouteRegistrar.
func (h *SkillsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/skills", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/skills", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/skills/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/skills/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/skills/{id}", protected(http.HandlerFunc(h.delete)))
}

func (h *SkillsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := repo.SkillFilter{
		Q:        q.Get("q"),
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

func (h *SkillsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var s domain.Skill
	if err := decodeJSON(w, r, &s); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, r, &s) {
		return
	}
	s.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &s); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, s)
}

func (h *SkillsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	s, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, s)
}

func (h *SkillsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.Skill
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

func (h *SkillsHandler) delete(w http.ResponseWriter, r *http.Request) {
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
