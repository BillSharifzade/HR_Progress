# HR Progress Platform — Tech Task (v1)

**Status:** draft for build
**Last updated:** 2026-04-27 (revised after parsing the source MC xlsx)
**Source of product requirements:** `DESCRIPTION.md` (read first if anything below is ambiguous)
**Source of seed data and import format:** `Матрица_Компетенций_для_Квантума.xlsx` at repo root

---

## 1. Product summary

A web platform for the HR department of a single big corporation. Tracks every worker's competencies in a Matrix of Competency (MC), compares those marks against per-`(department, grade)` requirements, and drives Plans of Individual Development (PIDs) that route workers through the company's training center (AtS) or a preceptor (mentor).

### Glossary

| Term | Meaning |
| --- | --- |
| **Worker** | An employee in the platform; has a department, grade, specialization, MC, PID(s), history. Logs in. |
| **Assessment** | A scheduled evaluation event for one worker, supervised by ≥3 assessors, who fill in marks per competency with comments, plus advantages/disadvantages. |
| **Interview** | The post-assessment phase where assessors gather more info and produce an interview total. Belongs to one Assessment. |
| **MC (Matrix of Competency)** | The consolidated `(worker, competency) → mark` data set, fed by completed assessments. |
| **Competency** | A named skill, scored 0.0–10.0. Extensible (CRUD by HR admin). |
| **Requirement** | The minimum mark and "critical" flag of one competency for one `(department, grade)` pair. |
| **PID (Plan of Individual Development)** | A page per worker containing files, links, comments, text — what the worker should do/read to develop. |
| **AtS** | The company's training center; a catalog of courses tied to competencies. |
| **Preceptor** | A specialist mentoring another worker on specific competencies; sets a predicted post-mentoring mark. Earns a "good job" badge if the worker's next assessment meets-or-exceeds the prediction. |
| **Specialization** | Free-text professional specialization, complementary to the global grade ladder. |
| **Отдел / Section** | Sub-unit inside a Department. Departments contain many sections; each worker belongs to one. v1 has no per-section requirement overrides — sections inherit their parent department's requirements. |
| **Evaluator role** | The capacity in which an assessor evaluates a worker (Section Head, Department Head, HR Analyst, HR Department Head, Direct Manager, Peer Department Head). Determined by the worker's grade per a documented HR policy and **auto-pre-filled** when scheduling, but HR can override. |

---

## 2. Scope

### In scope (v1)

- Full CRUD for users, departments, **sections (отделы)**, grades, competencies, requirements, AtS courses.
- Manual assessment scheduling (calendar).
- Assessment + interview workflow → MC writes.
- PID page per worker with files, links, comments, text.
- Preceptor assignments with target competencies, predicted marks, "good job" badge.
- Reports: gap analysis, dept/grade overviews, trend charts, exports (PDF, XLSX, CSV).
- Audit log (HR sees all; dept heads see their dept).
- Russian-language UI.
- JWT auth with role-based access control.
- Single-VM Docker Compose deployment.

### Out of scope (v1, but designed-for)

- Auto-matching of preceptors.
- Auto-PID generation from gaps.
- **Senior leadership grades (Заместитель Руководителя Департамента, Руководитель Департамента) — out of scope for MC.** They exist in the `grades` lookup but have no `competency_requirements` and don't appear in MC importers/exports. v1 covers grades 1–5 only.
- **Per-section requirement overrides.** Sections inherit their parent department's requirements. (The schema leaves room to add `section_id` to `competency_requirements` later without a breaking change.)
- Telegram-bot notifications (reserve `telegram_id` field on user; no integration yet).
- Gamified-quiz assessment (v1: simple form).
- Multi-tenant (single corporation only).
- Multi-language UI (RU only; design schema with i18n in mind but ship single-language).

### Defaults committed (flag if wrong)

- File uploads: max 100 MB, types restricted to common docs/images/video (`pdf, doc, docx, xls, xlsx, ppt, pptx, txt, md, png, jpg, jpeg, gif, webp, mp4, webm, zip`).
- Audit log retention: indefinite (no purging in v1).
- Soft-delete on users, departments, grades, competencies (so historical records keep referential integrity); hard-delete forbidden via UI.
- Password policy: min 10 chars, no other rules (small user base).
- Session: JWT access token (15 min) + refresh token in HTTP-only cookie (30 days).
- Time zone: store all timestamps in UTC; render in `Europe/Moscow` (`Europe/Moscow` is the default — configurable).
- Marks precision: `numeric(3,1)` — one decimal (4.2, 9.8 fits the spec).

---

## 3. Roles & permissions

### Roles

- **HR_ADMIN** — system god mode; only HR admins grant/revoke roles.
- **DEPT_HEAD** — full perms on workers/PIDs/assessments inside their assigned department(s).
- **ASSESSOR** — can be assigned to assessments; can communicate with worker (chat/comments on worker profile).
- **PRECEPTOR** — can be assigned to mentor a worker; can write preceptor notes and predicted marks.
- **WORKER** — implicit, every user; can view own profile, MC, PID, assessment history, course enrollments, preceptor assignments.

A user can hold any combination of roles. `WORKER` is implicit — it is not stored as a row.

### Permission matrix (high-level)

| Resource | HR_ADMIN | DEPT_HEAD (own dept) | ASSESSOR | PRECEPTOR | WORKER (self) |
| --- | --- | --- | --- | --- | --- |
| Users — create/edit/delete | ✅ | ✅ within dept | — | — | edit own non-sensitive fields |
| Roles — grant/revoke | ✅ | — | — | — | — |
| Departments / Grades — CRUD | ✅ | — | — | — | — |
| Competencies — CRUD | ✅ | — | — | — | — |
| Requirements — CRUD | ✅ | — | — | — | — |
| Assessments — schedule/create | ✅ | ✅ within dept | — | — | — |
| Assessment marks — write | ✅ | ✅ within dept | ✅ if assigned | — | — |
| Interview — fill | ✅ | ✅ within dept | ✅ if assigned | — | — |
| MC — view | ✅ | ✅ within dept | ✅ if assigned to that worker's assessment | ✅ for own students | ✅ own |
| MC — edit | ✅ | ✅ within dept | — | — | — |
| PID — view | ✅ | ✅ within dept | ✅ for that worker | ✅ for own students | ✅ own |
| PID — edit | ✅ | ✅ within dept | — | ✅ for own students | — |
| AtS courses — CRUD | ✅ | — | — | — | — |
| Course enrollments — manage | ✅ | ✅ within dept | — | — | view own |
| Preceptor assignments — manage | ✅ | ✅ within dept | — | view/work-on own | view own |
| Preceptor predictions — set | — | — | — | ✅ for own students | — |
| Calendar — manage | ✅ | ✅ within dept | view own events | view own events | view own events |
| Audit log — view | ✅ all | ✅ own dept only | — | — | — |
| Reports — run | ✅ all | ✅ own dept | — | — | — |

