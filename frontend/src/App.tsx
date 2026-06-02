import { lazy, Suspense } from 'react';
import { App as AntdApp } from 'antd';
import { Routes, Route, Navigate } from 'react-router-dom';

import { LoginPage } from './pages/login/LoginPage';
import { ProtectedRoute } from './auth/ProtectedRoute';
import { AppShell } from './components/AppShell';
import { PageSkeleton } from './components/PageSkeleton';
import { useAuth } from './auth/useAuth';

const DashboardPage = lazy(() => import('./pages/dashboard/DashboardPage').then(m => ({ default: m.DashboardPage })));
const CompetencyMatrixPage = lazy(() => import('./pages/competency/CompetencyMatrixPage').then(m => ({ default: m.CompetencyMatrixPage })));
const WorkersList = lazy(() => import('./pages/workers/WorkersList').then(m => ({ default: m.WorkersList })));
const WorkerProfile = lazy(() => import('./pages/workers/WorkerProfile').then(m => ({ default: m.WorkerProfile })));
const AdminPage = lazy(() => import('./pages/admin/AdminPage').then(m => ({ default: m.AdminPage })));
const MyAssessmentsPage = lazy(() => import('./pages/assessments/MyAssessmentsPage').then(m => ({ default: m.MyAssessmentsPage })));
const MyPeriodScoringPage = lazy(() => import('./pages/assessments/MyPeriodScoringPage').then(m => ({ default: m.MyPeriodScoringPage })));

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <AppShell>
        <Suspense fallback={<PageSkeleton type="list" />}>{children}</Suspense>
      </AppShell>
    </ProtectedRoute>
  );
}

function AdminOnly({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  if (!user?.roles.includes('HR_ADMIN')) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <AntdApp>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<Shell><DashboardPage /></Shell>} />
        <Route path="/workers" element={<Shell><WorkersList /></Shell>} />
        <Route path="/workers/:id" element={<Shell><WorkerProfile /></Shell>} />
        <Route path="/competencies" element={<Shell><CompetencyMatrixPage /></Shell>} />
        <Route path="/assessments" element={<Shell><MyAssessmentsPage /></Shell>} />
        <Route path="/assessments/:periodId" element={<Shell><MyPeriodScoringPage /></Shell>} />
        <Route path="/admin" element={<Shell><AdminOnly><AdminPage /></AdminOnly></Shell>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AntdApp>
  );
}
