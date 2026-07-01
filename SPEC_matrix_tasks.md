# Spec — Matrix & scoring enhancements (6 tasks)

Status: **implemented** (frontend tsc+build clean, backend build+vet clean). Date: 2026-06-30. Not yet committed or deployed.

Decisions locked with user:
- Comment templates → **reuse existing Интерпретации** (dept+grade+competency+mark).
- Matrix delete → **remove from this department's matrix only** (competency stays global).
- Critical/key competency color → **purple/violet** (red is freed for divergence).
- Divergence red → **any two role marks differ by >4**, shown in the admin scoring matrix.

---

## Task 1 — Delete a competency from a department's matrix
**Where:** `CompetencyMatrixPage.tsx`, matrix edit mode (`editMatrixColumns`).
**Backend:** none. `UpsertRequirements` already replaces all of a dept's requirements on save, so excluding a competency removes it.
**Change:** add a trash icon per row in the edit table. Click → confirm → clears that competency's cells in `matrixEditState` and visually marks the row as removed (greyed). On **Save**, removed competencies have no cells → dropped from the dept matrix. Cancel restores.

## Task 2 — Rename "HR-аналитик" → "Ассессор"
**Where:** `types.ts`.
**Change:** `AssessorRoleLabel.HRA` and `ParticipantRoleLabel.HRA` → `'Ассессор'`. This relabels the HRA column in the scoring matrix and role tags. (Note: the standalone `ASSESSOR` role is also "Ассессор"; they don't co-occur in the matrix scoring grid, so no collision there.)

## Task 3 — Critical → purple; divergence → red
**Where:** `PeriodScoringModal.tsx` (and key-star color unified in `MyPeriodScoringPage.tsx`).
**Critical color:** replace `token.colorError` used for key competencies (star + row highlight) with purple (`#722ed1`).
**Divergence flag — two places:**
- **Admin scoring window** (`PeriodScoringModal`, frontend-only): per competency row, among the entered role marks {HEAD, DEPT_HEAD, Ассессор/HRA, DCR_HEAD}, if **max − min > 4** → render that row red (red bg + red left border, reusing old critical styling) with tooltip "Расхождение оценок > 4". Raw values, strict `> 4`.
- **Worker's results page** (`MyResultsPage`, needs backend): add a `divergent bool` to `MyPublishedResults`/`EmployeeResult`, computed server-side as `max(score)−min(score) > 4` over non-null `assessment_scores` for roles {HEAD, DEPT_HEAD, HRA, DCR_HEAD} for that (period, employee, competency). Frontend shows a red tag on the competency row.

## Task 4 — Replace number-stepper arrows with a comment icon
**Where:** mark `InputNumber` in `PeriodScoringModal.tsx` and `MyPeriodScoringPage.tsx`.
**Change:** set `controls={false}` (removes the up/down arrows). Add a comment icon button next to each mark field that opens the comment modal (Task 5). Icon shows filled when a comment exists, outline when empty.

## Task 5 — Comment modal with prefilled interpretation
**Trigger:** when a mark is committed (field blur / Enter after entering a value), the comment modal opens automatically prefilled with the interpretation; it can be reopened anytime via the Task-4 icon.
**Prefill:** `lookupInterpretation(worker, competency, roundMark(score))` (existing endpoint).
**Round rule:** `roundMark(x)` = floor if fractional part ≤ 0.5, else ceil → 5.5→5, 5.6→6. Replaces the current `Math.round` calls.
**Editing:** the modal shows the prefilled text editable; the evaluator accepts (save) or edits. Saved as `feedback` on their own role's score.
**Per-role visibility:** evaluator sees only their own comment. **Admin** (in `PeriodScoringModal`, which already holds all roles' scores) gets a role switcher inside the modal to view each role's comment for that (worker, competency). Read-only for roles other than the one being scored.
**Template CRUD lives in the modal (not the Admin page):** when an admin has the modal open, they can create/edit/delete the prefilled-comment **template** for that (dept, grade, competency, mark) right there — it becomes the default everyone sees. Wires to the existing `Upsert/DeleteInterpretation` endpoints (already admin-only). No Администрирование-page section is added.
**Inline cleanup:** remove the old inline "Системная интерпретация" Alert + textarea from `MyPeriodScoringPage`; the modal replaces it.

## Task 6 — Previous / Next worker for all roles
**Where:** both scoring surfaces.
- `MyPeriodScoringPage.tsx`: add "← Предыдущий" beside the existing "Следующий →"; disable at ends.
- `PeriodScoringModal.tsx`: add ◀ / ▶ buttons beside the employee `Select` to step through the employee list; disable at ends.

---

## Build order (phased)
1. **Phase A (frontend-only, low risk):** Task 2, Task 6, Task 1.
2. **Phase B:** Task 3 — purple critical + divergence red. Frontend (admin window) **+ small backend** (`divergent` flag on published results for the worker results page).
3. **Phase C:** Tasks 4 + 5 (comment icon, comment modal with prefill + accept/edit, roundMark, admin role-switch, in-modal template CRUD).

Backend touched only by Task 3 (one added `divergent` field). Each phase: `go build`/`tsc --noEmit` clean before moving on.
