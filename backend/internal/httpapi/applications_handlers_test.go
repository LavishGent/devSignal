package httpapi

import (
	"encoding/json"
	"fmt"
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

// newModuleTestServer stands up the full stack with the applications module
// mounted, and returns a logged-in client.
func newModuleTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger, memory.NewLookupRepo())
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewApplicationsHandler(base, memory.NewApplicationRepo()))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "one@example.com")
	return ts, c
}

// loggedInClient registers a fresh user and returns a cookie-carrying client.
func loggedInClient(t *testing.T, ts *httptest.Server, email string) *http.Client {
	t.Helper()
	c := &http.Client{Jar: newCookieJar(t)}
	body := fmt.Sprintf(`{"email":%q,"displayName":"Tester","password":"sup3r-secret"}`, email)
	resp, err := c.Post(ts.URL+"/api/v1/auth/register", "application/json", strings.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("register %s: %v %d", email, err, resp.StatusCode)
	}
	_ = resp.Body.Close()
	return c
}

const validApplication = `{"company":"Initech","position":"Backend Engineer","status":"applied","appliedOn":"2026-07-01","workMode":"remote","salaryMin":120000,"salaryMax":150000}`

func createApplication(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/applications", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create application = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestApplicationCRUD(t *testing.T) {
	ts, c := newModuleTestServer(t)

	created := createApplication(t, c, ts.URL, validApplication)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created application should have an id")
	}
	if created["appliedOn"] != "2026-07-01" {
		t.Errorf("appliedOn round-trip = %v", created["appliedOn"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/applications/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["company"] != "Initech" {
		t.Errorf("company = %v", got["company"])
	}

	// Update the status.
	updated := strings.Replace(validApplication, `"status":"applied"`, `"status":"interviewing"`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/applications/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["status"] != "interviewing" {
		t.Errorf("status after update = %v", afterUpdate["status"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/applications/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/applications/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestApplicationValidationRejected(t *testing.T) {
	ts, c := newModuleTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing company", `{"position":"Dev","status":"applied"}`, "company"},
		{"bad status", `{"company":"X","position":"Dev","status":"bogus"}`, "status"},
		{"bad url", `{"company":"X","position":"Dev","status":"applied","url":"javascript:alert(1)"}`, "url"},
		{"salary order", `{"company":"X","position":"Dev","status":"applied","salaryMin":100,"salaryMax":50}`, "salaryMax"},
		{"negative salary", `{"company":"X","position":"Dev","status":"applied","salaryMin":-1}`, "salaryMin"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/applications", "application/json", strings.NewReader(tc.body))
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

func TestApplicationSearchAndFilter(t *testing.T) {
	ts, c := newModuleTestServer(t)

	seed := []string{
		`{"company":"Initech","position":"Backend Engineer","status":"applied","notes":"Go and Postgres role"}`,
		`{"company":"Globex","position":"Platform Engineer","status":"interviewing","notes":"Kubernetes heavy"}`,
		`{"company":"Hooli","position":"Frontend Developer","status":"applied","notes":"React position"}`,
	}
	for _, s := range seed {
		createApplication(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/applications" + query)
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
	if got := list("?status=applied"); len(got) != 2 {
		t.Errorf("status filter = %d rows, want 2", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=ENGINEER"); len(got) != 2 {
		t.Errorf("q=ENGINEER = %d rows, want 2", len(got))
	}
	if got := list("?q=postgres"); len(got) != 1 {
		t.Errorf("q=postgres (notes) = %d rows, want 1", len(got))
	}
	if got := list("?q=engineer&status=interviewing"); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestApplicationOwnershipIsolation(t *testing.T) {
	ts, alice := newModuleTestServer(t)
	bob := loggedInClient(t, ts, "bob@example.com")

	created := createApplication(t, alice, ts.URL, validApplication)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/applications/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/applications/"+id, strings.NewReader(validApplication))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/applications/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/applications")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestApplicationRequiresAuth(t *testing.T) {
	ts, _ := newModuleTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/applications")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}

func TestApplicationMalformedJSON(t *testing.T) {
	ts, c := newModuleTestServer(t)
	resp, err := c.Post(ts.URL+"/api/v1/applications", "application/json", strings.NewReader(`{"company": <<<`))
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
