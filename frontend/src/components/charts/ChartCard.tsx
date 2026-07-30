import { useState, type ReactNode } from 'react';
import { Card, Empty, Segmented, Space, Table, Tooltip, Typography } from 'antd';
import { BarChartOutlined, TableOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

interface Props<T extends object> {
  title: string;
  subtitle?: ReactNode;
  /** The chart itself. */
  children: ReactNode;
  /**
   * Every chart ships a table twin — it is the WCAG-clean way to read the same
   * numbers when color or hover is not available to the reader.
   */
  tableColumns: ColumnsType<T>;
  tableData: T[];
  rowKey: keyof T & string;
  emptyText?: string;
  extra?: ReactNode;
}

export function ChartCard<T extends object>({
  title, subtitle, children, tableColumns, tableData, rowKey, emptyText, extra,
}: Props<T>) {
  const [view, setView] = useState<'chart' | 'table'>('chart');
  const isEmpty = tableData.length === 0;

  return (
    <Card
      size="small"
      style={{ height: '100%' }}
      styles={{ body: { paddingTop: 8 } }}
      title={
        <Space direction="vertical" size={0} style={{ padding: '4px 0' }}>
          <Text strong>{title}</Text>
          {subtitle && <Text type="secondary" style={{ fontSize: 12, fontWeight: 400 }}>{subtitle}</Text>}
        </Space>
      }
      extra={
        <Space size={8}>
          {extra}
          {!isEmpty && (
            <Segmented
              size="small"
              value={view}
              onChange={v => setView(v as 'chart' | 'table')}
              options={[
                { value: 'chart', icon: <Tooltip title="График"><BarChartOutlined /></Tooltip> },
                { value: 'table', icon: <Tooltip title="Таблица"><TableOutlined /></Tooltip> },
              ]}
            />
          )}
        </Space>
      }
    >
      {isEmpty ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={emptyText ?? 'Нет данных'}
          style={{ margin: '32px 0' }}
        />
      ) : view === 'chart' ? (
        children
      ) : (
        <Table
          size="small"
          rowKey={rowKey}
          columns={tableColumns}
          dataSource={tableData}
          pagination={false}
          scroll={{ y: 260 }}
        />
      )}
    </Card>
  );
}
