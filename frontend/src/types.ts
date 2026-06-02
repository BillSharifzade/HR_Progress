export type Role = 'HR_ADMIN' | 'DEPT_HEAD' | 'ASSESSOR' | 'PRECEPTOR' | 'SECTION_HEAD' | 'ATS' | 'BOOK_SPACE';

export interface User {
  id: string;
  personnel_number?: string | null;
  username: string;
  email?: string | null;
  full_name: string;
  birth_date?: string | null;
  department_id?: string | null;
  section_id?: string | null;
  grade_id?: string | null;
  position_id?: string | null;
  position_name?: string | null;
  position?: string | null;
  specialization?: string | null;
  telegram_id?: number | null;
  hired_at?: string | null;
  hobbies?: string | null;
  must_change_password: boolean;
  is_active: boolean;
  roles: Role[];
  scope_department_ids?: string[];
  scope_section_ids?: string[];
}

export interface Worker {
  id: string;
  username: string;
  employee_no: number;
  personnel_number?: string | null;
  one_f_user_id?: number | null;
  one_f_is_manager?: boolean;
  phone_number?: string | null;
  last_synced_at?: string | null;
  full_name: string;
  email?: string | null;
  birth_date?: string | null;
  department_id?: string | null;
  department_name?: string | null;
  section_id?: string | null;
  section_name?: string | null;
  grade_id?: string | null;
  grade_name?: string | null;
  grade_level?: number | null;
  position_id?: string | null;
  position_name?: string | null;
  position?: string | null;
  specialization?: string | null;
  telegram_id?: number | null;
  hired_at?: string | null;
  hobbies?: string | null;
  is_active: boolean;
  roles: string[];
}

export interface WorkerSummary {
  id: string;
  employee_no: number;
  personnel_number?: string | null;
  one_f_user_id?: number | null;
  full_name: string;
  department_name?: string | null;
  section_name?: string | null;
  grade_name?: string | null;
  grade_level?: number | null;
  position_name?: string | null;
  hired_at?: string | null;
  is_active: boolean;
}

export interface WorkerCertification {
  id: string;
  user_id: string;
  title: string;
  issued_by?: string | null;
  issued_at?: string | null;
  expires_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface WorkerHistory {
  id: string;
  user_id: string;
  event_kind: string;
  event_date: string;
  title: string;
  description?: string | null;
  meta?: Record<string, unknown> | null;
  created_at: string;
}

export interface Position {
  id: string;
  name: string;
  is_active: boolean;
}

export interface Section {
  id: string;
  department_id: string;
  code?: string | null;
  name: string;
  description?: string | null;
  is_active: boolean;
}

export interface LoginResponse {
  access_token: string;
  expires_at: string;
  user: User;
}

export interface ApiError {
  error: { code: string; message: string };
}

export interface Department {
  id: string;
  code: string;
  name: string;
  description?: string;
  is_active: boolean;
}

export interface Employee {
  id: string;
  full_name: string;
  grade_id?: string | null;
  grade_name?: string | null;
  grade_level?: number | null;
  section_id?: string | null;
}

export interface Grade {
  id: string;
  name: string;
  level: number;
  description?: string | null;
  is_active: boolean;
}

export type CompetencyKind = 'LK' | 'UK' | 'PK';

export const CompetencyKindLabel: Record<CompetencyKind, string> = {
  LK: 'Личностные',
  UK: 'Управленческие',
  PK: 'Профессиональные',
};

export interface Competency {
  id: string;
  code: string;
  kind: CompetencyKind;
  name: string;
  description?: string | null;
  why_important?: string | null;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Requirement {
  id: string;
  department_id: string;
  competency_id: string;
  competency_code: string;
  competency_name: string;
  competency_kind: CompetencyKind;
  grade_id: string;
  grade_level: number;
  grade_name: string;
  required_min: number | null;
  is_key: boolean;
  description?: string | null;
}

export type AssessorRole = 'HEAD' | 'DEPT_HEAD' | 'HRA' | 'DCR_HEAD';

export const AssessorRoleLabel: Record<AssessorRole, string> = {
  HEAD:      'Рук. отдела',
  DEPT_HEAD: 'Рук. дпт',
  HRA:       'HR-аналитик',
  DCR_HEAD:  'Рук. ДЧР',
};

export interface AssessmentScore {
  id: string;
  period_id: string;
  employee_id: string;
  competency_id: string;
  assessor_role: AssessorRole;
  score: number | null;
  feedback?: string | null;
  assessed_by?: string | null;
  assessed_at?: string | null;
  updated_at: string;
}

export interface AssessmentPeriod {
  id: string;
  title: string;
  department_id?: string | null;
  period_start: string;
  period_end: string;
  is_active: boolean;
  created_by?: string | null;
  created_at: string;
  updated_at: string;
}

export type UserRole =
  | 'HR_ADMIN'
  | 'DEPT_HEAD'
  | 'SECTION_HEAD'
  | 'ASSESSOR'
  | 'PRECEPTOR'
  | 'ATS'
  | 'BOOK_SPACE';

export const UserRoleLabel: Record<UserRole, string> = {
  HR_ADMIN:     'HR-администратор',
  DEPT_HEAD:    'Руководитель департамента',
  SECTION_HEAD: 'Руководитель отдела',
  ASSESSOR:     'Ассессор',
  PRECEPTOR:    'Наставник',
  ATS:          'Центр обучения (AtS)',
  BOOK_SPACE:   'Библиотека материалов',
};

export interface RoleAssignment {
  id: string;
  user_id: string;
  role: UserRole;
  scope_department_id?: string | null;
  scope_department?: string | null;
  scope_section_id?: string | null;
  scope_section?: string | null;
  granted_by_id?: string | null;
  granted_by_name?: string | null;
  granted_at: string;
}

export type ParticipantRole = 'HEAD' | 'DEPT_HEAD' | 'HRA' | 'DCR_HEAD' | 'ASSESSOR';

export const ParticipantRoleLabel: Record<ParticipantRole, string> = {
  HEAD:      'Рук. отдела',
  DEPT_HEAD: 'Рук. дпт',
  HRA:       'HR-аналитик',
  DCR_HEAD:  'Рук. ДЧР',
  ASSESSOR:  'Ассессор',
};

export interface AssessmentParticipant {
  id: string;
  period_id: string;
  user_id: string;
  role: ParticipantRole;
  user_name?: string;
  full_name?: string;
  added_at: string;
}

export interface MyAssessmentPeriod {
  period_id: string;
  title: string;
  period_start: string;
  period_end: string;
  is_active: boolean;
  roles: ParticipantRole[];
  department?: string | null;
  department_id?: string | null;
}

export interface ConsolidatedScore {
  id: string;
  period_id: string;
  employee_id: string;
  competency_id: string;
  avg_score: number;
  finalized_at: string;
}

export interface PeriodWithScores {
  period: AssessmentPeriod;
  scores: AssessmentScore[];
}
