import { client } from './client';
import type {
  Worker,
  WorkerSummary,
  WorkerCertification,
  WorkerHistory,
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
}

export async function createCertification(
  workerId: string,
  payload: CertificationPayload,
): Promise<WorkerCertification> {
  const r = await client.post<WorkerCertification>(`/workers/${workerId}/certifications/`, payload);
  return r.data;
}

export async function deleteCertification(workerId: string, certId: string): Promise<void> {
  await client.delete(`/workers/${workerId}/certifications/${certId}`);
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