Implementation: a single `rbac` package returns `bool` for `(actor, action, target)` checks. All HTTP handlers delegate to it. No permission logic inlined elsewhere.

---

## 4. Domain model & ERD (text)

```
users ──┬─< user_roles
        ├─< user_history
        ├─ (department) departments
        ├─ (grade) grades
        ├─< assessments (as worker)
        ├─< assessment_assessors (as assessor)
        ├─< assessment_marks (as assessor)
        ├─< pids (as worker)
        ├─< preceptor_assignments (as worker OR preceptor)
        └─< course_enrollments (as worker)

departments ──< users
            ──< requirements

grades ──< users
       ──< requirements

competencies ──< requirements
             ──< assessment_marks
             ──< mc_entries
             ──< pid_items.competency_id
             ──< course_competencies
             ──< preceptor_target_competencies

assessments 1──1 interviews
            ──< assessment_assessors
            ──< assessment_marks
            ──< assessment_strengths
            ──< mc_entries (source)

pids ──< pid_items
     ──< (linked from preceptor_assignments / course_enrollments)

files ──< pid_items
      ──< preceptor_notes

ats_courses ──< course_competencies
            ──< course_enrollments

preceptor_assignments ──< preceptor_target_competencies
                      ──< preceptor_notes
                      ──< preceptor_rewards

calendar_events ──< calendar_event_participants
                ──? assessments (1:1 optional)

audit_logs (cross-cutting)
```

---

## 5. Database schema (PostgreSQL)

Conventions:
- All PKs are `uuid` (`gen_random_uuid()` via `pgcrypto`).
- All tables have `created_at timestamptz NOT NULL DEFAULT now()`. Mutable tables also have `updated_at timestamptz NOT NULL DEFAULT now()` (touched by trigger).
- Soft-deletable tables have `deleted_at timestamptz NULL`. Default queries filter `WHERE deleted_at IS NULL`.
- Marks: `numeric(3,1) CHECK (mark >= 0 AND mark <= 10)`.
- Money/identity values: none. Telegram id: `bigint`.
- Enums implemented as Postgres `ENUM` types where stable; otherwise `text` + `CHECK` for ones we expect to evolve.

### 5.1 Enums

```
role_kind:           HR_ADMIN | DEPT_HEAD | ASSESSOR | PRECEPTOR
competency_kind:     LK | UK | PK
                     -- ЛК Личностные (soft + cognitive), УК Управленческие, ПК Профессиональные/dept-specific
evaluator_role:      SECTION_HEAD | DEPT_HEAD | HRA | HR_DEPT_HEAD
                   | DIRECT_MANAGER | PEER_DEPT_HEAD | OTHER
                     -- Рук Отд / Рук Дпт / HR Analyst / Рук ДЧР / Manager / heads of other depts / catch-all
assessment_status:   DRAFT | SCHEDULED | ASSESSMENT_IN_PROGRESS | ASSESSMENT_DONE
                   | INTERVIEW_IN_PROGRESS | INTERVIEW_DONE | COMPLETED | CANCELLED
strength_kind:       ADVANTAGE | DISADVANTAGE
pid_status:          DRAFT | ACTIVE | COMPLETED | ARCHIVED
pid_item_kind:       TEXT | LINK | FILE | COMMENT | COURSE_REF | PRECEPTOR_REF
enrollment_status:   ENROLLED | IN_PROGRESS | COMPLETED | FAILED | CANCELLED
assignment_status:   ACTIVE | COMPLETED | CANCELLED
history_event_kind:  HIRED | PROMOTED | TRANSFERRED | EXTERNAL_EXPERIENCE | COMMENT | OTHER
calendar_event_kind: ASSESSMENT | INTERVIEW | OTHER
participant_response:PENDING | ACCEPTED | DECLINED
audit_action:        CREATE | UPDATE | DELETE | LOGIN | LOGIN_FAILED | ROLE_GRANT | ROLE_REVOKE
```

### 5.2 Tables

#### `users`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| username | text UNIQUE NOT NULL | login handle |
| email | text UNIQUE | optional |
| password_hash | text NOT NULL | argon2id |
| full_name | text NOT NULL | |
| department_id | uuid FK departments NULL | |
| section_id | uuid FK sections NULL | if set, its `department_id` must equal the user's `department_id` (enforced in service) |
| grade_id | uuid FK grades NULL | |
| specialization | text NULL | free-form |
| telegram_id | bigint NULL | reserved for future bot |
| hired_at | date NULL | |
| is_active | bool NOT NULL DEFAULT true | |
| deleted_at | timestamptz NULL | soft delete |
| created_at, updated_at | timestamptz | |

Index: `(department_id)`, `(section_id)`, `(grade_id)`, `(deleted_at)` partial.

#### `user_roles`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| user_id | uuid FK users NOT NULL | |
| role | role_kind NOT NULL | |
| scope_department_id | uuid FK departments NULL | only for DEPT_HEAD; otherwise NULL |
| granted_by | uuid FK users NULL | nullable for seed admin |
| granted_at | timestamptz NOT NULL DEFAULT now() | |

Unique: `(user_id, role, scope_department_id)`.

#### `user_history`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| user_id | uuid FK users NOT NULL | |
| event_kind | history_event_kind NOT NULL | |
| event_date | date NOT NULL | |
| title | text NOT NULL | |
| description | text NULL | |
| meta | jsonb NULL | flexible (e.g. `{from_grade, to_grade}`) |
| created_by | uuid FK users NULL | |
| created_at | timestamptz | |

Index: `(user_id, event_date DESC)`.

#### `departments`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| code | text UNIQUE NOT NULL | short code from xlsx (e.g. `ФЭД`, `ДИТ`) |
| name | text UNIQUE NOT NULL | full name (e.g. `Финансово-Экономический Департамент`) |
| description | text NULL | |
| is_active | bool NOT NULL DEFAULT true | |
| deleted_at | timestamptz NULL | |
| created_at, updated_at | timestamptz | |

