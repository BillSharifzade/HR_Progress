import { useState } from 'react';
import { Alert, Button, Drawer, Space, Table, Tag, Tooltip, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { CloudSyncOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';

import {
  getOneFStatus,
  listOneFRuns,
  triggerOneFSync,
  type OneFSyncRun,
} from '../../api/onef';

const { Text } = Typography;

const runColumns: ColumnsType<OneFSyncRun> = [
  {
    title: 'Время',
    dataIndex: 'started_at',
    width: 140,
    render: (v: string) => dayjs(v).format('DD.MM.YYYY HH:mm'),
  },
  {
    title: 'Триггер',
    dataIndex: 'trigger_kind',
    width: 110,
    render: (v: string) => (v === 'manual' ? 'Вручную' : 'По расписанию'),
  },
  {
    title: 'Статус',
    dataIndex: 'status',
    width: 110,
    render: (v: string) => {
      if (v === 'success') return <Tag color="green">Успешно</Tag>;
      if (v === 'failed') return <Tag color="red">Ошибка</Tag>;
      return <Tag color="blue">В процессе</Tag>;
    },
  },
  { title: 'Получено',  dataIndex: 'fetched_count', width: 90 },
  { title: 'Создано',   dataIndex: 'created_count', width: 90 },
  { title: 'Обновлено', dataIndex: 'updated_count', width: 100 },
  { title: 'Пропущено', dataIndex: 'skipped_count', width: 100 },
  {
    title: 'Длительность',
    dataIndex: 'duration_ms',
    width: 120,
    render: (v?: number | null) => (v == null ? '—' : `${v} мс`),
  },
  {
    title: 'Ошибка',
    dataIndex: 'error_message',
    render: (v?: string | null) =>
      v ? (
        <Tooltip title={v}>
          <Text type="danger" style={{ fontSize: 12 }}>
            {v.length > 60 ? v.slice(0, 60) + '…' : v}
          </Text>
        </Tooltip>
      ) : (
        <Text type="secondary">—</Text>
      ),
  },
];

export function OneFSyncButton() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [messageApi, contextHolder] = message.useMessage();

  const { data: status } = useQuery({
    queryKey: ['onef', 'status'],
    queryFn: getOneFStatus,
  });

  const { data: runs = [] } = useQuery<OneFSyncRun[]>({
    queryKey: ['onef', 'runs'],
    queryFn: () => listOneFRuns(20),
    enabled: open,
    refetchInterval: open ? 5_000 : false,
  });

  const syncMut = useMutation({
    mutationFn: triggerOneFSync,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['onef', 'runs'] });
      qc.invalidateQueries({ queryKey: ['workers'] });
      messageApi.success(
        `1F: получено ${res.fetched_count}, создано ${res.created_count}, обновлено ${res.updated_count}, пропущено ${res.skipped_count}`,
      );
    },
    onError: (err: { response?: { data?: { error?: { message?: string } } } }) => {
      messageApi.error(err?.response?.data?.error?.message ?? 'Не удалось синхронизироваться с 1F');
    },
  });

  const lastRun = runs[0];

  return (
    <>
      {contextHolder}
      <Button icon={<CloudSyncOutlined />} onClick={() => setOpen(true)}>
        Синхронизация с 1F
        {status && !status.configured && (
          <Tag color="orange" style={{ marginLeft: 8 }}>Не настроено</Tag>
        )}
      </Button>

      <Drawer
        title={
          <Space>
            <CloudSyncOutlined />
            <span>Синхронизация с 1F (Первая форма)</span>
          </Space>
        }
        placement="right"
        width={Math.min(window.innerWidth - 64, 1100)}
        open={open}
        onClose={() => setOpen(false)}
        extra={
          <Button
            type="primary"
            icon={<CloudSyncOutlined />}
            loading={syncMut.isPending}
            disabled={!status?.configured}
            onClick={() => syncMut.mutate()}
          >
            Синхронизировать сейчас
          </Button>
        }
      >
        {!status?.configured && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            message="Адрес 1F не задан"
            description="Установите переменную окружения ONEF_BASE_URL в конфигурации сервера, чтобы включить автоматическую ежедневную синхронизацию."
          />
        )}

        {lastRun && (
          <div style={{ marginBottom: 16, fontSize: 13 }}>
            <Text type="secondary">Последний запуск: </Text>
            <Text>{dayjs(lastRun.started_at).format('DD.MM.YYYY HH:mm')}</Text>
            {' · '}
            {lastRun.status === 'success' && <Tag color="green">Успешно</Tag>}
            {lastRun.status === 'failed' && <Tag color="red">Ошибка</Tag>}
            {lastRun.status === 'running' && <Tag color="blue">В процессе</Tag>}
            {' · '}
            <Text type="secondary">
              получено {lastRun.fetched_count}, создано {lastRun.created_count}, обновлено{' '}
              {lastRun.updated_count}, пропущено {lastRun.skipped_count}
            </Text>
          </div>
        )}

        <Table
          dataSource={runs}
          columns={runColumns}
          rowKey="id"
          size="small"
          pagination={false}
          locale={{ emptyText: 'Запусков ещё не было' }}
          scroll={{ x: true }}
        />
      </Drawer>
    </>
  );
}
