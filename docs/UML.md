# DevSignal — UML Class Diagram

This class diagram documents DevSignal's domain model and the three
object-oriented pillars the application is built on. It is rendered with
[Mermaid](https://mermaid.js.org/); GitLab and GitHub render it inline, and it
can also be pasted into the [Mermaid Live Editor](https://mermaid.live) to
export a PNG/SVG.

- **Encapsulation** — `Password` and `Date` keep their state unexported and
  expose only safe operations. `User.PasswordHash` is excluded from JSON so a
  credential can never leak through an API response.
- **Inheritance** — every career record embeds `BaseRecord` (struct embedding
  is Go's inheritance mechanism), inheriting identity, ownership, and
  audit-timestamp behavior from one place.
- **Polymorphism** — every entity implements the `Validator` interface, so a
  single handler path (`BaseHandler.validate`) validates any record through one
  contract. Persistence is equally polymorphic: the `Repository` interfaces have
  both a PostgreSQL implementation (production) and an in-memory implementation
  (tests).

## Domain class diagram

```mermaid
classDiagram
    direction TB

    class Validator {
        <<interface>>
        +Validate() Errors
    }

    class BaseRecord {
        +UUID ID
        +UUID UserID
        +time CreatedAt
        +time UpdatedAt
        +StampNew(userID, now)
        +Touch(now)
        +IsOwnedBy(userID) bool
    }

    class Application {
        +string Company
        +string Position
        +string Status
        +Date AppliedOn
        +string WorkMode
        +int SalaryMin
        +int SalaryMax
        +Validate() Errors
    }

    class Contact {
        +string Name
        +string Company
        +string Relationship
        +string Email
        +Date LastContactedOn
        +Date NextFollowupOn
        +Validate() Errors
    }

    class Interview {
        +UUID ApplicationID
        +string Stage
        +time ScheduledAt
        +int DurationMinutes
        +string Outcome
        +Validate() Errors
    }

    class Skill {
        +string Name
        +string Category
        +int Proficiency
        +int TargetProficiency
        +Date LastPracticedOn
        +Validate() Errors
    }

    class StudySession {
        +UUID SkillID
        +string Topic
        +int Minutes
        +Date StudiedOn
        +Validate() Errors
    }

    class Certification {
        +string Name
        +string Issuer
        +string Status
        +Date EarnedOn
        +Date ExpiresOn
        +Validate() Errors
    }

    class Project {
        +string Name
        +string Status
        +string TechStack
        +string RepoURL
        +Date LastActivityOn
        +Validate() Errors
    }

    class Goal {
        +string Title
        +string Category
        +string Status
        +int Priority
        +Date TargetDate
        +Validate() Errors
    }

    class User {
        +UUID ID
        +string Email
        +string DisplayName
        -string PasswordHash
        +Validate() Errors
    }

    class Password {
        -byte[] hash
        +Matches(plain) bool
        +Hash() string
    }

    %% Inheritance: every career record embeds BaseRecord
    BaseRecord <|-- Application
    BaseRecord <|-- Contact
    BaseRecord <|-- Interview
    BaseRecord <|-- Skill
    BaseRecord <|-- StudySession
    BaseRecord <|-- Certification
    BaseRecord <|-- Project
    BaseRecord <|-- Goal

    %% Polymorphism: every entity satisfies Validator
    Validator <|.. Application
    Validator <|.. Contact
    Validator <|.. Interview
    Validator <|.. Skill
    Validator <|.. StudySession
    Validator <|.. Certification
    Validator <|.. Project
    Validator <|.. Goal
    Validator <|.. User

    %% Associations (ownership and record links)
    User "1" o-- "many" Application : owns
    User "1" o-- "many" Contact : owns
    User "1" o-- "many" Skill : owns
    User "1" o-- "many" Goal : owns
    User "1" --> "1" Password : authenticates with
    Application "1" o-- "many" Interview : has
    Skill "1" o-- "many" StudySession : practiced in
```

## Repository polymorphism

Handlers and services depend only on the `repo` interfaces, never on a concrete
database. The same code runs against PostgreSQL in production and against
in-memory stores in unit tests.

```mermaid
classDiagram
    direction LR

    class ApplicationRepo {
        <<interface>>
        +Create(ctx, app) error
        +Get(ctx, userID, id) Application
        +List(ctx, userID, filter) Application[]
        +Update(ctx, app) error
        +Delete(ctx, userID, id) error
    }

    class PostgresApplications {
        -pool
        +Create() error
        +List() Application[]
    }

    class MemoryApplications {
        -map data
        +Create() error
        +List() Application[]
    }

    ApplicationRepo <|.. PostgresApplications
    ApplicationRepo <|.. MemoryApplications
```

## Recommendation-engine polymorphism

The rule-based recommendation engine treats every rule through the `Rule`
interface. Rules share identity via the embedded `BaseRule` (inheritance) and
keep their tuning thresholds unexported (encapsulation). Adding a rule requires
no change to the engine.

```mermaid
classDiagram
    direction TB

    class Rule {
        <<interface>>
        +Name() string
        +Evaluate(snapshot, now) Recommendation[]
    }

    class BaseRule {
        -string name
        -string module
        +Name() string
        +rec(priority, title, detail, id) Recommendation
    }

    class Engine {
        -Rule[] rules
        +Run(snapshot, now) Recommendation[]
    }

    BaseRule <|-- ContactFollowUpRule
    BaseRule <|-- InterviewPrepRule
    BaseRule <|-- ApplicationFollowUpRule
    BaseRule <|-- StaleSkillRule
    BaseRule <|-- OverdueGoalRule
    BaseRule <|-- InactiveProjectRule
    BaseRule <|-- CertificationExpiryRule
    BaseRule <|-- StudyGapRule

    Rule <|.. ContactFollowUpRule
    Rule <|.. InterviewPrepRule
    Rule <|.. ApplicationFollowUpRule
    Rule <|.. StaleSkillRule
    Rule <|.. OverdueGoalRule
    Rule <|.. InactiveProjectRule
    Rule <|.. CertificationExpiryRule
    Rule <|.. StudyGapRule

    Engine o-- "many" Rule : runs
```

## Source of truth

The diagram is generated from the code in `backend/internal/domain/`,
`backend/internal/repo/`, and `backend/internal/recommend/`. When an entity or
rule changes, update this file to match.
