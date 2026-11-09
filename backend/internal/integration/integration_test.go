//go:build integration

// Package integration exercises the real stack — postgres repositories,
// embedded migrations, and the full HTTP server — against a genuine
// PostgreSQL instance.
//
// Run with a database available (docker compose up -d db):
//
//	TEST_DATABASE_URL=postgres://devsignal:devsignal@localhost:5432/devsignal?sslmode=disable \
//	  go test -tags=integration ./internal/integration/ -v
//
// The suite creates a throwaway database for the run and drops it after, so
// it never touches existing data. Without TEST_DATABASE_URL it skips.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"devsignal/internal/auth"
	"devsignal/internal/database"
	"devsignal/internal/httpapi"
	"devsignal/internal/recommend"
	"devsignal/internal/repo/postgres"
	"devsignal/internal/search"
	"devsignal/internal/snapshot"
	"devsignal/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// startPostgres connects to TEST_DATABASE_URL, creates a scratch database
// for this run, migrates it, and returns a connected pool.
func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set; start postgres (docker compose up -d db) and set it to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connecting to TEST_DATABASE_URL: %v", err)
	}

	dbName := fmt.Sprintf("devsignal_it_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("creating scratch database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+dbName+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})

	cfg, err := pgx.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parsing TEST_DATABASE_URL: %v", err)
	}
	// Replace the last occurrence: the username and password can also be
	// "devsignal", and the database name is the final path segment.
	scratchURL := adminURL
	if i := strings.LastIndex(adminURL, "/"+cfg.Database); i >= 0 {
		scratchURL = adminURL[:i] + "/" + dbName + adminURL[i+len("/"+cfg.Database):]
	}

	pool, err := database.Connect(ctx, scratchURL)
	if err != nil {
		t.Fatalf("connecting to scratch database: %v", err)
	}
	t.Cleanup(pool.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.Migrate(ctx, pool, migrations.FS, logger); err != nil {
		t.Fatalf("migrating: %v", err)
	}
	// Re-running migrations must be a no-op, not an error.
	if err := database.Migrate(ctx, pool, migrations.FS, logger); err != nil {
		t.Fatalf("second migrate should be idempotent: %v", err)
	}
	return pool
}

// newRealServer assembles the production wiring (postgres store, real
// session manager) exactly as cmd/api does.
func newRealServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := postgres.NewStore(pool)
	sessions := auth.NewSessionManager(store.Sessions, time.Hour)
	base := httpapi.NewBaseHandler(logger)
	authH := httpapi.NewAuthHandler(base, store.Users, sessions, 10, false)
	srv := httpapi.NewServer(logger, false, "", authH)

	srv.AddModule(httpapi.NewApplicationsHandler(base, store.Applications))
	srv.AddModule(httpapi.NewContactsHandler(base, store.Contacts))
	srv.AddModule(httpapi.NewInterviewsHandler(base, store.Interviews, store.Applications))
	srv.AddModule(httpapi.NewSkillsHandler(base, store.Skills))
	srv.AddModule(httpapi.NewStudySessionsHandler(base, store.StudySessions, store.Skills))
	srv.AddModule(httpapi.NewCertificationsHandler(base, store.Certifications))
	srv.AddModule(httpapi.NewProjectsHandler(base, store.Projects))
	srv.AddModule(httpapi.NewGoalsHandler(base, store.Goals))
	loader := snapshot.Loader{
		Applications: store.Applications, Contacts: store.Contacts, Interviews: store.Interviews,
		Skills: store.Skills, StudySessions: store.StudySessions, Certifications: store.Certifications,
		Projects: store.Projects, Goals: store.Goals,
	}
	searcher := &search.Service{
		Applications: store.Applications, Contacts: store.Contacts, Interviews: store.Interviews,
		Skills: store.Skills, StudySessions: store.StudySessions, Certifications: store.Certifications,
		Projects: store.Projects, Goals: store.Goals,
	}
	srv.AddModule(httpapi.NewInsightsHandler(base, loader, recommend.NewEngine(), searcher))

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts
}

type client struct {
	t    *testing.T
	http *http.Client
	base string
}

func newClient(t *testing.T, ts *httptest.Server) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, http: &http.Client{Jar: jar}, base: ts.URL}
}

func (c *client) do(method, path, body string, wantStatus int) map[string]any {
	c.t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, wantStatus, raw)
	}
	var m map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &m) == nil {
		return m
	}
	return nil
}

