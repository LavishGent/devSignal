package reports

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/snapshot"
)

// Builder produces one report from a snapshot. All builders are pure
// functions of (snapshot, now), which makes them trivially unit-testable.
type Builder func(s *snapshot.Snapshot, now time.Time) *Report

// Catalog lists every available report. The map key is the URL id.
func Catalog() []Info {
	return []Info{
		{ID: "application-pipeline", Title: "Application Pipeline", Description: "Every tracked application with status, dates, and days in flight."},
		{ID: "interview-outcomes", Title: "Interview Outcomes", Description: "All interviews with schedule, stage, and result."},
		{ID: "study-activity", Title: "Study Activity", Description: "Study time aggregated per skill."},
		{ID: "skill-gap", Title: "Skill Gap Analysis", Description: "Proficiency versus target for every tracked skill."},
		{ID: "goal-progress", Title: "Goal Progress", Description: "Career goals with status, priority, and overdue flags."},
	}
}

// ByID resolves a report id to its builder.
func ByID(id string) (Builder, bool) {
	switch id {
	case "application-pipeline":
		return BuildApplicationPipeline, true
	case "interview-outcomes":
		return BuildInterviewOutcomes, true
	case "study-activity":
		return BuildStudyActivity, true
	case "skill-gap":
		return BuildSkillGap, true
	case "goal-progress":
		return BuildGoalProgress, true
	default:
		return nil, false
	}
}

const dateTimeDisplay = "2006-01-02 15:04 MST"

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// BuildApplicationPipeline reports every application and its age.
func BuildApplicationPipeline(s *snapshot.Snapshot, now time.Time) *Report {
	r := &Report{
		ID:          "application-pipeline",
		Title:       "Application Pipeline",
		GeneratedAt: now,
		Columns: []Column{
			{Key: "company", Label: "Company"}, {Key: "position", Label: "Position"},
			{Key: "status", Label: "Status"}, {Key: "appliedOn", Label: "Applied On"},
			{Key: "daysSince", Label: "Days Since Applied"}, {Key: "location", Label: "Location"},
			{Key: "updatedAt", Label: "Last Updated"},
		},
	}
	apps := append([]domain.Application(nil), s.Applications...)
	sort.Slice(apps, func(i, j int) bool { return apps[i].Company < apps[j].Company })
	for _, a := range apps {
		days := "—"
		if !a.AppliedOn.IsZero() {
			days = strconv.Itoa(a.AppliedOn.DaysSince(now))
		}
		r.Rows = append(r.Rows, []string{
			a.Company, a.Position, a.Status, dash(a.AppliedOn.String()),
			days, dash(a.Location), a.UpdatedAt.Format(dateTimeDisplay),
		})
	}
	return r
}

// BuildInterviewOutcomes reports every interview with its application.
func BuildInterviewOutcomes(s *snapshot.Snapshot, now time.Time) *Report {
	apps := s.ApplicationByID()
	r := &Report{
		ID:          "interview-outcomes",
		Title:       "Interview Outcomes",
		GeneratedAt: now,
		Columns: []Column{
			{Key: "company", Label: "Company"}, {Key: "position", Label: "Position"},
			{Key: "stage", Label: "Stage"}, {Key: "scheduledAt", Label: "Scheduled At"},
			{Key: "duration", Label: "Duration (min)"}, {Key: "interviewer", Label: "Interviewer"},
			{Key: "outcome", Label: "Outcome"},
		},
	}
	ivs := append([]domain.Interview(nil), s.Interviews...)
	sort.Slice(ivs, func(i, j int) bool { return ivs[i].ScheduledAt.Before(ivs[j].ScheduledAt) })
	for _, iv := range ivs {
		company, position := "—", "—"
		if app, ok := apps[iv.ApplicationID]; ok {
			company, position = app.Company, app.Position
		}
		duration := "—"
		if iv.DurationMinutes > 0 {
			duration = strconv.Itoa(iv.DurationMinutes)
		}
		r.Rows = append(r.Rows, []string{
			company, position, iv.Stage, iv.ScheduledAt.Format(dateTimeDisplay),
			duration, dash(iv.Interviewer), iv.Outcome,
		})
	}
	return r
}

