import { client } from './client';

export interface OneFStatus {
  configured: boolean;
}

export interface OneFSyncResult {
  run_id: string;
  status: 'running' | 'success' | 'failed';
  fetched_count: number;
  created_count: number;
  updated_count: number;
  skipped_count: number;
  duration_ms: number;
  error?: string;
}

export interface OneFSyncRun {
  id: string;
  started_at: string;
  finished_at?: string | null;
  triggered_by?: string | null;
  trigger_kind: 'cron' | 'manual';
  status: 'running' | 'success' | 'failed';
  fetched_count: number;
  created_count: number;
  updated_count: number;
  skipped_count: number;
  error_message?: string | null;
  duration_ms?: number | null;
}

export async function getOneFStatus(): Promise<OneFStatus> {
  const r = await client.get<OneFStatus>('/onef/status');
  return r.data;
}

export async function triggerOneFSync(): Promise<OneFSyncResult> {
  const r = await client.post<OneFSyncResult>('/onef/sync');
  return r.data;
}

export async function listOneFRuns(limit = 20): Promise<OneFSyncRun[]> {
  const r = await client.get<OneFSyncRun[]>('/onef/runs', { params: { limit } });
  return r.data;
}
