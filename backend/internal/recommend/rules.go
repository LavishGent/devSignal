package recommend

import (
	"fmt"
	"time"

	"devsignal/internal/snapshot"

	"github.com/google/uuid"
)

// Rule thresholds. Deliberately package-private: tuning them is a code
// change with tests, not runtime configuration.
const (
	contactStaleDays      = 21
	skillStaleDays        = 14
	interviewPrepDays     = 7
	applicationQuietDays  = 10
	projectInactiveDays   = 30
	certExpiryWindowDays  = 60
	studyGapDays          = 7
)

// ContactFollowUpRule flags contacts whose follow-up date has arrived and
// contacts that have gone quiet for too long.
type ContactFollowUpRule struct{ BaseRule }

// Evaluate implements Rule.
func (r ContactFollowUpRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	var out []Recommendation
	for _, c := range s.Contacts {
		switch {
		case !c.NextFollowupOn.IsZero() && c.NextFollowupOn.DaysSince(now) >= 0:
			out = append(out, r.rec(PriorityHigh,
				"Follow up with "+c.Name,
				fmt.Sprintf("A follow-up was scheduled for %s.", c.NextFollowupOn),
				c.ID))
		case c.NextFollowupOn.IsZero() && !c.LastContactedOn.IsZero() && c.LastContactedOn.DaysSince(now) >= contactStaleDays:
			out = append(out, r.rec(PriorityMedium,
				"Reconnect with "+c.Name,
				fmt.Sprintf("Last contact was %d days ago.", c.LastContactedOn.DaysSince(now)),
				c.ID))
		}
	}
	return out
}

// InterviewPrepRule flags pending interviews coming up within a week.
type InterviewPrepRule struct{ BaseRule }

// Evaluate implements Rule.
func (r InterviewPrepRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	apps := s.ApplicationByID()
	var out []Recommendation
	for _, iv := range s.Interviews {
		if iv.Outcome != "pending" || iv.ScheduledAt.Before(now) {
			continue
		}
		if iv.ScheduledAt.After(now.Add(interviewPrepDays * 24 * time.Hour)) {
			continue
		}
		where := "an application"
		if app, ok := apps[iv.ApplicationID]; ok {
			where = app.Company + " (" + app.Position + ")"
		}
		out = append(out, r.rec(PriorityHigh,
			"Prepare for your "+humanStage(iv.Stage)+" interview",
			fmt.Sprintf("Scheduled %s with %s.", iv.ScheduledAt.Format("Jan 2, 3:04 PM"), where),
			iv.ID))
	}
	return out
}

// ApplicationFollowUpRule flags early-stage applications that have gone
// quiet with no upcoming interview.
type ApplicationFollowUpRule struct{ BaseRule }

// Evaluate implements Rule.
func (r ApplicationFollowUpRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	upcoming := map[uuid.UUID]bool{}
	for _, iv := range s.Interviews {
		if iv.Outcome == "pending" && !iv.ScheduledAt.Before(now) {
			upcoming[iv.ApplicationID] = true
		}
	}
	var out []Recommendation
	for _, a := range s.Applications {
		if a.Status != "applied" && a.Status != "phone_screen" {
			continue
		}
		if upcoming[a.ID] {
			continue
		}
		quietDays := int(now.UTC().Sub(a.UpdatedAt).Hours() / 24)
		if quietDays < applicationQuietDays {
			continue
		}
		out = append(out, r.rec(PriorityMedium,
			"Follow up on your "+a.Company+" application",
			fmt.Sprintf("No activity for %d days on the %s role.", quietDays, a.Position),
			a.ID))
	}
	return out
}

// StaleSkillRule flags skills below target that have not been practiced.
type StaleSkillRule struct{ BaseRule }

