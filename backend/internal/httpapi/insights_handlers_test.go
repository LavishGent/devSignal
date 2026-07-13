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
	"devsignal/internal/recommend"
	"devsignal/internal/repo/memory"
	"devsignal/internal/search"
	"devsignal/internal/snapshot"
)

// newInsightsTestServer mounts every module plus the insights endpoints over
// shared in-memory repos, so cross-module behavior is exercised end to end.
func newInsightsTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := memory.NewUserRepo()
	sessions := auth.NewSessionManager(memory.NewSessionRepo(), time.Hour)
	base := NewBaseHandler(logger)
	authH := NewAuthHandler(base, users, sessions, 10, false)
	srv := NewServer(logger, false, "", authH)

	apps := memory.NewApplicationRepo()
	contacts := memory.NewContactRepo()
	interviews := memory.NewInterviewRepo()
	skills := memory.NewSkillRepo()
	studySessions := memory.NewStudySessionRepo()
	certs := memory.NewCertificationRepo()
	projects := memory.NewProjectRepo()
	goals := memory.NewGoalRepo()

	srv.AddModule(NewApplicationsHandler(base, apps))
	srv.AddModule(NewContactsHandler(base, contacts))
	srv.AddModule(NewInterviewsHandler(base, interviews, apps))
	srv.AddModule(NewSkillsHandler(base, skills))
	srv.AddModule(NewStudySessionsHandler(base, studySessions, skills))
	srv.AddModule(NewCertificationsHandler(base, certs))
	srv.AddModule(NewProjectsHandler(base, projects))
	srv.AddModule(NewGoalsHandler(base, goals))

	loader := snapshot.Loader{
		Applications: apps, Contacts: contacts, Interviews: interviews,
		Skills: skills, StudySessions: studySessions, Certifications: certs,
		Projects: projects, Goals: goals,
	}
	searcher := &search.Service{
		Applications: apps, Contacts: contacts, Interviews: interviews,
		Skills: skills, StudySessions: studySessions, Certifications: certs,
		Projects: projects, Goals: goals,
	}
	srv.AddModule(NewInsightsHandler(base, loader, recommend.NewEngine(), searcher))

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, loggedInClient(t, ts, "insights@example.com")
}

// seedInsights creates records across modules through the real API.
func seedInsights(t *testing.T, ts *httptest.Server, c *http.Client) {
	t.Helper()
	post := func(path, body string) {
		t.Helper()
		resp, err := c.Post(ts.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("seed %s = %d: %s", path, resp.StatusCode, b)
		}
		_ = resp.Body.Close()
	}
	post("/api/v1/applications", `{"company":"Initech","position":"Backend Engineer","status":"applied","notes":"Go role"}`)
	post("/api/v1/contacts", `{"name":"Rachel Recruiter","relationship":"recruiter","company":"Initech"}`)
	post("/api/v1/skills", `{"name":"Go","category":"language","proficiency":2,"targetProficiency":5}`)
	post("/api/v1/goals", `{"title":"Land a Go role","category":"job_search","status":"in_progress","priority":1}`)
	post("/api/v1/projects", `{"name":"go-tracker","status":"active","techStack":"Go, PostgreSQL"}`)
}

func TestDashboardEndpoint(t *testing.T) {
	ts, c := newInsightsTestServer(t)
	seedInsights(t, ts, c)

	resp, err := c.Get(ts.URL + "/api/v1/dashboard")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard: %v %d", err, resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["totalApplications"] != float64(1) {
		t.Errorf("totalApplications = %v", body["totalApplications"])
	}
	byStatus, _ := body["applicationsByStatus"].(map[string]any)
	if byStatus["applied"] != float64(1) {
		t.Errorf("applicationsByStatus = %v", byStatus)
	}
	if body["generatedAt"] == nil {
		t.Error("dashboard should carry a generated-at timestamp")
	}
	if days, _ := body["studyByDay"].([]any); len(days) != 28 {
		t.Errorf("studyByDay length = %d, want 28", len(days))
	}
}

