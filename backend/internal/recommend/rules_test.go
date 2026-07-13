package recommend

import (
	"strings"
	"testing"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/snapshot"

	"github.com/google/uuid"
)

// now is fixed so day arithmetic in every rule is deterministic.
var now = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

func date(y int, m time.Month, d int) domain.Date { return domain.NewDate(y, m, d) }

func stamped(userID uuid.UUID) domain.BaseRecord {
	var b domain.BaseRecord
	b.StampNew(userID, now.Add(-30*24*time.Hour)) // created a month ago by default
	return b
}

func TestContactFollowUpRule(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{Contacts: []domain.Contact{
		{BaseRecord: stamped(user), Name: "Due Dana", NextFollowupOn: date(2026, time.July, 14)},   // due yesterday
		{BaseRecord: stamped(user), Name: "Today Tom", NextFollowupOn: date(2026, time.July, 15)},  // due today
		{BaseRecord: stamped(user), Name: "Future Fay", NextFollowupOn: date(2026, time.August, 1)}, // not yet
		{BaseRecord: stamped(user), Name: "Quiet Quinn", LastContactedOn: date(2026, time.June, 1)}, // 44 days quiet
		{BaseRecord: stamped(user), Name: "Fresh Finn", LastContactedOn: date(2026, time.July, 10)}, // recent
		{BaseRecord: stamped(user), Name: "Blank Bo"}, // no dates at all
	}}

	recs := ContactFollowUpRule{BaseRule{"contact_follow_up", "contacts"}}.Evaluate(s, now)
	if len(recs) != 3 {
		t.Fatalf("got %d recommendations, want 3: %+v", len(recs), recs)
	}
	titles := strings.Join([]string{recs[0].Title, recs[1].Title, recs[2].Title}, "|")
	for _, want := range []string{"Due Dana", "Today Tom", "Quiet Quinn"} {
		if !strings.Contains(titles, want) {
			t.Errorf("expected a recommendation about %s, got %s", want, titles)
		}
	}
	// Scheduled follow-ups are high priority; quiet contacts are medium.
	for _, r := range recs {
		if strings.Contains(r.Title, "Quiet Quinn") && r.Priority != PriorityMedium {
			t.Error("stale contact should be medium priority")
		}
		if strings.Contains(r.Title, "Due Dana") && r.Priority != PriorityHigh {
			t.Error("due follow-up should be high priority")
		}
	}
}

func TestInterviewPrepRule(t *testing.T) {
	user := uuid.New()
	app := domain.Application{BaseRecord: stamped(user), Company: "Initech", Position: "Backend Engineer", Status: "interviewing"}
	s := &snapshot.Snapshot{
		Applications: []domain.Application{app},
		Interviews: []domain.Interview{
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "system_design", Outcome: "pending", ScheduledAt: now.Add(48 * time.Hour)},
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "final", Outcome: "pending", ScheduledAt: now.Add(10 * 24 * time.Hour)}, // too far out
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "technical", Outcome: "passed", ScheduledAt: now.Add(24 * time.Hour)},   // already decided
			{BaseRecord: stamped(user), ApplicationID: app.ID, Stage: "onsite", Outcome: "pending", ScheduledAt: now.Add(-24 * time.Hour)},    // in the past
		},
	}

	recs := InterviewPrepRule{BaseRule{"interview_prep", "interviews"}}.Evaluate(s, now)
	if len(recs) != 1 {
		t.Fatalf("got %d recommendations, want 1: %+v", len(recs), recs)
	}
	if !strings.Contains(recs[0].Title, "system design") {
		t.Errorf("title should mention the stage: %q", recs[0].Title)
	}
	if !strings.Contains(recs[0].Detail, "Initech") {
		t.Errorf("detail should mention the company: %q", recs[0].Detail)
	}
	if recs[0].Priority != PriorityHigh {
		t.Error("upcoming interview prep should be high priority")
	}
}

