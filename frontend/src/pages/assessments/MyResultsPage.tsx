import { Card, Empty, Progress, Space, Table, Tag, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import { myAssessmentResults } from '../../api/competency';
import type { EmployeeResult } from '../../types';

const { Text } = Typography;

export function MyResultsPage() {
  const { data: results = [], isLoading } = useQuery({
    queryKey: ['my-assessment-results'],
    queryFn: myAssessmentResults,
  });

  if (isLoading) return <PageSkeleton type="list" />;

  // Group by campaign.
  const byPeriod = new Map<string, EmployeeResult[]>();
  for (const r of results) {
    if (!byPeriod.has(r.period_id)) byPeriod.set(r.period_id, []);
    byPeriod.get(r.period_id)!.push(r);
  }

  return (
    <>
      <PageHeader title="Мои результаты ассессмента" subtitle="Опубликованный профиль компетенций" />
      {byPeriod.size === 0 ? (
        <Card><Empty description="Опубликованных результатов пока нет" /></Card>
      ) : (
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          {[...byPeriod.entries()].map(([periodId, rows]) => (
            <Card
              key={periodId}
              title={rows[0].period_title}
              extra={rows[0].published_at && (
                <Text type="secondary">Опубликовано: {dayjs(rows[0].published_at).format('DD.MM.YYYY')}</Text>
              )}
            >
              <Table
                rowKey="competency_id" size="small" pagination={false} dataSource={rows}
                columns={[
                  { title: 'Компетенция', dataIndex: 'competency_name', key: 'name' },
                  {
                    title: 'Итоговый балл', dataIndex: 'avg_score', key: 'avg', width: 280,
                    render: (a: number) => (
                      <Space>
                        <Progress percent={Math.round(a * 10)} steps={10} size="small" showInfo={false} />
                        <Tag color={a >= 7 ? 'green' : a >= 4 ? 'gold' : 'red'}>{a.toFixed(2)}</Tag>
                      </Space>
                    ),
                  },
                ]}
              />
            </Card>
          ))}
        </Space>
      )}
    </>
  );
}
