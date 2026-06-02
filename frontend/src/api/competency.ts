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

export async function createPeriod(payload: {
  title: string;
  department_id?: string;
  period_start: string;
  period_end: string;
}): Promise<AssessmentPeriod> {
  const r = await client.post<AssessmentPeriod>('/assessment-periods/', payload);
  return r.data;
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
