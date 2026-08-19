import { client } from './client';
import type {
  Worker,
  WorkerSummary,
  WorkerCertification,
  WorkerHistory,
  WorkerLanguage,
  WorkerExperience,
  WorkerSurveyAnswer,
  WorkerProfileData,
  CefrLevel,
  Position,
  Section,
  RoleAssignment,
  UserRole,
} from '../types';

export interface ListWorkersParams {
  department_id?: string;
  section_id?: string;
  grade_id?: string;
  search?: string;
  include_inactive?: boolean;
}

export async function listWorkers(params?: ListWorkersParams): Promise<WorkerSummary[]> {
  const r = await client.get<WorkerSummary[]>('/workers/', { params });
  return r.data;
}

export async function getWorker(id: string): Promise<Worker> {
  const r = await client.get<Worker>(`/workers/${id}/`);
  return r.data;
}

export async function listCertifications(workerId: string): Promise<WorkerCertification[]> {
  const r = await client.get<WorkerCertification[]>(`/workers/${workerId}/certifications/`);
  return r.data;
}

export interface CertificationPayload {
  title: string;
  issued_by?: string | null;
  issued_at?: string | null;
  expires_at?: string | null;
  source_url?: string | null;
}

export async function createCertification(
  workerId: string,
  payload: CertificationPayload,
): Promise<WorkerCertification> {
  const r = await client.post<WorkerCertification>(`/workers/${workerId}/certifications/`, payload);
  return r.data;
}

export async function updateCertification(
  workerId: string,
  certId: string,
  payload: CertificationPayload,
): Promise<WorkerCertification> {
  const r = await client.patch<WorkerCertification>(
    `/workers/${workerId}/certifications/${certId}`, payload);
  return r.data;
}

export async function deleteCertification(workerId: string, certId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/certifications/${certId}`);
}

export async function uploadCertificationFile(
  workerId: string, certId: string, file: File,
): Promise<WorkerCertification> {
  const body = new FormData();
  body.append('file', file);
  const r = await client.post<WorkerCertification>(
    `/workers/${workerId}/certifications/${certId}/file`, body,
    { headers: { 'Content-Type': 'multipart/form-data' } });
  return r.data;
}

export async function deleteCertificationFile(workerId: string, certId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/certifications/${certId}/file`);
}

/**
 * Hands a downloaded blob to the browser as a save. Endpoints that serve files
 * need the Authorization header, so the bytes come through the axios client and
 * are offered as an object URL rather than linked to directly.
 */
function saveBlob(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}

/**
 * Reads the file name off a Content-Disposition header. The server sends both
 * an ASCII fallback and the RFC 5987 form; the latter carries the real Cyrillic
 * name and is preferred whenever it decodes.
 */
