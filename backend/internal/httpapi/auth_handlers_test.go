package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devsignal/internal/auth"
	"devsignal/internal/repo/memory"
)

// newTestServer wires the full middleware/handler stack against in-memory
// repositories, so handler tests exercise exactly what production runs.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger)
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts
}

// client returns an http client that keeps cookies, like a browser.
func client(t *testing.T) *http.Client {
	t.Helper()
	jar := newCookieJar(t)
	return &http.Client{Jar: jar}
}

func postJSON(t *testing.T, c *http.Client, url, body string) *http.Response {
	t.Helper()
	resp, err := c.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return m
}

const registerBody = `{"email":"dev@example.com","displayName":"Darrell","password":"sup3r-secret"}`

func TestRegisterLoginMeLogoutFlow(t *testing.T) {
	ts := newTestServer(t)
	c := client(t)

	// Register signs the user in.
	resp := postJSON(t, c, ts.URL+"/api/v1/auth/register", registerBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["email"] != "dev@example.com" {
		t.Errorf("register response email = %v", body["email"])
	}
	if _, leaked := body["passwordHash"]; leaked {
		t.Error("password hash must never appear in a response")
	}

	// Me works with the session cookie.
	resp, err := c.Get(ts.URL + "/api/v1/auth/me")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("me after register: %v %d", err, resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Logout revokes the session.
	resp = postJSON(t, c, ts.URL+"/api/v1/auth/logout", "")
	_ = resp.Body.Close()
	resp, _ = c.Get(ts.URL + "/api/v1/auth/me")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("me after logout = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Login works again with the right password.
	resp = postJSON(t, c, ts.URL+"/api/v1/auth/login", `{"email":"dev@example.com","password":"sup3r-secret"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("login = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestRegisterValidation(t *testing.T) {
	ts := newTestServer(t)
	c := client(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"bad email", `{"email":"nope","displayName":"D","password":"longenough"}`, "email"},
		{"missing display name", `{"email":"a@b.com","password":"longenough"}`, "displayName"},
		{"short password", `{"email":"a@b.com","displayName":"D","password":"short"}`, "password"},
	}
	for _, tc := range cases {
		resp := postJSON(t, c, ts.URL+"/api/v1/auth/register", tc.body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, resp.StatusCode)
		}
		body := decodeBody(t, resp)
		fields, _ := body["fields"].(map[string]any)
		if fields[tc.field] == nil {
			t.Errorf("%s: expected problem on field %q, got %v", tc.name, tc.field, body)
		}
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ts := newTestServer(t)
	c := client(t)

	resp := postJSON(t, c, ts.URL+"/api/v1/auth/register", registerBody)
	_ = resp.Body.Close()
	// Same email, different case — normalization must still catch it.
	resp = postJSON(t, c, ts.URL+"/api/v1/auth/register",
		`{"email":"DEV@example.com","displayName":"Other","password":"sup3r-secret"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate register = %d, want 400", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	fields, _ := body["fields"].(map[string]any)
	if fields["email"] == nil {
		t.Error("duplicate email should be reported on the email field")
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	ts := newTestServer(t)
	c := client(t)
	resp := postJSON(t, c, ts.URL+"/api/v1/auth/register", registerBody)
	_ = resp.Body.Close()

	for name, body := range map[string]string{
		"wrong password": `{"email":"dev@example.com","password":"wrong-password"}`,
		"unknown email":  `{"email":"ghost@example.com","password":"sup3r-secret"}`,
	} {
		fresh := client(t)
		resp := postJSON(t, fresh, ts.URL+"/api/v1/auth/login", body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, resp.StatusCode)
		}
		// The error must not reveal which part was wrong.
		b := decodeBody(t, resp)
		if b["error"] != "invalid email or password" {
			t.Errorf("%s: error message %q leaks detail", name, b["error"])
		}
	}
}

func TestProtectedRouteRequiresSession(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("me without cookie = %d, want 401", resp.StatusCode)
	}
}

func TestAuthRateLimiting(t *testing.T) {
	ts := newTestServer(t)
	c := client(t)

	var last int
	for range 12 {
		resp := postJSON(t, c, ts.URL+"/api/v1/auth/login", `{"email":"x@y.com","password":"whatever!"}`)
		last = resp.StatusCode
		_ = resp.Body.Close()
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("12th rapid login attempt = %d, want 429", last)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	} {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy header missing")
	}
}
