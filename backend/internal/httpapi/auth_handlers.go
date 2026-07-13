package httpapi

import (
	"errors"
	"net/http"
	"time"

	"devsignal/internal/auth"
	"devsignal/internal/domain"
	"devsignal/internal/repo"
	"devsignal/internal/validation"
)

// sessionCookie is the browser cookie carrying the bearer token.
const sessionCookie = "devsignal_session"

// dummyBcryptHash is compared when a login email is unknown so that unknown
// and known emails take similar time (prevents account enumeration by timing).
const dummyBcryptHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeO8oYKpXV0Fw11AxwSFYUCCJZLp2rSx7u"

// AuthHandler serves register/login/logout/me. It embeds BaseHandler for the
// shared request plumbing (inheritance) and keeps its collaborators
// unexported (encapsulation).
type AuthHandler struct {
	BaseHandler
	users      repo.UserRepo
	sessions   *auth.SessionManager
	bcryptCost int
	secure     bool // Secure cookie flag; true in production
}

// NewAuthHandler wires the authentication endpoints.
func NewAuthHandler(base BaseHandler, users repo.UserRepo, sessions *auth.SessionManager, bcryptCost int, secure bool) *AuthHandler {
	return &AuthHandler{BaseHandler: base, users: users, sessions: sessions, bcryptCost: bcryptCost, secure: secure}
}

// Register mounts the auth routes. Login and register sit behind the
// rate limiter; logout/me require a session.
func (h *AuthHandler) Register(mux *http.ServeMux, limited Middleware, protected Middleware) {
	mux.Handle("POST /api/v1/auth/register", limited(http.HandlerFunc(h.register)))
	mux.Handle("POST /api/v1/auth/login", limited(http.HandlerFunc(h.login)))
	mux.Handle("POST /api/v1/auth/logout", http.HandlerFunc(h.logout))
	mux.Handle("GET /api/v1/auth/me", protected(http.HandlerFunc(h.me)))
}

type credentialsInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var in credentialsInput
	if err := decodeJSON(w, r, &in); err != nil {
		h.badRequest(w, err)
		return
	}

	now := h.Now()
	user := &domain.User{
		Email:       domain.NormalizeEmail(in.Email),
		DisplayName: in.DisplayName,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	errs := user.Validate()
	if len(in.Password) < auth.MinPasswordLength {
		errs.Add("password", "must be at least 8 characters")
	}
	if !errs.Ok() {
		respondValidation(w, errs)
		return
	}

	pw, err := auth.HashPassword(in.Password, h.bcryptCost)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordPolicy) {
			e := validation.New()
			e.Add("password", err.Error())
			respondValidation(w, e)
			return
		}
		h.repoError(w, err)
		return
	}
	user.PasswordHash = pw.Hash()
	user.ID = domain.NewID()

	if err := h.users.Create(r.Context(), user); err != nil {
		if errors.Is(err, repo.ErrConflict) {
			e := validation.New()
			e.Add("email", "is already registered")
			respondValidation(w, e)
			return
		}
		h.repoError(w, err)
		return
	}

	if !h.startSession(w, r, user) {
		return
	}
	respondJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var in credentialsInput
	if err := decodeJSON(w, r, &in); err != nil {
		h.badRequest(w, err)
		return
	}

	user, err := h.users.GetByEmail(r.Context(), in.Email)
	if err != nil {
		// Burn comparable time before answering so unknown emails are not
		// distinguishable from wrong passwords.
		auth.PasswordFromHash(dummyBcryptHash).Matches(in.Password)
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if !auth.PasswordFromHash(user.PasswordHash).Matches(in.Password) {
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !h.startSession(w, r, user) {
		return
	}
	respondJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = h.sessions.Revoke(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
	respondJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		h.repoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, user)
}

// startSession issues a token and sets the session cookie.
func (h *AuthHandler) startSession(w http.ResponseWriter, r *http.Request, user *domain.User) bool {
	token, err := h.sessions.Issue(r.Context(), user.ID)
	if err != nil {
		h.repoError(w, err)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.sessions.TTL() / time.Second),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
	return true
}

// RequireAuth verifies the session cookie and injects the user id into the
// request context. Module routes are wrapped with it.
func (h *AuthHandler) RequireAuth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(sessionCookie)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			s, err := h.sessions.Verify(r.Context(), c.Value)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "session is invalid or expired")
				return
			}
			next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), s.UserID)))
		})
	}
}
