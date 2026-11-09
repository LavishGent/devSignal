package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// InterviewsHandler serves /api/v1/interviews. It embeds BaseHandler for the
// shared plumbing and depends only on the repo interfaces, so it runs
// identically over postgres in production and memory in tests. It also holds
// the applications repo to verify that linked applications belong to the user.
type InterviewsHandler struct {
	BaseHandler
	repo         repo.InterviewRepo
	applications repo.ApplicationRepo
}

// NewInterviewsHandler wires the module.
func NewInterviewsHandler(base BaseHandler, interviews repo.InterviewRepo, applications repo.ApplicationRepo) *InterviewsHandler {
	return &InterviewsHandler{BaseHandler: base, repo: interviews, applications: applications}
}

// Register implements RouteRegistrar.
func (h *InterviewsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/interviews", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/interviews", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/interviews/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/interviews/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/interviews/{id}", protected(http.HandlerFunc(h.delete)))
}

// checkApplicationLink verifies the referenced application exists and belongs
// to the user; linking to another user's application is a validation failure,
// not a 404, so the caller learns which field is wrong.
func (h *InterviewsHandler) checkApplicationLink(w http.ResponseWriter, r *http.Request, userID, applicationID uuid.UUID) bool {
	if _, err := h.applications.Get(r.Context(), userID, applicationID); err != nil {
		errs := validation.New()
		errs.Add("applicationId", "must reference one of your applications")
		respondValidation(w, errs)
		return false
	}
	return true
}

func (h *InterviewsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	appID, _ := uuid.Parse(q.Get("applicationId")) // absent or invalid means all
	f := repo.InterviewFilter{
		Q:             q.Get("q"),
		Stage:         q.Get("stage"),
		Outcome:       q.Get("outcome"),
		ApplicationID: appID,
		Page:          h.page(r),
	}
	items, err := h.repo.List(r.Context(), userID, f)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *InterviewsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var i domain.Interview
	if err := decodeJSON(w, r, &i); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, r, &i) {
		return
	}
	if !h.checkApplicationLink(w, r, userID, i.ApplicationID) {
		return
	}
	i.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &i); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, i)
}

func (h *InterviewsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	i, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, i)
}

func (h *InterviewsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.Interview
	if err := decodeJSON(w, r, &in); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, r, &in) {
		return
	}
	if !h.checkApplicationLink(w, r, userID, in.ApplicationID) {
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

func (h *InterviewsHandler) delete(w http.ResponseWriter, r *http.Request) {
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
