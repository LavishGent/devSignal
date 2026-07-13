package reports

import (
	"strings"
	"testing"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/snapshot"

	"github.com/google/uuid"
)

var now = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

func stamped(userID uuid.UUID) domain.BaseRecord {
	var b domain.BaseRecord
	b.StampNew(userID, now.Add(-10*24*time.Hour))
	return b
}

func testSnapshot() *snapshot.Snapshot {
	user := uuid.New()
	app1 := domain.Application{BaseRecord: stamped(user), Company: "Initech, Inc.", Position: "Backend Engineer", Status: "applied", AppliedOn: domain.NewDate(2026, time.July, 1), Location: "Remote"}
	app2 := domain.Application{BaseRecord: stamped(user), Company: "Globex", Position: "Platform Engineer", Status: "interviewing"}
	skillGo := domain.Skill{BaseRecord: stamped(user), Name: "Go", Category: "language", Proficiency: 3, TargetProficiency: 5, LastPracticedOn: domain.NewDate(2026, time.July, 10)}
	skillSQL := domain.Skill{BaseRecord: stamped(user), Name: "SQL", Category: "database", Proficiency: 4, TargetProficiency: 4}
	sid := skillGo.ID
	return &snapshot.Snapshot{
		Applications: []domain.Application{app1, app2},
		Interviews: []domain.Interview{
			{BaseRecord: stamped(user), ApplicationID: app2.ID, Stage: "technical", ScheduledAt: now.Add(24 * time.Hour), DurationMinutes: 60, Interviewer: "Ada", Outcome: "pending"},
			{BaseRecord: stamped(user), ApplicationID: app2.ID, Stage: "phone_screen", ScheduledAt: now.Add(-72 * time.Hour), Outcome: "passed"},
		},
		Skills: []domain.Skill{skillGo, skillSQL},
		StudySessions: []domain.StudySession{
			{BaseRecord: stamped(user), SkillID: &sid, Topic: "Goroutines", Minutes: 90, StudiedOn: domain.NewDate(2026, time.July, 10)},
			{BaseRecord: stamped(user), SkillID: &sid, Topic: "Channels", Minutes: 30, StudiedOn: domain.NewDate(2026, time.July, 12)},
			{BaseRecord: stamped(user), Topic: "System design", Minutes: 45, StudiedOn: domain.NewDate(2026, time.July, 11)},
		},
		Goals: []domain.Goal{
			{BaseRecord: stamped(user), Title: "Get an offer", Category: "job_search", Status: "in_progress", Priority: 1, TargetDate: domain.NewDate(2026, time.July, 1)},
			{BaseRecord: stamped(user), Title: "Finish cert", Category: "certification", Status: "completed", Priority: 2, TargetDate: domain.NewDate(2026, time.June, 1), CompletedOn: domain.NewDate(2026, time.May, 20)},
		},
	}
}

func TestEveryReportHasTitleTimestampColumnsAndRows(t *testing.T) {
	s := testSnapshot()
	for _, info := range Catalog() {
		build, ok := ByID(info.ID)
		if !ok {
			t.Fatalf("catalog report %q has no builder", info.ID)
		}
		rep := build(s, now)
		if rep.Title == "" {
			t.Errorf("%s: empty title", info.ID)
		}
		if !rep.GeneratedAt.Equal(now) {
			t.Errorf("%s: GeneratedAt not stamped", info.ID)
		}
		if len(rep.Columns) < 3 {
			t.Errorf("%s: expected multiple columns, got %d", info.ID, len(rep.Columns))
		}
		if len(rep.Rows) < 1 {
			t.Errorf("%s: expected rows, got none", info.ID)
		}
		for i, row := range rep.Rows {
			if len(row) != len(rep.Columns) {
				t.Errorf("%s row %d: %d cells for %d columns", info.ID, i, len(row), len(rep.Columns))
			}
		}
	}
}

func TestByIDRejectsUnknown(t *testing.T) {
	if _, ok := ByID("nonsense"); ok {
		t.Error("unknown report id should not resolve")
	}
}

