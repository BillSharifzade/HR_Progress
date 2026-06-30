import { client } from './client';
import type {
  AssessmentPeriod,
  AssessmentScore,
  Competency,
  CompetencyKind,
  Department,
  Employee,
  Grade,
  PeriodWithScores,
  Requirement,
} from '../types';

export interface CreateCompetencyPayload {
  kind: CompetencyKind;
  name: string;
  description?: string | null;
  why_important?: string | null;
}

export interface UpdateCompetencyPayload extends CreateCompetencyPayload {
  is_active: boolean;
}

export async function listDepartments(): Promise<Department[]> {
  const r = await client.get<Department[]>('/departments/');
  return r.data;
}

export async function listAllDepartments(): Promise<Department[]> {
  const r = await client.get<Department[]>('/departments/', { params: { include_inactive: true } });
  return r.data;
}

export async function createDepartment(payload: { name: string; description?: string | null }): Promise<Department> {
  const r = await client.post<Department>('/departments/', payload);
  return r.data;
}

export async function updateDepartment(id: string, payload: { name: string; description?: string | null; is_active: boolean }): Promise<Department> {
  const r = await client.put<Department>(`/departments/${id}/`, payload);
  return r.data;
}

export async function deleteDepartment(id: string): Promise<void> {
  await client.delete(`/departments/${id}/`);
}

export async function listGrades(): Promise<Grade[]> {
  const r = await client.get<Grade[]>('/grades/');
  return r.data;
}

export async function listEmployees(deptId: string): Promise<Employee[]> {
  const r = await client.get<Employee[]>(`/departments/${deptId}/employees`);
  return r.data;
}

export interface BulkScorePayload {
  employee_id: string;
  competency_id: string;
  assessor_role: string;
  score: number | null;
  feedback?: string | null;
}

export async function upsertScoresBulk(periodId: string, scores: BulkScorePayload[]): Promise<void> {
  await client.post(`/assessment-periods/${periodId}/scores/bulk`, scores);
}

export interface UpsertRequirementPayload {
  competency_id: string;
  grade_id: string;
  required_min: number | null;
  is_key: boolean;
}

export async function upsertRequirements(deptId: string, reqs: UpsertRequirementPayload[]): Promise<void> {
  await client.put(`/departments/${deptId}/requirements/`, reqs);
}

export async function listCompetencies(): Promise<Competency[]> {
  const r = await client.get<Competency[]>('/competencies/');
  return r.data;
}

export async function createCompetency(payload: CreateCompetencyPayload): Promise<Competency> {
  const r = await client.post<Competency>('/competencies/', payload);
  return r.data;
}

export async function updateCompetency(id: string, payload: UpdateCompetencyPayload): Promise<Competency> {
  const r = await client.put<Competency>(`/competencies/${id}/`, payload);
  return r.data;
}

export async function deleteCompetency(id: string): Promise<void> {
  await client.delete(`/competencies/${id}/`);
}

export async function reorderCompetencies(ids: string[]): Promise<void> {
  await client.post('/competencies/reorder', { ids });
}

export async function listRequirements(deptId: string): Promise<Requirement[]> {
  const r = await client.get<Requirement[]>(`/departments/${deptId}/requirements/`);
  return r.data;
}

export async function listPeriods(deptId?: string): Promise<AssessmentPeriod[]> {
  const params = deptId ? { department_id: deptId } : {};
  const r = await client.get<AssessmentPeriod[]>('/assessment-periods/', { params });
  return r.data;
}

export interface CriterionInput {
  competency_id: string;
  name?: string | null;
  description?: string | null;
  min_score?: number | null;
}

export async function createPeriod(payload: {
  title: string;
  department_id?: string;
  period_start: string;
  period_end: string;
  group_size?: number;
  department_ids?: string[];
  section_ids?: string[];
  criteria?: CriterionInput[];
}): Promise<AssessmentPeriod> {
  const r = await client.post<AssessmentPeriod>('/assessment-periods/', payload);
  return r.data;
}