function fileNameFromDisposition(header: unknown): string | null {
  if (typeof header !== 'string') return null;
  const utf8 = /filename\*=UTF-8''([^;]+)/i.exec(header);
  if (utf8) {
    try {
      return decodeURIComponent(utf8[1]);
    } catch {
      // Malformed encoding — fall through to the ASCII form.
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(header);
  return plain ? plain[1] : null;
}

/**
 * Turns a failed blob-typed request into a readable message. With
 * `responseType: 'blob'` even the JSON error body arrives as a Blob, so it has
 * to be read back before the usual error shape is there to look at.
 */
export async function readBlobError(err: unknown): Promise<string | null> {
  const data = (err as { response?: { data?: unknown } })?.response?.data;
  if (!(data instanceof Blob)) return null;
  try {
    const parsed = JSON.parse(await data.text()) as { error?: { message?: string } };
    return parsed?.error?.message ?? null;
  } catch {
    return null;
  }
}

export async function downloadCertificationFile(
  workerId: string, certId: string, fileName: string,
): Promise<void> {
  const r = await client.get<Blob>(
    `/workers/${workerId}/certifications/${certId}/file`, { responseType: 'blob' });
  saveBlob(r.data, fileName || 'certificate');
}

/**
 * Downloads the worker register as a formatted XLSX workbook. The same filter
 * params the list view uses are passed straight through, so the file always
 * covers exactly the population shown on screen.
 */
export async function exportWorkers(params?: ListWorkersParams): Promise<string> {
  const r = await client.get<Blob>('/workers/export', { params, responseType: 'blob' });
  const name = fileNameFromDisposition(r.headers?.['content-disposition']) ?? 'workers.xlsx';
  saveBlob(r.data, name);
  return name;
}

// ─── Digital profile ────────────────────────────────────────────────────────

export async function getWorkerProfile(workerId: string): Promise<WorkerProfileData> {
  const r = await client.get<WorkerProfileData>(`/workers/${workerId}/profile/`);
  return r.data;
}

export type ProfilePayload = Omit<
  WorkerProfileData, 'user_id' | 'submitted_at' | 'source' | 'created_at' | 'updated_at'
>;

export async function saveWorkerProfile(
  workerId: string, payload: ProfilePayload,
): Promise<WorkerProfileData> {
  const r = await client.put<WorkerProfileData>(`/workers/${workerId}/profile/`, payload);
  return r.data;
}

export async function listLanguages(workerId: string): Promise<WorkerLanguage[]> {
  const r = await client.get<WorkerLanguage[]>(`/workers/${workerId}/languages/`);
  return r.data ?? [];
}

export interface LanguagePayload { language: string; level: CefrLevel }

export async function upsertLanguage(
  workerId: string, payload: LanguagePayload,
): Promise<WorkerLanguage> {
  const r = await client.post<WorkerLanguage>(`/workers/${workerId}/languages/`, payload);
  return r.data;
}

export async function updateLanguage(
  workerId: string, languageId: string, payload: LanguagePayload,
): Promise<WorkerLanguage> {
  const r = await client.patch<WorkerLanguage>(
    `/workers/${workerId}/languages/${languageId}`, payload);
  return r.data;
}

export async function deleteLanguage(workerId: string, languageId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/languages/${languageId}`);
}

export async function listExperience(workerId: string): Promise<WorkerExperience[]> {
  const r = await client.get<WorkerExperience[]>(`/workers/${workerId}/experience/`);
  return r.data ?? [];
}

export interface ExperiencePayload {
  company: string;
  position?: string | null;
  started_on?: string | null;
  ended_on?: string | null;
  description?: string | null;
  sort_order?: number | null;
}

export async function createExperience(
  workerId: string, payload: ExperiencePayload,
): Promise<WorkerExperience> {
  const r = await client.post<WorkerExperience>(`/workers/${workerId}/experience/`, payload);
  return r.data;
}

export async function updateExperience(
  workerId: string, experienceId: string, payload: ExperiencePayload,
): Promise<WorkerExperience> {
  const r = await client.patch<WorkerExperience>(
    `/workers/${workerId}/experience/${experienceId}`, payload);
  return r.data;
}

export async function deleteExperience(workerId: string, experienceId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/experience/${experienceId}`);
}

export async function listSurveyAnswers(workerId: string): Promise<WorkerSurveyAnswer[]> {
  const r = await client.get<WorkerSurveyAnswer[]>(`/workers/${workerId}/survey/`);
  return r.data ?? [];
}

export interface SurveyAnswerPayload {
  form_key?: string;
  question_code: string;
  question_text: string;
  answer_text: string;
  position?: number | null;
}

export async function upsertSurveyAnswer(
  workerId: string, payload: SurveyAnswerPayload,
): Promise<WorkerSurveyAnswer> {
  const r = await client.post<WorkerSurveyAnswer>(`/workers/${workerId}/survey/`, payload);
  return r.data;
}

export async function updateSurveyAnswer(
  workerId: string, answerId: string, payload: SurveyAnswerPayload,
): Promise<WorkerSurveyAnswer> {
  const r = await client.patch<WorkerSurveyAnswer>(
    `/workers/${workerId}/survey/${answerId}`, payload);
  return r.data;
}

export async function deleteSurveyAnswer(workerId: string, answerId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/survey/${answerId}`);
}

export async function listWorkerRoles(workerId: string): Promise<RoleAssignment[]> {
  const r = await client.get<RoleAssignment[]>(`/workers/${workerId}/roles/`);
  return r.data ?? [];
}

export interface GrantRolePayload {
  role: UserRole;
  scope_department_id?: string | null;
  scope_section_id?: string | null;
}

export async function grantRole(workerId: string, payload: GrantRolePayload): Promise<RoleAssignment> {
  const r = await client.post<RoleAssignment>(`/workers/${workerId}/roles/`, payload);
  return r.data;
}

export async function revokeRole(workerId: string, assignmentId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/roles/${assignmentId}`);
}