// TestFullStackAgainstRealPostgres drives the complete user journey through
// the production code paths: registration, CRUD in several modules, linked
// records, search, reports, dashboard, recommendations, and deletion
// cascades — all against real PostgreSQL with real migrations.
func TestFullStackAgainstRealPostgres(t *testing.T) {
	pool := startPostgres(t)
	ts := newRealServer(t, pool)

	alice := newClient(t, ts)
	alice.do("POST", "/api/v1/auth/register",
		`{"email":"alice@example.com","displayName":"Alice","password":"sup3r-secret"}`, http.StatusCreated)

	// Application CRUD round-trip.
	app := alice.do("POST", "/api/v1/applications",
		`{"company":"Initech","position":"Backend Engineer","status":"applied","appliedOn":"2026-07-01","workMode":"remote","salaryMin":120000,"salaryMax":150000,"notes":"Go and Postgres role"}`,
		http.StatusCreated)
	appID := app["id"].(string)
	if app["appliedOn"] != "2026-07-01" {
		t.Errorf("date round-trip through postgres failed: %v", app["appliedOn"])
	}

	got := alice.do("GET", "/api/v1/applications/"+appID, "", http.StatusOK)
	if got["company"] != "Initech" || got["salaryMin"] != float64(120000) {
		t.Errorf("application read-back wrong: %v", got)
	}

	updated := alice.do("PUT", "/api/v1/applications/"+appID,
		`{"company":"Initech","position":"Backend Engineer","status":"interviewing","appliedOn":"2026-07-01","workMode":"remote","salaryMin":120000,"salaryMax":150000,"notes":"moved forward"}`,
		http.StatusOK)
	if updated["status"] != "interviewing" {
		t.Errorf("update through postgres failed: %v", updated["status"])
	}

	// Linked interview; the FK ownership check runs against real rows.
	iv := alice.do("POST", "/api/v1/interviews",
		`{"applicationId":"`+appID+`","stage":"technical","scheduledAt":"2026-07-20T15:00:00Z","durationMinutes":60,"interviewer":"Ada","outcome":"pending"}`,
		http.StatusCreated)
	ivID := iv["id"].(string)

	// Skill + linked study session (SET NULL cascade checked later).
	skill := alice.do("POST", "/api/v1/skills",
		`{"name":"Go","category":"language","proficiency":2,"targetProficiency":5}`, http.StatusCreated)
	skillID := skill["id"].(string)
	session := alice.do("POST", "/api/v1/study-sessions",
		`{"skillId":"`+skillID+`","topic":"Goroutines","minutes":90,"studiedOn":"2026-07-10"}`, http.StatusCreated)
	sessionID := session["id"].(string)

	alice.do("POST", "/api/v1/goals",
		`{"title":"Land a Go role","category":"job_search","status":"in_progress","priority":1,"targetDate":"2026-06-01"}`,
		http.StatusCreated)

	// Search hits multiple modules through real ILIKE queries.
	res := alice.do("GET", "/api/v1/search?q=go", "", http.StatusOK)
	groups := res["groups"].([]any)
	if len(groups) < 3 {
		t.Errorf("expected multi-module search hits, got %d groups", len(groups))
	}

	// List filtering with SQL predicates. The update above replaced the
	// notes, so match on the position, which the update preserved.
	list := alice.do("GET", "/api/v1/applications?status=interviewing&q=backend", "", http.StatusOK)
	if items := list["items"].([]any); len(items) != 1 {
		t.Errorf("filtered list = %d rows, want 1", len(items))
	}

	// Dashboard and recommendations run their aggregation over real data.
	dash := alice.do("GET", "/api/v1/dashboard", "", http.StatusOK)
	if dash["totalApplications"] != float64(1) {
		t.Errorf("dashboard totals wrong: %v", dash["totalApplications"])
	}
	recs := alice.do("GET", "/api/v1/recommendations", "", http.StatusOK)
	if items := recs["items"].([]any); len(items) == 0 {
		t.Error("expected recommendations (stale skill, overdue goal) from seeded data")
	}

	// Reports generate with rows.
	rep := alice.do("GET", "/api/v1/reports/application-pipeline", "", http.StatusOK)
	if rows := rep["rows"].([]any); len(rows) != 1 {
		t.Errorf("report rows = %d, want 1", len(rows))
	}

	// A second user must see none of Alice's data (real WHERE user_id).
	bob := newClient(t, ts)
	bob.do("POST", "/api/v1/auth/register",
		`{"email":"bob@example.com","displayName":"Bob","password":"sup3r-secret"}`, http.StatusCreated)
	bob.do("GET", "/api/v1/applications/"+appID, "", http.StatusNotFound)
	bobDash := bob.do("GET", "/api/v1/dashboard", "", http.StatusOK)
	if bobDash["totalApplications"] != float64(0) {
		t.Error("cross-user isolation failed on real postgres")
	}
	// Bob cannot link an interview to Alice's application.
	bob.do("POST", "/api/v1/interviews",
		`{"applicationId":"`+appID+`","stage":"technical","scheduledAt":"2026-07-21T15:00:00Z","outcome":"pending"}`,
		http.StatusBadRequest)

	// Duplicate email surfaces the unique constraint as a field error.
	dup := newClient(t, ts)
	dup.do("POST", "/api/v1/auth/register",
		`{"email":"ALICE@example.com","displayName":"Imposter","password":"sup3r-secret"}`, http.StatusBadRequest)

	// Deleting the skill SET NULLs the study session's link.
	alice.do("DELETE", "/api/v1/skills/"+skillID, "", http.StatusOK)
	after := alice.do("GET", "/api/v1/study-sessions/"+sessionID, "", http.StatusOK)
	if after["skillId"] != nil {
		t.Errorf("skill delete should null the session link, got %v", after["skillId"])
	}

	// Deleting the application cascades to its interview.
	alice.do("DELETE", "/api/v1/applications/"+appID, "", http.StatusOK)
	alice.do("GET", "/api/v1/interviews/"+ivID, "", http.StatusNotFound)

	// Logout revokes the session in the sessions table.
	alice.do("POST", "/api/v1/auth/logout", "", http.StatusOK)
	alice.do("GET", "/api/v1/auth/me", "", http.StatusUnauthorized)
}
