package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// StudySessionsHandler serves /api/v1/study-sessions. It embeds BaseHandler
// for the shared plumbing and depends only on the repo interfaces, so it runs
// identically over postgres in production and memory in tests. It also holds
// the skills repo to verify that linked skills belong to the user.
type StudySessionsHandler struct {
	BaseHandler
	repo   repo.StudySessionRepo
	skills repo.SkillRepo
}

// NewStudySessionsHandler wires the module.
func NewStudySessionsHandler(base BaseHandler, sessions repo.StudySessionRepo, skills repo.SkillRepo) *StudySessionsHandler {
	return &StudySessionsHandler{BaseHandler: base, repo: sessions, skills: skills}
}

// Register implements RouteRegistrar.
func (h *StudySessionsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/study-sessions", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/study-sessions", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/study-sessions/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/study-sessions/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/study-sessions/{id}", protected(http.HandlerFunc(h.delete)))
}

// checkSkillLink verifies an optional skill reference exists and belongs to
// the user; linking to another user's skill is a validation failure, not a
// 404, so the caller learns which field is wrong. A nil skillID is fine — the
// link is optional.
func (h *StudySessionsHandler) checkSkillLink(w http.ResponseWriter, r *http.Request, userID uuid.UUID, skillID *uuid.UUID) bool {
	if skillID == nil {
		return true
	}
	if _, err := h.skills.Get(r.Context(), userID, *skillID); err != nil {
		errs := validation.New()
		errs.Add("skillId", "must reference one of your skills")
		respondValidation(w, errs)
		return false
	}
	return true
}

func (h *StudySessionsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	var skillID *uuid.UUID
	if id, err := uuid.Parse(q.Get("skillId")); err == nil { // absent or invalid means all
		skillID = &id
	}
	f := repo.StudySessionFilter{
		Q:       q.Get("q"),
		SkillID: skillID,
		Page:    h.page(r),
	}
	items, err := h.repo.List(r.Context(), userID, f)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *StudySessionsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var s domain.StudySession
	if err := decodeJSON(w, r, &s); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, &s) {
		return
	}
	if !h.checkSkillLink(w, r, userID, s.SkillID) {
		return
	}
	s.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &s); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, s)
}

func (h *StudySessionsHandler) get(w http.ResponseWriter, r *http.Request) {
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

func (h *StudySessionsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.StudySession
	if err := decodeJSON(w, r, &in); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, &in) {
		return
	}
	if !h.checkSkillLink(w, r, userID, in.SkillID) {
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

func (h *StudySessionsHandler) delete(w http.ResponseWriter, r *http.Request) {
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