// Evaluate implements Rule.
func (r StaleSkillRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	var out []Recommendation
	for _, sk := range s.Skills {
		if sk.TargetProficiency <= sk.Proficiency {
			continue
		}
		if !sk.LastPracticedOn.IsZero() && sk.LastPracticedOn.DaysSince(now) < skillStaleDays {
			continue
		}
		detail := fmt.Sprintf("You're at %d/5 with a target of %d/5", sk.Proficiency, sk.TargetProficiency)
		if sk.LastPracticedOn.IsZero() {
			detail += " and no practice logged yet."
		} else {
			detail += fmt.Sprintf("; last practiced %d days ago.", sk.LastPracticedOn.DaysSince(now))
		}
		out = append(out, r.rec(PriorityMedium, "Practice "+sk.Name, detail, sk.ID))
	}
	return out
}

// OverdueGoalRule flags open goals whose target date has passed.
type OverdueGoalRule struct{ BaseRule }

// Evaluate implements Rule.
func (r OverdueGoalRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	var out []Recommendation
	for _, g := range s.Goals {
		open := g.Status == "not_started" || g.Status == "in_progress"
		if !open || g.TargetDate.IsZero() || g.TargetDate.DaysSince(now) <= 0 {
			continue
		}
		out = append(out, r.rec(PriorityHigh,
			"Goal overdue: "+g.Title,
			fmt.Sprintf("The target date was %s (%d days ago).", g.TargetDate, g.TargetDate.DaysSince(now)),
			g.ID))
	}
	return out
}

// InactiveProjectRule flags active projects with no recent activity.
type InactiveProjectRule struct{ BaseRule }

// Evaluate implements Rule.
func (r InactiveProjectRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	var out []Recommendation
	for _, p := range s.Projects {
		if p.Status != "active" {
			continue
		}
		last := p.LastActivityOn
		if last.IsZero() {
			last = p.StartedOn
		}
		if last.IsZero() || last.DaysSince(now) < projectInactiveDays {
			continue
		}
		out = append(out, r.rec(PriorityLow,
			"Update your project "+p.Name,
			fmt.Sprintf("No recorded activity for %d days.", last.DaysSince(now)),
			p.ID))
	}
	return out
}

// CertificationExpiryRule flags earned certifications near or past expiry.
type CertificationExpiryRule struct{ BaseRule }

// Evaluate implements Rule.
func (r CertificationExpiryRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	var out []Recommendation
	for _, c := range s.Certifications {
		if c.Status != "earned" || c.ExpiresOn.IsZero() {
			continue
		}
		days := c.ExpiresOn.DaysUntil(now)
		switch {
		case days < 0:
			out = append(out, r.rec(PriorityHigh,
				"Certification expired: "+c.Name,
				fmt.Sprintf("Expired %s. Renew it or mark it expired.", c.ExpiresOn),
				c.ID))
		case days <= certExpiryWindowDays:
			out = append(out, r.rec(PriorityMedium,
				"Renew "+c.Name,
				fmt.Sprintf("Expires %s (in %d days).", c.ExpiresOn, days),
				c.ID))
		}
	}
	return out
}

// StudyGapRule nudges users who track skills but have not studied lately.
type StudyGapRule struct{ BaseRule }

// Evaluate implements Rule.
func (r StudyGapRule) Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation {
	if len(s.Skills) == 0 && len(s.StudySessions) == 0 {
		return nil // not using the study features at all
	}
	for _, ss := range s.StudySessions {
		if !ss.StudiedOn.IsZero() && ss.StudiedOn.DaysSince(now) < studyGapDays {
			return nil
		}
	}
	return []Recommendation{r.rec(PriorityMedium,
		"Log some study time",
		fmt.Sprintf("No study sessions in the last %d days.", studyGapDays),
		uuid.Nil)}
}

// humanStage renders an interview stage enum for display.
func humanStage(stage string) string {
	switch stage {
	case "phone_screen":
		return "phone screen"
	case "system_design":
		return "system design"
	default:
		return stage
	}
}