#### `sections`
Sub-units (отделы) inside a department. Each worker can belong to one.
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| department_id | uuid FK departments NOT NULL | |
| name | text NOT NULL | |
| description | text NULL | |
| is_active | bool NOT NULL DEFAULT true | |
| deleted_at | timestamptz NULL | |
| created_at, updated_at | timestamptz | |

Unique: `(department_id, name)` (partial: `WHERE deleted_at IS NULL`).
Index: `(department_id)`.

#### `grades` (global)
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| name | text UNIQUE NOT NULL | e.g. `Junior`, `Middle`, `Senior`, `Lead` |
| level | int NOT NULL UNIQUE | ordering, 1..N |
| description | text NULL | |
| is_active | bool NOT NULL DEFAULT true | |
| deleted_at | timestamptz NULL | |
| created_at, updated_at | timestamptz | |

#### `competencies`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| name | text UNIQUE NOT NULL | Russian, e.g. `Коммуникация` |
| description | text NULL | |
| kind | competency_kind NOT NULL | `LK` / `UK` / `PK` |
| importance | text NULL | optional sub-grouping (e.g. `Личностно-поведенческие`, `Когнитивные`) — informational |
| is_active | bool NOT NULL DEFAULT true | |
| deleted_at | timestamptz NULL | |
| created_at, updated_at | timestamptz | |

Index: `(kind)`.

#### `competency_requirements`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| department_id | uuid FK departments NOT NULL | |
| grade_id | uuid FK grades NOT NULL | |
| competency_id | uuid FK competencies NOT NULL | |
| min_mark | numeric(3,1) NOT NULL CHECK 0..10 | |
| is_critical | bool NOT NULL DEFAULT false | |
| created_at, updated_at | timestamptz | |

Unique: `(department_id, grade_id, competency_id)`.
Indexes: `(department_id, grade_id)`, `(competency_id)`.

#### `assessments`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| worker_id | uuid FK users NOT NULL | |
| status | assessment_status NOT NULL DEFAULT DRAFT | |
| scheduled_at | timestamptz NULL | |
| location | text NULL | |
| assessment_started_at | timestamptz NULL | |
| assessment_completed_at | timestamptz NULL | |
| interview_started_at | timestamptz NULL | |
| interview_completed_at | timestamptz NULL | |
| created_by | uuid FK users NOT NULL | |
| created_at, updated_at | timestamptz | |

Index: `(worker_id, scheduled_at DESC)`.

#### `assessment_assessors`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| assessment_id | uuid FK assessments NOT NULL | |
| assessor_id | uuid FK users NOT NULL | |
| evaluator_role | evaluator_role NOT NULL DEFAULT OTHER | role in which the assessor is evaluating |
| is_lead | bool NOT NULL DEFAULT false | one per assessment |

Unique: `(assessment_id, assessor_id, evaluator_role)`.
**Pre-fill rule (service layer):** when an assessment is created, the service auto-adds evaluator slots based on the worker's grade:
- Стажёр / Специалист / Ведущий Специалист → `SECTION_HEAD`, `HRA`
- Главный Специалист → `SECTION_HEAD`, `DEPT_HEAD`, `HRA`
- Руководитель Отдела → `DIRECT_MANAGER`, `HR_DEPT_HEAD`, `HRA`

The slot is pre-filled with the inferable user (e.g. the worker's section head from `users.section_id` → that section's head). If no user can be inferred, the slot is left empty for HR to fill. HR can add/remove/swap slots at any time before the assessment starts.

#### `assessment_marks`
Per-assessor mark per competency for the worker.
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| assessment_id | uuid FK assessments NOT NULL | |
| assessor_id | uuid FK users NOT NULL | |
| competency_id | uuid FK competencies NOT NULL | |
| mark | numeric(3,1) NULL CHECK 0..10 | nullable until set |
| comment | text NULL | |
| created_at, updated_at | timestamptz | |

Unique: `(assessment_id, assessor_id, competency_id)`.

#### `assessment_strengths`
Free-form advantages/disadvantages noted by an assessor.
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| assessment_id | uuid FK assessments NOT NULL | |
| assessor_id | uuid FK users NOT NULL | |
| kind | strength_kind NOT NULL | |
| body | text NOT NULL | |
| created_at | timestamptz | |

#### `interviews`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| assessment_id | uuid FK assessments UNIQUE NOT NULL | 1:1 |
| summary | text NULL | overall narrative |
| interview_total | numeric(3,1) NULL CHECK 0..10 | aggregate score |
| conducted_at | timestamptz NULL | |
| created_by | uuid FK users NOT NULL | |
| created_at, updated_at | timestamptz | |

#### `interview_questions`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| interview_id | uuid FK interviews NOT NULL | |
| asked_by | uuid FK users NOT NULL | |
| question | text NOT NULL | |
| answer | text NULL | |
| created_at | timestamptz | |

#### `mc_entries`
Each row is one consolidated `(worker, competency)` mark produced by a completed assessment (or manual HR adjustment). Append-only — full history kept. "Current MC" = latest row per `(worker, competency)`.

| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| worker_id | uuid FK users NOT NULL | |
| competency_id | uuid FK competencies NOT NULL | |
| mark | numeric(3,1) NOT NULL CHECK 0..10 | |
| source_assessment_id | uuid FK assessments NULL | NULL = manual adjustment |
| comment | text NULL | reason if manual |
| created_by | uuid FK users NOT NULL | |
| created_at | timestamptz | |

Indexes: `(worker_id, competency_id, created_at DESC)`, `(source_assessment_id)`.

A view `mc_current` exposes the latest row per `(worker, competency)`:
```sql
CREATE VIEW mc_current AS
SELECT DISTINCT ON (worker_id, competency_id) *
FROM mc_entries
ORDER BY worker_id, competency_id, created_at DESC;
```

#### `pids`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| worker_id | uuid FK users NOT NULL | |
| title | text NOT NULL | |
| status | pid_status NOT NULL DEFAULT DRAFT | |
| deadline_at | timestamptz NULL | |
| created_by | uuid FK users NOT NULL | |
| completed_at | timestamptz NULL | |
| created_at, updated_at | timestamptz | |

Index: `(worker_id, status)`.

#### `pid_items`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| pid_id | uuid FK pids NOT NULL | |
| kind | pid_item_kind NOT NULL | |
| title | text NULL | |
| body | text NULL | for TEXT/COMMENT |
| url | text NULL | for LINK |
| file_id | uuid FK files NULL | for FILE |
| course_id | uuid FK ats_courses NULL | for COURSE_REF |
| preceptor_assignment_id | uuid FK preceptor_assignments NULL | for PRECEPTOR_REF |
| competency_id | uuid FK competencies NULL | optional gap link |
| author_id | uuid FK users NOT NULL | |
| position | int NOT NULL DEFAULT 0 | display order |
| created_at, updated_at | timestamptz | |

