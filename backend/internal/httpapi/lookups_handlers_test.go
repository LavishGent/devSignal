package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devsignal/internal/auth"
	"devsignal/internal/domain"
	"devsignal/internal/repo/memory"
)

// newLookupServer wires auth, applications, and the options endpoint over a
// lookup repo the test can mutate.
func newLookupServer(t *testing.T) (*httptest.Server, *memory.LookupRepo) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	lookups := memory.NewLookupRepo()
	base := NewBaseHandler(logger, lookups)
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	authH := NewAuthHandler(base, memory.NewUserRepo(), sessions, 10, false)
	srv := NewServer(logger, false, "", authH)
	srv.AddModule(NewApplicationsHandler(base, memory.NewApplicationRepo()))
	srv.AddModule(NewLookupsHandler(base))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, lookups
}

func TestOptionsEndpointServesEveryKind(t *testing.T) {
	ts, _ := newLookupServer(t)
	c := client(t)
	_ = postJSON(t, c, ts.URL+"/api/v1/auth/register", registerBody).Body.Close()

	resp, err := c.Get(ts.URL + "/api/v1/options")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	for _, kind := range domain.LookupKinds {
		opts, ok := body[kind].([]any)
		if !ok || len(opts) == 0 {
			t.Errorf("kind %q missing or empty in response", kind)
			continue
		}
		first := opts[0].(map[string]any)
		if first["value"] == nil || first["label"] == nil {
			t.Errorf("kind %q option lacks value/label: %v", kind, first)
		}
	}
}

func TestOptionsEndpointRequiresAuth(t *testing.T) {
	ts, _ := newLookupServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/options")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// Adding an option to the lookup store is enough for the API to accept it;
// removing one is enough for the API to reject it. No code change needed.
func TestEnumValidationFollowsLookupTable(t *testing.T) {
	ts, lookups := newLookupServer(t)
	c := client(t)
	_ = postJSON(t, c, ts.URL+"/api/v1/auth/register", registerBody).Body.Close()

	body := `{"company":"X","position":"Dev","status":"ghosted"}`

	resp := postJSON(t, c, ts.URL+"/api/v1/applications", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown status: got %d, want 400", resp.StatusCode)
	}
	if fields := decodeBody(t, resp)["fields"].(map[string]any); fields["status"] == nil {
		t.Errorf("expected a status field error, got %v", fields)
	}

	l := domain.DefaultLookups()
	l[domain.KindApplicationStatus] = append(l[domain.KindApplicationStatus], domain.Option{Value: "ghosted", Label: "Ghosted"})
	lookups.Set(l)

	resp = postJSON(t, c, ts.URL+"/api/v1/applications", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("after adding option: got %d, want 201", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
