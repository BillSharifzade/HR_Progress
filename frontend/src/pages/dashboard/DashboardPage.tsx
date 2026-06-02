import { Card, Col, Row, Statistic, Typography, Tag, Space } from 'antd';
import { useAuth } from '../../auth/useAuth';
import { PageHeader } from '../../components/PageHeader';
import { UserRoleLabel, type UserRole } from '../../types';

export function DashboardPage() {
  const { user } = useAuth();
  if (!user) return null;
  return (
    <>
      <PageHeader
        title={`Здравствуйте, ${user.full_name}`}
        subtitle={<>@{user.username}</>}
      />

      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={8}>
            <Card>
              <Statistic title="Сотрудников" value="—" />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8}>
            <Card>
              <Statistic title="Активных оценок" value="—" />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8}>
            <Card>
              <Statistic title="Открытых ПИР" value="—" />
            </Card>
          </Col>
        </Row>

        <Card title="Ваш профиль">
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <div>
              <Typography.Text type="secondary">Роли:</Typography.Text>{' '}
              {user.roles.length === 0 ? (
                <Typography.Text>—</Typography.Text>
              ) : (
                user.roles.map((r) => (
                  <Tag color="blue" key={r}>{UserRoleLabel[r as UserRole] ?? r}</Tag>
                ))
              )}
            </div>
            <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
              Это первая фаза платформы. Дальнейшие разделы — Сотрудники, Компетенции, Матрица Компетенций, Оценки —
              будут добавлены в следующих этапах.
            </Typography.Paragraph>
          </Space>
        </Card>
      </Space>
    </>
  );
}
