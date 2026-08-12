# Spec — Digital profile questionnaire import

Status: **draft, awaiting approval**. Date: 2026-08-12.
Source file: `Анкета формирования цифрового профиля сотрудника  (Ответы).xlsx` — 146 responses, 30 columns.

Decisions locked with user (2026-08-12):

- Import mechanism → **in-app module with staging + review UI**, modelled on the 1F sync. Not a one-off script.
- Storage → **hybrid**: normalize what the platform queries or acts on, keep every answer verbatim in JSONB.
- Sequencing → **everything through the UI**. No partial back-door import of the confident rows.
- `AtS` (9 responses) and `ХИРАД` (2) → **out of scope**, skipped like `Дусти Фарма` / ДИЭ.
- `ДФП` → **back in scope**, must arrive from both 1F and the form.
- `БИЮД` → a mistake; the department must be `БЮД`. Merge and delete the `БИЮД` record.
- Test departments (`ТЕСТ`, `МММ`, `ДРСНР`) → remove.

---

## Why a module and not a script

The form is the standing intake for new hires, so this runs again. 28 rows needed human judgement this round and those judgements must be *stored*, not re-derived. The codebase already has the right shape for it — `onef_sync_runs` + dry-run counts + a manual trigger button — so this reuses an idiom rather than inventing a second one.

**Identity is fixed before data lands.** Writing profile rows against a wrong `user_id` is far harder to undo than to prevent, so Phase 0 is org cleanup with no profile data involved.

---

## Phase 0 — Org cleanup (migration 0016 + `onef/departments.go`)

Two seeded departments are mis-named shells. Migration 0005 expanded the xlsx's bare abbreviations by guess, so the competency matrices sit on departments with no staff, while the departments with staff have no matrix.

| code | name today | users | requirements | verdict |
|---|---|---|---|---|
| `ДФП` | Департамент Финансового Планирования | 0 | 51 | rename — real ДФП is Фармацевтической Промоции |
| `БЮД` | Бюджетный Департамент | 0 | 51 | keep the row, it holds the matrix |
| `БИЮД` | Бухгалтерский и Юридический  Департамент | 19 | 0 | merge into `БЮД`, then delete |

Neither "Финансового Планирования" nor "Бюджетный" exists in 1F. The xlsx never spells either name out — it only ever writes `ДФП` and `БЮД`.

### 0.1 Rename ДФП in place

```
UPDATE departments SET name = 'Департамент Фармацевтической Промоции' WHERE code = 'ДФП';
```

Renaming rather than delete-and-recreate is deliberate: it keeps the code `ДФП`, so `uniqueDeptCode` never has to disambiguate, and the 51 requirements stay attached. Creating a new row instead is exactly what produced `ДФП2` on 2026-07-29 — `deriveDeptCode` is plain initials (`competency/codes.go`) and both names initial to `ДФП`.

### 0.2 Merge БИЮД → БЮД, then delete

`БЮД` keeps its code and its 51 requirements, and takes the real name so future syncs match it by normalization:

```
UPDATE departments SET name = 'Бухгалтерский и Юридический Департамент' WHERE code = 'БЮД';
```

> **Assumption to confirm.** The user said the department "must be БЮД", which reads as the *code* being wrong on the auto-created row, not the name. 1F and the questionnaire both call this department Бухгалтерский/Бухгалтерско-Юридический, and its 19 employees will recognise that name — so the code stays `БЮД` and the name becomes the real one. If "Бюджетный Департамент" was meant to be the display name, only this one line changes.

Repoint everything that references the `БИЮД` row, then delete it. Verified against `pg_constraint` — six tables carry a department FK, and only three have rows:

| table | column | rows to move |
|---|---|---|
| `users` | `department_id` | 19 |
| `sections` | `department_id` | 2 (`ЮО` 5 users, `БО` 13) |
| `user_roles` | `scope_department_id` | 2 |
| `audit_logs` | `department_scope_id` | 0 |
| `dept_competency_requirements` | `department_id` | 0 |
| `assessment_periods` | `department_id` | 0 |

Section codes are unique per `(department_id, lower(code))` (migration 0010) and `БЮД` has no sections, so `ЮО`/`БО` move without collision. Delete `БИЮД` only after asserting it has zero referencing rows — the migration should fail loudly rather than cascade.

### 0.3 Keep the sync from re-creating it