// ── Criteria (FR-AS3) ────────────────────────────────────────────────────────
export async function listCriteria(periodId: string): Promise<import('../types').Criterion[]> {
  const r = await client.get<import('../types').Criterion[]>(`/assessment-periods/${periodId}/criteria`);
  return r.data ?? [];
}

export async function setCriteria(periodId: string, criteria: CriterionInput[]): Promise<import('../types').Criterion[]> {
  const r = await client.put<import('../types').Criterion[]>(`/assessment-periods/${periodId}/criteria`, { criteria });
  return r.data ?? [];
}

// ── Assessees (FR-AS2) ───────────────────────────────────────────────────────
export async function listAssessees(periodId: string): Promise<import('../types').Assessee[]> {
  const r = await client.get<import('../types').Assessee[]>(`/assessment-periods/${periodId}/assessees`);
  return r.data ?? [];
}

export async function addAssessees(
  periodId: string,
  payload: { user_ids?: string[]; department_ids?: string[]; section_ids?: string[] },
): Promise<{ added: number; assessees: import('../types').Assessee[] }> {
  const r = await client.post(`/assessment-periods/${periodId}/assessees`, payload);
  return r.data;
}

export async function removeAssessee(periodId: string, userId: string): Promise<void> {
  await client.delete(`/assessment-periods/${periodId}/assessees/${userId}`);
}

// ── Per-assessee assessors (FR-AS4) ──────────────────────────────────────────
export async function listAssesseeAssessors(periodId: string): Promise<import('../types').AssesseeAssessor[]> {
  const r = await client.get<import('../types').AssesseeAssessor[]>(`/assessment-periods/${periodId}/assessee-assessors`);
  return r.data ?? [];
}

export async function setAssesseeAssessors(
  periodId: string,
  assesseeId: string,
  assessorUserIds: string[],
): Promise<import('../types').AssesseeAssessor[]> {
  const r = await client.put<import('../types').AssesseeAssessor[]>(
    `/assessment-periods/${periodId}/assessees/${assesseeId}/assessors`,
    { assessor_user_ids: assessorUserIds },
  );
  return r.data ?? [];
}

// ── Lifecycle (Section 5, FR-AS9, FR-AS10) ───────────────────────────────────
export async function transitionPeriod(
  periodId: string,
  to: import('../types').CampaignStatus,
): Promise<AssessmentPeriod> {
  const r = await client.post<AssessmentPeriod>(`/assessment-periods/${periodId}/transition`, { to });
  return r.data;
}

// ── Learning groups (FR-AS13) ────────────────────────────────────────────────
export async function listGroups(periodId: string): Promise<import('../types').LearningGroup[]> {
  const r = await client.get<import('../types').LearningGroup[]>(`/assessment-periods/${periodId}/groups`);
  return r.data ?? [];
}

export async function listGroupJournal(periodId: string): Promise<import('../types').GroupJournalEntry[]> {
  const r = await client.get<import('../types').GroupJournalEntry[]>(`/assessment-periods/${periodId}/groups/journal`);
  return r.data ?? [];
}

export async function regenerateGroups(periodId: string, groupSize?: number): Promise<import('../types').LearningGroup[]> {
  const r = await client.post<import('../types').LearningGroup[]>(
    `/assessment-periods/${periodId}/groups/regenerate`,
    groupSize ? { group_size: groupSize } : {},
  );
  return r.data ?? [];
}

export async function moveGroupMember(
  periodId: string,
  userId: string,
  toGroupId: string,
): Promise<import('../types').LearningGroup[]> {
  const r = await client.post<import('../types').LearningGroup[]>(`/assessment-periods/${periodId}/groups/move`, {
    user_id: userId,
    to_group_id: toGroupId,
  });
  return r.data ?? [];
}

export async function confirmGroups(periodId: string): Promise<import('../types').LearningGroup[]> {
  const r = await client.post<import('../types').LearningGroup[]>(`/assessment-periods/${periodId}/groups/confirm`, {});
  return r.data ?? [];
}