func TestApplicationFollowUpRule(t *testing.T) {
	user := uuid.New()
	quiet := domain.Application{BaseRecord: stamped(user), Company: "Globex", Position: "Platform Engineer", Status: "applied"}
	withInterview := domain.Application{BaseRecord: stamped(user), Company: "Initech", Position: "Backend", Status: "phone_screen"}
	fresh := domain.Application{BaseRecord: stamped(user), Company: "Hooli", Position: "Frontend", Status: "applied"}
	fresh.Touch(now.Add(-2 * 24 * time.Hour)) // recent activity
	offer := domain.Application{BaseRecord: stamped(user), Company: "Umbrella", Position: "SRE", Status: "offer"} // late stage

	s := &snapshot.Snapshot{
		Applications: []domain.Application{quiet, withInterview, fresh, offer},
		Interviews: []domain.Interview{
			{BaseRecord: stamped(user), ApplicationID: withInterview.ID, Stage: "technical", Outcome: "pending", ScheduledAt: now.Add(72 * time.Hour)},
		},
	}

	recs := ApplicationFollowUpRule{BaseRule{"application_follow_up", "applications"}}.Evaluate(s, now)
	if len(recs) != 1 {
		t.Fatalf("got %d recommendations, want 1: %+v", len(recs), recs)
	}
	if !strings.Contains(recs[0].Title, "Globex") {
		t.Errorf("only the quiet application should be flagged: %q", recs[0].Title)
	}
}

func TestStaleSkillRule(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{Skills: []domain.Skill{
		{BaseRecord: stamped(user), Name: "Go", Proficiency: 3, TargetProficiency: 5, LastPracticedOn: date(2026, time.June, 1)},   // stale, gap
		{BaseRecord: stamped(user), Name: "SQL", Proficiency: 2, TargetProficiency: 4},                                              // never practiced, gap
		{BaseRecord: stamped(user), Name: "React", Proficiency: 3, TargetProficiency: 5, LastPracticedOn: date(2026, time.July, 10)}, // fresh
		{BaseRecord: stamped(user), Name: "Bash", Proficiency: 4, TargetProficiency: 4, LastPracticedOn: date(2026, time.January, 1)}, // no gap
	}}

	recs := StaleSkillRule{BaseRule{"stale_skill", "skills"}}.Evaluate(s, now)
	if len(recs) != 2 {
		t.Fatalf("got %d recommendations, want 2: %+v", len(recs), recs)
	}
	joined := recs[0].Title + recs[1].Title
	if !strings.Contains(joined, "Go") || !strings.Contains(joined, "SQL") {
		t.Errorf("expected Go and SQL to be flagged: %v", recs)
	}
}

func TestOverdueGoalRule(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{Goals: []domain.Goal{
		{BaseRecord: stamped(user), Title: "Ship portfolio", Status: "in_progress", Priority: 1, TargetDate: date(2026, time.July, 1)}, // overdue
		{BaseRecord: stamped(user), Title: "Learn K8s", Status: "not_started", Priority: 2, TargetDate: date(2026, time.December, 1)},  // future
		{BaseRecord: stamped(user), Title: "Done deal", Status: "completed", TargetDate: date(2026, time.June, 1), CompletedOn: date(2026, time.May, 30)},
		{BaseRecord: stamped(user), Title: "Gave up", Status: "abandoned", TargetDate: date(2026, time.June, 1)},
		{BaseRecord: stamped(user), Title: "No date", Status: "in_progress"},
	}}

	recs := OverdueGoalRule{BaseRule{"overdue_goal", "goals"}}.Evaluate(s, now)
	if len(recs) != 1 {
		t.Fatalf("got %d recommendations, want 1: %+v", len(recs), recs)
	}
	if !strings.Contains(recs[0].Title, "Ship portfolio") || recs[0].Priority != PriorityHigh {
		t.Errorf("overdue open goal should be flagged high: %+v", recs[0])
	}
}

func TestInactiveProjectRule(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{Projects: []domain.Project{
		{BaseRecord: stamped(user), Name: "old-tracker", Status: "active", LastActivityOn: date(2026, time.May, 1)},   // inactive
		{BaseRecord: stamped(user), Name: "fresh-api", Status: "active", LastActivityOn: date(2026, time.July, 10)},   // active recently
		{BaseRecord: stamped(user), Name: "paused-thing", Status: "paused", LastActivityOn: date(2026, time.March, 1)}, // not active status
		{BaseRecord: stamped(user), Name: "started-long-ago", Status: "active", StartedOn: date(2026, time.April, 1)},  // falls back to startedOn
	}}

	recs := InactiveProjectRule{BaseRule{"inactive_project", "projects"}}.Evaluate(s, now)
	if len(recs) != 2 {
		t.Fatalf("got %d recommendations, want 2: %+v", len(recs), recs)
	}
	joined := recs[0].Title + recs[1].Title
	if !strings.Contains(joined, "old-tracker") || !strings.Contains(joined, "started-long-ago") {
		t.Errorf("wrong projects flagged: %v", recs)
	}
}

