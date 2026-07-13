# DevSignal

DevSignal is a career-management platform for software engineers, built as the WGU
D424 Software Engineering Capstone project. It centralizes job applications,
recruiter contacts, interviews, skills, certifications, study sessions, personal
projects, and career goals into one web application — then turns that data into
dashboards, reports, and rule-based recommendations.

**Business problem.** Technical job seekers manage career advancement across
disconnected tools: applications in spreadsheets, contacts in email, interviews on
a calendar, study notes elsewhere. DevSignal replaces that fragmentation with a
single system that shows what has been done, what is overdue, and what to do next.

## Stack

| Layer      | Technology                                        |
|------------|---------------------------------------------------|
| Frontend   | React 19 + TypeScript (Vite)                      |
| Backend    | Go (net/http, pgx)                                |
| Database   | PostgreSQL 16                                     |
| Packaging  | Docker, docker-compose, Fly.io config, GitLab CI  |

## Repository layout

```
backend/    Go API — domain models, validation, auth, repositories, rules, reports
frontend/   React/TypeScript SPA
docs/       All project documentation (design, guides, test plan/results, rubric map)
```

## Quick start

See [docs/MAINTENANCE_GUIDE.md](docs/MAINTENANCE_GUIDE.md) for full setup. Short version:

```bash
docker compose up -d db        # PostgreSQL 16 on localhost:5432
cd backend && go run ./cmd/api # API on localhost:8080 (runs migrations at startup)
cd frontend && npm install && npm run dev   # UI on localhost:5173
```

## Repository and version

This build lives in a local git repository with full commit history
(`git log`, exported to [docs/evidence/git-history.txt](docs/evidence/git-history.txt)).
For the WGU submission, push to the course GitLab and record the URL and
submitted commit here:

- **Repository URL:** `https://gitlab.com/wgu-gitlab-environment/student-repos/<subgroup>/devsignal` *(fill in after push)*
- **Submitted version:** the commit tagged `submission-1` (`git tag submission-1 && git push --tags`)

## Documentation index

| Document | Purpose |
|----------|---------|
| [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) | Functional & nonfunctional requirements |
| [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md) | Implementation plan and task breakdown |
| [docs/DESIGN.md](docs/DESIGN.md) | Architecture, class diagram, design diagrams, ERD |
| [docs/USER_GUIDE.md](docs/USER_GUIDE.md) | End-user guide |
| [docs/MAINTENANCE_GUIDE.md](docs/MAINTENANCE_GUIDE.md) | Setup & operations guide for maintainers |
| [docs/TEST_PLAN.md](docs/TEST_PLAN.md) | Unit test plan |
| [docs/TEST_RESULTS.md](docs/TEST_RESULTS.md) | Test results and change summaries |
| [docs/SECURITY.md](docs/SECURITY.md) | Security features and rationale |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Docker / Fly.io deployment guide |
| [docs/RUBRIC_MAPPING.md](docs/RUBRIC_MAPPING.md) | Where each rubric requirement is satisfied |
