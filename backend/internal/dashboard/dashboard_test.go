package dashboard

import (
	"encoding/json"
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

func TestBuildSummary(t *testing.T) {
	user := uuid.New()
	app := domain.Application{BaseRecord: stamped(user), Company: "Initech", Position: "Backend", Status: "interviewing"}
	s := &snapshot.Snapshot{
		Applications: []domain.Application{
			app,
			{BaseRecord: stamped(user), Company: "Globex", Position: "Platform", Status: "applied"},
			{BaseRecord: stamped(user), Company: "Hooli", Position: "Frontend", Status: "applied"},
		},
		Interviews: []domain.Interview{
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "technical", Outcome: "pending", ScheduledAt: now.Add(3 * 24 * time.Hour)},
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "onsite", Outcome: "pending", ScheduledAt: now.Add(20 * 24 * time.Hour)}, // outside window
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "phone_screen", Outcome: "passed", ScheduledAt: now.Add(24 * time.Hour)}, // decided
		},
		Skills: []domain.Skill{
			{BaseRecord: stamped(user), Name: "Go", Category: "language", Proficiency: 2, TargetProficiency: 5},
			{BaseRecord: stamped(user), Name: "SQL", Category: "database", Proficiency: 3, TargetProficiency: 4},
			{BaseRecord: stamped(user), Name: "Bash", Category: "tooling", Proficiency: 4, TargetProficiency: 4}, // no gap
		},
		StudySessions: []domain.StudySession{
			{BaseRecord: stamped(user), Topic: "Recent", Minutes: 60, StudiedOn: domain.DateOf(now.Add(-24 * time.Hour))},
			{BaseRecord: stamped(user), Topic: "Also recent", Minutes: 30, StudiedOn: domain.DateOf(now.Add(-24 * time.Hour))},
			{BaseRecord: stamped(user), Topic: "Old", Minutes: 500, StudiedOn: domain.DateOf(now.Add(-45 * 24 * time.Hour))}, // outside 30d
		},
		Goals: []domain.Goal{
			{BaseRecord: stamped(user), Title: "Overdue", Status: "in_progress", Priority: 1, TargetDate: domain.NewDate(2026, time.July, 1)},
			{BaseRecord: stamped(user), Title: "Open", Status: "not_started", Priority: 2},
			{BaseRecord: stamped(user), Title: "Done", Status: "completed", Priority: 3},
		},
		Projects: []domain.Project{
			{BaseRecord: stamped(user), Name: "a", Status: "active"},
			{BaseRecord: stamped(user), Name: "b", Status: "archived"},
		},
		Certifications: []domain.Certification{
			{BaseRecord: stamped(user), Name: "AWS", Status: "earned", ExpiresOn: domain.NewDate(2026, time.August, 1)}, // expiring
			{BaseRecord: stamped(user), Name: "CKA", Status: "earned"},
			{BaseRecord: stamped(user), Name: "GCP", Status: "planned"},
		},
	}

	sum := Build(s, now)

	if sum.TotalApplications != 3 || sum.ApplicationsByStatus["applied"] != 2 || sum.ApplicationsByStatus["interviewing"] != 1 {
		t.Errorf("application counts wrong: %+v", sum.ApplicationsByStatus)
	}
	if len(sum.UpcomingInterviews) != 1 || sum.UpcomingInterviews[0].Company != "Initech" {
		t.Errorf("upcoming interviews wrong: %+v", sum.UpcomingInterviews)
	}
	if sum.StudyMinutesLast30 != 90 {
		t.Errorf("StudyMinutesLast30 = %d, want 90", sum.StudyMinutesLast30)
	}
	if len(sum.StudyByDay) != 28 {
		t.Fatalf("StudyByDay should have 28 zero-filled days, got %d", len(sum.StudyByDay))
	}
	if sum.StudyByDay[27].Date != domain.DateOf(now).String() {
		t.Errorf("last chart day should be today, got %s", sum.StudyByDay[27].Date)
	}
	if sum.StudyByDay[26].Minutes != 90 {
		t.Errorf("yesterday should aggregate to 90 minutes, got %d", sum.StudyByDay[26].Minutes)
	}
	if len(sum.TopSkillGaps) != 2 || sum.TopSkillGaps[0].Name != "Go" || sum.TopSkillGaps[0].Gap != 3 {
		t.Errorf("skill gaps wrong: %+v", sum.TopSkillGaps)
	}
	if sum.OpenGoals != 2 || sum.OverdueGoals != 1 {
		t.Errorf("goal counts wrong: open=%d overdue=%d", sum.OpenGoals, sum.OverdueGoals)
	}
	if sum.ActiveProjects != 1 {
		t.Errorf("ActiveProjects = %d, want 1", sum.ActiveProjects)
	}
	if sum.EarnedCertifications != 2 || sum.ExpiringCertifications != 1 {
		t.Errorf("cert counts wrong: earned=%d expiring=%d", sum.EarnedCertifications, sum.ExpiringCertifications)
	}
}

func TestBuildEmptySnapshotIsSafe(t *testing.T) {
	sum := Build(&snapshot.Snapshot{}, now)
	if sum.TotalApplications != 0 || len(sum.StudyByDay) != 28 {
		t.Error("empty snapshot should produce a zeroed, chartable summary")
	}
	if sum.UpcomingInterviews == nil {
		t.Error("UpcomingInterviews should be an empty slice, not null JSON")
	}
}

// Empty summary encodes its lists as arrays, not null.
func TestEmptySummaryEncodesArraysNotNull(t *testing.T) {
	raw, err := json.Marshal(Build(&snapshot.Snapshot{}, now))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"upcomingInterviews", "studyByDay", "topSkillGaps"} {
		if !strings.HasPrefix(string(decoded[key]), "[") {
			t.Errorf("%s should encode as a JSON array, got %s", key, decoded[key])
		}
	}
}
