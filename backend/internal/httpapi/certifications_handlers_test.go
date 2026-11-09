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

// newCertificationsTestServer stands up the full stack with the
// certifications module mounted, and returns a logged-in client.
func newCertificationsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger, memory.NewLookupRepo())
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewCertificationsHandler(base, memory.NewCertificationRepo()))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "cert-one@example.com")
	return ts, c
}

const validCertification = `{"name":"AWS Solutions Architect","issuer":"Amazon Web Services","status":"earned","earnedOn":"2026-03-15","expiresOn":"2029-03-15","credentialId":"SAA-12345","credentialUrl":"https://aws.amazon.com/verification/12345"}`

func createCertification(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/certifications", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create certification = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestCertificationCRUD(t *testing.T) {
	ts, c := newCertificationsTestServer(t)

	created := createCertification(t, c, ts.URL, validCertification)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created certification should have an id")
	}
	if created["earnedOn"] != "2026-03-15" || created["expiresOn"] != "2029-03-15" {
		t.Errorf("date round-trip = %v / %v", created["earnedOn"], created["expiresOn"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/certifications/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["issuer"] != "Amazon Web Services" {
		t.Errorf("issuer = %v", got["issuer"])
	}

	// Update the status.
	updated := strings.Replace(validCertification, `"status":"earned"`, `"status":"expired"`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/certifications/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["status"] != "expired" {
		t.Errorf("status after update = %v", afterUpdate["status"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/certifications/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/certifications/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestCertificationValidationRejected(t *testing.T) {
	ts, c := newCertificationsTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing name", `{"issuer":"AWS","status":"planned"}`, "name"},
		{"bad status", `{"name":"CKA","issuer":"CNCF","status":"bogus"}`, "status"},
		{"earned without earnedOn", `{"name":"CKA","issuer":"CNCF","status":"earned"}`, "earnedOn"},
		{"expires before earned", `{"name":"CKA","issuer":"CNCF","status":"earned","earnedOn":"2026-05-01","expiresOn":"2026-04-01"}`, "expiresOn"},
		{"bad credential url", `{"name":"CKA","issuer":"CNCF","status":"planned","credentialUrl":"javascript:alert(1)"}`, "credentialUrl"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/certifications", "application/json", strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, resp.StatusCode)
			_ = resp.Body.Close()
			continue
		}
		body := decodeBody(t, resp)
		fields, _ := body["fields"].(map[string]any)
		if fields[tc.field] == nil {
			t.Errorf("%s: expected problem on %q, got %v", tc.name, tc.field, body)
		}
	}
}

func TestCertificationSearchAndFilter(t *testing.T) {
	ts, c := newCertificationsTestServer(t)

	seed := []string{
		`{"name":"AWS Solutions Architect","issuer":"Amazon Web Services","status":"earned","earnedOn":"2026-01-10","notes":"cloud architecture track"}`,
		`{"name":"Certified Kubernetes Administrator","issuer":"Cloud Native Computing Foundation","status":"in_progress","notes":"container orchestration"}`,
		`{"name":"Google Cloud Professional Architect","issuer":"Google","status":"in_progress","notes":"multi-cloud design"}`,
	}
	for _, s := range seed {
		createCertification(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/certifications" + query)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("list %s: %v %d", query, err, resp.StatusCode)
		}
		body := decodeBody(t, resp)
		items, _ := body["items"].([]any)
		return items
	}

	if got := list(""); len(got) != 3 {
		t.Errorf("unfiltered list = %d rows, want 3", len(got))
	}
	if got := list("?status=in_progress"); len(got) != 2 {
		t.Errorf("status filter = %d rows, want 2", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=ARCHITECT"); len(got) != 2 {
		t.Errorf("q=ARCHITECT = %d rows, want 2", len(got))
	}
	if got := list("?q=amazon"); len(got) != 1 {
		t.Errorf("q=amazon (issuer) = %d rows, want 1", len(got))
	}
	if got := list("?q=orchestration"); len(got) != 1 {
		t.Errorf("q=orchestration (notes) = %d rows, want 1", len(got))
	}
	if got := list("?q=architect&status=in_progress"); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestCertificationOwnershipIsolation(t *testing.T) {
	ts, alice := newCertificationsTestServer(t)
	bob := loggedInClient(t, ts, "cert-bob@example.com")

	created := createCertification(t, alice, ts.URL, validCertification)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/certifications/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/certifications/"+id, strings.NewReader(validCertification))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/certifications/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/certifications")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestCertificationRequiresAuth(t *testing.T) {
	ts, _ := newCertificationsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/certifications")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}

func TestCertificationMalformedJSON(t *testing.T) {
	ts, c := newCertificationsTestServer(t)
	resp, err := c.Post(ts.URL+"/api/v1/certifications", "application/json", strings.NewReader(`{"name": <<<`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("malformed JSON = %d, want 400", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] == nil {
		t.Error("malformed JSON should return an error message")
	}
}