// ── Interpretation reference (FR-AS7.2) ──────────────────────────────────────
export async function listInterpretations(params: {
  department_id?: string;
  grade_id?: string;
  competency_id?: string;
}): Promise<import('../types').Interpretation[]> {
  const r = await client.get<import('../types').Interpretation[]>('/interpretations/', { params });
  return r.data ?? [];
}

export async function lookupInterpretation(
  assesseeId: string,
  competencyId: string,
  score: number,
): Promise<import('../types').InterpretationLookup> {
  const r = await client.get<import('../types').InterpretationLookup>('/interpretations/lookup', {
    params: { assessee_id: assesseeId, competency_id: competencyId, score },
  });
  return r.data;
}

export async function upsertInterpretation(payload: {
  department_id: string;
  grade_id: string;
  competency_id: string;
  score: number;
  text: string;
}): Promise<import('../types').Interpretation> {
  const r = await client.post<import('../types').Interpretation>('/interpretations/', payload);
  return r.data;
}

export async function deleteInterpretation(id: string): Promise<void> {
  await client.delete(`/interpretations/${id}`);
}

export async function copyInterpretations(payload: {
  from_department_id: string;
  to_department_id: string;
  from_grade_id?: string;
  to_grade_id?: string;
  overwrite?: boolean;
}): Promise<{ copied: number }> {
  const r = await client.post<{ copied: number }>('/interpretations/copy', payload);
  return r.data;
}

export async function interpretationHistory(params: {
  department_id?: string;
  grade_id?: string;
  competency_id?: string;
  score?: number;
}): Promise<import('../types').InterpretationHistoryEntry[]> {
  const r = await client.get<import('../types').InterpretationHistoryEntry[]>('/interpretations/history', { params });
  return r.data ?? [];
}

// ── Worker results (FR-AS10/AS11) ────────────────────────────────────────────
export async function myAssessmentResults(): Promise<import('../types').EmployeeResult[]> {
  const r = await client.get<import('../types').EmployeeResult[]>('/me/assessment-results/');
  return r.data ?? [];
}

export async function getPeriodWithScores(periodId: string): Promise<PeriodWithScores> {
  const r = await client.get<PeriodWithScores>(`/assessment-periods/${periodId}/`);
  return r.data;
}

export async function listPeriodParticipants(periodId: string): Promise<import('../types').AssessmentParticipant[]> {
  const r = await client.get<import('../types').AssessmentParticipant[]>(
    `/assessment-periods/${periodId}/participants/`,
  );
  return r.data ?? [];
}

export async function addPeriodParticipants(
  periodId: string,
  participants: { user_id: string; role: import('../types').ParticipantRole }[],
): Promise<import('../types').AssessmentParticipant[]> {
  const r = await client.post<import('../types').AssessmentParticipant[]>(
    `/assessment-periods/${periodId}/participants/`,
    { participants },
  );
  return r.data ?? [];
}

export async function listMyAssessmentPeriods(): Promise<import('../types').MyAssessmentPeriod[]> {
  const r = await client.get<import('../types').MyAssessmentPeriod[]>('/me/assessment-periods/');
  return r.data ?? [];
}

export async function listMyScoresIn(periodId: string): Promise<AssessmentScore[]> {
  const r = await client.get<AssessmentScore[]>(`/assessment-periods/${periodId}/my-scores`);
  return r.data ?? [];
}

export async function listConsolidatedScores(periodId: string): Promise<import('../types').ConsolidatedScore[]> {
  const r = await client.get<import('../types').ConsolidatedScore[]>(
    `/assessment-periods/${periodId}/consolidated`,
  );
  return r.data ?? [];
}

export async function listUsersWithRole(role: string): Promise<Employee[]> {
  const r = await client.get<Employee[]>('/users/with-role', { params: { role } });
  return r.data ?? [];
}

export async function upsertScore(
  periodId: string,
  payload: {
    employee_id: string;
    competency_id: string;
    assessor_role: string;
    score: number | null;
    feedback?: string | null;
  },
): Promise<AssessmentScore> {
  const r = await client.post<AssessmentScore>(`/assessment-periods/${periodId}/scores`, payload);
  return r.data;
}
