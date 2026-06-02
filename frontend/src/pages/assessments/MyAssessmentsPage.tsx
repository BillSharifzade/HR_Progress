import { Card, Empty, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import { listMyAssessmentPeriods } from '../../api/competency';
import type { MyAssessmentPeriod, ParticipantRole } from '../../types';
import { ParticipantRoleLabel } from '../../types';

const { Text } = Typography;

const ROLE_COLOR: Record<ParticipantRole, string> = {
  HEAD:      'geekblue',
  DEPT_HEAD: 'purple',
  HRA:       'cyan',
  DCR_HEAD:  'orange',
  ASSESSOR:  'green',
};

export function MyAssessmentsPage() {
  const navigate = useNavigate();
  const { data: periods = [], isLoading } = useQuery({
    queryKey: ['my-assessment-periods'],
    queryFn: listMyAssessmentPeriods,
  });

  if (isLoading) return <PageSkeleton type="list" />;

  const columns: ColumnsType<MyAssessmentPeriod> = [
    {
      title: 'Период',
      dataIndex: 'title',
      render: (title: string, row) => (
        <div>
          <Text strong>{title}</Text>
          {row.department && (
            <Text type="secondary" style={{ fontSize: 11, marginLeft: 8 }}>
              {row.department}
            </Text>
          )}
        </div>
      ),
    },
    {
      title: 'Даты',
      width: 220,
      render: (_, row) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {dayjs(row.period_start).format('DD.MM.YYYY')}
          {' — '}
          {dayjs(row.period_end).format('DD.MM.YYYY')}
        </Text>
      ),
    },
    {
      title: 'Моя роль',
      width: 240,
      render: (_, row) => (
        <Space size={4} wrap>
          {row.roles.map(role => (
            <Tag key={role} color={ROLE_COLOR[role]}>{ParticipantRoleLabel[role]}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: 'Статус',
      width: 110,
      render: (_, row) => (
        <Tag color={row.is_active ? 'green' : 'default'}>
          {row.is_active ? 'Активен' : 'Завершён'}
        </Tag>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title="Мои оценки"
        subtitle="Периоды, в которых вы участвуете как оценщик"
      />
      <Card size="small">
        {periods.length === 0 ? (
          <Empty description="Вы пока не назначены ни в один период" />
        ) : (
          <Table
            dataSource={periods}
            columns={columns}
            rowKey="period_id"
            size="small"
            pagination={false}
            onRow={(row) => ({
              style: { cursor: 'pointer' },
              onClick: () => navigate(`/assessments/${row.period_id}`),
            })}
          />
        )}
      </Card>
    </>
  );
}
