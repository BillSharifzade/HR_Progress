import { client } from './client';

export interface DashboardKPIs {
  active_workers: number;
  departments: number;
  competencies: number;
  active_campaigns: number;
  assessed_workers: number;
  avg_score: number | null;
}

export interface DashboardBucket {
  label: string;
  value: number;
}

export interface CompetencyGap {
  competency_id: string;
  name: string;
  kind: string;
  avg_score: number;
  required_min: number;
  delta: number;
  assessed: number;
}

export interface CampaignProgress {
  period_id: string;
  title: string;
  status: string;
  assessees: number;
  criteria: number;
  done: number;
  expected: number;
}

export interface TrendPoint {
  month: string;
  hired: number;
  total: number;
}

export interface DashboardOverview {
  kpis: DashboardKPIs;
  headcount_by_department: DashboardBucket[];
  headcount_by_grade: DashboardBucket[];
  competency_gaps: CompetencyGap[];
  score_distribution: DashboardBucket[];
  campaign_progress: CampaignProgress[];
  hiring_trend: TrendPoint[];
}

export async function getDashboardOverview(): Promise<DashboardOverview> {
  const r = await client.get<DashboardOverview>('/dashboard/overview');
  return r.data;
}
