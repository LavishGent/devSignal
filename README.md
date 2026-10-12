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

This build lives in a git repository with full commit history (`git log`). For
the WGU submission it is pushed to the course GitLab:

- **Repository URL:** `https://gitlab.com/wgu-gitlab-environment/student-repos/<subgroup>/devsignal`
- **Submitted version:** the commit tagged `submission-1`

## Documentation index

| Document | Purpose | Rubric |
|----------|---------|--------|
| [docs/UML.md](docs/UML.md) | UML class diagram (domain model, OOP pillars) | C — UML diagram |
| [docs/USER_GUIDE.md](docs/USER_GUIDE.md) | Running the application from a user's perspective | C — user guide |
| [docs/MAINTENANCE_GUIDE.md](docs/MAINTENANCE_GUIDE.md) | Technical guide for building, running, and updating the code | C — technical documentation |
| [docs/TEST_PLAN.md](docs/TEST_PLAN.md) | Unit test plan and test cases | D — test plan |
| [docs/TEST_RESULTS.md](docs/TEST_RESULTS.md) | Test results and summary of changes from testing | D — test results |
