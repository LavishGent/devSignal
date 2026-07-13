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

// newProjectsTestServer stands up the full stack with the projects module
// mounted, and returns a logged-in client.
func newProjectsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger)
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewProjectsHandler(base, memory.NewProjectRepo()))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "one@example.com")
	return ts, c
}

const validProject = `{"name":"DevSignal","description":"Career tracking dashboard","status":"active","techStack":"Go, Postgres","repoUrl":"https://github.com/tester/devsignal","startedOn":"2026-05-01","lastActivityOn":"2026-07-01"}`

func createProject(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/projects", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create project = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestProjectCRUD(t *testing.T) {
	ts, c := newProjectsTestServer(t)

	created := createProject(t, c, ts.URL, validProject)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created project should have an id")
	}
	if created["startedOn"] != "2026-05-01" {
		t.Errorf("startedOn round-trip = %v", created["startedOn"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/projects/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["name"] != "DevSignal" {
		t.Errorf("name = %v", got["name"])
	}

	// Update the status.
	updated := strings.Replace(validProject, `"status":"active"`, `"status":"paused"`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/projects/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["status"] != "paused" {
		t.Errorf("status after update = %v", afterUpdate["status"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/projects/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/projects/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestProjectValidationRejected(t *testing.T) {
	ts, c := newProjectsTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing name", `{"description":"No name","status":"idea"}`, "name"},
		{"bad status", `{"name":"X","status":"bogus"}`, "status"},
		{"bad repo url", `{"name":"X","status":"idea","repoUrl":"javascript:alert(1)"}`, "repoUrl"},
		{"bad demo url", `{"name":"X","status":"idea","demoUrl":"not-a-url"}`, "demoUrl"},
		{"date order", `{"name":"X","status":"idea","startedOn":"2026-07-01","lastActivityOn":"2026-06-01"}`, "lastActivityOn"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/projects", "application/json", strings.NewReader(tc.body))
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

func TestProjectSearchAndFilter(t *testing.T) {
	ts, c := newProjectsTestServer(t)

	seed := []string{
		`{"name":"DevSignal","description":"Career tracking dashboard","status":"active","techStack":"Go, Postgres","notes":"main portfolio piece"}`,
		`{"name":"Blogfolio","description":"Static site playground","status":"completed","techStack":"Hugo","notes":"markdown experiments"}`,
		`{"name":"TraderBot","description":"Quant trading experiments","status":"active","techStack":"Python, pandas","notes":"paper trading only"}`,
	}
	for _, s := range seed {
		createProject(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/projects" + query)
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
	if got := list("?status=active"); len(got) != 2 {
		t.Errorf("status filter = %d rows, want 2", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=EXPERIMENTS"); len(got) != 2 {
		t.Errorf("q=EXPERIMENTS = %d rows, want 2", len(got))
	}
	if got := list("?q=postgres"); len(got) != 1 {
		t.Errorf("q=postgres (tech stack) = %d rows, want 1", len(got))
	}
	if got := list("?q=experiments&status=active"); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestProjectOwnershipIsolation(t *testing.T) {
	ts, alice := newProjectsTestServer(t)
	bob := loggedInClient(t, ts, "bob@example.com")

	created := createProject(t, alice, ts.URL, validProject)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/projects/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/projects/"+id, strings.NewReader(validProject))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/projects/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/projects")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestProjectRequiresAuth(t *testing.T) {
	ts, _ := newProjectsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/projects")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}

func TestProjectMalformedJSON(t *testing.T) {
	ts, c := newProjectsTestServer(t)
	resp, err := c.Post(ts.URL+"/api/v1/projects", "application/json", strings.NewReader(`{"name": <<<`))
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
