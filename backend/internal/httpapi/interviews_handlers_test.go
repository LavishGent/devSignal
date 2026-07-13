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

// newInterviewsTestServer stands up the full stack with the applications and
// interviews modules mounted (interviews link to applications), and returns a
// logged-in client.
func newInterviewsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger)
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	applications := memory.NewApplicationRepo()
	srv.AddModule(NewApplicationsHandler(base, applications))
	srv.AddModule(NewInterviewsHandler(base, memory.NewInterviewRepo(), applications))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "interviews@example.com")
	return ts, c
}

const validInterviewTemplate = `{"applicationId":%q,"stage":"technical","scheduledAt":"2026-07-20T14:00:00Z","durationMinutes":60,"interviewer":"Jane Smith","location":"Zoom","outcome":"pending","prepNotes":"Review Go internals"}`

// validInterview fills the template with the application to link to.
func validInterview(applicationID string) string {
	return fmt.Sprintf(validInterviewTemplate, applicationID)
}

// createInterviewApplication seeds an application and returns its id, so
// interviews have something valid to link to.
func createInterviewApplication(t *testing.T, c *http.Client, url string) string {
	t.Helper()
	created := createApplication(t, c, url, validApplication)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("seeded application should have an id")
	}
	return id
}

func createInterview(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/interviews", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create interview = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestInterviewCRUD(t *testing.T) {
	ts, c := newInterviewsTestServer(t)
	appID := createInterviewApplication(t, c, ts.URL)

	created := createInterview(t, c, ts.URL, validInterview(appID))
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created interview should have an id")
	}
	if created["applicationId"] != appID {
		t.Errorf("applicationId round-trip = %v", created["applicationId"])
	}
	if created["scheduledAt"] != "2026-07-20T14:00:00Z" {
		t.Errorf("scheduledAt round-trip = %v", created["scheduledAt"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/interviews/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["interviewer"] != "Jane Smith" {
		t.Errorf("interviewer = %v", got["interviewer"])
	}

	// Update the outcome.
	updated := strings.Replace(validInterview(appID), `"outcome":"pending"`, `"outcome":"passed"`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/interviews/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["outcome"] != "passed" {
		t.Errorf("outcome after update = %v", afterUpdate["outcome"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/interviews/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/interviews/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestInterviewValidationRejected(t *testing.T) {
	ts, c := newInterviewsTestServer(t)
	appID := createInterviewApplication(t, c, ts.URL)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing applicationId", `{"stage":"technical","scheduledAt":"2026-07-20T14:00:00Z","outcome":"pending"}`, "applicationId"},
		{"bad stage", fmt.Sprintf(`{"applicationId":%q,"stage":"bogus","scheduledAt":"2026-07-20T14:00:00Z","outcome":"pending"}`, appID), "stage"},
		{"missing scheduledAt", fmt.Sprintf(`{"applicationId":%q,"stage":"technical","outcome":"pending"}`, appID), "scheduledAt"},
		{"bad outcome", fmt.Sprintf(`{"applicationId":%q,"stage":"technical","scheduledAt":"2026-07-20T14:00:00Z","outcome":"maybe"}`, appID), "outcome"},
		{"duration too long", fmt.Sprintf(`{"applicationId":%q,"stage":"technical","scheduledAt":"2026-07-20T14:00:00Z","outcome":"pending","durationMinutes":2000}`, appID), "durationMinutes"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/interviews", "application/json", strings.NewReader(tc.body))
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

func TestInterviewApplicationLinkRejected(t *testing.T) {
	ts, alice := newInterviewsTestServer(t)
	bob := loggedInClient(t, ts, "bob-interviews@example.com")

	aliceAppID := createInterviewApplication(t, alice, ts.URL)

	// A well-formed but unknown application id is rejected on the link check.
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
		if fields["applicationId"] == nil {
			t.Errorf("%s: expected problem on applicationId, got %v", name, body)
		}
	}

	unknown := validInterview("11111111-2222-3333-4444-555555555555")
	resp, err := alice.Post(ts.URL+"/api/v1/interviews", "application/json", strings.NewReader(unknown))
	assertLinkRejected("unknown application", resp, err)

	// Bob cannot link an interview to Alice's application.
	resp, err = bob.Post(ts.URL+"/api/v1/interviews", "application/json", strings.NewReader(validInterview(aliceAppID)))
	assertLinkRejected("cross-user link on create", resp, err)

	// Nor can Bob update one of his interviews to point at Alice's application.
	bobAppID := createInterviewApplication(t, bob, ts.URL)
	created := createInterview(t, bob, ts.URL, validInterview(bobAppID))
	id := created["id"].(string)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/interviews/"+id, strings.NewReader(validInterview(aliceAppID)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = bob.Do(req)
	assertLinkRejected("cross-user link on update", resp, err)
}

func TestInterviewSearchAndFilter(t *testing.T) {
	ts, c := newInterviewsTestServer(t)
	appA := createInterviewApplication(t, c, ts.URL)
	appB := createInterviewApplication(t, c, ts.URL)

	seed := []string{
		fmt.Sprintf(`{"applicationId":%q,"stage":"technical","scheduledAt":"2026-07-20T14:00:00Z","outcome":"pending","interviewer":"Jane Smith","prepNotes":"Go concurrency drills"}`, appA),
		fmt.Sprintf(`{"applicationId":%q,"stage":"behavioral","scheduledAt":"2026-07-21T10:00:00Z","outcome":"passed","interviewer":"Raj Patel","location":"Onsite HQ"}`, appA),
		fmt.Sprintf(`{"applicationId":%q,"stage":"technical","scheduledAt":"2026-07-22T09:00:00Z","outcome":"pending","interviewer":"Sam Lee","feedback":"strong on Go, weak on SQL"}`, appB),
	}
	for _, s := range seed {
		createInterview(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/interviews" + query)
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
	if got := list("?stage=technical"); len(got) != 2 {
		t.Errorf("stage filter = %d rows, want 2", len(got))
	}
	if got := list("?outcome=passed"); len(got) != 1 {
		t.Errorf("outcome filter = %d rows, want 1", len(got))
	}
	if got := list("?applicationId=" + appA); len(got) != 2 {
		t.Errorf("applicationId filter = %d rows, want 2", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=GO"); len(got) != 2 {
		t.Errorf("q=GO = %d rows, want 2", len(got))
	}
	if got := list("?q=onsite"); len(got) != 1 {
		t.Errorf("q=onsite (location) = %d rows, want 1", len(got))
	}
	if got := list("?q=go&stage=technical&applicationId=" + appB); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestInterviewOwnershipIsolation(t *testing.T) {
	ts, alice := newInterviewsTestServer(t)
	bob := loggedInClient(t, ts, "bob-isolation@example.com")

	appID := createInterviewApplication(t, alice, ts.URL)
	created := createInterview(t, alice, ts.URL, validInterview(appID))
	id := created["id"].(string)

	// Bob cannot see Alice's interview — and cannot even learn that it
	// exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/interviews/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/interviews")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestInterviewRequiresAuth(t *testing.T) {
	ts, _ := newInterviewsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/interviews")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}