func TestRecommendationsEndpoint(t *testing.T) {
	ts, c := newInsightsTestServer(t)
	seedInsights(t, ts, c)

	resp, err := c.Get(ts.URL + "/api/v1/recommendations")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("recommendations: %v %d", err, resp.StatusCode)
	}
	body := decodeBody(t, resp)
	items, _ := body["items"].([]any)
	// The seeded Go skill (gap, never practiced) must produce a suggestion.
	found := false
	for _, it := range items {
		rec := it.(map[string]any)
		if rec["rule"] == "stale_skill" && strings.Contains(rec["title"].(string), "Go") {
			found = true
			if rec["priority"] == nil || rec["module"] != "skills" {
				t.Errorf("recommendation shape wrong: %v", rec)
			}
		}
	}
	if !found {
		t.Errorf("expected a stale_skill recommendation, got %v", items)
	}

	// limit caps the list.
	resp, _ = c.Get(ts.URL + "/api/v1/recommendations?limit=1")
	body = decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) > 1 {
		t.Errorf("limit=1 returned %d items", len(items))
	}
}

func TestReportEndpoints(t *testing.T) {
	ts, c := newInsightsTestServer(t)
	seedInsights(t, ts, c)

	// Catalog lists all five reports.
	resp, _ := c.Get(ts.URL + "/api/v1/reports")
	body := decodeBody(t, resp)
	if items, _ := body["items"].([]any); len(items) != 5 {
		t.Fatalf("report catalog = %d entries, want 5", len(items))
	}

	// JSON report has title, timestamp, columns, rows.
	resp, _ = c.Get(ts.URL + "/api/v1/reports/application-pipeline")
	rep := decodeBody(t, resp)
	if rep["title"] != "Application Pipeline" || rep["generatedAt"] == nil {
		t.Errorf("report metadata wrong: %v", rep)
	}
	if rows, _ := rep["rows"].([]any); len(rows) != 1 {
		t.Errorf("report rows = %d, want 1", len(rows))
	}

	// CSV export sets download headers and includes the header row.
	resp, _ = c.Get(ts.URL + "/api/v1/reports/skill-gap?format=csv")
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("csv content type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "skill-gap.csv") {
		t.Errorf("csv disposition = %q", cd)
	}
	data, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(data), "Skill,Category,Proficiency") {
		t.Errorf("csv missing header: %s", data)
	}

	// Unknown report is a 404.
	resp, _ = c.Get(ts.URL + "/api/v1/reports/bogus")
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown report = %d, want 404", resp.StatusCode)
	}
}

func TestGlobalSearchEndpoint(t *testing.T) {
	ts, c := newInsightsTestServer(t)
	seedInsights(t, ts, c)

	// "go" matches the application (notes), skill (name), goal (title),
	// and project (name + tech stack) — grouped multi-module results.
	resp, err := c.Get(ts.URL + "/api/v1/search?q=" + "go")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("search: %v %d", err, resp.StatusCode)
	}
	body := decodeBody(t, resp)
	groups, _ := body["groups"].([]any)
	moduleSet := map[string]bool{}
	total := 0
	for _, g := range groups {
		grp := g.(map[string]any)
		items, _ := grp["items"].([]any)
		if len(items) == 0 {
			t.Errorf("group %v has no items but was returned", grp["module"])
		}
		total += len(items)
		moduleSet[grp["module"].(string)] = true
	}
	for _, want := range []string{"applications", "skills", "goals", "projects"} {
		if !moduleSet[want] {
			t.Errorf("expected a %s group in search results, got %v", want, moduleSet)
		}
	}
	if total < 4 {
		t.Errorf("expected multi-row results, got %d", total)
	}

	// Empty query is rejected.
	resp, _ = c.Get(ts.URL + "/api/v1/search?q=")
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty q = %d, want 400", resp.StatusCode)
	}
}

func TestInsightsIsolatedPerUser(t *testing.T) {
	ts, alice := newInsightsTestServer(t)
	seedInsights(t, ts, alice)
	bob := loggedInClient(t, ts, fmt.Sprintf("bob-%d@example.com", time.Now().UnixNano()))

	resp, _ := bob.Get(ts.URL + "/api/v1/dashboard")
	body := decodeBody(t, resp)
	if body["totalApplications"] != float64(0) {
		t.Errorf("bob sees alice's applications: %v", body["totalApplications"])
	}
	resp, _ = bob.Get(ts.URL + "/api/v1/search?q=Initech")
	body = decodeBody(t, resp)
	if groups, _ := body["groups"].([]any); len(groups) != 0 {
		t.Errorf("bob's search sees alice's records: %v", groups)
	}
}