func TestCertificationExpiryRule(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{Certifications: []domain.Certification{
		{BaseRecord: stamped(user), Name: "AWS SAA", Status: "earned", ExpiresOn: date(2026, time.August, 1)},   // 17 days out
		{BaseRecord: stamped(user), Name: "CKA", Status: "earned", ExpiresOn: date(2026, time.July, 1)},         // already past
		{BaseRecord: stamped(user), Name: "GCP PCA", Status: "earned", ExpiresOn: date(2027, time.July, 1)},     // far future
		{BaseRecord: stamped(user), Name: "Terraform", Status: "in_progress", ExpiresOn: date(2026, time.July, 20)}, // not earned
		{BaseRecord: stamped(user), Name: "No expiry", Status: "earned"},
	}}

	recs := CertificationExpiryRule{BaseRule{"certification_expiry", "certifications"}}.Evaluate(s, now)
	if len(recs) != 2 {
		t.Fatalf("got %d recommendations, want 2: %+v", len(recs), recs)
	}
	for _, r := range recs {
		switch {
		case strings.Contains(r.Title, "CKA") && r.Priority != PriorityHigh:
			t.Error("expired cert should be high priority")
		case strings.Contains(r.Title, "AWS SAA") && r.Priority != PriorityMedium:
			t.Error("soon-to-expire cert should be medium priority")
		}
	}
}

func TestStudyGapRule(t *testing.T) {
	user := uuid.New()
	rule := StudyGapRule{BaseRule{"study_gap", "study-sessions"}}

	// Recent study: no nudge.
	active := &snapshot.Snapshot{StudySessions: []domain.StudySession{
		{BaseRecord: stamped(user), Topic: "Goroutines", Minutes: 60, StudiedOn: date(2026, time.July, 12)},
	}}
	if recs := rule.Evaluate(active, now); len(recs) != 0 {
		t.Errorf("recent study should not nudge: %+v", recs)
	}

	// Stale study history: nudge.
	stale := &snapshot.Snapshot{StudySessions: []domain.StudySession{
		{BaseRecord: stamped(user), Topic: "Old topic", Minutes: 30, StudiedOn: date(2026, time.June, 1)},
	}}
	if recs := rule.Evaluate(stale, now); len(recs) != 1 {
		t.Errorf("stale study history should nudge once: %+v", recs)
	}

	// Skills tracked but never studied: nudge.
	skillsOnly := &snapshot.Snapshot{Skills: []domain.Skill{{BaseRecord: stamped(user), Name: "Go", Proficiency: 1, TargetProficiency: 5}}}
	if recs := rule.Evaluate(skillsOnly, now); len(recs) != 1 {
		t.Errorf("skills without study should nudge: %+v", recs)
	}

	// Not using study features at all: stay quiet.
	if recs := rule.Evaluate(&snapshot.Snapshot{}, now); len(recs) != 0 {
		t.Errorf("empty snapshot should not nudge: %+v", recs)
	}
}

func TestEngineOrdersByPriorityThenModuleThenTitle(t *testing.T) {
	user := uuid.New()
	s := &snapshot.Snapshot{
		Contacts: []domain.Contact{
			{BaseRecord: stamped(user), Name: "Quiet Quinn", LastContactedOn: date(2026, time.May, 1)}, // medium
		},
		Goals: []domain.Goal{
			{BaseRecord: stamped(user), Title: "Overdue goal", Status: "in_progress", Priority: 1, TargetDate: date(2026, time.July, 1)}, // high
		},
		Projects: []domain.Project{
			{BaseRecord: stamped(user), Name: "dusty", Status: "active", LastActivityOn: date(2026, time.January, 1)}, // low
		},
	}

	recs := NewEngine().Run(s, now)
	if len(recs) != 3 {
		t.Fatalf("got %d recommendations, want 3: %+v", len(recs), recs)
	}
	if recs[0].Priority > recs[1].Priority || recs[1].Priority > recs[2].Priority {
		t.Errorf("recommendations not ordered by priority: %+v", recs)
	}
	if recs[0].Rule != "overdue_goal" || recs[2].Rule != "inactive_project" {
		t.Errorf("unexpected order: %s, %s, %s", recs[0].Rule, recs[1].Rule, recs[2].Rule)
	}
}