Index: `(pid_id, position)`.
**Check:** the right column is populated for the kind (enforced in service layer).

#### `files`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| original_name | text NOT NULL | |
| stored_path | text NOT NULL UNIQUE | relative path under `FILES_DIR` |
| size_bytes | bigint NOT NULL | |
| mime_type | text NOT NULL | |
| sha256 | text NOT NULL | for de-dup later |
| uploaded_by | uuid FK users NOT NULL | |
| created_at | timestamptz | |

#### `ats_courses`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| title | text NOT NULL | |
| description | text NULL | |
| duration_hours | numeric(6,1) NULL | |
| is_active | bool NOT NULL DEFAULT true | |
| deleted_at | timestamptz NULL | |
| created_at, updated_at | timestamptz | |

#### `course_competencies`
| col | type | notes |
| --- | --- | --- |
| course_id | uuid FK ats_courses NOT NULL | |
| competency_id | uuid FK competencies NOT NULL | |

PK: `(course_id, competency_id)`.

#### `course_enrollments`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| worker_id | uuid FK users NOT NULL | |
| course_id | uuid FK ats_courses NOT NULL | |
| pid_id | uuid FK pids NULL | optional link |
| status | enrollment_status NOT NULL DEFAULT ENROLLED | |
| enrolled_at | timestamptz NOT NULL DEFAULT now() | |
| completed_at | timestamptz NULL | |
| score | numeric(3,1) NULL | |
| comment | text NULL | |
| created_by | uuid FK users NOT NULL | |
| updated_at | timestamptz | |

Index: `(worker_id, status)`, `(course_id)`.

#### `preceptor_assignments`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| worker_id | uuid FK users NOT NULL | the student |
| preceptor_id | uuid FK users NOT NULL | |
| status | assignment_status NOT NULL DEFAULT ACTIVE | |
| pid_id | uuid FK pids NULL | optional link |
| started_at | timestamptz NOT NULL DEFAULT now() | |
| ended_at | timestamptz NULL | |
| methodology | text NULL | preceptor's plan |
| created_by | uuid FK users NOT NULL | |
| updated_at | timestamptz | |

Index: `(worker_id, status)`, `(preceptor_id, status)`.

#### `preceptor_target_competencies`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| preceptor_assignment_id | uuid FK preceptor_assignments NOT NULL | |
| competency_id | uuid FK competencies NOT NULL | |
| baseline_mark | numeric(3,1) NULL | mark at assignment start |
| predicted_mark | numeric(3,1) NOT NULL CHECK 0..10 | preceptor's prediction |
| created_at, updated_at | timestamptz | |

Unique: `(preceptor_assignment_id, competency_id)`.

#### `preceptor_notes`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| preceptor_assignment_id | uuid FK preceptor_assignments NOT NULL | |
| body | text NOT NULL | |
| file_id | uuid FK files NULL | |
| created_by | uuid FK users NOT NULL | |
| created_at | timestamptz | |

#### `preceptor_rewards`
Resolved on each assessment completion. One row per `(target_competency)` resolved.

| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| preceptor_assignment_id | uuid FK preceptor_assignments NOT NULL | |
| target_competency_id | uuid FK preceptor_target_competencies NOT NULL | |
| resolved_assessment_id | uuid FK assessments NOT NULL | the next-assessment that resolved it |
| predicted_mark | numeric(3,1) NOT NULL | snapshot |
| actual_mark | numeric(3,1) NOT NULL | snapshot from MC |
| earned | bool NOT NULL | actual ≥ predicted |
| awarded_at | timestamptz NOT NULL DEFAULT now() | |

Unique: `(target_competency_id, resolved_assessment_id)`.

#### `calendar_events`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| kind | calendar_event_kind NOT NULL | |
| title | text NOT NULL | |
| description | text NULL | |
| starts_at | timestamptz NOT NULL | |
| ends_at | timestamptz NOT NULL | |
| location | text NULL | |
| assessment_id | uuid FK assessments NULL | |
| department_id | uuid FK departments NULL | for dept-head visibility filter |
| created_by | uuid FK users NOT NULL | |
| created_at, updated_at | timestamptz | |

Index: `(starts_at)`, `(department_id)`.

#### `calendar_event_participants`
| col | type | notes |
| --- | --- | --- |
| event_id | uuid FK calendar_events NOT NULL | |
| user_id | uuid FK users NOT NULL | |
| response | participant_response NOT NULL DEFAULT PENDING | |
| responded_at | timestamptz NULL | |

PK: `(event_id, user_id)`.

#### `audit_logs`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| actor_id | uuid FK users NULL | NULL for unauthenticated events |
| action | audit_action NOT NULL | |
| entity_type | text NOT NULL | e.g. `users`, `competencies` |
| entity_id | uuid NULL | |
| before_data | jsonb NULL | |
| after_data | jsonb NULL | |
| department_scope_id | uuid FK departments NULL | for dept-head filtering |
| ip | inet NULL | |
| user_agent | text NULL | |
| created_at | timestamptz | |

Indexes: `(created_at DESC)`, `(entity_type, entity_id, created_at DESC)`, `(department_scope_id, created_at DESC)`.

#### `refresh_tokens`
| col | type | notes |
| --- | --- | --- |
| id | uuid PK | |
| user_id | uuid FK users NOT NULL | |
| token_hash | text NOT NULL UNIQUE | SHA-256 of the random token |
| issued_at | timestamptz NOT NULL DEFAULT now() | |
| expires_at | timestamptz NOT NULL | |
| revoked_at | timestamptz NULL | |
| user_agent | text NULL | |
| ip | inet NULL | |

### 5.3 Migrations

Tool: **`golang-migrate`**, SQL files under `backend/migrations/NNNN_name.up.sql` / `.down.sql`. The server runs `migrate up` on startup.

Seed migration: creates one HR_ADMIN user (`admin / admin` — must be changed on first login).

---

## 6. Backend (Go) architecture

### 6.1 Stack

