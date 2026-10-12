# DevSignal — User Guide

DevSignal is a career-management platform for software engineers. It centralizes
job applications, recruiter contacts, interviews, skills, certifications, study
sessions, personal projects, and career goals into one place, then turns that
data into a dashboard, reports, and rule-based recommendations.

This guide explains how to use the running application from an end-user
perspective. For installation and operation, see
[MAINTENANCE_GUIDE.md](MAINTENANCE_GUIDE.md).

## 1. Getting started

### Create an account
1. Open the application in your browser (locally, `http://localhost:5173` in
   development or `http://localhost:8080` when running the full container).
2. Click **Register**.
3. Enter your display name, email, and a password of at least 8 characters.
4. Submit the form. You are logged in automatically and taken to the Dashboard.

### Log in and out
- **Log in:** from the Login page, enter your email and password.
- **Log out:** use the **Log out** control in the navigation. Your session is a
  secure, HttpOnly cookie, so logging out fully revokes access on the server.

Your data is private to your account. Every screen shows only the records you
own; you can never see another user's data.

## 2. The Dashboard

The Dashboard is the landing page after login. It summarizes your current
career state at a glance:

- **Total applications** and a breakdown by pipeline status.
- **Upcoming interviews** scheduled in the near future.
- **Study minutes in the last 30 days**, charted by day.
- **Top skill gaps** (where current proficiency is furthest below your target).
- **Open goals, overdue goals, active projects**, and certification counts.

Below the summary, the **Recommendations** panel lists prioritized next steps
(for example, "Follow up with a contact," "Prepare for an upcoming interview,"
or "A goal is overdue"). Each recommendation links to the relevant record.

## 3. Managing records (the eight modules)

DevSignal has eight record types, each with its own page reachable from the
navigation. All of them share the same workflow:

| Page | What it tracks |
|------|----------------|
| **Applications** | Job applications and their pipeline status |
| **Contacts** | Recruiters, hiring managers, mentors, and peers |
| **Interviews** | Interview events linked to an application |
| **Skills** | Technical skills with current and target proficiency |
| **Study Sessions** | Logged study time, optionally tied to a skill |
| **Certifications** | Professional credentials and their lifecycle |
| **Projects** | Personal or portfolio projects |
| **Goals** | Career goals with priority and target date |

### Add a record
1. Open a module page (e.g., **Applications**).
2. Click **New** (or **Add**).
3. Fill in the form. Required fields are marked, and some fields offer fixed
   choices (for example, an application's status is one of *wishlist, applied,
   phone screen, interviewing, offer, accepted, rejected, withdrawn*).
4. Click **Save**. If anything is invalid, the form highlights the specific
   field and explains the problem; nothing is saved until the form is valid.

### Edit a record
1. Click a row (or its **Edit** control) in the table.
2. Change the fields in the dialog.
3. Click **Save**.

### Delete a record
1. Click the **Delete** control on a row.
2. Confirm in the dialog that appears. Deletion is permanent.

### Field rules you will encounter
- **Emails** must be well-formed (e.g., contacts).
- **URLs** must start with `http://` or `https://` (e.g., application links,
  certification and project URLs).
- **Proficiency / target** on a skill are rated 1–5.
- **Study minutes** must be between 1 and 1440 (one day).
- **Dates** that form a range are checked for order — for example, a
  certification's expiry date cannot be before the date it was earned, and a
  goal's completion date is only allowed when the goal is marked completed.

## 4. Searching

The **Search** page runs a single query across every module at once. Type a
term (a company, a person, a skill, a topic) and results are grouped by module —
Applications, Contacts, Interviews, Skills, Study Sessions, Certifications,
Projects, and Goals. Each result links straight to its module so you can open or
edit it. Groups with no matches are hidden.

## 5. Reports

The **Reports** page lets you generate and export tabular reports built from
your data. Each report has a title and a generated-at timestamp. Available
reports:

| Report | Shows |
|--------|-------|
| **Application Pipeline** | Every application with status, dates, and days in flight |
| **Interview Outcomes** | All interviews with schedule, stage, and result |
| **Study Activity** | Study time aggregated per skill |
| **Skill Gap Analysis** | Proficiency versus target for every skill |
| **Goal Progress** | Goals with status, priority, and overdue flags |

Each report can be **exported as CSV** for use in a spreadsheet. The export is
safe to open in Excel or Google Sheets — DevSignal neutralizes spreadsheet
formula injection automatically.

## 6. Tips

- Keep **contact follow-up dates** current — they drive the follow-up
  recommendations on the Dashboard.
- Set a **target proficiency** on each skill so the Skill Gap report and
  dashboard tiles can highlight where to focus.
- Log **study sessions** regularly; a gap in study activity produces a
  recommendation to get back on track.
- Mark interview **outcomes** as they resolve so the pipeline and reports stay
  accurate.

## 7. Signing out and data safety

When you are done, log out — especially on a shared computer. Your session
cookie is HttpOnly and expires automatically (72 hours by default), and logging
out revokes it immediately on the server.