func TestApplicationPipelineContent(t *testing.T) {
	rep := BuildApplicationPipeline(testSnapshot(), now)
	if len(rep.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rep.Rows))
	}
	// Sorted by company: Globex before Initech.
	if rep.Rows[0][0] != "Globex" || rep.Rows[1][0] != "Initech, Inc." {
		t.Errorf("unexpected company order: %v", rep.Rows)
	}
	// Days since applied computed from AppliedOn (2026-07-01 → 14 days).
	if rep.Rows[1][4] != "14" {
		t.Errorf("days since applied = %q, want 14", rep.Rows[1][4])
	}
	// Missing applied date renders an em dash.
	if rep.Rows[0][3] != "—" || rep.Rows[0][4] != "—" {
		t.Errorf("missing dates should render —: %v", rep.Rows[0])
	}
}

func TestStudyActivityAggregation(t *testing.T) {
	rep := BuildStudyActivity(testSnapshot(), now)
	if len(rep.Rows) != 2 {
		t.Fatalf("expected 2 aggregate rows (Go + unlinked), got %d: %v", len(rep.Rows), rep.Rows)
	}
	// Go has 120 minutes across 2 sessions and sorts first.
	if rep.Rows[0][0] != "Go" || rep.Rows[0][2] != "2" || rep.Rows[0][3] != "120" {
		t.Errorf("Go aggregate wrong: %v", rep.Rows[0])
	}
	if rep.Rows[0][5] != "2026-07-12" {
		t.Errorf("last studied should be the max date: %v", rep.Rows[0])
	}
	if rep.Rows[1][0] != "(No linked skill)" || rep.Rows[1][3] != "45" {
		t.Errorf("unlinked aggregate wrong: %v", rep.Rows[1])
	}
}

func TestSkillGapOrdering(t *testing.T) {
	rep := BuildSkillGap(testSnapshot(), now)
	// Go (gap +2) sorts before SQL (gap 0).
	if rep.Rows[0][0] != "Go" || rep.Rows[0][4] != "+2" {
		t.Errorf("largest gap should sort first: %v", rep.Rows)
	}
	if rep.Rows[1][0] != "SQL" || rep.Rows[1][4] != "0" {
		t.Errorf("zero gap renders 0: %v", rep.Rows[1])
	}
}

func TestGoalProgressOverdueFlag(t *testing.T) {
	rep := BuildGoalProgress(testSnapshot(), now)
	if rep.Rows[0][0] != "Get an offer" || rep.Rows[0][6] != "yes" {
		t.Errorf("open past-target goal should be overdue: %v", rep.Rows[0])
	}
	if rep.Rows[1][6] != "no" {
		t.Errorf("completed goal must not be overdue: %v", rep.Rows[1])
	}
}

func TestCSVOutput(t *testing.T) {
	rep := BuildApplicationPipeline(testSnapshot(), now)
	data, err := rep.CSV()
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "Application Pipeline\n") {
		t.Errorf("CSV should start with the title: %q", text[:40])
	}
	if !strings.Contains(text, "Generated at,"+now.Format(time.RFC3339)) {
		t.Error("CSV should carry the generated-at timestamp")
	}
	if !strings.Contains(text, "Company,Position,Status") {
		t.Error("CSV should include the header row")
	}
	// Company with a comma must be quoted per RFC 4180.
	if !strings.Contains(text, `"Initech, Inc."`) {
		t.Error("CSV should quote fields containing commas")
	}
}

func TestCSVNeutralizesFormulaInjection(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{Applications: []domain.Application{
		{BaseRecord: stamped(user), Company: "=HYPERLINK(\"evil\")", Position: "@cmd", Status: "applied"},
	}}
	data, err := BuildApplicationPipeline(s, now).CSV()
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "\n=HYPERLINK") || strings.Contains(text, ",=HYPERLINK") ||
		strings.Contains(text, `"=HYPERLINK`) {
		t.Error("cell starting with = must be escaped in CSV output")
	}
	if !strings.Contains(text, "'=HYPERLINK") || !strings.Contains(text, "'@cmd") {
		t.Errorf("formula-leading cells should be quote-prefixed: %s", text)
	}
	// The skill-gap report's own "+2" cells get the same treatment: values
	// remain readable, just inert in spreadsheets.
}