export interface ResetCredentialsResult {
  username: string;
  password: string;
}

export async function resetWorkerCredentials(
  workerId: string,
): Promise<ResetCredentialsResult> {
  const r = await client.post<ResetCredentialsResult>(
    `/workers/${workerId}/credentials/reset`,
    {},
  );
  return r.data;
}

export async function listHistory(workerId: string): Promise<WorkerHistory[]> {
  const r = await client.get<WorkerHistory[]>(`/workers/${workerId}/history/`);
  return r.data;
}

export interface HistoryPayload {
  event_kind: string;
  event_date: string;
  title: string;
  description?: string | null;
  meta?: Record<string, unknown> | null;
}

export async function createHistory(workerId: string, payload: HistoryPayload): Promise<WorkerHistory> {
  const r = await client.post<WorkerHistory>(`/workers/${workerId}/history/`, payload);
  return r.data;
}

export async function listPositions(): Promise<Position[]> {
  const r = await client.get<Position[]>('/positions/');
  return r.data;
}

export async function listSections(departmentId?: string): Promise<Section[]> {
  const params = departmentId ? { department_id: departmentId } : {};
  const r = await client.get<Section[]>('/sections/', { params });
  return r.data;
}

export interface CreateSectionPayload {
  department_id: string;
  name: string;
  description?: string | null;
}

export interface UpdateSectionPayload {
  name: string;
  description?: string | null;
  is_active: boolean;
}

export async function createSection(payload: CreateSectionPayload): Promise<Section> {
  const r = await client.post<Section>('/sections/', payload);
  return r.data;
}

export async function updateSection(id: string, payload: UpdateSectionPayload): Promise<Section> {
  const r = await client.patch<Section>(`/sections/${id}`, payload);
  return r.data;
}

export async function deleteSection(id: string): Promise<void> {
  await client.delete(`/sections/${id}`);
}

export interface CreateWorkerPayload {
  username?: string;
  full_name: string;
  email?: string | null;
  personnel_number?: string | null;
  birth_date?: string | null;
  department_id?: string | null;
  section_id?: string | null;
  grade_id?: string | null;
  position?: string | null;
  specialization?: string | null;
  telegram_id?: number | null;
  hired_at?: string | null;
  hobbies?: string | null;
}

export interface CreateWorkerResult {
  worker: Worker;
  username: string;
  password: string;
}

export async function createWorker(payload: CreateWorkerPayload): Promise<CreateWorkerResult> {
  const r = await client.post<CreateWorkerResult>('/workers/', payload);
  return r.data;
}

export interface UpdateWorkerPayload {
  full_name: string;
  email?: string | null;
  personnel_number?: string | null;
  birth_date?: string | null;
  department_id?: string | null;
  section_id?: string | null;
  grade_id?: string | null;
  position?: string | null;
  specialization?: string | null;
  telegram_id?: number | null;
  hired_at?: string | null;
  hobbies?: string | null;
}

export async function updateWorker(id: string, payload: UpdateWorkerPayload): Promise<Worker> {
  const r = await client.patch<Worker>(`/workers/${id}/`, payload);
  return r.data;
}

export async function activateWorker(id: string): Promise<void> {
  await client.post(`/workers/${id}/activate`);
}

export async function deactivateWorker(id: string): Promise<void> {
  await client.post(`/workers/${id}/deactivate`);
}
