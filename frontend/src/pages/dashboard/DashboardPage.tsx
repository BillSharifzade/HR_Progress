import { Alert, Card, Space, Tag, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';

import { useAuth } from '../../auth/useAuth';
import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import { UserRoleLabel, type UserRole } from '../../types';
import { getDashboardOverview } from '../../api/dashboard';
import { DashboardAnalytics } from './DashboardAnalytics';

const { Text, Paragraph } = Typography;

export function DashboardPage() {
  const { user } = useAuth();

  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard-overview'],
    queryFn: getDashboardOverview,
    retry: false,
  });

  // Analytics are leadership-scoped; everyone else just gets their profile.
  const forbidden = (error as any)?.response?.status === 403;

  if (!user) return null;

  const profileCard = (
    <Card title="Ваш профиль" size="small">
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <div>
          <Text type="secondary">Роли:</Text>{' '}
          {user.roles.length === 0
            ? <Text>—</Text>
            : user.roles.map(r => (
                <Tag color="blue" key={r}>{UserRoleLabel[r as UserRole] ?? r}</Tag>
              ))}
        </div>
        {forbidden && (
          <Paragraph type="secondary" style={{ margin: 0 }}>
            Сводная аналитика доступна руководителям и сотрудникам ДЧР.
            Свои результаты вы найдёте в разделе «Мои результаты».
          </Paragraph>
        )}
      </Space>
    </Card>
  );

  const header = (
    <PageHeader title={`Здравствуйте, ${user.full_name}`} subtitle={<>@{user.username}</>} />
  );

  if (forbidden) return <>{header}{profileCard}</>;
  if (isLoading) return <>{header}<PageSkeleton type="list" /></>;
  if (error || !data) {
    return (
      <>
        {header}
        <Alert type="error" showIcon message="Не удалось загрузить аналитику" style={{ marginBottom: 16 }} />
        {profileCard}
      </>
    );
  }

  return (
    <>
      {header}
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <DashboardAnalytics data={data} />
        {profileCard}
      </Space>
    </>
  );
}