| Concern | Choice |
| --- | --- |
| HTTP router | `github.com/go-chi/chi/v5` |
| DB driver | `github.com/jackc/pgx/v5` (pool) |
| Query layer | `sqlc` (generates type-safe Go from `.sql` files) |
| Migrations | `github.com/golang-migrate/migrate/v4` |
| Validation | `github.com/go-playground/validator/v10` |
| Auth | JWT via `github.com/golang-jwt/jwt/v5` |
| Password hash | `golang.org/x/crypto/argon2` (argon2id) |
| Logging | `log/slog` (stdlib, JSON handler) |
| Config | env vars loaded into a typed struct (no Viper) |
| File hashing | `crypto/sha256` |
| Excel export | `github.com/xuri/excelize/v2` |
| PDF export | `github.com/jung-kurt/gofpdf` (or `unidoc/unipdf` if needed) |
| Charts (server-side, optional) | none — frontend renders charts; backend serves data |
| Testing | stdlib + `github.com/stretchr/testify` |

### 6.2 Package layout

```
backend/
├── cmd/
│   └── server/
│       └── main.go               # wires config, db, router, starts http server
├── internal/
│   ├── config/                   # env-based config struct
│   ├── db/                       # pgx pool, migration runner
│   ├── auth/                     # jwt issue/verify, password hashing, middleware
│   ├── rbac/                     # Allow(actor, action, target) -> bool
│   ├── audit/                    # AuditWriter; helpers to record before/after
│   ├── files/                    # local-disk storage; sha256, mime sniff
│   ├── httpx/                    # router setup, error JSON, request id, recover, cors
│   ├── pagination/               # offset/limit + cursor helpers
│   ├── i18n/                     # error messages (RU)
│   ├── reports/                  # report generators (xlsx, pdf, csv)
│   └── domain/
│       ├── users/                # handler.go, service.go, repository.go (sqlc), models.go
│       ├── departments/
│       ├── grades/
│       ├── competencies/
│       ├── requirements/
│       ├── assessments/
│       ├── interviews/
│       ├── mc/
│       ├── pids/
│       ├── ats/
│       ├── enrollments/
│       ├── preceptors/
│       ├── calendar/
│       └── history/
├── migrations/                   # *.up.sql / *.down.sql
├── queries/                      # sqlc input .sql files (one per domain)
├── sqlc.yaml
├── Dockerfile
└── go.mod
```

Each domain package follows the same shape:

```go
// repository.go — thin wrapper around sqlc-generated code
type Repository struct { q *sqlcgen.Queries; db *pgxpool.Pool }
// service.go — business rules, transactions, calls audit, calls rbac
type Service struct { repo *Repository; rbac *rbac.RBAC; audit *audit.Writer; ... }
// handler.go — chi handlers, parses input (validator), calls service, writes JSON
type Handler struct { svc *Service }
// models.go — DTOs, request/response shapes (separate from sqlc row types)
```

### 6.3 Conventions

