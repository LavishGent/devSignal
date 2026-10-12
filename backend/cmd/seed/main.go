// Command seed populates the database with a realistic demo account for
// screencasts and manual walkthroughs. It is idempotent: on each run it
// deletes the existing demo user (cascading to all their records) and
// recreates the full data set with dates relative to "now", so the dashboard,
// reports, and recommendations always look alive.
//
// Usage:
//
//	go run ./cmd/seed                 # seeds demo@devsignal.app / demopass123
//	go run ./cmd/seed -email a@b.com -password secret12
//
// It reuses the same configuration, repositories, and validation as the API,
// so seeded records are guaranteed to satisfy every domain rule.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"devsignal/internal/auth"
	"devsignal/internal/config"
	"devsignal/internal/database"
	"devsignal/internal/domain"
	"devsignal/internal/repo/postgres"
	"devsignal/migrations"

	"github.com/google/uuid"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	email := flag.String("email", "demo@devsignal.app", "demo account email")
	password := flag.String("password", "demopass123", "demo account password (min 8 chars)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, migrations.FS, logger); err != nil {
		return err
	}

	normalized := domain.NormalizeEmail(*email)

	// Idempotent reset: deleting the user cascades to every owned record
	// (ON DELETE CASCADE), so a re-run starts from a clean demo account
	// without touching any other user's data.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, normalized); err != nil {
		return fmt.Errorf("clearing existing demo user: %w", err)
	}

	store := postgres.NewStore(pool)
	s := &seeder{ctx: ctx, store: store, now: time.Now().UTC()}

	user, err := s.createUser(normalized, *password, cfg.BcryptCost)
	if err != nil {
		return err
	}
	s.userID = user.ID

	if err := s.seedAll(); err != nil {
		return err
	}

	logger.Info("demo data seeded",
		"email", normalized,
		"password", *password,
		"applications", s.counts["applications"],
		"contacts", s.counts["contacts"],
		"interviews", s.counts["interviews"],
		"skills", s.counts["skills"],
		"study_sessions", s.counts["study_sessions"],
		"certifications", s.counts["certifications"],
		"projects", s.counts["projects"],
		"goals", s.counts["goals"],
	)
	fmt.Printf("\n  Demo account ready — log in with:\n    email:    %s\n    password: %s\n\n", normalized, *password)
	return nil
}

type seeder struct {
	ctx    context.Context
	store  *postgres.Store
	userID uuid.UUID
	now    time.Time
	counts map[string]int
}

func (s *seeder) createUser(email, password string, cost int) (*domain.User, error) {
	pw, err := auth.HashPassword(password, cost)
	if err != nil {
		return nil, fmt.Errorf("hashing demo password: %w", err)
	}
	u := &domain.User{
		ID:           domain.NewID(),
		Email:        email,
		DisplayName:  "Demo Engineer",
		PasswordHash: pw.Hash(),
		CreatedAt:    s.now,
		UpdatedAt:    s.now,
	}
	if errs := u.Validate(); !errs.Ok() {
		return nil, fmt.Errorf("invalid demo user: %v", errs)
	}
	if err := s.store.Users.Create(s.ctx, u); err != nil {
		return nil, fmt.Errorf("creating demo user: %w", err)
	}
	return u, nil
}

// days returns a Date offset from today by d days (negative = past).
func (s *seeder) days(d int) domain.Date { return domain.DateOf(s.now.AddDate(0, 0, d)) }

// at returns a timestamp offset from now by d days at the given hour (UTC).
func (s *seeder) at(d, hour int) time.Time {
	base := s.now.AddDate(0, 0, d)
	return time.Date(base.Year(), base.Month(), base.Day(), hour, 0, 0, 0, time.UTC)
}

func ptr[T any](v T) *T { return &v }

func (s *seeder) seedAll() error {
	s.counts = map[string]int{}

	appIDs, err := s.seedApplications()
	if err != nil {
		return err
	}
	if err := s.seedContacts(); err != nil {
		return err
	}
	if err := s.seedInterviews(appIDs); err != nil {
		return err
	}
	skillIDs, err := s.seedSkills()
	if err != nil {
		return err
	}
	if err := s.seedStudySessions(skillIDs); err != nil {
		return err
	}
	if err := s.seedCertifications(); err != nil {
		return err
	}
	if err := s.seedProjects(); err != nil {
		return err
	}
	return s.seedGoals()
}

func (s *seeder) stamp(r interface{ StampNew(uuid.UUID, time.Time) }) {
	r.StampNew(s.userID, s.now)
}