// BuildStudyActivity aggregates study time per skill.
func BuildStudyActivity(s *snapshot.Snapshot, now time.Time) *Report {
	skills := s.SkillByID()

	type agg struct {
		name, category string
		sessions       int
		minutes        int
		last           domain.Date
	}
	byKey := map[string]*agg{}
	for _, ss := range s.StudySessions {
		key, name, category := "unlinked", "(No linked skill)", "—"
		if ss.SkillID != nil {
			if sk, ok := skills[*ss.SkillID]; ok {
				key, name, category = sk.ID.String(), sk.Name, sk.Category
			}
		}
		a, ok := byKey[key]
		if !ok {
			a = &agg{name: name, category: category}
			byKey[key] = a
		}
		a.sessions++
		a.minutes += ss.Minutes
		if a.last.IsZero() || a.last.Before(ss.StudiedOn) {
			a.last = ss.StudiedOn
		}
	}

	r := &Report{
		ID:          "study-activity",
		Title:       "Study Activity by Skill",
		GeneratedAt: now,
		Columns: []Column{
			{Key: "skill", Label: "Skill"}, {Key: "category", Label: "Category"},
			{Key: "sessions", Label: "Sessions"}, {Key: "minutes", Label: "Total Minutes"},
			{Key: "hours", Label: "Total Hours"}, {Key: "lastStudied", Label: "Last Studied"},
		},
	}
	rows := make([]*agg, 0, len(byKey))
	for _, a := range byKey {
		rows = append(rows, a)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].minutes > rows[j].minutes })
	for _, a := range rows {
		r.Rows = append(r.Rows, []string{
			a.name, a.category, strconv.Itoa(a.sessions), strconv.Itoa(a.minutes),
			fmt.Sprintf("%.1f", float64(a.minutes)/60), a.last.String(),
		})
	}
	return r
}

// BuildSkillGap reports proficiency versus target for every skill.
func BuildSkillGap(s *snapshot.Snapshot, now time.Time) *Report {
	r := &Report{
		ID:          "skill-gap",
		Title:       "Skill Gap Analysis",
		GeneratedAt: now,
		Columns: []Column{
			{Key: "skill", Label: "Skill"}, {Key: "category", Label: "Category"},
			{Key: "proficiency", Label: "Proficiency"}, {Key: "target", Label: "Target"},
			{Key: "gap", Label: "Gap"}, {Key: "lastPracticed", Label: "Last Practiced"},
		},
	}
	skills := append([]domain.Skill(nil), s.Skills...)
	sort.Slice(skills, func(i, j int) bool {
		gi := skills[i].TargetProficiency - skills[i].Proficiency
		gj := skills[j].TargetProficiency - skills[j].Proficiency
		if gi != gj {
			return gi > gj
		}
		return skills[i].Name < skills[j].Name
	})
	for _, sk := range skills {
		gap := sk.TargetProficiency - sk.Proficiency
		gapStr := "0"
		if gap > 0 {
			gapStr = "+" + strconv.Itoa(gap)
		}
		r.Rows = append(r.Rows, []string{
			sk.Name, sk.Category, strconv.Itoa(sk.Proficiency), strconv.Itoa(sk.TargetProficiency),
			gapStr, dash(sk.LastPracticedOn.String()),
		})
	}
	return r
}

// BuildGoalProgress reports goals with overdue flags.
func BuildGoalProgress(s *snapshot.Snapshot, now time.Time) *Report {
	r := &Report{
		ID:          "goal-progress",
		Title:       "Goal Progress",
		GeneratedAt: now,
		Columns: []Column{
			{Key: "title", Label: "Goal"}, {Key: "category", Label: "Category"},
			{Key: "status", Label: "Status"}, {Key: "priority", Label: "Priority"},
			{Key: "targetDate", Label: "Target Date"}, {Key: "completedOn", Label: "Completed On"},
			{Key: "overdue", Label: "Overdue"},
		},
	}
	goals := append([]domain.Goal(nil), s.Goals...)
	sort.Slice(goals, func(i, j int) bool {
		if goals[i].Priority != goals[j].Priority {
			return goals[i].Priority < goals[j].Priority
		}
		return goals[i].Title < goals[j].Title
	})
	for _, g := range goals {
		open := g.Status == "not_started" || g.Status == "in_progress"
		overdue := "no"
		if open && !g.TargetDate.IsZero() && g.TargetDate.DaysSince(now) > 0 {
			overdue = "yes"
		}
		r.Rows = append(r.Rows, []string{
			g.Title, g.Category, g.Status, "P" + strconv.Itoa(g.Priority),
			dash(g.TargetDate.String()), dash(g.CompletedOn.String()), overdue,
		})
	}
	return r
}
