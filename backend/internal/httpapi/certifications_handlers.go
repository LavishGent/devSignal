package httpapi

import (
	"net/http"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
)

// CertificationsHandler serves /api/v1/certifications. It embeds BaseHandler
// for the shared plumbing and depends only on the repo interface, so it runs
// identically over postgres in production and memory in tests.
type CertificationsHandler struct {
	BaseHandler
	repo repo.CertificationRepo
}

// NewCertificationsHandler wires the module.
func NewCertificationsHandler(base BaseHandler, r repo.CertificationRepo) *CertificationsHandler {
	return &CertificationsHandler{BaseHandler: base, repo: r}
}

// Register implements RouteRegistrar.
func (h *CertificationsHandler) Register(mux *http.ServeMux, protected Middleware) {
	mux.Handle("GET /api/v1/certifications", protected(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/certifications", protected(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/certifications/{id}", protected(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/certifications/{id}", protected(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/certifications/{id}", protected(http.HandlerFunc(h.delete)))
}

func (h *CertificationsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := repo.CertificationFilter{
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

func (h *CertificationsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var c domain.Certification
	if err := decodeJSON(w, r, &c); err != nil {
		h.badRequest(w, err)
		return
	}
	if !h.validate(w, &c) {
		return
	}
	c.StampNew(userID, h.Now())
	if err := h.repo.Create(r.Context(), &c); err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, c)
}

func (h *CertificationsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, c)
}

func (h *CertificationsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var in domain.Certification
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

func (h *CertificationsHandler) delete(w http.ResponseWriter, r *http.Request) {
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
