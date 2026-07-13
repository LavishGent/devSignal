package httpapi

import (
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

// newStudySessionsTestServer stands up the full stack with the skills and
// study-sessions modules mounted (study sessions optionally link to skills),
// and returns a logged-in client.
func newStudySessionsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger)
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	skills := memory.NewSkillRepo()
	srv.AddModule(NewSkillsHandler(base, skills))
	srv.AddModule(NewStudySessionsHandler(base, memory.NewStudySessionRepo(), skills))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "study@example.com")
	return ts, c
}

const validStudySession = `{"topic":"Go generics deep dive","minutes":90,"studiedOn":"2026-07-10","resource":"https://go.dev/blog/intro-generics","notes":"Type parameter constraints"}`

// validStudySessionWithSkill builds a payload linked to the given skill.
func validStudySessionWithSkill(skillID string) string {
	return fmt.Sprintf(`{"skillId":%q,"topic":"Go generics deep dive","minutes":90,"studiedOn":"2026-07-10","resource":"https://go.dev/blog/intro-generics","notes":"Type parameter constraints"}`, skillID)
}

// createStudySessionSkill seeds a skill and returns its id, so study sessions
// have something valid to link to.
func createStudySessionSkill(t *testing.T, c *http.Client, url string) string {
	t.Helper()
	created := createSkill(t, c, url, validSkill)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("seeded skill should have an id")
	}
	return id
}

func createStudySession(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/study-sessions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create study session = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestStudySessionCRUD(t *testing.T) {
	ts, c := newStudySessionsTestServer(t)
	skillID := createStudySessionSkill(t, c, ts.URL)

	created := createStudySession(t, c, ts.URL, validStudySessionWithSkill(skillID))
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created study session should have an id")
	}
	if created["skillId"] != skillID {
		t.Errorf("skillId round-trip = %v", created["skillId"])
	}
	if created["studiedOn"] != "2026-07-10" {
		t.Errorf("studiedOn round-trip = %v", created["studiedOn"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// The skill link is optional: creating without one yields a null skillId.
	unlinked := createStudySession(t, c, ts.URL, validStudySession)
	if unlinked["skillId"] != nil {
		t.Errorf("unlinked skillId = %v, want null", unlinked["skillId"])
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/study-sessions/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["topic"] != "Go generics deep dive" {
		t.Errorf("topic = %v", got["topic"])
	}

	// Update the minutes.
	updated := strings.Replace(validStudySessionWithSkill(skillID), `"minutes":90`, `"minutes":120`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/study-sessions/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["minutes"] != float64(120) {
		t.Errorf("minutes after update = %v", afterUpdate["minutes"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/study-sessions/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/study-sessions/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestStudySessionValidationRejected(t *testing.T) {
	ts, c := newStudySessionsTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing topic", `{"minutes":30,"studiedOn":"2026-07-10"}`, "topic"},
		{"minutes too low", `{"topic":"Go","minutes":0,"studiedOn":"2026-07-10"}`, "minutes"},
		{"minutes too high", `{"topic":"Go","minutes":2000,"studiedOn":"2026-07-10"}`, "minutes"},
		{"missing studiedOn", `{"topic":"Go","minutes":30}`, "studiedOn"},
		{"topic too long", fmt.Sprintf(`{"topic":%q,"minutes":30,"studiedOn":"2026-07-10"}`, strings.Repeat("x", 301)), "topic"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/study-sessions", "application/json", strings.NewReader(tc.body))
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

func TestStudySessionSkillLinkRejected(t *testing.T) {
	ts, alice := newStudySessionsTestServer(t)
	bob := loggedInClient(t, ts, "bob-study@example.com")

	aliceSkillID := createStudySessionSkill(t, alice, ts.URL)

	// A well-formed but unknown skill id is rejected on the link check.
	assertLinkRejected := func(name string, resp *http.Response, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, resp.StatusCode)
			_ = resp.Body.Close()
			return
		}
		body := decodeBody(t, resp)
		fields, _ := body["fields"].(map[string]any)
		if fields["skillId"] == nil {
			t.Errorf("%s: expected problem on skillId, got %v", name, body)
		}
	}

	unknown := validStudySessionWithSkill("11111111-2222-3333-4444-555555555555")
	resp, err := alice.Post(ts.URL+"/api/v1/study-sessions", "application/json", strings.NewReader(unknown))
	assertLinkRejected("unknown skill", resp, err)

	// Bob cannot link a study session to Alice's skill.
	resp, err = bob.Post(ts.URL+"/api/v1/study-sessions", "application/json", strings.NewReader(validStudySessionWithSkill(aliceSkillID)))
	assertLinkRejected("cross-user link on create", resp, err)

	// Nor can Bob update one of his sessions to point at Alice's skill.
	created := createStudySession(t, bob, ts.URL, validStudySession)
	id := created["id"].(string)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/study-sessions/"+id, strings.NewReader(validStudySessionWithSkill(aliceSkillID)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = bob.Do(req)
	assertLinkRejected("cross-user link on update", resp, err)
}

func TestStudySessionSearchAndFilter(t *testing.T) {
	ts, c := newStudySessionsTestServer(t)
	skillID := createStudySessionSkill(t, c, ts.URL)

	seed := []string{
		fmt.Sprintf(`{"skillId":%q,"topic":"Go concurrency patterns","minutes":60,"studiedOn":"2026-07-08","notes":"Channels and select"}`, skillID),
		`{"topic":"SQL window functions","minutes":45,"studiedOn":"2026-07-09","resource":"PostgreSQL docs"}`,
		fmt.Sprintf(`{"skillId":%q,"topic":"Go testing","minutes":30,"studiedOn":"2026-07-10","notes":"Table-driven tests"}`, skillID),
	}
	for _, s := range seed {
		createStudySession(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/study-sessions" + query)
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
	if got := list("?skillId=" + skillID); len(got) != 2 {
		t.Errorf("skillId filter = %d rows, want 2", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=GO"); len(got) != 2 {
		t.Errorf("q=GO = %d rows, want 2", len(got))
	}
	if got := list("?q=postgresql"); len(got) != 1 {
		t.Errorf("q=postgresql (resource) = %d rows, want 1", len(got))
	}
	if got := list("?q=channels"); len(got) != 1 {
		t.Errorf("q=channels (notes) = %d rows, want 1", len(got))
	}
	if got := list("?q=testing&skillId=" + skillID); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestStudySessionOwnershipIsolation(t *testing.T) {
	ts, alice := newStudySessionsTestServer(t)
	bob := loggedInClient(t, ts, "bob-study-isolation@example.com")

	created := createStudySession(t, alice, ts.URL, validStudySession)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/study-sessions/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/study-sessions/"+id, strings.NewReader(validStudySession))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/study-sessions/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/study-sessions")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestStudySessionRequiresAuth(t *testing.T) {
	ts, _ := newStudySessionsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/study-sessions")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}