- **All writes go through services**, not handlers. Handlers only marshal/unmarshal and call `svc.X(ctx, input)`.
- **Transactions**: services that touch ≥2 tables open a `pgx.Tx`; pass through context.
- **Audit**: any service mutation calls `audit.Record(ctx, action, entity, before, after, deptScope)`. Audit is best-effort (logged on failure but doesn't roll back the tx).
- **Errors**: domain errors are typed (`ErrNotFound`, `ErrConflict`, `ErrForbidden`, `ErrValidation`). `httpx.WriteError` maps to HTTP codes (404, 409, 403, 422). 500 only for unexpected.
- **Time**: services accept `time.Time` in UTC; handlers parse RFC3339.
- **IDs**: UUIDs as strings on the wire; `uuid.UUID` internally.
- **Logging**: `slog` with request id, actor id, route. No PII in logs beyond user id.

### 6.4 Auth flow

1. `POST /api/v1/auth/login {username, password}` → verify argon2id → issue access JWT (15 min, signed HS256, claims `{sub, roles[], dept_ids[], exp, jti}`) + refresh token (random 32 bytes, store SHA-256 in `refresh_tokens`, 30 d) → set refresh token in `Set-Cookie: HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth`.
2. Frontend keeps access JWT in memory only. Re-issues on tab focus or 401 via `POST /api/v1/auth/refresh` (cookie-based).
3. `POST /api/v1/auth/logout` revokes the current refresh token.
4. Middleware `auth.Required` parses JWT, loads user (cached for request), populates `ctx`. `auth.RequireRole(role)` does role checks; finer-grained checks go through `rbac`.

### 6.5 RBAC engine

Single function:
```go
rbac.Allow(actor User, action Action, target Target) bool
```

`Action` is a typed string (e.g. `users.create`, `assessment.write_marks`, `pid.edit`).
`Target` carries optional `DepartmentID`, `WorkerID`, `AssessmentID`.

Rules implemented as a switch by action; each case checks roles against target. No dynamic policy DSL — small enough that explicit code is clearer.

Tests: a fixture-driven matrix of `(actor, action, target) -> expected`.

---

## 7. REST API surface (`/api/v1`)

All endpoints return JSON. Standard envelope:
- Success: the resource (or `{ items, total, limit, offset }` for lists).
- Error: `{ error: { code, message, details? } }`.

### Auth
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `GET /auth/me`
- `POST /auth/me/password` — change own password

### Users
- `GET /users` — list (filter by `department_id`, `grade_id`, `role`, `q`)
- `POST /users` — HR_ADMIN, DEPT_HEAD (own dept)
- `GET /users/{id}`
- `PATCH /users/{id}`
- `DELETE /users/{id}` — soft-delete
- `GET /users/{id}/history`
- `POST /users/{id}/history`
- `PATCH /users/{id}/history/{eventId}`
- `DELETE /users/{id}/history/{eventId}`

### Roles
- `GET /users/{id}/roles`
- `POST /users/{id}/roles {role, scope_department_id?}`
- `DELETE /users/{id}/roles/{roleAssignmentId}`

### Departments / Grades / Competencies
- Standard CRUD: `GET / POST / GET/{id} / PATCH/{id} / DELETE/{id}`

### Requirements
- `GET /requirements?department_id=&grade_id=&competency_id=`
- `POST /requirements`
- `PATCH /requirements/{id}`
- `DELETE /requirements/{id}`
- `GET /requirements/matrix?department_id=&grade_id=` — returns the full requirements grid for a (dept, grade)

### Assessments
- `GET /assessments?worker_id=&status=&from=&to=`
- `POST /assessments` — create draft
- `GET /assessments/{id}`
- `PATCH /assessments/{id}` — schedule / update meta / change status
- `DELETE /assessments/{id}` — only DRAFT/CANCELLED
- `POST /assessments/{id}/assessors {user_id, is_lead}`
- `DELETE /assessments/{id}/assessors/{userId}`
- `GET /assessments/{id}/marks` — all marks across assessors
- `PUT /assessments/{id}/marks/{competencyId}` — own mark only (unless HR/DH override) `{mark, comment}`
- `GET /assessments/{id}/strengths` (advantages/disadvantages)
- `POST /assessments/{id}/strengths {kind, body}`
- `DELETE /assessments/{id}/strengths/{strengthId}`
- `POST /assessments/{id}/complete` — server runs aggregation: per-competency average → writes to `mc_entries`; resolves preceptor rewards; transitions to `COMPLETED`.

### Interviews
- `GET /assessments/{id}/interview`
- `PUT /assessments/{id}/interview {summary, interview_total}`
- `GET /assessments/{id}/interview/questions`
- `POST /assessments/{id}/interview/questions {question, answer}`
- `PATCH /interviews/questions/{id}`
- `DELETE /interviews/questions/{id}`

### MC
- `GET /workers/{id}/mc` — current values per competency, joined with requirements for the worker's dept/grade (`mark`, `min_mark`, `is_critical`, `gap`)
- `GET /workers/{id}/mc/history?competency_id=` — time series
- `POST /workers/{id}/mc/adjust {competency_id, mark, comment}` — manual HR adjustment (audit logged)

### PIDs
- `GET /workers/{id}/pids`
- `POST /workers/{id}/pids {title, deadline_at}`
- `GET /pids/{id}`
- `PATCH /pids/{id}`
- `DELETE /pids/{id}`
- `POST /pids/{id}/items {kind, ...payload, competency_id?}`
- `PATCH /pids/{id}/items/{itemId}`
- `DELETE /pids/{id}/items/{itemId}`
- `POST /pids/{id}/items/reorder {ids: []}`

### Files
- `POST /files` — multipart upload → returns `{id, original_name, size_bytes, mime_type}`
- `GET /files/{id}` — streams file with original filename
- `DELETE /files/{id}` — only if not referenced

### AtS courses
- CRUD on `/courses`
- `POST /courses/{id}/competencies {competency_id}`
- `DELETE /courses/{id}/competencies/{competencyId}`

### Course enrollments
- `GET /enrollments?worker_id=&course_id=&status=`
- `POST /enrollments`
- `PATCH /enrollments/{id}`
- `DELETE /enrollments/{id}`

### Preceptor assignments
- `GET /preceptor-assignments?worker_id=&preceptor_id=&status=`
- `POST /preceptor-assignments`
- `PATCH /preceptor-assignments/{id}`
- `POST /preceptor-assignments/{id}/targets {competency_id, predicted_mark, baseline_mark?}`
- `PATCH /preceptor-assignments/{id}/targets/{targetId}` — only before first resolution
- `DELETE /preceptor-assignments/{id}/targets/{targetId}`
- `GET /preceptor-assignments/{id}/notes`
- `POST /preceptor-assignments/{id}/notes {body, file_id?}`
- `DELETE /preceptor-assignments/{id}/notes/{noteId}`
- `GET /preceptor-assignments/{id}/rewards`

### Calendar
- `GET /calendar/events?from=&to=&department_id=&user_id=`
- `POST /calendar/events`
- `PATCH /calendar/events/{id}`
- `DELETE /calendar/events/{id}`
- `POST /calendar/events/{id}/participants {user_id}`
- `DELETE /calendar/events/{id}/participants/{userId}`
- `POST /calendar/events/{id}/respond {response}` — current user RSVP

### Reports
- `GET /reports/gap-analysis?department_id=&grade_id=` — gap per competency for each worker vs requirement
- `GET /reports/department-overview?department_id=` — counts by grade, avg marks, critical-fail count
- `GET /reports/worker/{id}/profile` — full PDF-friendly profile
- `GET /reports/preceptor/{id}/effectiveness` — predictions vs actuals over time
- `GET /reports/competency/{id}/distribution` — histogram across the org
- `GET /reports/assessments/activity?from=&to=` — counts and statuses
- All accept `format=json|xlsx|pdf|csv` (where applicable). Default `json`.

### Audit log
- `GET /audit?actor_id=&entity_type=&entity_id=&from=&to=&action=&department_id=` — paginated, ordered by `created_at DESC`. DEPT_HEAD scoped to own dept.

---

## 8. Frontend (React) architecture

### 8.1 Stack

| Concern | Choice |
| --- | --- |
| Build | Vite + TypeScript |
| UI | **Ant Design 5** (full RU locale, dense tables, forms, calendar, upload — fits admin dashboards and ships fast with a professional look) |
| Charts | `@ant-design/charts` (also used by AntD ecosystem) |
| Routing | `react-router-dom` v6 |
| Server state | TanStack Query v5 |
| Client state | Zustand |
| HTTP | axios with interceptors (attach JWT, refresh on 401) |
| Forms | AntD `Form` + `zod` for shared schema validation |
| Date/time | dayjs (bundled with AntD) |
| Tables | AntD `Table` with server-side pagination |
| Markdown rendering (PID text items) | `react-markdown` + `remark-gfm` |
| File upload | AntD `Upload` |
| i18n | `react-i18next` with one `ru` bundle (designed for future expansion) |
| Lint/format | ESLint, Prettier |

### 8.2 Folder layout

```
frontend/
├── src/
│   ├── api/
│   │   ├── client.ts            # axios instance, refresh logic
│   │   ├── auth.ts
│   │   ├── users.ts
│   │   ├── ... (one per resource)
│   │   └── reports.ts
│   ├── auth/
│   │   ├── AuthProvider.tsx
│   │   ├── useAuth.ts
│   │   └── ProtectedRoute.tsx
│   ├── rbac/
│   │   └── permissions.ts       # mirror of backend rbac for UI affordances
│   ├── components/
│   │   ├── AppShell.tsx         # sider + header + content
│   │   ├── DataTable.tsx
│   │   ├── EmptyState.tsx
│   │   ├── Markdown.tsx
│   │   └── ...
│   ├── pages/
│   │   ├── login/
│   │   ├── dashboard/
│   │   ├── workers/
│   │   │   ├── WorkersList.tsx
│   │   │   ├── WorkerProfile.tsx
│   │   │   ├── WorkerMC.tsx
│   │   │   ├── WorkerPIDs.tsx
│   │   │   ├── WorkerHistory.tsx
│   │   │   └── WorkerAssessments.tsx
│   │   ├── departments/
│   │   ├── grades/
│   │   ├── competencies/
│   │   ├── requirements/        # matrix editor (dept × grade × competency)
│   │   ├── assessments/
│   │   │   ├── AssessmentsList.tsx
│   │   │   ├── AssessmentDetail.tsx
│   │   │   ├── AssessmentMarksForm.tsx
│   │   │   └── InterviewForm.tsx
│   │   ├── pids/
│   │   │   └── PIDPage.tsx
│   │   ├── ats/
│   │   ├── preceptors/
│   │   ├── calendar/
│   │   ├── reports/
│   │   ├── audit/
│   │   └── settings/
│   ├── hooks/
│   ├── lib/
│   ├── locales/ru/translation.json
│   ├── types/
│   ├── routes.tsx
│   ├── theme.ts                 # AntD ConfigProvider theme tokens
│   └── main.tsx
├── public/
├── index.html
├── vite.config.ts
├── tsconfig.json
└── package.json
```

### 8.3 UX guidelines

- Single AppShell with collapsible sidebar (AntD `Layout`). Nav grouped: «Сотрудники», «Оценка», «Развитие», «Календарь», «Отчёты», «Администрирование».
- All tables: server-side pagination, sticky header, column filters/sort.
- Forms: validation messages in Russian; `Form.Item` with required markers; submit disabled until valid.
- Uploads: drag-and-drop area; show progress; preview file name + size.
- Empty states: friendly Russian copy + primary action.
- Confirmation modals for destructive actions; `dangerouslySetInnerHTML` is forbidden (use `Markdown` component).
- AntD theme: light by default; dark mode via toggle (AntD 5 supports algorithm). Brand color: deep blue (`#1F5EFF`) — adjustable.

### 8.4 Page list (v1)

| Page | Route | Roles |
| --- | --- | --- |
| Login | `/login` | public |
| Dashboard | `/` | all |
| Workers list | `/workers` | HR/DH |
| Worker profile | `/workers/:id` | scoped |
| Worker MC | `/workers/:id/mc` | scoped |
| Worker PIDs | `/workers/:id/pids` | scoped |
| PID editor | `/pids/:id` | scoped |
| Worker history | `/workers/:id/history` | scoped |
| Departments | `/departments` | HR |
| Grades | `/grades` | HR |
| Competencies | `/competencies` | HR |
| Requirements matrix | `/requirements` | HR |
| Assessments list | `/assessments` | HR/DH |
| Assessment detail | `/assessments/:id` | scoped |
| AtS courses | `/courses` | HR |
| Course detail / enrollments | `/courses/:id` | HR |
| Preceptor assignments | `/preceptors` | HR/DH/preceptor |
| Calendar | `/calendar` | all |
| Reports | `/reports` | HR/DH |
| Audit log | `/audit` | HR/DH |
| Settings (own profile, password) | `/settings` | all |

---

## 9. Reports — detail

Each report has a JSON shape and an export shape (xlsx/pdf/csv). The frontend's Reports section composes filters and renders charts; the backend endpoint owns aggregation SQL.

### 9.1 Gap analysis
Inputs: `department_id`, `grade_id`. Output rows: `worker_id, full_name, competency_id, name, mark, min_mark, gap (= min_mark - mark, 0 if positive), is_critical_fail`. Frontend renders heatmap (workers × competencies), color-graded by gap; below-critical cells flagged red.

### 9.2 Department overview
Inputs: `department_id`. Output: count of workers by grade; average mark per competency; number of "critical fails"; top 5 weakest and strongest competencies.

### 9.3 Worker profile (PDF-ready)
Inputs: `worker_id`. Sections: identity, history timeline, current MC vs requirements, last 5 assessments, active PIDs, course enrollments, preceptor assignments. Used as an export-friendly endpoint that the PDF generator consumes.

### 9.4 Preceptor effectiveness
Inputs: `preceptor_id` (or list all). Output: rows of `(assignment, target_competency, predicted, actual, earned, resolved_at)`; aggregate accuracy %.

### 9.5 Competency distribution
Inputs: `competency_id`, optional `department_id`/`grade_id`. Output: histogram in 0.5-mark buckets; average; std dev.

### 9.6 Assessment activity
Inputs: `from`, `to`. Output: counts by status, by department, by week; list of overdue scheduled assessments.

Export libraries: `excelize` for XLSX, `gofpdf` for PDF (using a CJK-capable font like DejaVu — RU support), stdlib `encoding/csv` for CSV. Reports stream large outputs.

---

## 10. Audit log — detail

- Captured automatically by service-layer wrappers. Every mutation includes `before` (NULL for CREATE) and `after` (NULL for DELETE) JSONB snapshots.
- Login successes/failures and role grants/revokes are explicit calls to the audit writer.
- Dept-head scope: services attach `department_scope_id` based on the affected entity (e.g. user's department, assessment's worker's department, PID's worker's department). NULL means "system-wide" → only HR_ADMIN sees.
- UI: filterable table; click row → modal with diff view (before/after side-by-side, JSON diff).

---

## 11. Calendar — detail

- Backed by `calendar_events` + `calendar_event_participants`.
- Creating an Assessment also creates a linked `calendar_events` row of kind `ASSESSMENT` (auto-managed: keeping `starts_at` synced with `assessments.scheduled_at`).
- Frontend page: AntD `Calendar` for month view; switch to week/day list view; clicking event opens drawer with details and (if linked) shortcut to the assessment.
- Conflict warnings (frontend-only, advisory): when adding a participant, fetch their events overlapping the time range and show a warning if any.

---

## 12. Localization

- Backend error messages: bilingual code (`USER_NOT_FOUND`) + RU `message`. UI maps codes through `react-i18next`; backend message used as fallback.
- DB stores names in Russian as plain text (e.g. `competencies.name = "Бухгалтерский учёт"`). No `_translations` tables in v1.
- All AntD components wrapped in `<ConfigProvider locale={ruRU}>`.

---

## 13. File storage

- Local disk under `FILES_DIR` (env-configured, e.g. `/var/lib/hrprogress/files`).
- Storage layout: `{FILES_DIR}/{yyyy}/{mm}/{uuid}{ext}`.
- Access: only via authenticated `GET /files/{id}` — never exposed by static path.
- Mime sniffing on upload (`net/http.DetectContentType`); rejected types → 415.
- SHA-256 stored for future de-duplication; v1 still creates a new row even on duplicate hash.

---

## 14. Deployment

### Compose layout

```
deploy/
├── docker-compose.yml
├── Caddyfile                # TLS terminator + reverse proxy
├── .env.example
└── backups/                 # bind-mount for pg_dump output
```

Services:
- `db` — `postgres:16`, named volume `pgdata`, healthcheck `pg_isready`.
- `api` — built from `backend/Dockerfile` (multi-stage, `gcr.io/distroless/static`). Mounts `files` volume at `/data/files`. Runs migrations on startup. Exposes `:8080`.
- `web` — built from `frontend/Dockerfile`, just an Nginx serving the Vite build (`/usr/share/nginx/html`). Has a single `try_files` rule to support React Router.
- `caddy` — TLS + reverse proxy: `/api/*` → `api:8080`, everything else → `web:80`.

Backups: nightly cron container runs `pg_dump` into `./backups`, keeps last 14.

Env vars (excerpt):
```
APP_ENV=production
DATABASE_URL=postgres://hrprogress:...@db:5432/hrprogress
JWT_SECRET=<random 64 bytes>
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
FILES_DIR=/data/files
LOG_LEVEL=info
DEFAULT_TIMEZONE=Europe/Moscow
```

Healthchecks: `/api/v1/healthz` (DB ping + version).

---

## 15. Testing strategy

- Backend: unit tests for services with a real Postgres (testcontainers-go). Each domain has at least: happy path, validation error, RBAC denial. RBAC tested with a fixture matrix.
- Frontend: component tests for forms with Vitest + Testing Library; route smoke tests with Playwright (login → create user → create assessment → fill marks → complete assessment → see MC update).
- A seed script populates 3 departments, 4 grades, 20 competencies, 10 workers, 1 HR admin, 2 dept heads, 4 assessors, 2 preceptors, requirements for every (dept, grade) — used both in dev and as Playwright fixture.

---

## 16. Phased delivery plan

Each phase ends in a runnable, demoable system. No phase is "scaffolding only".

**Phase 1 — Auth + admin login**
*Tightest possible slice that lets the admin log in.*
- Repo skeleton (backend + frontend + deploy).
- Postgres + migrations 0001–0005: extensions + enums, departments + sections, grades, users + user_roles + user_history, refresh_tokens + audit_logs, seed HR_ADMIN.
- Auth API: `POST /auth/login`, `/auth/refresh`, `/auth/logout`, `GET /auth/me`, `POST /auth/me/password`.
- RBAC engine + tests.
- Audit log infrastructure (LOGIN / LOGIN_FAILED captured).
- React: AppShell, login page (RU), AuthProvider, axios refresh interceptor, empty dashboard at `/`.
- Docker Compose: `db` + `api` + `web` + `caddy`. `docker compose up` → log in as `admin / admin` (must change on first login).

**Phase 2 — MC ingestion (manual + xlsx import)**
- Migrations: competencies, competency_requirements, files.
- Backend: CRUD for departments/sections/grades/competencies/requirements; `POST /mc/import` (xlsx upload → preview → confirm) targeted at the layout of `Матрица_Компетенций_для_Квантума.xlsx`; `POST /workers/{id}/mc/adjust` for manual marks.
- Frontend: requirements matrix editor, per-worker MC view, manual adjustment form, xlsx upload with parsed-rows preview and validation errors.

**Phase 3 — Assessments & calendar**
- `assessments`, `assessment_assessors`, `assessment_marks`, `assessment_strengths`, `interviews`, `interview_questions`, `mc_entries` (append-only) and `mc_current` view.
- Workflow APIs incl. `complete` endpoint that aggregates marks → MC.
- Auto-pre-fill of evaluator slots per the §5.2 rule, with override.
- Calendar: events auto-linked to assessments.
- Frontend: assessments list, detail, assessor marks form, interview form, calendar page.
- Worker MC history chart.

**Phase 4 — PIDs, AtS, Preceptors**
- `pids`, `pid_items`, `files`, `ats_courses`, `course_competencies`, `course_enrollments`, `preceptor_assignments`, `preceptor_target_competencies`, `preceptor_notes`, `preceptor_rewards`.
- Reward resolution on assessment complete (predicted vs actual).
- Frontend: PID page (rich editor), AtS catalog + enrollments, preceptor assignment page with notes and predictions, "good job" badge surfaced on preceptor profile.
- File upload/download + ACL.

**Phase 5 — Reports & Audit UI**
- All six reports with JSON + XLSX/PDF/CSV exports.
- Reports landing page with filters and charts.
- Audit log UI with diff viewer.
- Worker history page.

**Phase 6 — Polish & hardening**
- Russian copy review.
- Theme + dark mode.
- Accessibility pass.
- Backups, healthchecks, log rotation.
- Documentation: admin handbook (RU), deployment runbook.

---

## 17. Open items / future work

- Telegram bot for notifications (reserved field already on `users`).
- Auto preceptor matching (based on competency expertise + load).
- Auto PID suggestions from gap analysis + course catalog.
- Gamified quiz format for assessments (replacing v1 "simple form").
- 2FA for HR admins.
- Multi-language UI.
- Workflow engine for approvals (e.g. PID approval by dept head before activation).

---

## 18. Decisions log (committed defaults — flag if wrong)

- Backend HTTP framework: chi.
- Query layer: sqlc (no ORM).
- Frontend UI: Ant Design 5.
- Auth: JWT access + refresh-cookie.
- File storage: local disk, served via authenticated endpoint.
- Soft delete via `deleted_at`.
- MC stored as append-only history; `mc_current` view exposes latest.
- Preceptor reward rule: actual ≥ predicted at next assessment → `earned = true` (badge).
- Single Docker Compose deployment, Caddy in front for TLS.
- File upload limit: 100 MB; allowed types listed in §2.
- Time zone: stored UTC, displayed `Europe/Moscow`.
- Departments are flat at the org level; **sections (отделы) nest inside a department**, but in v1 they only carry workers — no per-section requirements (sections inherit from their department).
- Evaluator slots are **auto-pre-filled** when scheduling an assessment, but **HR can override** before the assessment starts (§5.2 `assessment_assessors`).
- Senior leadership grades 6–7 are out of scope for MC and have no requirements rows.
- **Department seed (codes from xlsx):** ФЭД (Финансово-Экономический Департамент), ДФП (Департамент Финансового Планирования), БЮД (Бюджетный Департамент), АХД (Административно-Хозяйственный Департамент), ДЗЛ (Департамент Закупки и Логистики), ДИТ (Департамент Информационных Технологий), ДЧР (Департамент Человеческих Ресурсов).
