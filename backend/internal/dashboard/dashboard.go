// Package dashboard aggregates a snapshot into the summary the home screen
// renders: pipeline counts, upcoming interviews, study activity, skill gaps,
// and goal health.
package dashboard

import (
	"sort"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/snapshot"
)

// Summary is the dashboard payload.
type Summary struct {
	GeneratedAt           time.Time            `json:"generatedAt"`
	TotalApplications     int                  `json:"totalApplications"`
	ApplicationsByStatus  map[string]int       `json:"applicationsByStatus"`
	UpcomingInterviews    []UpcomingInterview  `json:"upcomingInterviews"`
	StudyMinutesLast30    int                  `json:"studyMinutesLast30"`
	StudyByDay            []DayMinutes         `json:"studyByDay"`
	TopSkillGaps          []SkillGap           `json:"topSkillGaps"`
	OpenGoals             int                  `json:"openGoals"`
	OverdueGoals          int                  `json:"overdueGoals"`
	ActiveProjects        int                  `json:"activeProjects"`
	EarnedCertifications  int                  `json:"earnedCertifications"`
	ExpiringCertifications int                 `json:"expiringCertifications"`
}

// UpcomingInterview is an interview within the next two weeks, joined with
// its application for display.
type UpcomingInterview struct {
	ID          string    `json:"id"`
	Company     string    `json:"company"`
	Position    string    `json:"position"`
	Stage       string    `json:"stage"`
	ScheduledAt time.Time `json:"scheduledAt"`
}

// DayMinutes is one day of study activity for the chart.
type DayMinutes struct {
	Date    string `json:"date"` // YYYY-MM-DD
	Minutes int    `json:"minutes"`
}

// SkillGap is one skill's distance from its target.
type SkillGap struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Proficiency int    `json:"proficiency"`
	Target      int    `json:"target"`
	Gap         int    `json:"gap"`
}

const (
	interviewWindowDays = 14
	studyWindowDays     = 30
	studyChartDays      = 28
	topGaps             = 5
	certExpiryDays      = 60
)

// Build computes the summary from a snapshot. Pure function of its inputs.
func Build(s *snapshot.Snapshot, now time.Time) *Summary {
	sum := &Summary{
		GeneratedAt:          now,
		TotalApplications:    len(s.Applications),
		ApplicationsByStatus: map[string]int{},
	}

	for _, a := range s.Applications {
		sum.ApplicationsByStatus[a.Status]++
	}

	// Interviews inside the window, soonest first.
	apps := s.ApplicationByID()
	horizon := now.Add(interviewWindowDays * 24 * time.Hour)
	for _, iv := range s.Interviews {
		if iv.Outcome != "pending" || iv.ScheduledAt.Before(now) || iv.ScheduledAt.After(horizon) {
			continue
		}
		u := UpcomingInterview{ID: iv.ID.String(), Stage: iv.Stage, ScheduledAt: iv.ScheduledAt}
		if app, ok := apps[iv.ApplicationID]; ok {
			u.Company, u.Position = app.Company, app.Position
		}
		sum.UpcomingInterviews = append(sum.UpcomingInterviews, u)
	}
	sort.Slice(sum.UpcomingInterviews, func(i, j int) bool {
		return sum.UpcomingInterviews[i].ScheduledAt.Before(sum.UpcomingInterviews[j].ScheduledAt)
	})

	// Study: 30-day total plus a 28-day per-day series (zero-filled).
	perDay := map[string]int{}
	for _, ss := range s.StudySessions {
		if ss.StudiedOn.IsZero() {
			continue
		}
		age := ss.StudiedOn.DaysSince(now)
		if age >= 0 && age < studyWindowDays {
			sum.StudyMinutesLast30 += ss.Minutes
		}
		if age >= 0 && age < studyChartDays {
			perDay[ss.StudiedOn.String()] += ss.Minutes
		}
	}
	for i := studyChartDays - 1; i >= 0; i-- {
		d := domain.DateOf(now.Add(-time.Duration(i) * 24 * time.Hour))
		sum.StudyByDay = append(sum.StudyByDay, DayMinutes{Date: d.String(), Minutes: perDay[d.String()]})
	}

	// Largest skill gaps.
	gaps := []SkillGap{}
	for _, sk := range s.Skills {
		if gap := sk.TargetProficiency - sk.Proficiency; gap > 0 {
			gaps = append(gaps, SkillGap{
				ID: sk.ID.String(), Name: sk.Name,
				Proficiency: sk.Proficiency, Target: sk.TargetProficiency, Gap: gap,
			})
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Gap != gaps[j].Gap {
			return gaps[i].Gap > gaps[j].Gap
		}
		return gaps[i].Name < gaps[j].Name
	})
	if len(gaps) > topGaps {
		gaps = gaps[:topGaps]
	}
	sum.TopSkillGaps = gaps

	// Goals, projects, certifications.
	for _, g := range s.Goals {
		if g.Status == "not_started" || g.Status == "in_progress" {
			sum.OpenGoals++
			if !g.TargetDate.IsZero() && g.TargetDate.DaysSince(now) > 0 {
				sum.OverdueGoals++
			}
		}
	}
	for _, p := range s.Projects {
		if p.Status == "active" {
			sum.ActiveProjects++
		}
	}
	for _, c := range s.Certifications {
		if c.Status != "earned" {
			continue
		}
		sum.EarnedCertifications++
		if !c.ExpiresOn.IsZero() {
			if d := c.ExpiresOn.DaysUntil(now); d >= 0 && d <= certExpiryDays {
				sum.ExpiringCertifications++
			}
		}
	}

	if sum.UpcomingInterviews == nil {
		sum.UpcomingInterviews = []UpcomingInterview{}
	}
	return sum
}