func (s *seeder) seedApplications() (map[string]uuid.UUID, error) {
	apps := []domain.Application{
		{Company: "Stripe", Position: "Senior Backend Engineer", Status: "interviewing", AppliedOn: s.days(-18), Location: "Remote (US)", WorkMode: "remote", URL: "https://stripe.com/jobs/123", SalaryMin: ptr(180000), SalaryMax: ptr(230000), Notes: "Referred by a former colleague. Strong Go role."},
		{Company: "Datadog", Position: "Platform Engineer", Status: "phone_screen", AppliedOn: s.days(-9), Location: "New York, NY", WorkMode: "hybrid", URL: "https://datadoghq.com/careers/456", SalaryMin: ptr(170000), SalaryMax: ptr(210000), Notes: "Recruiter screen went well."},
		{Company: "Vercel", Position: "Full Stack Engineer", Status: "applied", AppliedOn: s.days(-3), Location: "Remote", WorkMode: "remote", URL: "https://vercel.com/careers/789", Notes: "Applied through the careers page."},
		{Company: "Cloudflare", Position: "Systems Engineer", Status: "offer", AppliedOn: s.days(-40), Location: "Austin, TX", WorkMode: "hybrid", SalaryMin: ptr(175000), SalaryMax: ptr(220000), Notes: "Offer received — negotiating start date."},
		{Company: "HashiCorp", Position: "Software Engineer, Terraform", Status: "rejected", AppliedOn: s.days(-55), Location: "Remote", WorkMode: "remote", Notes: "Did not move past the take-home."},
		{Company: "Figma", Position: "Backend Engineer", Status: "wishlist", WorkMode: "onsite", Location: "San Francisco, CA", Notes: "Dream role — need to tailor resume first."},
	}
	ids := map[string]uuid.UUID{}
	for i := range apps {
		a := apps[i]
		if errs := a.Validate(); !errs.Ok() {
			return nil, fmt.Errorf("invalid application %q: %v", a.Company, errs)
		}
		s.stamp(&a)
		if err := s.store.Applications.Create(s.ctx, &a); err != nil {
			return nil, fmt.Errorf("creating application %q: %w", a.Company, err)
		}
		ids[a.Company] = a.ID
		s.counts["applications"]++
	}
	return ids, nil
}

func (s *seeder) seedContacts() error {
	contacts := []domain.Contact{
		{Name: "Priya Shah", Company: "Stripe", Role: "Engineering Manager", Relationship: "hiring_manager", Email: "priya@example.com", Phone: "+1 555 0100", LinkedinURL: "https://linkedin.com/in/priya", LastContactedOn: s.days(-5), NextFollowupOn: s.days(-1), Notes: "Owes me a scheduling update — follow up is due."},
		{Name: "Marcus Lee", Company: "Datadog", Role: "Technical Recruiter", Relationship: "recruiter", Email: "marcus@example.com", LinkedinURL: "https://linkedin.com/in/marcuslee", LastContactedOn: s.days(-9), NextFollowupOn: s.days(2), Notes: "Will send next-round details."},
		{Name: "Dana Wu", Company: "Independent", Role: "Staff Engineer", Relationship: "mentor", Email: "dana@example.com", LastContactedOn: s.days(-30), Notes: "Monthly career check-in. Overdue to reconnect."},
		{Name: "Sam Rivera", Company: "Cloudflare", Role: "Senior Engineer", Relationship: "engineer", Email: "sam@example.com", LinkedinURL: "https://linkedin.com/in/samrivera", LastContactedOn: s.days(-12), Notes: "Did the referral. Thank-you note sent."},
		{Name: "Jordan Kim", Company: "Vercel", Role: "Developer", Relationship: "peer", Email: "jordan@example.com", LastContactedOn: s.days(-20), Notes: "Met at a meetup."},
	}
	for i := range contacts {
		c := contacts[i]
		if errs := c.Validate(); !errs.Ok() {
			return fmt.Errorf("invalid contact %q: %v", c.Name, errs)
		}
		s.stamp(&c)
		if err := s.store.Contacts.Create(s.ctx, &c); err != nil {
			return fmt.Errorf("creating contact %q: %w", c.Name, err)
		}
		s.counts["contacts"]++
	}
	return nil
}

