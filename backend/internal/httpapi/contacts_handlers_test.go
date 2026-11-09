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

// newContactsTestServer stands up the full stack with the contacts module
// mounted, and returns a logged-in client.
func newContactsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger, memory.NewLookupRepo())
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewContactsHandler(base, memory.NewContactRepo()))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	c := loggedInClient(t, ts, "one@example.com")
	return ts, c
}

const validContact = `{"name":"Priya Sharma","company":"Initech","role":"Technical Recruiter","relationship":"recruiter","email":"priya@initech.example","linkedinUrl":"https://linkedin.com/in/priya","lastContactedOn":"2026-07-01","nextFollowupOn":"2026-07-15"}`

func createContact(t *testing.T, c *http.Client, url, body string) map[string]any {
	t.Helper()
	resp, err := c.Post(url+"/api/v1/contacts", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create contact = %d: %s", resp.StatusCode, b)
	}
	return decodeBody(t, resp)
}

func TestContactCRUD(t *testing.T) {
	ts, c := newContactsTestServer(t)

	created := createContact(t, c, ts.URL, validContact)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created contact should have an id")
	}
	if created["lastContactedOn"] != "2026-07-01" {
		t.Errorf("lastContactedOn round-trip = %v", created["lastContactedOn"])
	}
	if created["nextFollowupOn"] != "2026-07-15" {
		t.Errorf("nextFollowupOn round-trip = %v", created["nextFollowupOn"])
	}
	if created["createdAt"] == nil || created["updatedAt"] == nil {
		t.Error("audit timestamps should be set")
	}

	// Read it back.
	resp, _ := c.Get(ts.URL + "/api/v1/contacts/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get = %d", resp.StatusCode)
	}
	got := decodeBody(t, resp)
	if got["name"] != "Priya Sharma" {
		t.Errorf("name = %v", got["name"])
	}

	// Update the relationship.
	updated := strings.Replace(validContact, `"relationship":"recruiter"`, `"relationship":"mentor"`, 1)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/contacts/"+id, strings.NewReader(updated))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %v %d", err, resp.StatusCode)
	}
	afterUpdate := decodeBody(t, resp)
	if afterUpdate["relationship"] != "mentor" {
		t.Errorf("relationship after update = %v", afterUpdate["relationship"])
	}
	if afterUpdate["id"] != id {
		t.Error("update must not change the record id")
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/contacts/"+id, nil)
	resp, _ = c.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	resp, _ = c.Get(ts.URL + "/api/v1/contacts/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestContactValidationRejected(t *testing.T) {
	ts, c := newContactsTestServer(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"missing name", `{"relationship":"recruiter"}`, "name"},
		{"missing relationship", `{"name":"Priya"}`, "relationship"},
		{"bad relationship", `{"name":"Priya","relationship":"bogus"}`, "relationship"},
		{"bad email", `{"name":"Priya","relationship":"recruiter","email":"not-an-email"}`, "email"},
		{"bad linkedin url", `{"name":"Priya","relationship":"recruiter","linkedinUrl":"javascript:alert(1)"}`, "linkedinUrl"},
	}
	for _, tc := range cases {
		resp, err := c.Post(ts.URL+"/api/v1/contacts", "application/json", strings.NewReader(tc.body))
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

func TestContactSearchAndFilter(t *testing.T) {
	ts, c := newContactsTestServer(t)

	seed := []string{
		`{"name":"Priya Sharma","company":"Initech","role":"Technical Recruiter","relationship":"recruiter","notes":"met at Go meetup"}`,
		`{"name":"Marcus Webb","company":"Globex","role":"Staff Engineer","relationship":"engineer","notes":"referral for platform team"}`,
		`{"name":"Dana Cole","company":"Hooli","role":"Engineering Manager","relationship":"hiring_manager","email":"dana@hooli.example"}`,
	}
	for _, s := range seed {
		createContact(t, c, ts.URL, s)
	}

	list := func(query string) []any {
		resp, err := c.Get(ts.URL + "/api/v1/contacts" + query)
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
	if got := list("?relationship=recruiter"); len(got) != 1 {
		t.Errorf("relationship filter = %d rows, want 1", len(got))
	}
	// Case-insensitive text search across fields.
	if got := list("?q=ENGINEER"); len(got) != 2 {
		t.Errorf("q=ENGINEER = %d rows, want 2", len(got))
	}
	if got := list("?q=meetup"); len(got) != 1 {
		t.Errorf("q=meetup (notes) = %d rows, want 1", len(got))
	}
	if got := list("?q=hooli.example"); len(got) != 1 {
		t.Errorf("q=hooli.example (email) = %d rows, want 1", len(got))
	}
	if got := list("?q=engineer&relationship=engineer"); len(got) != 1 {
		t.Errorf("combined filter = %d rows, want 1", len(got))
	}
	if got := list("?q=nomatch"); len(got) != 0 {
		t.Errorf("no-match search = %d rows, want 0", len(got))
	}
}

func TestContactOwnershipIsolation(t *testing.T) {
	ts, alice := newContactsTestServer(t)
	bob := loggedInClient(t, ts, "bob@example.com")

	created := createContact(t, alice, ts.URL, validContact)
	id := created["id"].(string)

	// Bob cannot see, update, or delete Alice's record — and cannot even
	// learn that it exists (404, not 403).
	resp, _ := bob.Get(ts.URL + "/api/v1/contacts/" + id)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user get = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/contacts/"+id, strings.NewReader(validContact))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user update = %d, want 404", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/contacts/"+id, nil)
	resp, _ = bob.Do(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-user delete = %d, want 404", resp.StatusCode)
	}

	// Bob's list must not contain Alice's rows.
	resp, _ = bob.Get(ts.URL + "/api/v1/contacts")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's rows", len(items))
	}
}

func TestContactRequiresAuth(t *testing.T) {
	ts, _ := newContactsTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/contacts")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated list = %d, want 401", resp.StatusCode)
	}
}
