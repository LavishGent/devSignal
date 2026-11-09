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

// newGoalsTestServer stands up the full stack with the goals module mounted,
// and returns a logged-in client.
func newGoalsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger, memory.NewLookupRepo())
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewGoalsHandler(base, memory.NewGoalRepo()))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "goals@example.com")
	return ts, c
}

const validGoal = `{"title":"Ship portfolio site","description":"Personal site built in Go","category":"project","status":"in_progress","priority":2,"targetDate":"2026-12-31"}`

func createGoal(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/goals", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create goal = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestGoalCRUD(t *testing.T) {
	ts, c := newGoalsTestServer(t)

	created := createGoal(t, c, ts.URL, validGoal)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created goal should have an id")
	}
	if created["targetDate"] != "2026-12-31" {
		t.Errorf("targetDate round-trip = %v", created["targetDate"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/goals/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["title"] != "Ship portfolio site" {
		t.Errorf("title = %v", got["title"])
	}

	// Complete the goal.
	updated := strings.Replace(validGoal, `"status":"in_progress"`, `"status":"completed","completedOn":"2026-07-10"`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/goals/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["status"] != "completed" {
		t.Errorf("status after update = %v", afterUpdate["status"])
	}
	if afterUpdate["completedOn"] != "2026-07-10" {
		t.Errorf("completedOn after update = %v", afterUpdate["completedOn"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/goals/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/goals/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestGoalValidationRejected(t *testing.T) {
	ts, c := newGoalsTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing title", `{"category":"skill","status":"not_started","priority":1}`, "title"},
		{"bad category", `{"title":"X","category":"hobby","status":"not_started","priority":1}`, "category"},
		{"bad status", `{"title":"X","category":"skill","status":"bogus","priority":1}`, "status"},
		{"priority out of range", `{"title":"X","category":"skill","status":"not_started","priority":5}`, "priority"},
		{"completedOn without completed", `{"title":"X","category":"skill","status":"in_progress","priority":1,"completedOn":"2026-07-01"}`, "completedOn"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/goals", "application/json", strings.NewReader(tc.body))
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

func TestGoalSearchAndFilter(t *testing.T) {
	ts, c := newGoalsTestServer(t)

	seed := []string{
		`{"title":"Learn Go generics","description":"Deep dive on type parameters","category":"skill","status":"in_progress","priority":1}`,
		`{"title":"AWS certification","description":"Solutions Architect exam","category":"certification","status":"not_started","priority":2,"notes":"Study cloud fundamentals"}`,
		`{"title":"Build portfolio site","description":"Personal website in Go","category":"project","status":"in_progress","priority":3}`,
	}
	for _, s := range seed {
		createGoal(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/goals" + query)
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
	if got := list("?category=certification"); len(got) != 1 {
		t.Errorf("category filter = %d rows, want 1", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=GO"); len(got) != 2 {
		t.Errorf("q=GO = %d rows, want 2", len(got))
	}
	if got := list("?q=cloud"); len(got) != 1 {
		t.Errorf("q=cloud (notes) = %d rows, want 1", len(got))
	}
	if got := list("?q=go&category=project"); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestGoalOwnershipIsolation(t *testing.T) {
	ts, alice := newGoalsTestServer(t)
	bob := loggedInClient(t, ts, "goals-bob@example.com")

	created := createGoal(t, alice, ts.URL, validGoal)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/goals/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/goals/"+id, strings.NewReader(validGoal))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/goals/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/goals")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestGoalRequiresAuth(t *testing.T) {
	ts, _ := newGoalsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/goals")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}