func (s *seeder) seedInterviews(appIDs map[string]uuid.UUID) error {
	ivs := []domain.Interview{
		{ApplicationID: appIDs["Stripe"], Stage: "technical", ScheduledAt: s.at(3, 14), DurationMinutes: 60, Interviewer: "Priya Shah", Location: "Google Meet", Outcome: "pending", PrepNotes: "Review Go concurrency and system design for payments."},
		{ApplicationID: appIDs["Stripe"], Stage: "phone_screen", ScheduledAt: s.at(-7, 11), DurationMinutes: 45, Interviewer: "Recruiting", Location: "Phone", Outcome: "passed", Feedback: "Advanced to the technical round."},
		{ApplicationID: appIDs["Datadog"], Stage: "phone_screen", ScheduledAt: s.at(5, 10), DurationMinutes: 30, Interviewer: "Marcus Lee", Location: "Zoom", Outcome: "pending", PrepNotes: "Behavioral questions; prepare STAR stories."},
		{ApplicationID: appIDs["Cloudflare"], Stage: "onsite", ScheduledAt: s.at(-14, 9), DurationMinutes: 240, Interviewer: "Panel", Location: "Austin office", Outcome: "passed", Feedback: "Strong across the board — led to an offer."},
	}
	for i := range ivs {
		iv := ivs[i]
		if iv.ApplicationID == uuid.Nil {
			continue // skip if the parent application was not seeded
		}
		if errs := iv.Validate(); !errs.Ok() {
			return fmt.Errorf("invalid interview %q: %v", iv.Stage, errs)
		}
		s.stamp(&iv)
		if err := s.store.Interviews.Create(s.ctx, &iv); err != nil {
			return fmt.Errorf("creating interview %q: %w", iv.Stage, err)
		}
		s.counts["interviews"]++
	}
	return nil
}

func (s *seeder) seedSkills() (map[string]uuid.UUID, error) {
	skills := []domain.Skill{
		{Name: "Go", Category: "language", Proficiency: 4, TargetProficiency: 5, LastPracticedOn: s.days(-2), Notes: "Primary backend language."},
		{Name: "PostgreSQL", Category: "database", Proficiency: 3, TargetProficiency: 5, LastPracticedOn: s.days(-6), Notes: "Indexing and query tuning to improve."},
		{Name: "System Design", Category: "system_design", Proficiency: 3, TargetProficiency: 5, LastPracticedOn: s.days(-20), Notes: "Stale — needs practice before onsites."},
		{Name: "React", Category: "framework", Proficiency: 4, TargetProficiency: 4, LastPracticedOn: s.days(-4)},
		{Name: "Kubernetes", Category: "cloud", Proficiency: 2, TargetProficiency: 4, LastPracticedOn: s.days(-25), Notes: "Big gap — prioritize."},
		{Name: "TypeScript", Category: "language", Proficiency: 4, TargetProficiency: 5, LastPracticedOn: s.days(-4)},
		{Name: "Communication", Category: "soft_skill", Proficiency: 4, TargetProficiency: 5, LastPracticedOn: s.days(-10)},
	}
	ids := map[string]uuid.UUID{}
	for i := range skills {
		sk := skills[i]
		if errs := sk.Validate(); !errs.Ok() {
			return nil, fmt.Errorf("invalid skill %q: %v", sk.Name, errs)
		}
		s.stamp(&sk)
		if err := s.store.Skills.Create(s.ctx, &sk); err != nil {
			return nil, fmt.Errorf("creating skill %q: %w", sk.Name, err)
		}
		ids[sk.Name] = sk.ID
		s.counts["skills"]++
	}
	return ids, nil
}

func (s *seeder) seedStudySessions(skillIDs map[string]uuid.UUID) error {
	type plan struct {
		skill   string
		topic   string
		minutes int
		daysAgo int
		res     string
	}
	plans := []plan{
		{"Go", "Goroutines and channels deep dive", 90, 2, "https://go.dev/blog/pipelines"},
		{"Go", "Error handling patterns", 45, 5, ""},
		{"PostgreSQL", "EXPLAIN ANALYZE and index strategy", 60, 6, "https://use-the-index-luke.com"},
		{"System Design", "Designing a rate limiter", 75, 8, ""},
		{"System Design", "Caching strategies", 60, 20, ""},
		{"Kubernetes", "Pods, deployments, services", 50, 12, "https://kubernetes.io/docs"},
		{"React", "Hooks and memoization", 40, 4, ""},
		{"TypeScript", "Generics and utility types", 55, 4, ""},
		{"Go", "Testing and table-driven tests", 65, 1, ""},
		{"PostgreSQL", "Transactions and isolation levels", 50, 14, ""},
		{"Communication", "Mock behavioral interview", 30, 10, ""},
		{"Kubernetes", "Helm basics", 45, 24, ""},
	}
	for _, p := range plans {
		ss := domain.StudySession{
			Topic:     p.topic,
			Minutes:   p.minutes,
			StudiedOn: s.days(-p.daysAgo),
			Resource:  p.res,
		}
		if id, ok := skillIDs[p.skill]; ok {
			ss.SkillID = ptr(id)
		}
		if errs := ss.Validate(); !errs.Ok() {
			return fmt.Errorf("invalid study session %q: %v", p.topic, errs)
		}
		s.stamp(&ss)
		if err := s.store.StudySessions.Create(s.ctx, &ss); err != nil {
			return fmt.Errorf("creating study session %q: %w", p.topic, err)
		}
		s.counts["study_sessions"]++
	}
	return nil
}

