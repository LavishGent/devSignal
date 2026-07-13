package httpapi

import (
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

// newSkillsTestServer stands up the full stack with the skills module
// mounted, and returns a logged-in client.
func newSkillsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger)
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewSkillsHandler(base, memory.NewSkillRepo()))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "one@example.com")
	return ts, c
}

const validSkill = `{"name":"Go","category":"language","proficiency":3,"targetProficiency":5,"lastPracticedOn":"2026-07-01","notes":"Daily driver at work"}`

func createSkill(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/skills", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create skill = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestSkillCRUD(t *testing.T) {
	ts, c := newSkillsTestServer(t)

	created := createSkill(t, c, ts.URL, validSkill)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created skill should have an id")
	}
	if created["lastPracticedOn"] != "2026-07-01" {
		t.Errorf("lastPracticedOn round-trip = %v", created["lastPracticedOn"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/skills/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["name"] != "Go" {
		t.Errorf("name = %v", got["name"])
	}

	// Update the proficiency.
	updated := strings.Replace(validSkill, `"proficiency":3`, `"proficiency":4`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/skills/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["proficiency"] != float64(4) {
		t.Errorf("proficiency after update = %v", afterUpdate["proficiency"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/skills/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/skills/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestSkillValidationRejected(t *testing.T) {
	ts, c := newSkillsTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing name", `{"category":"language","proficiency":3,"targetProficiency":5}`, "name"},
		{"bad category", `{"name":"Go","category":"bogus","proficiency":3,"targetProficiency":5}`, "category"},
		{"proficiency too low", `{"name":"Go","category":"language","proficiency":0,"targetProficiency":5}`, "proficiency"},
		{"proficiency too high", `{"name":"Go","category":"language","proficiency":6,"targetProficiency":5}`, "proficiency"},
		{"target out of range", `{"name":"Go","category":"language","proficiency":3,"targetProficiency":9}`, "targetProficiency"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/skills", "application/json", strings.NewReader(tc.body))
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

func TestSkillSearchAndFilter(t *testing.T) {
	ts, c := newSkillsTestServer(t)

	seed := []string{
		`{"name":"Go","category":"language","proficiency":3,"targetProficiency":5,"notes":"Backend services"}`,
		`{"name":"PostgreSQL","category":"database","proficiency":2,"targetProficiency":4,"notes":"Query tuning practice"}`,
		`{"name":"Python","category":"language","proficiency":4,"targetProficiency":5,"notes":"Scripting and ML work"}`,
	}
	for _, s := range seed {
		createSkill(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/skills" + query)
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
	if got := list("?category=language"); len(got) != 2 {
		t.Errorf("category filter = %d rows, want 2", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=PY"); len(got) != 1 {
		t.Errorf("q=PY = %d rows, want 1", len(got))
	}
	if got := list("?q=practice"); len(got) != 1 {
		t.Errorf("q=practice (notes) = %d rows, want 1", len(got))
	}
	if got := list("?q=ml&category=language"); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestSkillOwnershipIsolation(t *testing.T) {
	ts, alice := newSkillsTestServer(t)
	bob := loggedInClient(t, ts, "bob@example.com")

	created := createSkill(t, alice, ts.URL, validSkill)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/skills/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/skills/"+id, strings.NewReader(validSkill))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/skills/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/skills")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestSkillRequiresAuth(t *testing.T) {
	ts, _ := newSkillsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/skills")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}