Deleting `БИЮД` is not enough on its own: the next poll sees `Бухгалтерский и Юридический  Департамент` (note 1F's double space) and auto-creates it again. Two layers, both cheap:

- The 0.2 rename makes `normalizeDeptName` match it directly — the normalizer already collapses whitespace runs.
- Add an explicit alias as a belt-and-braces guard, same pattern as ДЗЛ:
  `normalizeDeptName("Бухгалтерский и Юридический  Департамент") → "БЮД"`

### 0.4 Un-ignore ДФП

Drop `Департамент Фармацевтической Промоции` from `ignoredDepartments` in `backend/internal/onef/departments.go`. `Дусти Фарма` (17 people) and `Департамент Инженерной Экспертизы` (2) stay.

Expected on the next sync: **6 users created**, 3 sections auto-created (`Отдел Аналитики и Аудита`, `Отдел Развития и Науки`, `Отдел Фармацевтической Промоции`), 1 manager grant.

### 0.5 Remove test departments

`ТЕСТ` (TESTSTSTSTS, 12 requirements), `МММ` (Мы Можем Многое, 1 user `test head`), `ДРСНР` (already soft-deleted). One `assessment_periods` row points at ТЕСТ or МММ and must go with them; the `test head` user is deactivated, not deleted, so audit history stays intact.

### 0.6 Then run the sync

Picks up the three people added to 1F since the 10 Aug run — `Одинаев Насрулло` (ДЗЛ), `Кодиров Хабибулло` (АХД), `Нуров Азизджон` (ДИТ) — plus the 6 from ДФП.

### 0.7 Data hygiene, same pass

- Merge the duplicate `Кенджаева Фарзуна Хушвахтовна` (`emp#178`, `emp#179`) and find out why the `one_f_user_id` match missed it.
- Convert `emp#10` from Latin to Cyrillic: `Mansurov Aziz Jamshedovich` → `Мансуров Азиз Джамшедович`. It is the only Latin name left and it breaks every name comparison.

### Effect on the import

| | before Phase 0 | after |
|---|---|---|
| resolve automatically | 118 | **124** |
| skipped by policy | — | **11** (AtS 9, ХИРАД 2) |
| need a human | 28 | **11** |

The 11 remaining: rows 5, 78 (ДФП, absent from 1F), 13, 48, 136, 138, 140, 141 (ДИТ), 114, 133 (ДЗЛ), 132 (ФЭД).

---

## Phase 1 — Profile schema (migration 0017)

### Staging

```
profile_import_runs
  id uuid pk, source_filename text, uploaded_by uuid → users, uploaded_at timestamptz,
  status text,              -- 'parsed' | 'reviewing' | 'committed' | 'failed'
  total_rows, matched_count, skipped_count, unresolved_count int,
  committed_at timestamptz, error_message text

profile_import_rows
  id uuid pk, run_id uuid → profile_import_runs on delete cascade,
  source_row_no int,        -- the xlsx row, so findings stay traceable
  submitted_at timestamptz, -- the form's «Отметка времени»
  raw_full_name text, raw_department text,
  raw jsonb not null,       -- every answer verbatim, nothing dropped
  match_status text,        -- 'EXACT'|'PARTIAL'|'FUZZY'|'UNRESOLVED'|'SKIPPED'|'MANUAL'
  match_score numeric(4,3), match_candidates jsonb,
  matched_user_id uuid → users,
  resolved_by uuid → users, resolved_at timestamptz, resolution_note text,
  committed_at timestamptz
  unique (run_id, source_row_no)
```

`raw jsonb` is what makes the hybrid choice safe: normalization can be re-run, and a future survey engine can be built off stored data instead of re-uploading files.

### Normalized

```
user_profile                        -- 1:1 with users
  user_id uuid pk → users on delete cascade,
  education_levels text[], institution text, specialty text,
  prior_experience_band text, notable_projects text, previous_employers text,
  career_goal text, development_directions text[],
  mobility_department text, mobility_relocation text, internal_projects text,
  expertise_areas text, colleagues_ask_about text,
  teaching_readiness text, teaching_topics text,
  professional_interests text[], learning_formats text[], learning_hours_band text,
  extra_note text,
  source_run_id uuid → profile_import_runs, submitted_at timestamptz,
  created_at, updated_at

user_languages
  id uuid pk, user_id uuid → users on delete cascade,
  language text not null, level text not null check (level in ('A1','A2','B1','B2','C1','C2')),
  unique (user_id, language)
```

Reused as-is: `users.hobbies`, `user_certifications` (needs one added column, `source_url text`), `user_history` for anything HR chooses to promote to the career timeline.

`previous_employers` stays free text rather than being split into `user_history` rows — the answers are single blobs like `"Thefiftyfive Group - Ведущий специалист разработки, Technohub Dushanbe - главный ментор"` and no split is reliable. HR can promote entries manually.

---

## Phase 2 — Importer backend (`backend/internal/profile/`)

Package layout mirrors `onef/`: `models.go`, `parser.go`, `matcher.go`, `repository.go`, `service.go`, `handler.go`.

`POST /api/v1/profile-import/runs` (multipart xlsx) → parse + match + stage, returns dry-run counts.
`GET  /api/v1/profile-import/runs`, `GET .../runs/{id}/rows?status=` → review data.
`PATCH .../rows/{id}` → set `matched_user_id`, or mark skipped, with a note.
`POST .../runs/{id}/commit` → write-through in one transaction.
`DELETE .../runs/{id}` → discard an uncommitted run.

HR_ADMIN only, audited through the existing `audit` writer.

### Parser rules

These are all real defects in the current file, not hypotheticals:

1. **Language grid is broken.** It rendered as checkboxes, so 27 people ticked several CEFR levels for one language and 5 ticked all six. Rule: **take the highest ticked level.**
2. **Mixed alphabets in the level values.** `А1`/`А2` use a **Cyrillic А**; `B1`/`C1` use Latin. Normalize to Latin before the CHECK constraint sees them.
3. **Multi-selects are comma-joined strings** — including education (`"Высшее, Магистратура"`). Split on `", "`. No option value contains a comma.
4. **Certificates are Google Drive URLs**, comma-separated, 24 of 146 filled. Store the URLs in `user_certifications.source_url`; fetching waits for file storage in Phase 4.
5. **Whitespace.** 24 names carry leading/trailing spaces; no NBSP or double spaces occur. Trim on read regardless.
6. **Department answers never overwrite 1F.** They are self-reported and already wrong in 4 rows. Used only to flag disagreement in the review UI.

### Matching rules

Per-token alignment, order-independent, script-agnostic — whole-string similarity fails badly on missing patronymics (`Акопян Ованес` vs `Акопян Ованес Гагикович` scores 0.72 and would be discarded).

Normalize: trim, collapse whitespace, lowercase, ё→е, strip punctuation, hyphen→space. Align the shorter token list onto the longer, greedy best-match without reuse, score = mean token similarity, transliterate when scripts differ.

| status | rule |
|---|---|
| `EXACT` | ≥ 0.995, same token count |
| `PARTIAL` | ≥ 0.995, form omits a name part |
| `FUZZY` | ≥ 0.90 |
| `UNRESOLVED` | below 0.90, **or** a runner-up within 0.02 of the leader |

**Never auto-commit an ambiguous match.** The data contains real same-name traps — 1F holds two `Хакимов Илхом` in different departments with different patronymics and tenure, and rows 84/133 are two different `Хукматов`. The ambiguity guard exists specifically for these.

Skip rule applies to **unmatched rows only**, keyed on the typed department. Matched rows import regardless of what they typed — row 7 typed ДИЭ and row 66 typed ДФП, and both are real employees who mis-selected.

---

## Phase 3 — Review UI

`/admin/profile-import`, HR_ADMIN only. Upload → dry-run summary → row list filtered by status.

Unresolved rows show the parsed answers beside the top 3 candidates with scores, plus a search box to pick any user, and buttons for **Привязать**, **Пропустить**, **Создать сотрудника**. Department disagreements and duplicate submissions surface as warnings on the row. Commit is blocked while any row is still `UNRESOLVED`.

## Phase 4 — Commit and display

Write-through in one transaction, per resolved row: upsert `user_profile`, replace `user_languages`, upsert certifications, set `users.hobbies` if empty. Re-importing the same person updates their profile and bumps `submitted_at`; the previous run's staging rows are retained as history.

Then surface it: a **Цифровой профиль** tab on `WorkerProfile` and a languages/education block on the worker card.

## Phase 5 — Downstream (not in this spec)

The questionnaire feeds two unbuilt phases and should be treated as their input, not as decoration:

- **Preceptor pool** — columns 22–25. 69 people said yes to running internal training, 65 said "in future", with their topics and what colleagues already come to them for.
- **PID inputs** — columns 27–29. Professional interests, preferred learning format, monthly hours available.

---

## Open items

1. **`БЮД` display name** — confirm the assumption in §0.2.
2. **The 11 unresolved rows** — HR needs to say whether these are new hires not yet in 1F, or people who should be created manually.
3. **Next form round must carry `employee_no`.** Name matching is a one-time cost for this batch; a required табельный номер field removes it permanently. Prefilled per-employee links would be better still.
4. **Certificate files** — the Drive links are only reachable while the folder stays shared. Worth mirroring the files once file storage exists.
5. **`Дусти Фарма` (17 people)** — still ignored. Confirm that is still intended, given ДФП came back.

---

## Build order

**0** org cleanup + sync (migration 0016, `departments.go`) → **1** schema (migration 0017) → **2** parser, matcher, dry-run → **3** review UI → **4** commit + profile display.

Each phase ends `go build` / `go vet` / `tsc --noEmit` clean, and each migration is tested up **and** down before moving on.
