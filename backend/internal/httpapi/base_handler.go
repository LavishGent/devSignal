package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// ctxKey is unexported so outside packages cannot forge context values.
type ctxKey int

const userIDKey ctxKey = 1

// withUserID stores the authenticated user on the request context.
func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFrom extracts the authenticated user id set by the auth middleware.
func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// BaseHandler carries behavior shared by every module handler. Module
// handlers embed it (inheritance via struct embedding), so request parsing,
// error mapping, and clock access are written exactly once.
type BaseHandler struct {
	Logger  *slog.Logger
	Now     func() time.Time
	Lookups repo.LookupRepo // fixed-choice option lists, owned by the database
}

// NewBaseHandler builds the shared handler core.
func NewBaseHandler(logger *slog.Logger, lookups repo.LookupRepo) BaseHandler {
	return BaseHandler{Logger: logger, Now: time.Now, Lookups: lookups}
}

// userID returns the authenticated user, failing the request if the auth
// middleware did not run (a programming error, not a user error).
func (h BaseHandler) userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := UserIDFrom(r.Context())
	if !ok {
		h.Logger.Error("handler reached without auth middleware", "path", r.URL.Path)
		respondError(w, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, false
	}
	return id, true
}

// pathID parses the {id} path segment as a UUID.
func (h BaseHandler) pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusNotFound, "record not found")
		return uuid.Nil, false
	}
	return id, true
}

// page reads limit/offset query parameters into a clamped Page.
func (h BaseHandler) page(r *http.Request) repo.Page {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	return repo.Page{Limit: limit, Offset: offset}.Clamp()
}

// repoError maps persistence errors onto HTTP statuses uniformly.
func (h BaseHandler) repoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		respondError(w, http.StatusNotFound, "record not found")
	case errors.Is(err, repo.ErrConflict):
		respondError(w, http.StatusConflict, "a conflicting record already exists")
	default:
		h.Logger.Error("repository error", "err", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// badRequest reports a request-shape problem (malformed JSON, bad params).
func (h BaseHandler) badRequest(w http.ResponseWriter, err error) {
	respondError(w, http.StatusBadRequest, err.Error())
}

// validate runs any entity through the Validator interface, then checks its
// fixed-choice fields against the lookup table, and writes the field-keyed
// 400 response on failure. Every module handler validates its records
// polymorphically through this one contract. Structural and enumeration
// problems are reported together so the user fixes the form in one pass.
func (h BaseHandler) validate(w http.ResponseWriter, r *http.Request, v domain.Validator) bool {
	errs := v.Validate()
	if ent, ok := v.(domain.Enumerated); ok {
		lookups, err := h.Lookups.All(r.Context())
		if err != nil {
			h.repoError(w, err)
			return false
		}
		domain.ValidateEnums(errs, ent, lookups)
	}
	if !errs.Ok() {
		respondValidation(w, errs)
		return false
	}
	return true
}