func (s *seeder) seedCertifications() error {
	certs := []domain.Certification{
		{Name: "AWS Certified Solutions Architect – Associate", Issuer: "Amazon Web Services", Status: "earned", EarnedOn: s.days(-700), ExpiresOn: s.days(40), CredentialID: "AWS-SAA-12345", CredentialURL: "https://aws.amazon.com/verification", Notes: "Expiring soon — plan the recertification."},
		{Name: "Certified Kubernetes Administrator (CKA)", Issuer: "The Linux Foundation", Status: "in_progress", Notes: "Targeting the exam next quarter."},
		{Name: "HashiCorp Certified: Terraform Associate", Issuer: "HashiCorp", Status: "planned", Notes: "After CKA."},
	}
	for i := range certs {
		c := certs[i]
		if errs := c.Validate(); !errs.Ok() {
			return fmt.Errorf("invalid certification %q: %v", c.Name, errs)
		}
		s.stamp(&c)
		if err := s.store.Certifications.Create(s.ctx, &c); err != nil {
			return fmt.Errorf("creating certification %q: %w", c.Name, err)
		}
		s.counts["certifications"]++
	}
	return nil
}

func (s *seeder) seedProjects() error {
	projects := []domain.Project{
		{Name: "DevSignal", Description: "Career-management platform for software engineers.", Status: "active", TechStack: "Go, React, PostgreSQL", RepoURL: "https://gitlab.com/example/devsignal", DemoURL: "https://devsignal.fly.dev", StartedOn: s.days(-60), LastActivityOn: s.days(-1), Notes: "This capstone project."},
		{Name: "Distributed Cache", Description: "A toy Redis-like cache to practice systems programming.", Status: "paused", TechStack: "Go", RepoURL: "https://github.com/example/dcache", StartedOn: s.days(-120), LastActivityOn: s.days(-45), Notes: "Inactive — resume after job search."},
		{Name: "Portfolio Site", Description: "Personal site and blog.", Status: "completed", TechStack: "TypeScript, Astro", RepoURL: "https://github.com/example/portfolio", DemoURL: "https://example.dev", StartedOn: s.days(-200), LastActivityOn: s.days(-30)},
	}
	for i := range projects {
		p := projects[i]
		if errs := p.Validate(); !errs.Ok() {
			return fmt.Errorf("invalid project %q: %v", p.Name, errs)
		}
		s.stamp(&p)
		if err := s.store.Projects.Create(s.ctx, &p); err != nil {
			return fmt.Errorf("creating project %q: %w", p.Name, err)
		}
		s.counts["projects"]++
	}
	return nil
}

func (s *seeder) seedGoals() error {
	goals := []domain.Goal{
		{Title: "Land a senior backend role", Description: "Accept an offer at a top-tier company.", Category: "job_search", Status: "in_progress", Priority: 1, TargetDate: s.days(30), Notes: "Cloudflare offer in hand; still interviewing at Stripe."},
		{Title: "Pass the CKA exam", Description: "Earn the Certified Kubernetes Administrator cert.", Category: "certification", Status: "in_progress", Priority: 2, TargetDate: s.days(75)},
		{Title: "Reach Go proficiency level 5", Description: "Deepen concurrency and performance skills.", Category: "skill", Status: "in_progress", Priority: 2, TargetDate: s.days(-5), Notes: "Overdue — push to finish."},
		{Title: "Publish 3 portfolio projects", Description: "Ship and document three projects.", Category: "project", Status: "completed", Priority: 3, TargetDate: s.days(-10), CompletedOn: s.days(-12)},
	}
	for i := range goals {
		g := goals[i]
		if errs := g.Validate(); !errs.Ok() {
			return fmt.Errorf("invalid goal %q: %v", g.Title, errs)
		}
		s.stamp(&g)
		if err := s.store.Goals.Create(s.ctx, &g); err != nil {
			return fmt.Errorf("creating goal %q: %w", g.Title, err)
		}
		s.counts["goals"]++
	}
	return nil
}
