import { Card, Skeleton, Space } from 'antd';

interface PageSkeletonProps {
  type?: 'list' | 'profile' | 'form';
}

export function PageSkeleton({ type = 'list' }: PageSkeletonProps) {
  if (type === 'profile') {
    return (
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Skeleton active paragraph={{ rows: 1 }} style={{ maxWidth: 300 }} />
        <Card>
          <Skeleton avatar active paragraph={{ rows: 2 }} />
        </Card>
        <Card>
          <Skeleton active paragraph={{ rows: 5 }} />
        </Card>
      </Space>
    );
  }

  if (type === 'form') {
    return (
      <Card>
        <Skeleton active paragraph={{ rows: 8 }} />
      </Card>
    );
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Skeleton active paragraph={{ rows: 1 }} style={{ maxWidth: 240 }} />
      <Card bodyStyle={{ padding: 0 }}>
        <Skeleton active paragraph={{ rows: 6 }} style={{ padding: '16px 24px' }} />
      </Card>
    </Space>
  );
}
